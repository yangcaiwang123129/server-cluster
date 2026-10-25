// Package store 玩家数据与物品存储层。
// 高度封装：业务层只依赖 PlayerStore 与 BattleLootStore 接口，
// 不直接碰 Redis、WAL、MySQL、theine。
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"server-cluster/share/model"
	"sort"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"server-cluster/internal/store/flush"
	"server-cluster/internal/store/localcache"
	"server-cluster/internal/store/mysql"
	"server-cluster/internal/store/rediscache"
	"server-cluster/internal/store/version"
	"server-cluster/internal/store/wal"
	"server-cluster/internal/svc"

	"github.com/dobyte/due/v2/log"
)

// PlayerStore 大厅与战斗结算共用。
type PlayerStore interface {
	GetCore(ctx context.Context, playerID int64) (*model.PlayerCore, error)
	UpdateCore(ctx context.Context, playerID int64, patch model.CorePatch) (*model.PlayerCore, error)

	GetAssets(ctx context.Context, playerID int64) ([]model.PlayerAsset, error)
	UpdateAssets(ctx context.Context, playerID int64, changes []model.AssetChange) error

	GetProfile(ctx context.Context, playerID int64) (*model.PlayerProfile, error)
	UpdateProfile(ctx context.Context, playerID int64, patch model.ProfilePatch) error

	DeletePlayer(ctx context.Context, playerID int64) error

	Recover(ctx context.Context) error
	Close(ctx context.Context) error
}

// BattleLootStore 战斗服专用：拾取临时资产。实现位于 internal/battleloot。
type BattleLootStore interface {
	// Pickup 对局内拾取，写房间内存 + 本地 WAL（非严格）。
	Pickup(ctx context.Context, matchID int64, playerID int64, item model.LootItem) error

	// ListPicked 查询某玩家本局已拾取。
	ListPicked(ctx context.Context, matchID int64, playerID int64) ([]model.LootItem, error)

	// Settle 对局结算：汇总并通过 PlayerStore 持久化。
	Settle(ctx context.Context, matchID int64, playerID int64) error

	// Abort 对局异常结束：按规则丢弃或部分补偿。
	Abort(ctx context.Context, matchID int64) error

	// RecoverMatch 崩溃后回放 loot WAL，恢复未结算拾取。
	RecoverMatch(ctx context.Context, matchID int64) error
}

// playerStore PlayerStore 实现：组合 L1 / Redis / WAL / flushWorker / repo / versionGen。
type playerStore struct {
	cfg    *Config
	repo   *mysql.PlayerRepo
	walW   *wal.Writer
	rc     *rediscache.Cache // nil 表示 Redis 关闭（Node 形态）
	vg     version.Generator
	fw     *flush.Worker
	walID  string
	cancel context.CancelFunc

	coreL1    localcache.Cache[*model.PlayerCore]
	assetsL1  localcache.Cache[[]model.PlayerAsset]
	profileL1 localcache.Cache[*model.PlayerProfile]
}

