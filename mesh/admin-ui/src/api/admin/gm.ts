import request from '@/utils/request'
import type { ApiResponse, PageResult } from '@/types/api'
import type {
  GameCatalogEntry,
  GmEquip,
  GmHero,
  GmHeroAttribute,
  GmHeroListResp,
  GmItem,
  GmMail,
  GmMailAttachment,
  GmRecord,
  GmUpgradeResult
} from '@/types/admin'

/**
 * GM 接口：admin 经 mesh 转发到 game 服，每个方法对应 game 侧 logic 的一个函数。
 * 请求统一以 playerId 定位玩家，后台自动选择玩家所在实例。
 */

/** Game config catalog for hero/equip/item lists (Luban tables), paginated. */
export function listGameCatalog(params: {
  type: 'hero' | 'equip' | 'item'
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

/** 物品容器列表，containerType 为 0 时返回全部物品容器。 */
export function listGmItem(params: { playerId: number; containerType?: number }) {
  return request<ApiResponse<GmItem[]>>({
    url: '/api/v1/gm/item',
    method: 'get',
    params
  })
}

/** 物品容器内移动物品。itemId 为雪花 ID 字符串（JSON 大 int64 会丢精度）。 */
export function moveGmItem(data: {
  playerId: number
  containerType: number
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
export function rotateGmItem(data: { playerId: number; containerType: number; itemId: string }) {
  return request<ApiResponse<null>>({
    url: '/api/v1/gm/item/rotate',
    method: 'post',
    data
  })
}

/** 物品容器一键整理。 */
export function sortGmItem(data: { playerId: number; containerType: number }) {
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
  containerType: number
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
  return request<ApiResponse<GmHeroListResp>>({
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

/** 英雄累计属性；heroId 为 0 时取出战英雄。 */
export function getGmHeroAttribute(params: { playerId: number; heroId?: number }) {
  return request<ApiResponse<GmHeroAttribute>>({
    url: '/api/v1/gm/hero/attribute',
    method: 'get',
    params
  })
}

/** 英雄升级（消耗背包升级素材）。 */
export function upgradeGmHero(data: { playerId: number; cfgId: number }) {
  return request<ApiResponse<GmUpgradeResult>>({
    url: '/api/v1/gm/hero/upgrade',
    method: 'post',
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

export function addGmEquip(data: { playerId: number; equipId: number; heroId?: number }) {
  return request<ApiResponse<GmEquip>>({
    url: '/api/v1/gm/equip',
    method: 'post',
    data
  })
}

export function updateGmEquip(data: {
  playerId: number
  id: string
  heroId?: number
  level: number
  quality: number
  breakthrough: number
  equipped: boolean
}) {
  return request<ApiResponse<GmEquip>>({
    url: '/api/v1/gm/equip',
    method: 'put',
    data
  })
}

/** 装备穿戴（同英雄同配置自动顶替；heroId 为 0 时取当前出战英雄）。 */
export function wearGmEquip(data: { playerId: number; id: string; heroId?: number }) {
  return request<ApiResponse<GmEquip>>({
    url: '/api/v1/gm/equip/wear',
    method: 'post',
    data
  })
}

/** 装备卸载。 */
export function unwearGmEquip(data: { playerId: number; id: string }) {
  return request<ApiResponse<GmEquip>>({
    url: '/api/v1/gm/equip/unwear',
    method: 'post',
    data
  })
}

/** 装备升级（消耗背包升级素材；equipId 为装备实例ID 字符串）。 */
export function upgradeGmEquip(data: { playerId: number; equipId: string }) {
  return request<ApiResponse<GmUpgradeResult>>({
    url: '/api/v1/gm/equip/upgrade',
    method: 'post',
    data
  })
}

/** 装备升品（消耗升品素材；equipId 为装备实例ID 字符串）。 */
export function upQualityGmEquip(data: { playerId: number; equipId: string }) {
  return request<ApiResponse<GmUpgradeResult>>({
    url: '/api/v1/gm/equip/up-quality',
    method: 'post',
    data
  })
}

/** 装备突破（消耗突破素材；equipId 为装备实例ID 字符串）。 */
export function breakthroughGmEquip(data: { playerId: number; equipId: string }) {
  return request<ApiResponse<GmUpgradeResult>>({
    url: '/api/v1/gm/equip/breakthrough',
    method: 'post',
    data
  })
}

// ── 邮件 ──────────────────────────────────────────────────────────

/** 玩家邮件列表（按未读优先、创建时间倒序排序）。 */
export function listGmMail(params: { playerId: number }) {
  return request<ApiResponse<GmMail[]>>({
    url: '/api/v1/gm/mail',
    method: 'get',
    params
  })
}

/** 后台发送邮件（attachments 为空则为纯通知）。 */
export function sendGmMail(data: {
  playerId: number
  title: string
  content?: string
  attachments?: GmMailAttachment[]
}) {
  return request<ApiResponse<GmMail>>({
    url: '/api/v1/gm/mail',
    method: 'post',
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
