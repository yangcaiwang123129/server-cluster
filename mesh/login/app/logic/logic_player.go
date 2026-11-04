// Package logic 业务实现：实现 global 定义的接口；单例容器在本包，init 自注册。
package logic

import (
	"context"
	"errors"
	"fmt"
	"server-cluster/share/pb"
	"strconv"
	"time"

	"server-cluster/internal/svc"
	"server-cluster/mesh/login/app/global"
	"server-cluster/mesh/login/app/repo"
	"server-cluster/share/consts"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/singleton"
	"server-cluster/share/util"

	"github.com/dobyte/due/v2/etc"
	"github.com/dobyte/jwt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// clientIPKey 与 handler 注入 key 保持一致。
const clientIPKey = "client_ip"

// redis key prefixes
const (
	keySmsCode     = "sms:%s"          // 验证码
	keySmsCooldown = "sms_cooldown:%s" // 60s 防刷
	smsCodeTTL     = 5 * time.Minute
	smsCooldownTTL = 60 * time.Second
)

// gateAddr 下发给客户端的网关地址。
func gateAddr() string {
	if addr := etc.Get("etc.gate.addr").String(); addr != "" {
		return addr
	}
	return consts.DefaultGateAddr
}

type playerLogic struct{}

var playerLogicInst = singleton.NewContainer[global.PlayerLogic]("PlayerLogic")

func init() { playerLogicInst.Register(&playerLogic{}) }

func Player() global.PlayerLogic { return playerLogicInst.MustGet() }

// ──── 公共辅助 ────

// newPlayer 新建玩家基础表行（所有登录类型首次建 player 都走这里）。
func (this *playerLogic) newPlayer(ctx context.Context, deviceId string, zone int32) (*model.Player, error) {
	registerIP, _ := ctx.Value(clientIPKey).(string)
	m := &model.Player{
		PlayerID:   svc.NextID(),
		DeviceId:   deviceId,
		Zone:       zone,
		RegisterIP: registerIP,
		Status:     model.PlayerStatusNormal,
	}
	if err := repo.Repo().Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// issueToken 签发 JWT，payload 只有 player_id 字符串。
func (this *playerLogic) issueToken(pid int64) (string, error) {
	token, err := svc.JWT().GenerateToken(jwt.Payload{svc.PayloadPlayerID: strconv.FormatInt(pid, 10)})
	if err != nil {
		return "", err
	}
	return token.Token, nil
}

// checkNotBanned 校验玩家未被封号。
func (this *playerLogic) checkNotBanned(m *model.Player) error {
	if m.Status == model.PlayerStatusBanned {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_PLAYER_BANNED, "")
	}
	if m.BanUntil > 0 && util.NowUnix() < m.BanUntil {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_PLAYER_BANNED, "")
	}
	return nil
}

// ──── 统一登录入口 ────

func (this *playerLogic) PlayerLogin(ctx context.Context, req *dto.PlayerLoginReq) (*dto.PlayerLoginResp, error) {
	if req.DeviceId == "" {
		return nil, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "deviceId is required")
	}

	var (
		token string
		isNew bool
		err   error
	)

	switch req.LoginType {
	case consts.LoginTypeGuest:
		token, isNew, err = this.guestLogin(ctx, req)
	case consts.LoginTypeAccount:
		token, isNew, err = this.accountLogin(ctx, req)
	case consts.LoginTypePhoneSMS:
		token, isNew, err = this.phoneLogin(ctx, req)
	case consts.LoginTypeWechat:
		token, isNew, err = this.wechatLogin(ctx, req)
	case consts.LoginTypeApple:
		token, isNew, err = this.appleLogin(ctx, req)
	default:
		return nil, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "unknown loginType")
	}

	if err != nil {
		return nil, err
	}
	return &dto.PlayerLoginResp{
		Token:       token,
		GateAddr:    gateAddr(),
		IsNewPlayer: isNew,
	}, nil
}

// ──── 5 种登录分支 ────

// guestLogin 游客：只查 player.device_id，命中=复用，未命中=新建。不碰 player_auth。
func (this *playerLogic) guestLogin(ctx context.Context, req *dto.PlayerLoginReq) (string, bool, error) {
	m, err := repo.Repo().GetByDeviceID(ctx, req.DeviceId)
	if err == nil {
		if err := this.checkNotBanned(m); err != nil {
			return "", false, err
		}
		token, err := this.issueToken(m.PlayerID)
		return token, false, err
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false, err
	}
	// 未找到 → 新建
	if req.Zone == 0 {
		req.Zone = 86
	}
	m, err = this.newPlayer(ctx, req.DeviceId, req.Zone)
	if err != nil {
		return "", false, err
	}
	token, err := this.issueToken(m.PlayerID)
	return token, true, err
}