// New 组装存储层（不执行恢复；调用方须随后调用 Recover 再对外服务）。
// cfg 为 nil 时返回错误；配置加载见 Load（etc.store 段缺席时应跳过初始化）。
// MySQL 分片与 Redis 连接均从 svc 的命名实例管理器获取（本包不创建/关闭连接）。
func New(ctx context.Context, cfg *Config) (PlayerStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("store: config is nil")
	}
	if len(cfg.MySQL.Shards) == 0 {
		return nil, fmt.Errorf("store: at least one mysql shard instance is required")
	}
	shards := make([]*sql.DB, 0, len(cfg.MySQL.Shards))
	for _, name := range cfg.MySQL.Shards {
		inst, err := svc.Ctx().MySQL.Get(name)
		if err != nil {
			return nil, fmt.Errorf("store: mysql shard instance %q: %w", name, err)
		}
		shards = append(shards, inst.SQL())
	}
	s := &playerStore{cfg: cfg, walID: cfg.WALID}

	s.repo = mysql.NewPlayerRepo(shards, mysql.NewRouter(len(shards)))

	syncMode := wal.SyncStrict
	if cfg.WALSyncMode == "batch" {
		syncMode = wal.SyncBatch
	}
	var err error
	if s.walW, err = wal.Open(cfg.WALDir, cfg.WALID, syncMode, cfg.WALBatchSyncInterval); err != nil {
		return nil, err
	}

	var rdb redis.UniversalClient
	if cfg.Redis.Enabled {
		client, err := svc.Ctx().Redis.Get(cfg.Redis.Instance)
		if err != nil {
			_ = s.walW.Close()
			return nil, fmt.Errorf("store: redis instance %q: %w", cfg.Redis.Instance, err)
		}
		s.rc = rediscache.New(client, rediscache.Config{
			KeyPrefix:         cfg.Redis.KeyPrefix,
			TTL:               cfg.Redis.TTL,
			TTLJitter:         cfg.Redis.TTLJitter,
			InvalidateChannel: cfg.Redis.InvalidateChannel,
		})
		rdb = client
	}
	s.vg = version.New(cfg.Version.Driver, rdb, cfg.Redis.KeyPrefix, s.repo)

	l1cfg := localcache.Config{Mode: cfg.Local.Mode, MaxSize: cfg.Local.MaxSize, TTL: cfg.Local.TTL}
	if s.coreL1, err = localcache.New[*model.PlayerCore](l1cfg, s.loadCore); err != nil {
		s.closeParts()
		return nil, err
	}
	if s.assetsL1, err = localcache.New[[]model.PlayerAsset](l1cfg, s.loadAssets); err != nil {
		s.closeParts()
		return nil, err
	}
	if s.profileL1, err = localcache.New[*model.PlayerProfile](l1cfg, s.loadProfile); err != nil {
		s.closeParts()
		return nil, err
	}

	s.fw = flush.New(s, s, cfg.MySQL.FlushBatchSize, cfg.MySQL.FlushInterval)
	s.fw.Start()

	if s.rc != nil {
		var subCtx context.Context
		subCtx, s.cancel = context.WithCancel(context.Background())
		s.rc.SubscribeInvalidate(subCtx, s.onInvalidate)
	}
	return s, nil
}

func l1Key(playerID int64) string { return strconv.FormatInt(playerID, 10) }

func parseL1Key(key string) (int64, error) {
	return strconv.ParseInt(key, 10, 64)
}

// ---------------------------------------------------------------------------
// 读路径：L1 → loader（Redis → MySQL → 回填）
// ---------------------------------------------------------------------------

// GetCore 读核心数据。
func (s *playerStore) GetCore(ctx context.Context, playerID int64) (*model.PlayerCore, error) {
	return s.coreL1.Get(ctx, l1Key(playerID))
}

// GetAssets 读资产列表；玩家无任何资产时返回空切片。
func (s *playerStore) GetAssets(ctx context.Context, playerID int64) ([]model.PlayerAsset, error) {
	return s.assetsL1.Get(ctx, l1Key(playerID))
}

// GetProfile 读档案。
func (s *playerStore) GetProfile(ctx context.Context, playerID int64) (*model.PlayerProfile, error) {
	return s.profileL1.Get(ctx, l1Key(playerID))
}

