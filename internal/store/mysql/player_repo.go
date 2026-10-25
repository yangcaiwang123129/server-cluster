package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// ErrNotFound 行不存在；store 层转换为 store.ErrNotFound。
var ErrNotFound = errors.New("mysql: row not found")

// CoreRow player_core 行。
type CoreRow struct {
	PlayerID int64
	Nickname string
	Level    int32
	Exp      int64
	Gold     int64
	Version  uint64
}

// AssetRow player_assets 行。
type AssetRow struct {
	PlayerID  int64
	AssetType int32
	Amount    int64
	Version   uint64
}

// ProfileRow player_profile 行。
type ProfileRow struct {
	PlayerID     int64
	Signature    string
	AvatarID     int32
	SettingsJSON string
	Version      uint64
}

// PlayerRepo 玩家仓储：所有语句带 player_id 条件，表名一律经 ShardRouter 解析。
type PlayerRepo struct {
	shards []*sql.DB
	router ShardRouter
}

// NewPlayerRepo 构造仓储。
func NewPlayerRepo(shards []*sql.DB, router ShardRouter) *PlayerRepo {
	return &PlayerRepo{shards: shards, router: router}
}

func (r *PlayerRepo) db(ref ShardRef) (*sql.DB, error) {
	if ref.DBIndex < 0 || ref.DBIndex >= len(r.shards) {
		return nil, fmt.Errorf("mysql: shard index %d out of range", ref.DBIndex)
	}
	return r.shards[ref.DBIndex], nil
}

// GetCore 读取核心行；miss 返回 ErrNotFound。
func (r *PlayerRepo) GetCore(ctx context.Context, playerID int64) (*CoreRow, error) {
	ref := r.router.Route(TableCore, playerID)
	db, err := r.db(ref)
	if err != nil {
		return nil, err
	}
	row := &CoreRow{}
	err = db.QueryRowContext(ctx, fmt.Sprintf(
		"SELECT player_id, nickname, level, exp, gold, version FROM %s WHERE player_id = ?", ref.Table),
		playerID,
	).Scan(&row.PlayerID, &row.Nickname, &row.Level, &row.Exp, &row.Gold, &row.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("mysql: get core player=%d failed: %w", playerID, err)
	}
	return row, nil
}

