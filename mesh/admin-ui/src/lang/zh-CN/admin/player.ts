/**
 * Player management: read-only views over accounts and live presence.
 * Menu titles come from the database, not from here.
 */
export default {
  // ── Search ──────────────────────────────────────────────────────
  playerId: '玩家ID',
  playerIdPlaceholder: '搜索玩家ID（精确）',
  username: '账号',
  usernamePlaceholder: '按账号搜索',
  phoneNumberPlaceholder: '按手机号搜索（模糊）',
  status: '状态',
  statusNormal: '正常',
  statusBanned: '封号',

  // ── Columns ─────────────────────────────────────────────────────
  nickname: '昵称',
  zone: '区号',
  phoneNumber: '手机号',
  level: '等级',
  vipLevel: 'VIP等级',
  currencies: '货币',
  coupon: '点券',
  gold: '金',
  coin: '币',
  createdAt: '注册时间',
  registerIp: '注册IP',

  // ── Toolbar / empty state ────────────────────────────────────────
  onlineTip: '查看当前在线玩家',
  emptyTip: '没有匹配的玩家',

  // ── Copy feedback ────────────────────────────────────────────────
  copyPlayerId: '复制玩家ID',
  copyIp: '复制IP',
  copyOk: '已复制',
  copyFailed: '复制失败，请手动复制',

  // ── Online dialog ───────────────────────────────────────────────
  online: '在线玩家',
  onlineEmpty: '当前没有在线玩家',
  enterTime: '进入时间',
  lobbyNodeId: '大厅服务器',

  // ── Kick ────────────────────────────────────────────────────────
  kick: '踢下线',
  banKick: '封号踢下线',
  kickConfirm: '确认踢玩家 {playerId} 下线？',
  banKickConfirm: '确认封号并踢玩家 {playerId} 下线？',
  kickOk: '已踢下线',
  kickAll: '批量踢下线（{n}）',
  kickAllConfirm: '确认批量踢下线当前 {n} 位在线玩家？',
  kickAllOk: '全部踢下线成功',
  kickAllPartial: '部分成功（{ok}/{total}）',

  // ── Ban / Unban / ResetPwd ──────────────────────────────────────
  ban: '封号',
  banDuration: '封号时长',
  banReason: '封号原因',
  banReasonRequired: '请输入封号原因',
  banReasonPlaceholder: '请输入封号原因...',
  banOk: '已封号',
  unban: '解封',
  unbanConfirm: '确认解封玩家 {playerId}？',
  unbanOk: '已解封',
  resetPwd: '重置密码',
  newPassword: '新密码',
  newPasswordPlaceholder: '6-32 位',
  pwdRequired: '请输入新密码',
  pwdLength: '密码长度 6-32 位',
  pwdOk: '密码已重置',

  // ── Online: LoginIP ──────────────────────────────────────────────
  loginIp: '登录 IP'
}
