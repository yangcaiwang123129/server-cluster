package logic

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/dobyte/due/v2/log"
	"gorm.io/gorm"

	"server-cluster/internal/svc"
	"server-cluster/node/game/app/global"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
	"server-cluster/share/util"
)

type mailLogic struct{}

var mailLogicInst = singleton.NewContainer[global.MailLogic]("MailLogic")

func init() {
	mailLogicInst.Register(&mailLogic{})
}

// Mail 获取邮件业务单例。
func Mail() global.MailLogic { return mailLogicInst.MustGet() }

// EnterGame 进入游戏：加载玩家邮件持有，未建档则初始化空集合（仅预热缓存）。
func (this *mailLogic) EnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq) (*pb.EnterGameResp, error) {
	if _, err := loadPlayerMail(ctx, playerID); err != nil {
		return nil, err
	}
	return &pb.EnterGameResp{}, nil
}

// LeaveGame 离开游戏：邮件持有数据同步落库，并逐出内存缓存。
func (this *mailLogic) LeaveGame(ctx context.Context, playerID int64, _ *pb.LeaveGameReq) (*pb.LeaveGameResp, error) {
	if err := flushAndEvict(ctx, global.PlayerMailCache, playerID); err != nil {
		return nil, err
	}
	return &pb.LeaveGameResp{}, nil
}

// MailList 邮件列表：未读优先，同状态按创建时间倒序；返回未读数用于红点。
func (this *mailLogic) MailList(ctx context.Context, playerID int64) (*pb.MailListResp, error) {
	row, err := loadPlayerMail(ctx, playerID)
	if err != nil {
		return nil, err
	}
	entries := make([]*pb.MailEntry, 0, row.Mails.Len())
	unread := int32(0)
	row.Mails.Range(func(_ int64, m model.MailInfo) bool {
		entries = append(entries, toMailEntry(m))
		if m.Status == model.MailStatusUnread {
			unread++
		}
		return true
	})
	// 排序：未读优先，其次已读，最后已领取；同状态按创建时间倒序
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Status != entries[j].Status {
			return entries[i].Status < entries[j].Status
		}
		return entries[i].CreatedAt > entries[j].CreatedAt
	})
	return &pb.MailListResp{Entries: entries, UnreadCount: unread}, nil
}

// MailRead 标记单封已读；重复已读幂等。
func (this *mailLogic) MailRead(ctx context.Context, playerID int64, req *pb.MailReadReq) (*pb.MailReadResp, error) {
	row, err := loadPlayerMail(ctx, playerID)
	if err != nil {
		return nil, err
	}
	m, ok := row.Mails.Get(req.GetId())
	if !ok {
		return nil, errs.CodeInvalidParams.WithMessage("邮件不存在").Err()
	}
	if m.Status == model.MailStatusUnread {
		m.Status = model.MailStatusRead
		m.ReadAt = time.Now().UnixMilli()
		if !row.UpdateMail(m) {
			return nil, errs.CodeInvalidParams.WithMessage("邮件不存在").Err()
		}
		global.PlayerMailCache.Set(playerID, row)
	}
	return &pb.MailReadResp{Entry: toMailEntry(m)}, nil
}