func (s *playerStore) loadCore(ctx context.Context, key string) (*model.PlayerCore, error) {
	playerID, err := parseL1Key(key)
	if err != nil {
		return nil, err
	}
	if s.rc != nil {
		if h, err := s.rc.GetHash(ctx, playerID, rediscache.KindCore); err == nil {
			if h.Null {
				return nil, ErrNotFound
			}
			if h.Found {
				if c, err := coreFromHash(playerID, h.Fields); err == nil {
					return c, nil
				}
			}
		}
	}
	row, err := s.repo.GetCore(ctx, playerID)
	if errors.Is(err, mysql.ErrNotFound) {
		s.setNullBestEffort(ctx, playerID, rediscache.KindCore)
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c := coreFromRow(row)
	s.setRedisBestEffort(ctx, playerID, rediscache.KindCore, coreToHash(c))
	return c, nil
}

func (s *playerStore) loadAssets(ctx context.Context, key string) ([]model.PlayerAsset, error) {
	playerID, err := parseL1Key(key)
	if err != nil {
		return nil, err
	}
	if s.rc != nil {
		if h, err := s.rc.GetHash(ctx, playerID, rediscache.KindAssets); err == nil && h.Found {
			if assets, err := assetsFromHash(playerID, h.Fields); err == nil {
				return assets, nil
			}
		}
	}
	rows, err := s.repo.GetAssets(ctx, playerID)
	if err != nil {
		return nil, err
	}
	assets := make([]model.PlayerAsset, 0, len(rows))
	for _, r := range rows {
		assets = append(assets, *assetFromRow(r))
	}
	s.setRedisBestEffort(ctx, playerID, rediscache.KindAssets, assetsToHash(assets))
	return assets, nil
}

func (s *playerStore) loadProfile(ctx context.Context, key string) (*model.PlayerProfile, error) {
	playerID, err := parseL1Key(key)
	if err != nil {
		return nil, err
	}
	if s.rc != nil {
		if h, err := s.rc.GetHash(ctx, playerID, rediscache.KindProfile); err == nil {
			if h.Null {
				return nil, ErrNotFound
			}
			if h.Found {
				if p, err := profileFromHash(playerID, h.Fields); err == nil {
					return p, nil
				}
			}
		}
	}
	row, err := s.repo.GetProfile(ctx, playerID)
	if errors.Is(err, mysql.ErrNotFound) {
		s.setNullBestEffort(ctx, playerID, rediscache.KindProfile)
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p := profileFromRow(row)
	s.setRedisBestEffort(ctx, playerID, rediscache.KindProfile, profileToHash(p))
	return p, nil
}

// ---------------------------------------------------------------------------
// 写路径：version → WAL Write（失败即整体失败）→ L1 → 尽力 Redis → 失效广播 → flush.Notify
// ---------------------------------------------------------------------------

// UpdateCore 补丁合并后写入；玩家不存在时从零值创建。
func (s *playerStore) UpdateCore(ctx context.Context, playerID int64, patch model.CorePatch) (*model.PlayerCore, error) {
	ver, err := s.vg.Next(ctx, playerID)
	if err != nil {
		return nil, err
	}
	cur, err := s.GetCore(ctx, playerID)
	if errors.Is(err, ErrNotFound) {
		cur = &model.PlayerCore{PlayerID: playerID}
	} else if err != nil {
		return nil, err
	}
	merged := *cur
	if patch.Nickname != nil {
		merged.Nickname = *patch.Nickname
	}
	if patch.Level != nil {
		merged.Level = *patch.Level
	}
	if patch.Exp != nil {
		merged.Exp = *patch.Exp
	}
	if patch.Gold != nil {
		merged.Gold = *patch.Gold
	}
	merged.PlayerID = playerID
	merged.Version = ver

	entry, err := s.newEntry(wal.OpSetCore, playerID, ver, &merged)
	if err != nil {
		return nil, err
	}
	if err := s.walW.Write(entry); err != nil {
		return nil, err // WAL 失败 = 整个写失败
	}
	s.coreL1.Set(l1Key(playerID), &merged)
	s.setRedisBestEffort(ctx, playerID, rediscache.KindCore, coreToHash(&merged))
	s.publishBestEffort(ctx, playerID, rediscache.KindCore)
	s.fw.Notify(entry)
	return &merged, nil
}

// UpdateAssets 增量合并（正增负减）；扣减后小于 0 返回 ErrInsufficientAsset。
func (s *playerStore) UpdateAssets(ctx context.Context, playerID int64, changes []model.AssetChange) error {
	if len(changes) == 0 {
		return nil
	}
	ver, err := s.vg.Next(ctx, playerID)
	if err != nil {
		return err
	}
	cur, err := s.GetAssets(ctx, playerID)
	if errors.Is(err, ErrNotFound) {
		cur = nil
	} else if err != nil {
		return err
	}
	mergedMap := make(map[int32]model.PlayerAsset, len(cur))
	for _, a := range cur {
		mergedMap[a.AssetType] = a
	}
	changed := make([]model.PlayerAsset, 0, len(changes))
	for _, ch := range changes {
		a := mergedMap[ch.AssetType]
		a.PlayerID = playerID
		a.AssetType = ch.AssetType
		a.Amount += ch.Amount
		if a.Amount < 0 {
			return ErrInsufficientAsset
		}
		a.Version = ver
		mergedMap[ch.AssetType] = a
		changed = append(changed, a)
	}
	merged := make([]model.PlayerAsset, 0, len(mergedMap))
	for _, a := range mergedMap {
		merged = append(merged, a)
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].AssetType < merged[j].AssetType })

	entry, err := s.newEntry(wal.OpSetAssets, playerID, ver, changed)
	if err != nil {
		return err
	}
	if err := s.walW.Write(entry); err != nil {
		return err
	}
	s.assetsL1.Set(l1Key(playerID), merged)
	s.setRedisBestEffort(ctx, playerID, rediscache.KindAssets, assetsToHash(merged))
	s.publishBestEffort(ctx, playerID, rediscache.KindAssets)
	s.fw.Notify(entry)
	return nil
}

