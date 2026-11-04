package logic

import (
	"context"
	"sort"

	"github.com/dobyte/due/v2/log"

	"server-cluster/internal/cache"
	"server-cluster/internal/svc"
	"server-cluster/node/game/app/global"
	"server-cluster/share/convert"
	"server-cluster/share/enum"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
	"server-cluster/share/singleton"
	"server-cluster/share/tpl"
	"server-cluster/share/util"
)

// mailCache 玩家邮件（player_mail，一玩家一行，邮件 map JSON）；读走 LRU，写异步批量 upsert。
var mailCache = cache.NewPlayerTableCache[model.PlayerMail](cache.CacherConfig{Name: enum.TablePlayerMail.String()})

type mailLogic struct{}

var mailLogicInst = singleton.NewContainer[global.MailLogic]("MailLogic")

func init() {
	mailLogicInst.Register(&mailLogic{})
}

// Mail 获取邮件业务单例。
func Mail() global.MailLogic { return mailLogicInst.MustGet() }

func (this *mailLogic) OnEnterGame(ctx context.Context, playerID int64, _ *pb.EnterGameReq, _ *pb.EnterGameResp) error {
	if _, err := this.Load(ctx, playerID); err != nil {
		return err
	}
	return nil
}

func (this *mailLogic) OnLeaveGame(ctx context.Context, playerID int64, _ *pb.LeaveGameReq, _ *pb.LeaveGameResp) error {
	if err := flushAndEvict(ctx, mailCache, playerID); err != nil {
		return err
	}
	return nil
}

func (this *mailLogic) MailList(ctx context.Context, playerID int64) (*pb.MailListResp, error) {
	row, err := this.Load(ctx, playerID)
	if err != nil {
		return nil, err
	}
	entries := make([]*pb.MailEntry, 0, row.Mails.Len())
	unread := int32(0)
	row.Mails.Range(func(_ int64, m model.MailInfo) bool {
		if m.Status == model.MailStatusDeleted {
			return true
		}
		entries = append(entries, convert.MailEntry(m))
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

func (this *mailLogic) MailRead(ctx context.Context, playerID int64, req *pb.MailReadReq) (*pb.MailReadResp, error) {
	row, err := this.Load(ctx, playerID)
	if err != nil {
		return nil, err
	}
	m, ok := row.Mails.Get(req.GetId())
	if !ok || m.Status == model.MailStatusDeleted {
		return &pb.MailReadResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "邮件不存在")}, nil
	}
	if m.Status == model.MailStatusUnread {
		m.Status = model.MailStatusRead
		m.ReadAt = util.NowUnixMilli()
		if !row.UpdateMail(m) {
			return &pb.MailReadResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "邮件不存在")}, nil
		}
		this.Save(playerID, row)
	}
	return &pb.MailReadResp{Entry: convert.MailEntry(m)}, nil
}

func (this *mailLogic) MailReadAll(ctx context.Context, playerID int64, _ *pb.MailReadAllReq) (*pb.MailReadAllResp, error) {
	row, err := this.Load(ctx, playerID)
	if err != nil {
		return nil, err
	}
	now := util.NowUnixMilli()
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
	this.Save(playerID, row)
	return &pb.MailReadAllResp{}, nil
}

func (this *mailLogic) MailClaim(ctx context.Context, playerID int64, req *pb.MailClaimReq) (*pb.MailClaimResp, error) {
	row, err := this.Load(ctx, playerID)
	if err != nil {
		return nil, err
	}
	m, ok := row.Mails.Get(req.GetId())
	if !ok || m.Status == model.MailStatusDeleted {
		return &pb.MailClaimResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "邮件不存在")}, nil
	}
	if m.Status == model.MailStatusClaimed || !m.HasAttachments() {
		return &pb.MailClaimResp{Entry: convert.MailEntry(m)}, nil
	}
	originStatus, originClaimedAt := m.Status, m.ClaimedAt
	m.Status = model.MailStatusClaimed
	m.ClaimedAt = util.NowUnixMilli()
	if !row.UpdateMail(m) {
		return &pb.MailClaimResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "邮件不存在")}, nil
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
			this.Save(playerID, row)
			if r := codedResult(err); r != nil {
				return &pb.MailClaimResp{Result: r}, nil
			}
			return nil, err
		}
		granted = append(granted, &pb.AssetEntry{Type: a.Type, CfgId: a.CfgID, Count: count})
	}
	this.Save(playerID, row)
	return &pb.MailClaimResp{Entry: convert.MailEntry(m), Attachments: granted}, nil
}

