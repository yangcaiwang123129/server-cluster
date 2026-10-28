/**
 * Player management: read-only views over accounts and live presence.
 * Menu titles come from the database, not from here.
 */
export default {
  // ── Search ──────────────────────────────────────────────────────
  username: '账号',
  usernamePlaceholder: '请输入账号',

  // ── Columns ─────────────────────────────────────────────────────
  playerId: '玩家ID',
  zone: '区号',
  phoneNumber: '手机号',
  createdAt: '创建时间',

  // ── Online dialog ───────────────────────────────────────────────
  online: '在线玩家',
  enterTime: '进入时间',

  // ── Kick ────────────────────────────────────────────────────────
  kick: '踢下线',
  banKick: '封号踢下线',
  kickConfirm: '确认踢玩家 {playerId} 下线？',
  banKickConfirm: '确认封号并踢玩家 {playerId} 下线？',
  kickOk: '已踢下线'
}