// UpdateProfile 补丁合并后写入；玩家不存在时从零值创建。
func (s *playerStore) UpdateProfile(ctx context.Context, playerID int64, patch model.ProfilePatch) error {
	ver, err := s.vg.Next(ctx, playerID)
	if err != nil {
		return err
	}
	cur, err := s.GetProfile(ctx, playerID)
	if errors.Is(err, ErrNotFound) {
		cur = &model.PlayerProfile{PlayerID: playerID}
	} else if err != nil {
		return err
	}
	merged := *cur
	if patch.Signature != nil {
		merged.Signature = *patch.Signature
	}
	if patch.AvatarID != nil {
		merged.AvatarID = *patch.AvatarID
	}
	if patch.SettingsJSON != nil {
		merged.SettingsJSON = *patch.SettingsJSON
	}
	merged.PlayerID = playerID
	merged.Version = ver

	entry, err := s.newEntry(wal.OpSetProfile, playerID, ver, &merged)
	if err != nil {
		return err
	}
	if err := s.walW.Write(entry); err != nil {
		return err
	}
	s.profileL1.Set(l1Key(playerID), &merged)
	s.setRedisBestEffort(ctx, playerID, rediscache.KindProfile, profileToHash(&merged))
	s.publishBestEffort(ctx, playerID, rediscache.KindProfile)
	s.fw.Notify(entry)
	return nil
}

// DeletePlayer 删除玩家三张表数据并失效全部缓存。
func (s *playerStore) DeletePlayer(ctx context.Context, playerID int64) error {
	entry, err := s.newEntry(wal.OpDelete, playerID, 0, nil)
	if err != nil {
		return err
	}
	if err := s.walW.Write(entry); err != nil {
		return err
	}
	key := l1Key(playerID)
	s.coreL1.Delete(key)
	s.assetsL1.Delete(key)
	s.profileL1.Delete(key)
	if s.rc != nil {
		if err := s.rc.Delete(ctx, playerID,
			rediscache.KindCore, rediscache.KindAssets, rediscache.KindProfile); err != nil {
			log.Warnf("store: redis delete player=%d failed: %v", playerID, err)
		}
	}
	s.publishBestEffort(ctx, playerID,
		rediscache.KindCore, rediscache.KindAssets, rediscache.KindProfile)
	s.fw.Notify(entry)
	return nil
}

// ---------------------------------------------------------------------------
// flush.Sink / flush.Committer 实现
// ---------------------------------------------------------------------------

