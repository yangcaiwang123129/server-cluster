// Package logic 客户端相关业务逻辑：客户端版本管理、客户端模拟器。
package logic

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dobyte/due/network/ws/v2"
	"github.com/dobyte/due/v2/codes"
	"github.com/dobyte/due/v2/core/buffer"
	"github.com/dobyte/due/v2/etc"
	"github.com/dobyte/due/v2/network"
	"github.com/dobyte/due/v2/packet"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"server-cluster/internal/svc"
	"server-cluster/mesh/admin/app/repo"
	"server-cluster/share/consts"
	"server-cluster/share/dto"
	"server-cluster/share/errs"
	"server-cluster/share/model"
	"server-cluster/share/pb"
)

// ClientVersionList 客户端版本分页列表。
func ClientVersionList(req dto.ClientVersionListReq) ([]model.ClientVersion, int64, error) {
	return repo.R().ListClientVersions(repo.ClientVersionQuery{
		Version:  req.Version,
		Platform: req.Platform,
		Status:   req.Status,
		Order:    req.Order,
	}, req.PageIndex, req.PageSize)
}

// ClientVersionGet 客户端版本详情。
func ClientVersionGet(id int) (*model.ClientVersion, error) {
	return repo.R().GetClientVersion(id)
}

// ClientVersionCreate 新增客户端版本。
func ClientVersionCreate(req dto.ClientVersionSaveReq) (*model.ClientVersion, error) {
	m, err := clientVersionModel(req, 0)
	if err != nil {
		return nil, err
	}
	if err := repo.R().CreateClientVersion(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ClientVersionUpdate 修改客户端版本。
func ClientVersionUpdate(id int, req dto.ClientVersionSaveReq) (*model.ClientVersion, error) {
	if id == 0 {
		return nil, codes.NewCode(400, "版本ID不能为空").Err()
	}
	m, err := clientVersionModel(req, id)
	if err != nil {
		return nil, err
	}
	m.VersionId = id
	if err := repo.R().UpdateClientVersion(m); err != nil {
		return nil, err
	}
	// 重新读一次，返回带创建/更新时间的完整记录。
	return repo.R().GetClientVersion(id)
}

// ClientVersionDelete 批量删除客户端版本。
func ClientVersionDelete(ids []int) error { return repo.R().DeleteClientVersions(ids) }

// ClientVersionUpdateStatus 启用/停用客户端版本。
func ClientVersionUpdateStatus(req dto.ClientVersionStatusReq) error {
	if req.VersionId == 0 {
		return codes.NewCode(400, "版本ID不能为空").Err()
	}
	return repo.R().UpdateClientVersionStatus(req.VersionId, req.Status)
}

// clientVersionModel 校验请求并转模型；excludeId 用于修改时排除自身的重号检查。
func clientVersionModel(req dto.ClientVersionSaveReq, excludeId int) (*model.ClientVersion, error) {
	version := strings.TrimSpace(req.Version)
	platform := strings.TrimSpace(req.Platform)
	if version == "" {
		return nil, codes.NewCode(400, "版本号不能为空").Err()
	}
	if platform == "" {
		return nil, codes.NewCode(400, "平台不能为空").Err()
	}
	exists, err := repo.R().ExistsClientVersion(version, platform, excludeId)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, codes.NewCode(400, "该平台下已存在相同版本号").Err()
	}

	status := req.Status
	if status == "" {
		status = consts.StatusEnabled
	}

	return &model.ClientVersion{
		Version:     version,
		Platform:    platform,
		DownloadURL: strings.TrimSpace(req.DownloadURL),
		ForceUpdate: req.ForceUpdate,
		Status:      status,
		ReleaseNote: req.ReleaseNote,
	}, nil
}

const (
	simDefaultLoginAddr = "http://188.18.23.72:8080"
	simLoginTimeout     = 5 * time.Second
	simDialTimeout      = 5 * time.Second
	simStepTimeout      = 5 * time.Second
	simSessionMaxAge    = 30 * time.Minute
)

// simFrame 网关下发的一帧业务报文。
type simFrame struct {
	route   int32
	seq     int32
	payload []byte
}

// simSession 一次模拟客户端会话，持有网关连接与自增序号。
type simSession struct {
	id        string
	token     string
	gateAddr  string
	playerID  int64
	createdAt time.Time

	mu       sync.Mutex
	conn     network.Conn
	packer   packet.Packer
	seq      int32
	inbox    chan simFrame
	verified bool
	entered  bool
}

// simStore 会话表。
type simStore struct {
	mu       sync.Mutex
	sessions map[string]*simSession
}

var simSessions = &simStore{sessions: make(map[string]*simSession)}

func init() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			simSessions.sweep(simSessionMaxAge)
		}
	}()
}

