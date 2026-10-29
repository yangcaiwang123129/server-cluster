import request from '@/utils/request'
import type { ApiResponse, PageQuery, PageResult } from '@/types/api'
import type { Player, PlayerQuery, OnlinePlayer } from '@/types/admin'

/** Player management: read-only views over player_base and game-node presence. */

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