// Flush 批量落盘：解码 payload → repo 分组写入 → 推进 committed_seq。
// 任一失败返回错误，worker 将整批回队重试（version 乐观条件保证幂等）。
func (s *playerStore) Flush(ctx context.Context, entries []*wal.Entry) error {
	if len(entries) == 0 {
		return nil
	}
	var coreRows []*mysql.CoreRow
	var assetRows []*mysql.AssetRow
	var profileRows []*mysql.ProfileRow
	var deleteIDs []int64
	for _, e := range entries {
		switch e.Op {
		case wal.OpSetCore:
			var c model.PlayerCore
			if err := json.Unmarshal(e.Payload, &c); err != nil {
				return fmt.Errorf("store: decode core entry seq=%d failed: %w", e.Seq, err)
			}
			coreRows = append(coreRows, coreToRow(&c))
		case wal.OpSetAssets:
			var assets []model.PlayerAsset
			if err := json.Unmarshal(e.Payload, &assets); err != nil {
				return fmt.Errorf("store: decode assets entry seq=%d failed: %w", e.Seq, err)
			}
			for i := range assets {
				assetRows = append(assetRows, assetToRow(&assets[i]))
			}
		case wal.OpSetProfile:
			var p model.PlayerProfile
			if err := json.Unmarshal(e.Payload, &p); err != nil {
				return fmt.Errorf("store: decode profile entry seq=%d failed: %w", e.Seq, err)
			}
			profileRows = append(profileRows, profileToRow(&p))
		case wal.OpDelete:
			deleteIDs = append(deleteIDs, e.PlayerID)
		default:
			log.Warnf("store: unknown wal op=%q seq=%d, skipped", e.Op, e.Seq)
		}
	}
	if err := s.repo.BatchUpsertCore(ctx, coreRows); err != nil {
		return err
	}
	if err := s.repo.BatchUpsertAssets(ctx, assetRows); err != nil {
		return err
	}
	if err := s.repo.BatchUpsertProfile(ctx, profileRows); err != nil {
		return err
	}
	if err := s.repo.DeletePlayer(ctx, deleteIDs); err != nil {
		return err
	}
	return s.repo.SetCommittedSeq(ctx, s.walID, entries[len(entries)-1].Seq)
}

// Commit 已提交水位之前的 WAL 前缀截断。
func (s *playerStore) Commit(_ context.Context, seq uint64) error {
	return s.walW.TruncateFront(seq)
}

// ---------------------------------------------------------------------------
// Recover / Close
// ---------------------------------------------------------------------------

// Recover 阻塞式崩溃恢复：从 committed_seq+1 回放 WAL，分批落 MySQL，
// 尽力失效对应 Redis 键，成功后推进水位并截断前缀。须在对外服务前调用。
func (s *playerStore) Recover(ctx context.Context) error {
	committed, err := s.repo.GetCommittedSeq(ctx, s.walID)
	if err != nil {
		return err
	}
	batchSize := s.cfg.MySQL.FlushBatchSize
	if batchSize <= 0 {
		batchSize = 200
	}
	batch := make([]*wal.Entry, 0, batchSize)
	flushBatch := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := s.Flush(ctx, batch); err != nil {
			return err
		}
		for _, e := range batch {
			s.invalidateRedisBestEffort(ctx, e)
		}
		if err := s.walW.TruncateFront(batch[len(batch)-1].Seq); err != nil {
			return err
		}
		batch = batch[:0]
		return nil
	}
	last, err := s.walW.ReplayFrom(committed+1, func(e *wal.Entry) error {
		batch = append(batch, e)
		if len(batch) >= batchSize {
			return flushBatch()
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("store: recover replay failed: %w", err)
	}
	if err := flushBatch(); err != nil {
		return fmt.Errorf("store: recover flush failed: %w", err)
	}
	if last > committed {
		log.Infof("store: recover done, wal=%s replayed to seq=%d", s.walID, last)
	}
	return nil
}

// Close 刷完剩余 WAL 并释放全部资源。
func (s *playerStore) Close(_ context.Context) error {
	if s.fw != nil {
		s.fw.Close() // 最终 flush + commit + truncate
	}
	if s.cancel != nil {
		s.cancel()
	}
	s.closeParts()
	return nil
}

// closeParts 关闭缓存与 WAL；MySQL / Redis 连接由 svc 的命名实例管理器统一关闭。
func (s *playerStore) closeParts() {
	if s.coreL1 != nil {
		s.coreL1.Close()
	}
	if s.assetsL1 != nil {
		s.assetsL1.Close()
	}
	if s.profileL1 != nil {
		s.profileL1.Close()
	}
	if s.walW != nil {
		_ = s.walW.Close()
	}
}

// ---------------------------------------------------------------------------
// 内部辅助
// ---------------------------------------------------------------------------

// newEntry 构造 WAL 条目（Seq 由 Writer.Write 分配）。
func (s *playerStore) newEntry(op string, playerID int64, ver uint64, payload any) (*wal.Entry, error) {
	e := &wal.Entry{
		Op:        op,
		PlayerID:  playerID,
		Version:   ver,
		CreatedAt: time.Now().UnixMilli(),
	}
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("store: marshal payload failed: %w", err)
		}
		e.Payload = data
	}
	return e, nil
}