func (st *simStore) put(s *simSession) {
	st.mu.Lock()
	st.sessions[s.id] = s
	st.mu.Unlock()
}

func (st *simStore) get(id string) (*simSession, error) {
	st.mu.Lock()
	s, ok := st.sessions[id]
	st.mu.Unlock()
	if !ok {
		return nil, codes.NewCode(400, "会话不存在或已结束，请重新登录").Err()
	}
	return s, nil
}

func (st *simStore) remove(id string) {
	st.mu.Lock()
	delete(st.sessions, id)
	st.mu.Unlock()
}

// sweep 清理超时会话：先在锁内摘除，再逐个关闭连接，避免与步骤调用互相等锁。
func (st *simStore) sweep(maxAge time.Duration) {
	st.mu.Lock()
	stale := make([]*simSession, 0)
	for id, s := range st.sessions {
		if time.Since(s.createdAt) > maxAge {
			delete(st.sessions, id)
			stale = append(stale, s)
		}
	}
	st.mu.Unlock()

	for _, s := range stale {
		s.mu.Lock()
		if s.conn != nil {
			_ = s.conn.Close(true)
			s.conn = nil
		}
		s.mu.Unlock()
	}
}

// SimOptions 返回模拟器默认参数（页面可覆盖）。
func SimOptions() dto.SimOptionsResp {
	loginAddr := strings.TrimSpace(etc.Get("etc.sim.loginAddr").String())
	if loginAddr == "" {
		loginAddr = simDefaultLoginAddr
	}
	return dto.SimOptionsResp{
		LoginAddr: loginAddr,
	}
}

