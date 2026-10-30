import request from '@/utils/request'
import type { ApiResponse, PageResult } from '@/types/api'
import type { GameCatalogEntry, GmEquip, GmHero, GmItem, GmRecord, GmSkill, GmWeapon } from '@/types/admin'

/**
 * GM 接口：admin 经 mesh 转发到 game 服，每个方法对应 game 侧 logic 的一个函数。
 * 请求统一以 playerId 定位玩家，后台自动选择玩家所在实例。
 */

/** Game config catalog for hero/weapon/skill/equip/item lists (Luban tables), paginated. */
export function listGameCatalog(params: {
  type: 'hero' | 'weapon' | 'skill' | 'equip' | 'item'
  keyword?: string
  pageIndex: number
  pageSize: number
}) {
  return request<ApiResponse<PageResult<GameCatalogEntry>>>({
    url: '/api/v1/gm/catalog',
    method: 'get',
    params
  })
}

// ── 物品 ──────────────────────────────────────────────────────────

/** 物品容器列表，itemType 为 0 时返回全部物品容器。 */
export function listGmItem(params: { playerId: number; itemType?: number }) {
  return request<ApiResponse<GmItem[]>>({
    url: '/api/v1/gm/item',
    method: 'get',
    params
  })
}

/** 物品容器内移动物品。itemId 为雪花 ID 字符串（JSON 大 int64 会丢精度）。 */
export function moveGmItem(data: {
  playerId: number
  itemType: number
  itemId: string
  toX: number
  toY: number
  rotated: boolean
}) {
  return request<ApiResponse<null>>({
    url: '/api/v1/gm/item/move',
    method: 'post',
    data
  })
}

/** 物品容器内旋转物品。 */
export function rotateGmItem(data: { playerId: number; itemType: number; itemId: string }) {
  return request<ApiResponse<null>>({
    url: '/api/v1/gm/item/rotate',
    method: 'post',
    data
  })
}

/** 物品容器一键整理。 */
export function sortGmItem(data: { playerId: number; itemType: number }) {
  return request<ApiResponse<null>>({
    url: '/api/v1/gm/item/sort',
    method: 'post',
    data
  })
}

/** 物品容器间移动物品（返回变更后的源/目标容器）。 */
export function transferGmItem(data: {
  playerId: number
  fromType: number
  toType: number
  itemId: string
  toX: number
  toY: number
  rotated: boolean
}) {
  return request<ApiResponse<GmItem[]>>({
    url: '/api/v1/gm/item/transfer',
    method: 'post',
    data
  })
}

/** 向物品容器放入物品（返回变更后的容器）。 */
export function addGmItem(data: {
  playerId: number
  itemType: number
  itemCfgId: number
  count: number
  gridX: number
  gridY: number
  rotated: boolean
}) {
  return request<ApiResponse<GmItem[]>>({
    url: '/api/v1/gm/item/add',
    method: 'post',
    data
  })
}

// ── 英雄 ──────────────────────────────────────────────────────────

export function listGmHero(params: { playerId: number }) {
  return request<ApiResponse<GmHero[]>>({
    url: '/api/v1/gm/hero',
    method: 'get',
    params
  })
}

export function addGmHero(data: { playerId: number; heroId: number }) {
  return request<ApiResponse<GmHero>>({
    url: '/api/v1/gm/hero',
    method: 'post',
    data
  })
}

export function updateGmHero(data: {
  playerId: number
  heroId: number
  level: number
  exp: number
  star: number
  skinId: number
  equipped: boolean
}) {
  return request<ApiResponse<GmHero>>({
    url: '/api/v1/gm/hero',
    method: 'put',
    data
  })
}

// ── 武器 ──────────────────────────────────────────────────────────

export function listGmWeapon(params: { playerId: number }) {
  return request<ApiResponse<GmWeapon[]>>({
    url: '/api/v1/gm/weapon',
    method: 'get',
    params
  })
}

export function addGmWeapon(data: { playerId: number; weaponId: number }) {
  return request<ApiResponse<GmWeapon>>({
    url: '/api/v1/gm/weapon',
    method: 'post',
    data
  })
}

export function updateGmWeapon(data: {
  playerId: number
  weaponId: number
  level: number
  durability: number
  equipped: boolean
}) {
  return request<ApiResponse<GmWeapon>>({
    url: '/api/v1/gm/weapon',
    method: 'put',
    data
  })
}

// ── 技能 ──────────────────────────────────────────────────────────

export function listGmSkill(params: { playerId: number }) {
  return request<ApiResponse<GmSkill[]>>({
    url: '/api/v1/gm/skill',
    method: 'get',
    params
  })
}

export function addGmSkill(data: { playerId: number; skillId: number }) {
  return request<ApiResponse<GmSkill>>({
    url: '/api/v1/gm/skill',
    method: 'post',
    data
  })
}

export function updateGmSkill(data: { playerId: number; skillId: number; level: number; equipped: boolean }) {
  return request<ApiResponse<GmSkill>>({
    url: '/api/v1/gm/skill',
    method: 'put',
    data
  })
}

// ── 装备 ──────────────────────────────────────────────────────────

export function listGmEquip(params: { playerId: number }) {
  return request<ApiResponse<GmEquip[]>>({
    url: '/api/v1/gm/equip',
    method: 'get',
    params
  })
}

export function addGmEquip(data: { playerId: number; equipId: number }) {
  return request<ApiResponse<GmEquip>>({
    url: '/api/v1/gm/equip',
    method: 'post',
    data
  })
}

export function updateGmEquip(data: { playerId: number; equipId: number; level: number; equipped: boolean }) {
  return request<ApiResponse<GmEquip>>({
    url: '/api/v1/gm/equip',
    method: 'put',
    data
  })
}

// ── 打点记录 ──────────────────────────────────────────────────────

/** 获得/消耗记录分页列表（读 game_record 库）。 */
export function listGmRecord(params: {
  action: number
  module: 'item' | 'equip' | 'weapon' | 'skill' | 'hero'
  playerId?: number
  cfgId?: number
  begin?: number
  end?: number
  pageIndex: number
  pageSize: number
}) {
  return request<ApiResponse<PageResult<GmRecord>>>({
    url: '/api/v1/gm/record',
    method: 'get',
    params
  })
}