// setRedisBestEffort 尽力写 Redis；失败只记日志。
func (s *playerStore) setRedisBestEffort(ctx context.Context, playerID int64, kind string, fields map[string]string) {
	if s.rc == nil || len(fields) == 0 {
		return
	}
	if err := s.rc.SetHash(ctx, playerID, kind, fields); err != nil {
		log.Warnf("store: redis set player=%d kind=%s failed: %v", playerID, kind, err)
	}
}

// setNullBestEffort 尽力写空值标记防穿透。
func (s *playerStore) setNullBestEffort(ctx context.Context, playerID int64, kind string) {
	if s.rc == nil {
		return
	}
	if err := s.rc.SetNull(ctx, playerID, kind); err != nil {
		log.Warnf("store: redis set null player=%d kind=%s failed: %v", playerID, kind, err)
	}
}

// publishBestEffort 尽力发布失效广播。
func (s *playerStore) publishBestEffort(ctx context.Context, playerID int64, kinds ...string) {
	if s.rc == nil {
		return
	}
	if err := s.rc.PublishInvalidate(ctx, &rediscache.Message{PlayerID: playerID, Fields: kinds}); err != nil {
		log.Warnf("store: publish invalidate player=%d failed: %v", playerID, err)
	}
}

// invalidateRedisBestEffort 恢复时尽力失效对应 Redis 键（下次读从 MySQL 回源）。
func (s *playerStore) invalidateRedisBestEffort(ctx context.Context, e *wal.Entry) {
	if s.rc == nil {
		return
	}
	kind := ""
	switch e.Op {
	case wal.OpSetCore:
		kind = rediscache.KindCore
	case wal.OpSetAssets:
		kind = rediscache.KindAssets
	case wal.OpSetProfile:
		kind = rediscache.KindProfile
	case wal.OpDelete:
		s.invalidateRedisBestEffort(ctx, &wal.Entry{Op: wal.OpSetCore, PlayerID: e.PlayerID})
		s.invalidateRedisBestEffort(ctx, &wal.Entry{Op: wal.OpSetAssets, PlayerID: e.PlayerID})
		s.invalidateRedisBestEffort(ctx, &wal.Entry{Op: wal.OpSetProfile, PlayerID: e.PlayerID})
		return
	}
	if kind == "" {
		return
	}
	if err := s.rc.Delete(ctx, e.PlayerID, kind); err != nil {
		log.Warnf("store: recover invalidate player=%d kind=%s failed: %v", e.PlayerID, kind, err)
	}
}

// onInvalidate 订阅失效广播：删除本地 L1 对应键。
func (s *playerStore) onInvalidate(msg *rediscache.Message) {
	key := l1Key(msg.PlayerID)
	for _, f := range msg.Fields {
		switch f {
		case rediscache.KindCore:
			s.coreL1.Delete(key)
		case rediscache.KindAssets:
			s.assetsL1.Delete(key)
		case rediscache.KindProfile:
			s.profileL1.Delete(key)
		}
	}
}

// ---------------------------------------------------------------------------
// model ↔ mysql.Row / redis.Hash 显式转换（禁止反射 / json map 往返）
// ---------------------------------------------------------------------------

func coreToRow(c *model.PlayerCore) *mysql.CoreRow {
	return &mysql.CoreRow{
		PlayerID: c.PlayerID,
		Nickname: c.Nickname,
		Level:    c.Level,
		Exp:      c.Exp,
		Gold:     c.Gold,
		Version:  c.Version,
	}
}

func coreFromRow(r *mysql.CoreRow) *model.PlayerCore {
	return &model.PlayerCore{
		PlayerID: r.PlayerID,
		Nickname: r.Nickname,
		Level:    r.Level,
		Exp:      r.Exp,
		Gold:     r.Gold,
		Version:  r.Version,
	}
}

func assetToRow(a *model.PlayerAsset) *mysql.AssetRow {
	return &mysql.AssetRow{
		PlayerID:  a.PlayerID,
		AssetType: a.AssetType,
		Amount:    a.Amount,
		Version:   a.Version,
	}
}