// SimLogin 走登录服的注册/登录接口，拿到 token 与网关地址并建立会话。
func SimLogin(req dto.SimLoginReq) (*dto.SimLoginResp, error) {
	mode := req.Mode
	if mode == "" {
		mode = "login"
	}
	if mode != "login" && mode != "register" {
		return nil, codes.NewCode(400, "mode 只能是 login 或 register").Err()
	}
	if strings.TrimSpace(req.Username) == "" || req.Password == "" {
		return nil, codes.NewCode(400, "账号和密码不能为空").Err()
	}

	loginAddr := strings.TrimRight(strings.TrimSpace(req.LoginAddr), "/")
	if loginAddr == "" {
		loginAddr = strings.TrimRight(SimOptions().LoginAddr, "/")
	}
	if !strings.HasPrefix(loginAddr, "http://") && !strings.HasPrefix(loginAddr, "https://") {
		loginAddr = "http://" + loginAddr
	}

	path := "/api/v1/player/login"
	body := map[string]any{
		"loginType": dto.LoginTypePassword,
		"username":  req.Username,
		"password":  req.Password,
	}
	displayBody := map[string]any{
		"loginType": dto.LoginTypePassword,
		"username":  req.Username,
		"password":  "******",
	}
	if mode == "register" {
		path = "/api/v1/player/register"
		body = map[string]any{"username": req.Username, "password": req.Password}
		displayBody = map[string]any{"username": req.Username, "password": "******"}
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, errs.NewError(500, err.Error())
	}

	start := time.Now()
	httpReq, err := http.NewRequest(http.MethodPost, loginAddr+path, bytes.NewReader(payload))
	if err != nil {
		return nil, errs.NewError(400, "登录服地址不合法: "+err.Error())
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: simLoginTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, errs.NewError(502, "请求登录服失败: "+err.Error())
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, errs.NewError(502, "读取登录服响应失败: "+err.Error())
	}

	var env struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Token    string `json:"token"`
			GateAddr string `json:"gateAddr"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, errs.NewError(502, "登录服响应无法解析: "+err.Error())
	}
	if resp.StatusCode != http.StatusOK || env.Code != 0 {
		msg := env.Message
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return nil, errs.NewError(400, fmt.Sprintf("登录失败(code=%d): %s", env.Code, msg))
	}
	if env.Data.Token == "" || env.Data.GateAddr == "" {
		return nil, codes.NewCode(502, "登录服未返回 token 或网关地址").Err()
	}

	gateAddr := env.Data.GateAddr

	playerID, _ := svc.ParseToken(env.Data.Token)

	s := &simSession{
		id:        fmt.Sprintf("sim-%d", svc.NextID()),
		token:     env.Data.Token,
		gateAddr:  gateAddr,
		playerID:  playerID,
		createdAt: time.Now(),
		packer: packet.NewPacker(
			packet.WithByteOrder(binary.BigEndian),
			packet.WithRouteBytes(4),
			packet.WithSeqBytes(4),
			packet.WithBufferBytes(5000),
			packet.WithHeartbeatTime(true),
		),
		inbox: make(chan simFrame, 32),
	}
	simSessions.put(s)

	reqJSON, _ := json.Marshal(map[string]any{
		"method": http.MethodPost,
		"url":    loginAddr + path,
		"body":   displayBody,
	})
	respJSON, _ := json.Marshal(map[string]any{
		"code":    env.Code,
		"message": env.Message,
		"data":    map[string]any{"token": env.Data.Token, "gateAddr": env.Data.GateAddr},
	})

	return &dto.SimLoginResp{
		SessionID: s.id,
		PlayerID:  playerID,
		Token:     env.Data.Token,
		GateAddr:  gateAddr,
		TakenMs:   time.Since(start).Milliseconds(),
		Request:   reqJSON,
		Response:  respJSON,
	}, nil
}

// SimConnect 与网关建立 WebSocket 连接。
func SimConnect(sessionID string) (*dto.SimConnectResp, error) {
	s, err := simSessions.get(sessionID)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn != nil {
		return nil, codes.NewCode(400, "网关已连接，无需重复连接").Err()
	}

	url := simWSURL(s.gateAddr)
	packer, inbox := s.packer, s.inbox
	client := ws.NewClient(
		ws.WithClientHeartbeatInterval(10*time.Second),
		ws.WithClientDialTimeout(simDialTimeout),
	)
	client.OnReceive(func(_ network.Conn, buf buffer.Buffer) {
		route, seq, msg, err := packer.UnpackMessage(buf)
		if err != nil {
			buf.Release()
			return
		}
		frame := simFrame{route: route, seq: seq, payload: append([]byte(nil), msg.Bytes()...)}
		buf.Release()
		select {
		case inbox <- frame:
		default:
		}
	})

	start := time.Now()
	conn, err := client.Dial(url)
	if err != nil {
		return nil, errs.NewError(502, "连接网关失败("+url+"): "+err.Error())
	}
	s.conn = conn

	remoteAddr, localAddr := "", ""
	if addr, err := conn.RemoteAddr(); err == nil {
		remoteAddr = addr.String()
	}
	if addr, err := conn.LocalAddr(); err == nil {
		localAddr = addr.String()
	}

	return &dto.SimConnectResp{
		GateAddr:   s.gateAddr,
		RemoteAddr: remoteAddr,
		LocalAddr:  localAddr,
		TakenMs:    time.Since(start).Milliseconds(),
	}, nil
}

// SimStep 执行首包验证 / 进入游戏 / 退出游戏，并返回收发报文。
func SimStep(req dto.SimStepReq) (*dto.SimStepResp, error) {
	s, err := simSessions.get(req.SessionID)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn == nil {
		return nil, codes.NewCode(400, "请先连接网关").Err()
	}

	var (
		route int32
		reqP  proto.Message
		respP proto.Message
	)

	switch req.Step {
	case "verify":
		route = int32(pb.RouteCode_ROUTE_CODE_VERIFY)
		reqP = &pb.VerifyReq{Token: s.token}
		respP = &pb.VerifyResp{}
	case "enter":
		if !s.verified {
			return nil, codes.NewCode(400, "请先完成首包验证").Err()
		}
		route = int32(pb.RouteCode_ROUTE_CODE_ENTER_GAME)
		reqP = &pb.EnterGameReq{Token: s.token}
		respP = &pb.EnterGameResp{}
	case "leave":
		if !s.entered {
			return nil, codes.NewCode(400, "请先进入游戏").Err()
		}
		route = int32(pb.RouteCode_ROUTE_CODE_LEAVE_GAME)
		reqP = &pb.LeaveGameReq{}
		respP = &pb.LeaveGameResp{}
	default:
		return nil, errs.NewError(400, "未知步骤: "+req.Step)
	}

	payload, err := proto.Marshal(reqP)
	if err != nil {
		return nil, errs.NewError(500, "序列化请求失败: "+err.Error())
	}

	s.seq++
	seq := s.seq
	buf, err := s.packer.PackMessage(route, seq, payload)
	if err != nil {
		return nil, errs.NewError(500, "打包报文失败: "+err.Error())
	}

	start := time.Now()
	if err := s.conn.Push(buf); err != nil {
		buf.Release()
		return nil, errs.NewError(502, "发送报文失败: "+err.Error())
	}

	timeout := simStepTimeout
	if req.Step == "leave" {
		// 退出回包与断线存在竞态，稍等即可，不必占满整个超时窗口。
		timeout = time.Second
	}
	frame, received := s.waitFrame(route, seq, timeout)
	if !received && req.Step != "leave" {
		return nil, errs.NewError(504, fmt.Sprintf(
			"网关未在 %s 内响应 route=%d seq=%d（连接可能已断开，或服务端校验失败未能回包）",
			simStepTimeout, route, seq))
	}

	respJSON := json.RawMessage(`{}`)
	if received {
		if err := proto.Unmarshal(frame.payload, respP); err != nil {
			return nil, errs.NewError(502, fmt.Sprintf("网关响应解析失败(route=%d seq=%d): %v", route, seq, err))
		}
		respJSON = simProtoJSON(respP)
	} else {
		// 退出游戏时服务端回包后立刻断开连接，回包与断线存在竞态，收不到回包也视为发送成功。
		respJSON = json.RawMessage(`{"sent":true,"responseReceived":false,"note":"网关已断开连接，未收到退出回包（服务端回包与断开存在竞态）"}`)
	}

	switch req.Step {
	case "verify":
		s.verified = true
	case "enter":
		s.entered = true
	case "leave":
		// 服务端在回包后断开连接，这里同步收尾并移除会话。
		_ = s.conn.Close(true)
		s.conn = nil
		simSessions.remove(s.id)
	}

	return &dto.SimStepResp{
		Step:     req.Step,
		Route:    route,
		Seq:      seq,
		Request:  simProtoJSON(reqP),
		Response: respJSON,
		TakenMs:  time.Since(start).Milliseconds(),
	}, nil
}

// SimClose 主动关闭会话（幂等）。
func SimClose(sessionID string) error {
	simSessions.mu.Lock()
	s, ok := simSessions.sessions[sessionID]
	delete(simSessions.sessions, sessionID)
	simSessions.mu.Unlock()
	if !ok {
		return nil
	}

	s.mu.Lock()
	if s.conn != nil {
		_ = s.conn.Close(true)
		s.conn = nil
	}
	s.mu.Unlock()

	return nil
}

// waitFrame 等待与请求 route/seq 匹配的响应帧。
func (s *simSession) waitFrame(route, seq int32, timeout time.Duration) (simFrame, bool) {
	deadline := time.After(timeout)
	for {
		select {
		case frame := <-s.inbox:
			if frame.route == route && frame.seq == seq {
				return frame, true
			}
		case <-deadline:
			return simFrame{}, false
		}
	}
}

// simProtoJSON 把 proto 消息转成页面可读的 JSON（含零值字段）。
func simProtoJSON(message proto.Message) json.RawMessage {
	data, err := protojson.MarshalOptions{UseProtoNames: false, EmitUnpopulated: true}.Marshal(message)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return data
}

// simWSURL 把登录服下发的网关地址补全成 WebSocket 地址。
func simWSURL(addr string) string {
	addr = strings.TrimSpace(addr)
	switch {
	case strings.HasPrefix(addr, "ws://"), strings.HasPrefix(addr, "wss://"):
	case strings.HasPrefix(addr, "http://"):
		addr = "ws://" + strings.TrimPrefix(addr, "http://")
	case strings.HasPrefix(addr, "https://"):
		addr = "wss://" + strings.TrimPrefix(addr, "https://")
	default:
		addr = "ws://" + addr
	}
	if !strings.Contains(strings.TrimPrefix(strings.TrimPrefix(addr, "ws://"), "wss://"), "/") {
		addr += "/"
	}
	return addr
}