// GetAssets 读取资产行列表；miss 返回空切片（非错误）。
func (r *PlayerRepo) GetAssets(ctx context.Context, playerID int64) ([]*AssetRow, error) {
	ref := r.router.Route(TableAssets, playerID)
	db, err := r.db(ref)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, fmt.Sprintf(
		"SELECT player_id, asset_type, amount, version FROM %s WHERE player_id = ?", ref.Table),
		playerID,
	)
	if err != nil {
		return nil, fmt.Errorf("mysql: get assets player=%d failed: %w", playerID, err)
	}
	defer rows.Close()
	out := make([]*AssetRow, 0)
	for rows.Next() {
		row := &AssetRow{}
		if err := rows.Scan(&row.PlayerID, &row.AssetType, &row.Amount, &row.Version); err != nil {
			return nil, fmt.Errorf("mysql: scan asset row failed: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// GetProfile 读取档案行；miss 返回 ErrNotFound。
func (r *PlayerRepo) GetProfile(ctx context.Context, playerID int64) (*ProfileRow, error) {
	ref := r.router.Route(TableProfile, playerID)
	db, err := r.db(ref)
	if err != nil {
		return nil, err
	}
	row := &ProfileRow{}
	err = db.QueryRowContext(ctx, fmt.Sprintf(
		"SELECT player_id, signature, avatar_id, settings_json, version FROM %s WHERE player_id = ?", ref.Table),
		playerID,
	).Scan(&row.PlayerID, &row.Signature, &row.AvatarID, &row.SettingsJSON, &row.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("mysql: get profile player=%d failed: %w", playerID, err)
	}
	return row, nil
}

// BatchUpsertCore 批量写入核心行；按分片分组逐分片事务执行。
// version 乐观条件：仅当新 version 更大时覆盖业务列，version 取两者最大值（幂等，可安全重放）。
func (r *PlayerRepo) BatchUpsertCore(ctx context.Context, rows []*CoreRow) error {
	groups := r.groupByShard(len(rows), func(i int) int64 { return rows[i].PlayerID })
	for idx, indexes := range groups {
		if err := r.upsertCore(ctx, idx, rows, indexes); err != nil {
			return err
		}
	}
	return nil
}

func (r *PlayerRepo) upsertCore(ctx context.Context, shardIdx int, rows []*CoreRow, indexes []int) error {
	db := r.shards[shardIdx]
	table := r.router.Route(TableCore, rows[indexes[0]].PlayerID).Table
	var sb strings.Builder
	sb.WriteString("INSERT INTO ")
	sb.WriteString(table)
	sb.WriteString(" (player_id, nickname, level, exp, gold, version) VALUES ")
	args := make([]any, 0, len(indexes)*6)
	for i, ri := range indexes {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString("(?,?,?,?,?,?)")
		row := rows[ri]
		args = append(args, row.PlayerID, row.Nickname, row.Level, row.Exp, row.Gold, row.Version)
	}
	sb.WriteString(` ON DUPLICATE KEY UPDATE
	nickname = IF(VALUES(version) > version, VALUES(nickname), nickname),
	level    = IF(VALUES(version) > version, VALUES(level), level),
	exp      = IF(VALUES(version) > version, VALUES(exp), exp),
	gold     = IF(VALUES(version) > version, VALUES(gold), gold),
	version  = GREATEST(version, VALUES(version))`)
	return r.execInTx(ctx, db, sb.String(), args)
}

// BatchUpsertAssets 批量写入资产行；分片事务 + version 乐观条件。
func (r *PlayerRepo) BatchUpsertAssets(ctx context.Context, rows []*AssetRow) error {
	groups := r.groupByShard(len(rows), func(i int) int64 { return rows[i].PlayerID })
	for idx, indexes := range groups {
		if err := r.upsertAssets(ctx, idx, rows, indexes); err != nil {
			return err
		}
	}
	return nil
}

func (r *PlayerRepo) upsertAssets(ctx context.Context, shardIdx int, rows []*AssetRow, indexes []int) error {
	db := r.shards[shardIdx]
	table := r.router.Route(TableAssets, rows[indexes[0]].PlayerID).Table
	var sb strings.Builder
	sb.WriteString("INSERT INTO ")
	sb.WriteString(table)
	sb.WriteString(" (player_id, asset_type, amount, version) VALUES ")
	args := make([]any, 0, len(indexes)*4)
	for i, ri := range indexes {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString("(?,?,?,?)")
		row := rows[ri]
		args = append(args, row.PlayerID, row.AssetType, row.Amount, row.Version)
	}
	sb.WriteString(` ON DUPLICATE KEY UPDATE
	amount  = IF(VALUES(version) > version, VALUES(amount), amount),
	version = GREATEST(version, VALUES(version))`)
	return r.execInTx(ctx, db, sb.String(), args)
}

// BatchUpsertProfile 批量写入档案行；分片事务 + version 乐观条件。
func (r *PlayerRepo) BatchUpsertProfile(ctx context.Context, rows []*ProfileRow) error {
	groups := r.groupByShard(len(rows), func(i int) int64 { return rows[i].PlayerID })
	for idx, indexes := range groups {
		if err := r.upsertProfile(ctx, idx, rows, indexes); err != nil {
			return err
		}
	}
	return nil
}

func (r *PlayerRepo) upsertProfile(ctx context.Context, shardIdx int, rows []*ProfileRow, indexes []int) error {
	db := r.shards[shardIdx]
	table := r.router.Route(TableProfile, rows[indexes[0]].PlayerID).Table
	var sb strings.Builder
	sb.WriteString("INSERT INTO ")
	sb.WriteString(table)
	sb.WriteString(" (player_id, signature, avatar_id, settings_json, version) VALUES ")
	args := make([]any, 0, len(indexes)*5)
	for i, ri := range indexes {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString("(?,?,?,?,?)")
		row := rows[ri]
		args = append(args, row.PlayerID, row.Signature, row.AvatarID, row.SettingsJSON, row.Version)
	}
	sb.WriteString(` ON DUPLICATE KEY UPDATE
	signature     = IF(VALUES(version) > version, VALUES(signature), signature),
	avatar_id     = IF(VALUES(version) > version, VALUES(avatar_id), avatar_id),
	settings_json = IF(VALUES(version) > version, VALUES(settings_json), settings_json),
	version       = GREATEST(version, VALUES(version))`)
	return r.execInTx(ctx, db, sb.String(), args)
}

// DeletePlayer 删除玩家三张表全部数据；按分片分组执行。
func (r *PlayerRepo) DeletePlayer(ctx context.Context, ids []int64) error {
	groups := r.groupByShard(len(ids), func(i int) int64 { return ids[i] })
	for idx, indexes := range groups {
		db := r.shards[idx]
		placeholders := strings.Repeat("?,", len(indexes))
		placeholders = placeholders[:len(placeholders)-1]
		args := make([]any, 0, len(indexes))
		for _, ri := range indexes {
			args = append(args, ids[ri])
		}
		for _, base := range []string{TableCore, TableAssets, TableProfile} {
			table := r.router.Route(base, ids[indexes[0]]).Table
			if err := r.execInTx(ctx, db, fmt.Sprintf(
				"DELETE FROM %s WHERE player_id IN (%s)", table, placeholders), args); err != nil {
				return fmt.Errorf("mysql: delete %s failed: %w", table, err)
			}
		}
	}
	return nil
}

// GetCommittedSeq 读取 WAL 已提交水位；wal_id 无记录时返回 0。
// player_wal_meta 无 player_id，固定落 0 号分片。
func (r *PlayerRepo) GetCommittedSeq(ctx context.Context, walID string) (uint64, error) {
	if len(r.shards) == 0 {
		return 0, fmt.Errorf("mysql: no shards")
	}
	var seq uint64
	err := r.shards[0].QueryRowContext(ctx,
		"SELECT committed_seq FROM player_wal_meta WHERE wal_id = ?", walID).Scan(&seq)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("mysql: get committed_seq wal=%s failed: %w", walID, err)
	}
	return seq, nil
}

// SetCommittedSeq 推进 WAL 已提交水位（只增不减）。
func (r *PlayerRepo) SetCommittedSeq(ctx context.Context, walID string, seq uint64) error {
	if len(r.shards) == 0 {
		return fmt.Errorf("mysql: no shards")
	}
	_, err := r.shards[0].ExecContext(ctx,
		`INSERT INTO player_wal_meta (wal_id, committed_seq) VALUES (?, ?)
		 ON DUPLICATE KEY UPDATE committed_seq = GREATEST(committed_seq, VALUES(committed_seq))`,
		walID, seq)
	if err != nil {
		return fmt.Errorf("mysql: set committed_seq wal=%s seq=%d failed: %w", walID, seq, err)
	}
	return nil
}

// NextVersion 版本号生成（mysql_fallback 驱动）：事务内 version+1；
// 行不存在时插入仅含 player_id/version 的默认行，返回 1。
func (r *PlayerRepo) NextVersion(ctx context.Context, playerID int64) (uint64, error) {
	ref := r.router.Route(TableCore, playerID)
	db, err := r.db(ref)
	if err != nil {
		return 0, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("mysql: begin next_version tx failed: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var version uint64
	err = tx.QueryRowContext(ctx, fmt.Sprintf(
		"SELECT version FROM %s WHERE player_id = ? FOR UPDATE", ref.Table), playerID).Scan(&version)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(
			"INSERT INTO %s (player_id, version) VALUES (?, 1)", ref.Table), playerID); err != nil {
			return 0, fmt.Errorf("mysql: insert default core row player=%d failed: %w", playerID, err)
		}
		if err := tx.Commit(); err != nil {
			return 0, err
		}
		return 1, nil
	case err != nil:
		return 0, fmt.Errorf("mysql: lock core row player=%d failed: %w", playerID, err)
	}

	version++
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(
		"UPDATE %s SET version = ? WHERE player_id = ?", ref.Table), version, playerID); err != nil {
		return 0, fmt.Errorf("mysql: bump version player=%d failed: %w", playerID, err)
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return version, nil
}

// groupByShard 按路由结果把下标分组到分片。
func (r *PlayerRepo) groupByShard(n int, idAt func(i int) int64) map[int][]int {
	groups := make(map[int][]int)
	for i := 0; i < n; i++ {
		ref := r.router.Route(TableCore, idAt(i))
		groups[ref.DBIndex] = append(groups[ref.DBIndex], i)
	}
	return groups
}

// execInTx 单语句事务执行。
func (r *PlayerRepo) execInTx(ctx context.Context, db *sql.DB, query string, args []any) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("mysql: begin tx failed: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return err
	}
	return tx.Commit()
}
