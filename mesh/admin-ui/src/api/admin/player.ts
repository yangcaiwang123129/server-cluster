import request from '@/utils/request'
import type { ApiResponse, PageQuery, PageResult } from '@/types/api'
import type { Player, PlayerQuery, OnlinePlayer, PlayerBanReq, PlayerUnbanReq, PlayerResetPasswordReq } from '@/types/admin'

/** Player management: read-only views over player and game-node presence. */

export function listPlayer(query: PlayerQuery & Partial<PageQuery>) {
  return request<ApiResponse<PageResult<Player>>>({
    url: '/api/v1/gm/player',
    method: 'get',
    params: query
  })
}

/** One-shot list, not paginated: the game node returns everyone online. */
export function listOnlinePlayer() {
  return request<ApiResponse<OnlinePlayer[]>>({
    url: '/api/v1/gm/player/online',
    method: 'get'
  })
}

/** Kick a player offline on its game node. kickType mirrors pb.KickType: 1 active, 2 ban. */
export function kickPlayer(data: { playerId: number; nodeId: string; kickType: number }) {
  return request<ApiResponse<null>>({
    url: '/api/v1/gm/player/kick',
    method: 'post',
    data
  })
}

/** Ban a player: write ban fields + kick if online. duration in seconds, 0 = permanent. */
export function banPlayer(data: PlayerBanReq) {
  return request<ApiResponse<null>>({
    url: '/api/v1/gm/player/ban',
    method: 'post',
    data
  })
}

/** Unban a player. */
export function unbanPlayer(data: PlayerUnbanReq) {
  return request<ApiResponse<null>>({
    url: '/api/v1/gm/player/unban',
    method: 'post',
    data
  })
}

/** Reset player password (admin-side, MD5-hashed server-side). */
export function resetPlayerPassword(data: PlayerResetPasswordReq) {
  return request<ApiResponse<null>>({
    url: '/api/v1/gm/player/reset-password',
    method: 'post',
    data
  })
}