// accountLogin 账号密码：player_auth(type=1, identifier=username) 查，比对 bcrypt。
// 未找到时自动注册（player + player_auth）。
func (this *playerLogic) accountLogin(ctx context.Context, req *dto.PlayerLoginReq) (string, bool, error) {
	if req.Username == "" || req.Password == "" {
		return "", false, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "username and password are required")
	}

	auth, err := repo.AuthRepo().FindByTypeAndIdentifier(ctx, consts.LoginTypeAccount, req.Username)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false, err
	}

	if auth != nil {
		// 已有账号 → 校验密码
		if err := bcrypt.CompareHashAndPassword([]byte(auth.AuthSecret), []byte(req.Password)); err != nil {
			return "", false, errs.ErrOf(pb.ErrCode_ERR_CODE_PASSWORD_WRONG, "")
		}
		// 查 player → 校验状态 → 发 token
		m, err := repo.Repo().GetByID(ctx, auth.PlayerID)
		if err != nil {
			return "", false, err
		}
		if err := this.checkNotBanned(m); err != nil {
			return "", false, err
		}
		token, err := this.issueToken(m.PlayerID)
		return token, false, err
	}

	// 账号不存在 → 自动注册
	if req.Zone == 0 {
		req.Zone = 86
	}
	m, err := this.newPlayer(ctx, req.DeviceId, req.Zone)
	if err != nil {
		return "", false, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", false, err
	}
	if err := repo.AuthRepo().Create(ctx, &model.PlayerAuth{
		PlayerID:       m.PlayerID,
		AuthType:       consts.LoginTypeAccount,
		AuthIdentifier: req.Username,
		AuthSecret:     string(hash),
		Verified:       1,
	}); err != nil {
		return "", false, err
	}
	token, err := this.issueToken(m.PlayerID)
	return token, true, err
}

// phoneLogin 手机号验证码：Redis 验码 → player_auth(type=2, identifier=phone) 查/建。
func (this *playerLogic) phoneLogin(ctx context.Context, req *dto.PlayerLoginReq) (string, bool, error) {
	if req.PhoneNumber == "" || req.SmsCode == "" {
		return "", false, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "phoneNumber and smsCode are required")
	}

	// 1. Redis 验验证码
	if rdb, err := svc.Ctx().Redis.Default(); err == nil {
		key := fmt.Sprintf(keySmsCode, req.PhoneNumber)
		stored, err := rdb.Get(ctx, key).Result()
		if err != nil || stored != req.SmsCode {
			return "", false, errs.ErrOf(pb.ErrCode_ERR_CODE_SMS_CODE_INVALID, "")
		}
		// 一次性：用完删掉
		_ = rdb.Del(ctx, key).Err()
	}

	// 2. 查凭据
	auth, err := repo.AuthRepo().FindByTypeAndIdentifier(ctx, consts.LoginTypePhoneSMS, req.PhoneNumber)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false, err
	}
	if auth != nil {
		m, err := repo.Repo().GetByID(ctx, auth.PlayerID)
		if err != nil {
			return "", false, err
		}
		if err := this.checkNotBanned(m); err != nil {
			return "", false, err
		}
		token, err := this.issueToken(m.PlayerID)
		return token, false, err
	}

	// 未绑定 → 新建 player + player_auth
	if req.Zone == 0 {
		req.Zone = 86
	}
	m, err := this.newPlayer(ctx, req.DeviceId, req.Zone)
	if err != nil {
		return "", false, err
	}
	if err := repo.AuthRepo().Create(ctx, &model.PlayerAuth{
		PlayerID:       m.PlayerID,
		AuthType:       consts.LoginTypePhoneSMS,
		AuthIdentifier: req.PhoneNumber,
		Verified:       1,
	}); err != nil {
		return "", false, err
	}
	token, err := this.issueToken(m.PlayerID)
	return token, true, err
}