func assetFromRow(r *mysql.AssetRow) *model.PlayerAsset {
	return &model.PlayerAsset{
		PlayerID:  r.PlayerID,
		AssetType: r.AssetType,
		Amount:    r.Amount,
		Version:   r.Version,
	}
}

func profileToRow(p *model.PlayerProfile) *mysql.ProfileRow {
	return &mysql.ProfileRow{
		PlayerID:     p.PlayerID,
		Signature:    p.Signature,
		AvatarID:     p.AvatarID,
		SettingsJSON: p.SettingsJSON,
		Version:      p.Version,
	}
}

func profileFromRow(r *mysql.ProfileRow) *model.PlayerProfile {
	return &model.PlayerProfile{
		PlayerID:     r.PlayerID,
		Signature:    r.Signature,
		AvatarID:     r.AvatarID,
		SettingsJSON: r.SettingsJSON,
		Version:      r.Version,
	}
}

func coreToHash(c *model.PlayerCore) map[string]string {
	return map[string]string{
		"nickname": c.Nickname,
		"level":    strconv.FormatInt(int64(c.Level), 10),
		"exp":      strconv.FormatInt(c.Exp, 10),
		"gold":     strconv.FormatInt(c.Gold, 10),
		"version":  strconv.FormatUint(c.Version, 10),
	}
}

func coreFromHash(playerID int64, f map[string]string) (*model.PlayerCore, error) {
	level, err := strconv.ParseInt(f["level"], 10, 32)
	if err != nil {
		return nil, err
	}
	exp, err := strconv.ParseInt(f["exp"], 10, 64)
	if err != nil {
		return nil, err
	}
	gold, err := strconv.ParseInt(f["gold"], 10, 64)
	if err != nil {
		return nil, err
	}
	ver, err := strconv.ParseUint(f["version"], 10, 64)
	if err != nil {
		return nil, err
	}
	return &model.PlayerCore{
		PlayerID: playerID,
		Nickname: f["nickname"],
		Level:    int32(level),
		Exp:      exp,
		Gold:     gold,
		Version:  ver,
	}, nil
}

// assetsToHash asset_type -> amount，version 单独字段（取列表内最大版本）。
func assetsToHash(assets []model.PlayerAsset) map[string]string {
	fields := make(map[string]string, len(assets)+1)
	var maxVer uint64
	for _, a := range assets {
		fields[strconv.FormatInt(int64(a.AssetType), 10)] = strconv.FormatInt(a.Amount, 10)
		if a.Version > maxVer {
			maxVer = a.Version
		}
	}
	fields["version"] = strconv.FormatUint(maxVer, 10)
	return fields
}

func assetsFromHash(playerID int64, f map[string]string) ([]model.PlayerAsset, error) {
	ver, _ := strconv.ParseUint(f["version"], 10, 64)
	assets := make([]model.PlayerAsset, 0, len(f))
	for k, v := range f {
		if k == "version" {
			continue
		}
		assetType, err := strconv.ParseInt(k, 10, 32)
		if err != nil {
			return nil, err
		}
		amount, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, err
		}
		assets = append(assets, model.PlayerAsset{
			PlayerID:  playerID,
			AssetType: int32(assetType),
			Amount:    amount,
			Version:   ver,
		})
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].AssetType < assets[j].AssetType })
	return assets, nil
}

func profileToHash(p *model.PlayerProfile) map[string]string {
	return map[string]string{
		"signature":     p.Signature,
		"avatar_id":     strconv.FormatInt(int64(p.AvatarID), 10),
		"settings_json": p.SettingsJSON,
		"version":       strconv.FormatUint(p.Version, 10),
	}
}

func profileFromHash(playerID int64, f map[string]string) (*model.PlayerProfile, error) {
	avatarID, err := strconv.ParseInt(f["avatar_id"], 10, 32)
	if err != nil {
		return nil, err
	}
	ver, err := strconv.ParseUint(f["version"], 10, 64)
	if err != nil {
		return nil, err
	}
	return &model.PlayerProfile{
		PlayerID:     playerID,
		Signature:    f["signature"],
		AvatarID:     int32(avatarID),
		SettingsJSON: f["settings_json"],
		Version:      ver,
	}, nil
}