// pendingClaim 一键领取时先落库、后发放所需的原始状态快照。
type pendingClaim struct {
	mail            model.MailInfo
	originStatus    int32
	originClaimedAt int64
}

func (this *mailLogic) MailClaimAll(ctx context.Context, playerID int64, _ *pb.MailClaimAllReq) (*pb.MailClaimAllResp, error) {
	row, err := this.Load(ctx, playerID)
	if err != nil {
		return nil, err
	}
	now := util.NowUnixMilli()
	pending := make([]pendingClaim, 0, row.Mails.Len())
	if err := row.Mails.UpdateAll(func(m map[int64]model.MailInfo) error {
		for id, mail := range m {
			if mail.Status == model.MailStatusDeleted || mail.Status == model.MailStatusClaimed || !mail.HasAttachments() {
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
		claimed = append(claimed, convert.MailEntry(p.mail))
	}
	this.Save(playerID, row)
	return &pb.MailClaimAllResp{Entries: claimed}, nil
}

func (this *mailLogic) MailDelete(ctx context.Context, playerID int64, req *pb.MailDeleteReq) (*pb.MailDeleteResp, error) {
	row, err := this.Load(ctx, playerID)
	if err != nil {
		return nil, err
	}
	m, ok := row.Mails.Get(req.GetId())
	if !ok {
		return &pb.MailDeleteResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "邮件不存在")}, nil
	}
	if m.Status == model.MailStatusDeleted {
		return &pb.MailDeleteResp{}, nil
	}
	if !m.CanDelete() {
		return &pb.MailDeleteResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "奖励邮件领取后才能删除")}, nil
	}
	m.Status = model.MailStatusDeleted
	if !row.UpdateMail(m) {
		return &pb.MailDeleteResp{Result: errs.BuildErrResult(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "邮件不存在")}, nil
	}
	this.Save(playerID, row)
	return &pb.MailDeleteResp{}, nil
}

func (this *mailLogic) MailDeleteClean(ctx context.Context, playerID int64, _ *pb.MailDeleteCleanReq) (*pb.MailDeleteCleanResp, error) {
	row, err := this.Load(ctx, playerID)
	if err != nil {
		return nil, err
	}
	count := int32(0)
	if err := row.Mails.UpdateAll(func(m map[int64]model.MailInfo) error {
		for id, mail := range m {
			if !mail.CanDelete() {
				continue
			}
			mail.Status = model.MailStatusDeleted
			m[id] = mail
			count++
		}
		return nil
	}); err != nil {
		return nil, err
	}
	this.Save(playerID, row)
	return &pb.MailDeleteCleanResp{Count: count}, nil
}

func (this *mailLogic) MailSend(ctx context.Context, playerID int64, req *pb.GmMailSendReq) (*pb.MailEntry, error) {
	if req.GetTitle() == "" {
		return nil, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "邮件标题不能为空")
	}
	row, err := this.Load(ctx, playerID)
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
		CreatedAt:   util.NowUnixMilli(),
	}
	if !row.AddMail(m) {
		return nil, errs.ErrOf(pb.ErrCode_ERR_CODE_INTERNAL_ERROR, "邮件创建失败")
	}
	this.Save(playerID, row)
	return convert.MailEntry(m), nil
}

func (this *mailLogic) Load(ctx context.Context, playerID int64) (*model.PlayerMail, error) {
	return mailCache.GetOrInit(ctx, playerID,
		func(id int64) *model.PlayerMail {
			return &model.PlayerMail{PlayerID: id, Mails: util.NewSafeMap[int64, model.MailInfo]()}
		},
		func(row *model.PlayerMail) {
			if row.Mails == nil {
				row.Mails = util.NewSafeMap[int64, model.MailInfo]()
			}
		})
}

func (this *mailLogic) Save(playerID int64, row *model.PlayerMail) {
	mailCache.Set(playerID, row)
}

func (this *mailLogic) Flush(ctx context.Context, playerID int64) error {
	return flushAndEvict(ctx, mailCache, playerID)
}