// wechatLogin 微信：code2session 换 openid → 查/建 player_auth(type=3, openid)。
// code2session 依赖微信开放平台 API，开发阶段暂以 wechatCode 直接当 openid mock。
func (this *playerLogic) wechatLogin(ctx context.Context, req *dto.PlayerLoginReq) (string, bool, error) {
	if req.WechatCode == "" {
		return "", false, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "wechatCode is required")
	}

	openid, err := this.wechatCode2Openid(ctx, req.WechatCode)
	if err != nil {
		return "", false, err
	}

	auth, err := repo.AuthRepo().FindByTypeAndIdentifier(ctx, consts.LoginTypeWechat, openid)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false, err
	}
	if auth != nil {
		m, err := repo.Repo().GetByID(ctx, auth.PlayerID)
		if err != nil {
			return "", false, err
		}
		if err := this.checkNotBanned(m); err != nil {
			return "", false, err
		}
		token, err := this.issueToken(m.PlayerID)
		return token, false, err
	}

	if req.Zone == 0 {
		req.Zone = 86
	}
	m, err := this.newPlayer(ctx, req.DeviceId, req.Zone)
	if err != nil {
		return "", false, err
	}
	if err := repo.AuthRepo().Create(ctx, &model.PlayerAuth{
		PlayerID:       m.PlayerID,
		AuthType:       consts.LoginTypeWechat,
		AuthIdentifier: openid,
		Verified:       1,
	}); err != nil {
		return "", false, err
	}
	token, err := this.issueToken(m.PlayerID)
	return token, true, err
}

// appleLogin Apple ID：identityToken 验签解 sub → 查/建 player_auth(type=4, sub)。
// Apple 验签依赖 .well-known/jwks 公钥本地缓存，开发阶段暂 mock。
func (this *playerLogic) appleLogin(ctx context.Context, req *dto.PlayerLoginReq) (string, bool, error) {
	if req.AppleIdentityToken == "" {
		return "", false, errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "appleIdentityToken is required")
	}

	sub, err := this.appleVerifyAndExtractSub(ctx, req.AppleIdentityToken, req.AppleNonce)
	if err != nil {
		return "", false, err
	}

	auth, err := repo.AuthRepo().FindByTypeAndIdentifier(ctx, consts.LoginTypeApple, sub)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false, err
	}
	if auth != nil {
		m, err := repo.Repo().GetByID(ctx, auth.PlayerID)
		if err != nil {
			return "", false, err
		}
		if err := this.checkNotBanned(m); err != nil {
			return "", false, err
		}
		token, err := this.issueToken(m.PlayerID)
		return token, false, err
	}

	if req.Zone == 0 {
		req.Zone = 86
	}
	m, err := this.newPlayer(ctx, req.DeviceId, req.Zone)
	if err != nil {
		return "", false, err
	}
	if err := repo.AuthRepo().Create(ctx, &model.PlayerAuth{
		PlayerID:       m.PlayerID,
		AuthType:       consts.LoginTypeApple,
		AuthIdentifier: sub,
		Verified:       1,
	}); err != nil {
		return "", false, err
	}
	token, err := this.issueToken(m.PlayerID)
	return token, true, err
}

// ──── 第三方登录 mock（后续替换为真实实现） ────

// wechatCode2Openid 把 wx.login() 的 code 换成 openid。
// TODO: 生产环境调 https://api.weixin.qq.com/sns/jscode2session
func (this *playerLogic) wechatCode2Openid(_ context.Context, code string) (string, error) {
	// mock: 用 code 直接当 openid
	return "wx_" + code, nil
}

// appleVerifyAndExtractSub 验签 identityToken 并解出 "sub"。
// TODO: 生产环境拉 .well-known/jwks 本地缓存验签
func (this *playerLogic) appleVerifyAndExtractSub(_ context.Context, identityToken, nonce string) (string, error) {
	// mock: 直接返回 identityToken 当 sub（取前 32 字符模拟 sub 长度）
	if len(identityToken) > 32 {
		return "apple_" + identityToken[:32], nil
	}
	return "apple_" + identityToken, nil
}

// ──── SendPhoneCode ────

func (this *playerLogic) SendPhoneCode(ctx context.Context, req *dto.PlayerPhoneSendCodeReq) error {
	if req.PhoneNumber == "" {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "phoneNumber is required")
	}

	rdb, err := svc.Ctx().Redis.Default()
	if err != nil {
		// Redis 不可用时降级：开发环境直接放行（返回 nil，login 时用任意 code 也能过）
		return nil
	}

	// 60s 防刷
	cooldownKey := fmt.Sprintf(keySmsCooldown, req.PhoneNumber)
	if wait, err := rdb.TTL(ctx, cooldownKey).Result(); err == nil && wait > 0 {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_SMS_COOLDOWN, "")
	}

	// 生成 6 位随机码（TODO: 替换为 crypto/rand）
	code := strconv.Itoa(int(util.NowUnixNano() % 1000000))
	// 补前导零
	for len(code) < 6 {
		code = "0" + code
	}

	codeKey := fmt.Sprintf(keySmsCode, req.PhoneNumber)
	if err := rdb.Set(ctx, codeKey, code, smsCodeTTL).Err(); err != nil {
		return err
	}
	_ = rdb.Set(ctx, cooldownKey, "1", smsCooldownTTL).Err()

	// TODO: 调第三方 SMS 网关发送 code
	_ = code
	return nil
}

