import request from '@/utils/request'
import type { ApiResponse, PageQuery, PageResult } from '@/types/api'
import type {
  ItemCfg,
  ItemCfgQuery,
  PlayerContainer
} from '@/types/admin'

/**
 * Player item management: the Luban item catalog plus the grid operations on a
 * player's backpack/stash/secure container.
 */

/** Item catalog, paginated in memory by the admin server. */
export function listItem(query: ItemCfgQuery & Partial<PageQuery>) {
  return request<ApiResponse<PageResult<ItemCfg>>>({
    url: '/api/v1/item',
    method: 'get',
    params: query
  })
}

/** One container of one player. An uncreated player resolves to a default empty grid. */
export function getPlayerContainer(params: { playerId: number; containerType: number }) {
  return request<ApiResponse<PlayerContainer>>({
    url: '/api/v1/player/container',
    method: 'get',
    params
  })
}

/** Place a new item instance on the grid. Every mutation answers with the updated container. */
export function addContainerItem(data: {
  playerId: number
  containerType: number
  itemCfgId: number
  count: number
  gridX: number
  gridY: number
  rotated: boolean
}) {
  return request<ApiResponse<PlayerContainer>>({
    url: '/api/v1/player/container/item',
    method: 'post',
    data
  })
}

/** Move/rotate an existing item instance. */
export function moveContainerItem(data: {
  playerId: number
  containerType: number
  itemId: number
  gridX: number
  gridY: number
  rotated: boolean
}) {
  return request<ApiResponse<PlayerContainer>>({
    url: '/api/v1/player/container/item',
    method: 'put',
    data
  })
}

/** Auto-sort the container. */
export function sortPlayerContainer(data: { playerId: number; containerType: number }) {
  return request<ApiResponse<PlayerContainer>>({
    url: '/api/v1/player/container/sort',
    method: 'post',
    data
  })
}
