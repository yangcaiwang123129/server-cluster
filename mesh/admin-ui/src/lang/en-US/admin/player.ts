/**
 * Player management: read-only views over accounts and live presence.
 * Menu titles come from the database, not from here.
 */
export default {
  playerId: 'Player ID',
  playerIdPlaceholder: 'Search by exact player ID',
  username: 'Username',
  usernamePlaceholder: 'Search by username',
  phoneNumberPlaceholder: 'Search by phone (prefix/partial)',
  status: 'Status',
  statusNormal: 'Normal',
  statusBanned: 'Banned',

  nickname: 'Nickname',
  zone: 'Zone',
  phoneNumber: 'Phone Number',
  level: 'Level',
  vipLevel: 'VIP',
  currencies: 'Currency',
  coupon: 'Coupon',
  gold: 'Gold',
  coin: 'Coin',
  createdAt: 'Created At',
  registerIp: 'Register IP',

  onlineTip: 'View players currently online',
  emptyTip: 'No players match the filters',

  copyPlayerId: 'Copy player ID',
  copyIp: 'Copy IP',
  copyOk: 'Copied',
  copyFailed: 'Copy failed, please copy manually',

  online: 'Online Players',
  onlineEmpty: 'No players online right now',
  enterTime: 'Enter Time',
  lobbyNodeId: 'Lobby Node',

  kick: 'Kick',
  banKick: 'Ban & Kick',
  kickConfirm: 'Kick player {playerId} offline?',
  banKickConfirm: 'Ban and kick player {playerId} offline?',
  kickOk: 'Player kicked offline',
  kickAll: 'Kick All ({n})',
  kickAllConfirm: 'Kick all {n} online players offline?',
  kickAllOk: 'All players kicked',
  kickAllPartial: 'Partially kicked ({ok}/{total})',

  ban: 'Ban',
  banDuration: 'Duration',
  banReason: 'Reason',
  banReasonRequired: 'Please enter ban reason',
  banReasonPlaceholder: 'Enter ban reason...',
  banOk: 'Player banned',
  unban: 'Unban',
  unbanConfirm: 'Confirm unban player {playerId}?',
  unbanOk: 'Player unbanned',
  resetPwd: 'Reset Password',
  newPassword: 'New Password',
  newPasswordPlaceholder: '6-32 chars',
  pwdRequired: 'Please enter new password',
  pwdLength: 'Must be 6-32 chars',
  pwdOk: 'Password reset',

  loginIp: 'Login IP'
}