// ──── PlayerBind（游客升级 / 已登录玩家加绑） ────

func (this *playerLogic) PlayerBind(ctx context.Context, req *dto.PlayerBindReq) error {
	if req.PlayerId == 0 {
		return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "playerId is required")
	}
	m, err := repo.Repo().GetByID(ctx, req.PlayerId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrOf(pb.ErrCode_ERR_CODE_PLAYER_NOT_FOUND, "")
		}
		return err
	}
	switch req.AuthType {
	case consts.LoginTypeAccount:
		if req.Username == "" || req.Password == "" {
			return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "username and password are required")
		}
		// 检查 username 没被别人用
		if _, err := repo.AuthRepo().FindByTypeAndIdentifier(ctx, consts.LoginTypeAccount, req.Username); err == nil {
			return errs.ErrOf(pb.ErrCode_ERR_CODE_PLAYER_EXISTS, "username already taken")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		return repo.AuthRepo().Create(ctx, &model.PlayerAuth{
			PlayerID:       m.PlayerID,
			AuthType:       consts.LoginTypeAccount,
			AuthIdentifier: req.Username,
			AuthSecret:     string(hash),
			Verified:       1,
		})
	case consts.LoginTypePhoneSMS:
		if req.PhoneNumber == "" || req.SmsCode == "" {
			return errs.ErrOf(pb.ErrCode_ERR_CODE_INVALID_PARAMS, "phoneNumber and smsCode are required")
		}
		if rdb, err := svc.Ctx().Redis.Default(); err == nil {
			stored, _ := rdb.Get(ctx, fmt.Sprintf(keySmsCode, req.PhoneNumber)).Result()
			if stored != req.SmsCode {
				return errs.ErrOf(pb.ErrCode_ERR_CODE_SMS_CODE_INVALID, "")
			}
			_ = rdb.Del(ctx, fmt.Sprintf(keySmsCode, req.PhoneNumber)).Err()
		}
		// 检查手机号没被别人绑
		if _, err := repo.AuthRepo().FindByTypeAndIdentifier(ctx, consts.LoginTypePhoneSMS, req.PhoneNumber); err == nil {
			return errs.ErrOf(pb.ErrCode_ERR_CODE_PLAYER_EXISTS, "phone already bound")
		}
		return repo.AuthRepo().Create(ctx, &model.PlayerAuth{
			PlayerID:       m.PlayerID,
			AuthType:       consts.LoginTypePhoneSMS,
			AuthIdentifier: req.PhoneNumber,
			Verified:       1,
		})
	case consts.LoginTypeWechat:
		openid, err := this.wechatCode2Openid(ctx, req.WechatCode)
		if err != nil {
			return err
		}
		if _, err := repo.AuthRepo().FindByTypeAndIdentifier(ctx, consts.LoginTypeWechat, openid); err == nil {
			return errs.ErrOf(pb.ErrCode_ERR_CODE_PLAYER_EXISTS, "wechat already bound")
		}
		return repo.AuthRepo().Create(ctx, &model.PlayerAuth{
			PlayerID:       m.PlayerID,
			AuthType:       consts.LoginTypeWechat,
			AuthIdentifier: openid,
			Verified:       1,
		})
	case consts.LoginTypeApple:
		sub, err := this.appleVerifyAndExtractSub(ctx, req.AppleIdentityToken, req.AppleNonce)
		if err != nil {
			return err
		}
		if _, err := repo.AuthRepo().FindByTypeAndIdentifier(ctx, consts.LoginTypeApple, sub); err == nil {
			return errs.ErrOf(pb.ErrCode_ERR_CODE_PLAYER_EXISTS, "apple id already bound")
		}
		return repo.AuthRepo().Create(ctx, &model.PlayerAuth{
			PlayerID:       m.PlayerID,
			AuthType:       consts.LoginTypeApple,
			AuthIdentifier: sub,
			Verified:       1,
		})
	default:
		return errs.ErrOf(pb.ErrCode_ERR_CODE_AUTH_TYPE_UNKNOWN, "")
	}
}