// MailReadAll 全部未读标记已读。
func (this *mailLogic) MailReadAll(ctx context.Context, playerID int64, _ *pb.MailReadAllReq) (*pb.MailReadAllResp, error) {
	row, err := loadPlayerMail(ctx, playerID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	if err := row.Mails.UpdateAll(func(m map[int64]model.MailInfo) error {
		for id, mail := range m {
			if mail.Status == model.MailStatusUnread {
				mail.Status = model.MailStatusRead
				mail.ReadAt = now
				m[id] = mail
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	global.PlayerMailCache.Set(playerID, row)
	return &pb.MailReadAllResp{}, nil
}

// MailClaim 领取单封附件：先落库领取状态再发放资产，避免发放成功后状态落库失败导致重试重复发放。
// 残余风险：进程若在资产发放与状态回滚之间崩溃，仍可能有个别附件不一致；彻底解决需资产与邮件同事务。
func (this *mailLogic) MailClaim(ctx context.Context, playerID int64, req *pb.MailClaimReq) (*pb.MailClaimResp, error) {
	row, err := loadPlayerMail(ctx, playerID)
	if err != nil {
		return nil, err
	}
	m, ok := row.Mails.Get(req.GetId())
	if !ok {
		return nil, errs.CodeInvalidParams.WithMessage("邮件不存在").Err()
	}
	if m.Status == model.MailStatusClaimed || !m.HasAttachments() {
		return &pb.MailClaimResp{Entry: toMailEntry(m)}, nil
	}
	originStatus, originClaimedAt := m.Status, m.ClaimedAt
	m.Status = model.MailStatusClaimed
	m.ClaimedAt = time.Now().UnixMilli()
	if !row.UpdateMail(m) {
		return nil, errs.CodeInvalidParams.WithMessage("邮件不存在").Err()
	}
	// 状态已落库，再发放全部附件；失败时尽量回滚状态以便重试
	granted := make([]*pb.AssetEntry, 0, len(m.Attachments))
	for _, a := range m.Attachments {
		count := a.Count
		if count <= 0 {
			count = 1
		}
		if err := Asset().AssetAdd(ctx, playerID, &tpl.Asset{
			Type:  tpl.AssetType(a.Type),
			Id:    a.CfgID,
			Count: count,
		}); err != nil {
			log.Warnf("game: mail claim grant failed, player=%d mail=%d asset=%+v: %v", playerID, m.MailID, a, err)
			m.Status, m.ClaimedAt = originStatus, originClaimedAt
			if !row.UpdateMail(m) {
				log.Warnf("game: mail claim rollback failed, player=%d mail=%d", playerID, m.MailID)
			}
			global.PlayerMailCache.Set(playerID, row)
			return nil, err
		}
		granted = append(granted, &pb.AssetEntry{Type: a.Type, CfgId: a.CfgID, Count: count})
	}
	global.PlayerMailCache.Set(playerID, row)
	return &pb.MailClaimResp{Entry: toMailEntry(m), Attachments: granted}, nil
}

// pendingClaim 一键领取时先落库、后发放所需的原始状态快照。
type pendingClaim struct {
	mail            model.MailInfo
	originStatus    int32
	originClaimedAt int64
}

// MailClaimAll 一键领取全部未领取附件（跳过已领取/无附件邮件）。
// 先在 UpdateAll 内把邮件标记为已领取并落库，再逐封发放，避免中途失败后邮件仍可重复领取。
// 残余风险同 MailClaim：进程在发放与回滚之间崩溃仍可能不一致。
func (this *mailLogic) MailClaimAll(ctx context.Context, playerID int64, _ *pb.MailClaimAllReq) (*pb.MailClaimAllResp, error) {
	row, err := loadPlayerMail(ctx, playerID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	pending := make([]pendingClaim, 0, row.Mails.Len())
	if err := row.Mails.UpdateAll(func(m map[int64]model.MailInfo) error {
		for id, mail := range m {
			if mail.Status == model.MailStatusClaimed || !mail.HasAttachments() {
				continue
			}
			originStatus, originClaimedAt := mail.Status, mail.ClaimedAt
			mail.Status = model.MailStatusClaimed
			mail.ClaimedAt = now
			m[id] = mail
			pending = append(pending, pendingClaim{mail: mail, originStatus: originStatus, originClaimedAt: originClaimedAt})
		}
		return nil
	}); err != nil {
		return nil, err
	}
	claimed := make([]*pb.MailEntry, 0, len(pending))
	for _, p := range pending {
		granted := true
		for _, a := range p.mail.Attachments {
			count := a.Count
			if count <= 0 {
				count = 1
			}
			if err := Asset().AssetAdd(ctx, playerID, &tpl.Asset{
				Type:  tpl.AssetType(a.Type),
				Id:    a.CfgID,
				Count: count,
			}); err != nil {
				log.Warnf("game: mail claim-all grant failed, player=%d mail=%d asset=%+v: %v", playerID, p.mail.MailID, a, err)
				granted = false
				break
			}
		}
		if !granted {
			p.mail.Status, p.mail.ClaimedAt = p.originStatus, p.originClaimedAt
			if !row.UpdateMail(p.mail) {
				log.Warnf("game: mail claim-all rollback failed, player=%d mail=%d", playerID, p.mail.MailID)
			}
			continue
		}
		claimed = append(claimed, toMailEntry(p.mail))
	}
	global.PlayerMailCache.Set(playerID, row)
	return &pb.MailClaimAllResp{Entries: claimed}, nil
}

// MailDelete 删除单封邮件：奖励邮件未领取时禁止删除。
func (this *mailLogic) MailDelete(ctx context.Context, playerID int64, req *pb.MailDeleteReq) (*pb.MailDeleteResp, error) {
	row, err := loadPlayerMail(ctx, playerID)
	if err != nil {
		return nil, err
	}
	m, ok := row.Mails.Get(req.GetId())
	if !ok {
		return nil, errs.CodeInvalidParams.WithMessage("邮件不存在").Err()
	}
	if !m.CanDelete() {
		return nil, errs.CodeInvalidParams.WithMessage("奖励邮件领取后才能删除").Err()
	}
	if !row.DeleteMail(m.MailID) {
		return nil, errs.CodeInvalidParams.WithMessage("邮件不存在").Err()
	}
	global.PlayerMailCache.Set(playerID, row)
	return &pb.MailDeleteResp{}, nil
}

// MailDeleteClean 清理全部可删除邮件（无附件或已领取），返回删除数量。
func (this *mailLogic) MailDeleteClean(ctx context.Context, playerID int64, _ *pb.MailDeleteCleanReq) (*pb.MailDeleteCleanResp, error) {
	row, err := loadPlayerMail(ctx, playerID)
	if err != nil {
		return nil, err
	}
	count := int32(0)
	if err := row.Mails.UpdateAll(func(m map[int64]model.MailInfo) error {
		for id, mail := range m {
			if mail.CanDelete() {
				delete(m, id)
				count++
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	global.PlayerMailCache.Set(playerID, row)
	return &pb.MailDeleteCleanResp{Count: count}, nil
}

// MailSend 后台发送邮件（可带附件）：生成实例ID，状态未读。
func (this *mailLogic) MailSend(ctx context.Context, playerID int64, req *pb.GmMailSendReq) (*pb.MailEntry, error) {
	if req.GetTitle() == "" {
		return nil, errs.CodeInvalidParams.WithMessage("邮件标题不能为空").Err()
	}
	row, err := loadPlayerMail(ctx, playerID)
	if err != nil {
		return nil, err
	}
	atts := make([]model.MailAttachment, 0, len(req.GetAttachments()))
	for _, a := range req.GetAttachments() {
		atts = append(atts, model.MailAttachment{Type: a.GetType(), CfgID: a.GetCfgId(), Count: a.GetCount()})
	}
	m := model.MailInfo{
		MailID:      svc.NextID(),
		Title:       req.GetTitle(),
		Content:     req.GetContent(),
		Attachments: atts,
		Status:      model.MailStatusUnread,
		CreatedAt:   time.Now().UnixMilli(),
	}
	if !row.AddMail(m) {
		return nil, errs.CodeInternalError.WithMessage("邮件创建失败").Err()
	}
	global.PlayerMailCache.Set(playerID, row)
	return toMailEntry(m), nil
}

// loadPlayerMail 加载玩家邮件；未建档则初始化空 map 并挂入缓存（异步落库）。
func loadPlayerMail(ctx context.Context, playerID int64) (*model.PlayerMail, error) {
	row, err := global.PlayerMailCache.Get(ctx, playerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = &model.PlayerMail{PlayerID: playerID, Mails: util.NewSafeMap[int64, model.MailInfo]()}
		global.PlayerMailCache.Set(playerID, row)
		return row, nil
	}
	if err != nil {
		return nil, err
	}
	if row.Mails == nil {
		row.Mails = util.NewSafeMap[int64, model.MailInfo]()
	}
	return row, nil
}

// toMailEntry 持有数据 → wire 条目。
func toMailEntry(m model.MailInfo) *pb.MailEntry {
	atts := make([]*pb.AssetEntry, 0, len(m.Attachments))
	for _, a := range m.Attachments {
		atts = append(atts, &pb.AssetEntry{Type: a.Type, CfgId: a.CfgID, Count: a.Count})
	}
	return &pb.MailEntry{
		Id:          m.MailID,
		Title:       m.Title,
		Content:     m.Content,
		Attachments: atts,
		Status:      pb.MailStatus(m.Status),
		CreatedAt:   m.CreatedAt,
		ReadAt:      m.ReadAt,
		ClaimedAt:   m.ClaimedAt,
	}
}
