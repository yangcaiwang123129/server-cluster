/**
 * Domain types for the admin module.
 *
 * Fields are named as the Go side sends them. Most are optional because the
 * list endpoint returns a subset of what the detail endpoint does, and the same
 * type describes a row, a loaded record and a form being filled in.
 */

/** A user, as returned by /api/v1/sys-user. */
export interface SysUser {
  userId?: number
  username?: string
  nickName?: string
  password?: string
  phone?: string
  email?: string
  sex?: string
  /** '1' disabled, '2' enabled. A string, not a boolean -- the backend's choice. */
  status?: string
  avatar?: string
  remark?: string
  roleId?: number
  createdAt?: string
}

/** Filters the user list accepts, on top of the paging keys. */
export interface SysUserQuery {
  username?: string
  phone?: string
  status?: string
}

/** A role. */
export interface SysRole {
  roleId?: number
  roleName?: string
  roleKey?: string
  roleSort?: number
  status?: string
  remark?: string
  menuIds?: number[]
  createdAt?: string
}

export interface SysRoleQuery {
  roleName?: string
  roleKey?: string
  status?: string
}

/** A menu entry. The route table is generated from these; see stores/permission. */
export interface SysMenu {
  menuId?: number
  menuName?: string
  parentId?: number
  /** 'M' directory, 'C' page, 'F' button-level permission. */
  menuType?: string
  path?: string
  component?: string
  icon?: string
  sort?: number
  /** '0' shown, anything else hidden. Compared loosely -- the backend sends strings. */
  visible?: string
  isFrame?: string
  noCache?: boolean
  /** Permission code, in the 模块:资源:操作 form. */
  permission?: string
  /** Display title, which is what the tree shows -- menuName is the route name. */
  title?: string
  action?: string
  /** Ids of the backend routes this menu grants; `sysApi` is the same set, expanded. */
  apis?: number[]
  sysApi?: SysApi[]
  children?: SysMenu[]
  createdAt?: string
}

export interface SysMenuQuery {
  /** The tree is searched by display title, not by route name. */
  title?: string
  menuName?: string
  visible?: string
}

/** A registered backend route, managed under 接口管理. */
export interface SysApi {
  id?: number
  path?: string
  handle?: string
  title?: string
  type?: string
  action?: string
  createdAt?: string
}

export interface SysApiQuery {
  title?: string
  path?: string
  action?: string
  /** 'SYS' for infrastructure routes, 'BUS' for the grantable ones. */
  type?: string
}

/** A configuration entry. `configKey` is what getConfigKey looks up. */
export interface SysConfig {
  configName?: string
  configKey?: string
  configValue?: string
  configType?: string
  isFrontend?: string
  remark?: string
  createdAt?: string
}

export interface SysConfigQuery {
  configName?: string
  configKey?: string
  configType?: string
}

/** A dictionary type; its entries are SysDictData. */
export interface SysDictType {
  /**
   * `id` on the wire. The column is dict_id, but the struct tags it `json:"id"`,
   * so a page reading dictId gets undefined -- which is what the type used to
   * say, and why the data page's type picker keyed every option on undefined.
   */
  id?: number
  dictName?: string
  dictType?: string
  /** '1' disabled, '2' normal. Number on the wire; see the api module. */
  status?: string
  remark?: string
  createdAt?: string
}

export interface SysDictTypeQuery {
  dictName?: string
  dictType?: string
  status?: string
}

/** One entry of a dictionary. */
export interface SysDictData {
  dictCode?: number
  dictSort?: number
  dictLabel?: string
  dictValue?: string
  dictType?: string
  cssClass?: string
  listClass?: string
  isDefault?: string
  status?: string
  remark?: string
  createdAt?: string
}

export interface SysDictDataQuery {
  dictType?: string
  dictLabel?: string
  status?: string
}

/** Filters the login audit accepts. */
export interface SysLoginLogQuery {
  username?: string
  status?: string
  ipaddr?: string
}

/** Filters the operation audit accepts. */
export interface SysOperaLogQuery {
  operUrl?: string
  status?: string
  beginTime?: string
  endTime?: string
}

/** An audit row: one login attempt. */
export interface SysLoginLog {
  /** models.Model serialises the primary key as `id` -- there is no infoId. */
  id?: number
  username?: string
  ipaddr?: string
  loginLocation?: string
  browser?: string
  os?: string
  platform?: string
  status?: string
  msg?: string
  loginTime?: string
  createdAt?: string
}

/** An audit row: one write operation. */
export interface SysOperaLog {
  id?: number
  title?: string
  businessType?: string
  method?: string
  requestMethod?: string
  operUrl?: string
  operIp?: string
  operName?: string
  status?: string
  operTime?: string
  operLocation?: string
  operParam?: string
  jsonResult?: string
  latencyTime?: string
  createdAt?: string
}

/** Filters the player list accepts. */
export interface PlayerQuery {
  username?: string
}

/** A player account row from player_base. Timestamps are unix milliseconds. */
export interface Player {
  playerId?: number
  username?: string
  zone?: number
  phoneNumber?: number
  createdAt?: number
}

/** A player currently online, as reported by the game node. */
export interface OnlinePlayer {
  playerId?: number
  enterTime?: number
  /** game node instance the player resides on */
  nodeId?: string
}

/** Filters the Luban item catalog accepts. */
export interface ItemCfgQuery {
  keyword?: string
}

/** A row from the Luban itemconfig table. */
export interface ItemCfg {
  id?: number
  name?: string
  desc?: string
  type?: number
  iconKey?: string
  /** footprint in cells */
  width?: number
  height?: number
  rarity?: number
  maxCount?: number
  overlayMaxCount?: number
  storageType?: number
  price?: number
  sellingPrice?: number
  usedInBackpack?: boolean
}

/** GM 容器内物品（含配置展示信息）。itemId 为雪花 ID，后端按字符串下发。 */
export interface GmContainerItem {
  itemId?: string
  itemCfgId?: number
  count?: number
  gridX?: number
  gridY?: number
  rotated?: boolean
  name?: string
  iconKey?: string
  /** occupied cells in the current orientation */
  width?: number
  height?: number
  rarity?: number
  maxCount?: number
}

export interface GmContainer {
  playerId?: number
  containerType?: number
  gridW?: number
  gridH?: number
  version?: number
  items?: GmContainerItem[]
}

export interface GmHero {
  heroId?: number
  level?: number
  exp?: number
  star?: number
  skinId?: number
  equipped?: boolean
  /** config-derived display info */
  name?: string
  rarity?: number
}

export interface GmWeapon {
  weaponId?: number
  level?: number
  durability?: number
  equipped?: boolean
  /** config-derived display info */
  name?: string
  iconKey?: string
}

export interface GmSkill {
  skillId?: number
  level?: number
  equipped?: boolean
  /** config-derived display info */
  name?: string
  desc?: string
}

export interface GmEquip {
  equipId?: number
  level?: number
  equipped?: boolean
  /** config-derived display info */
  name?: string
  iconKey?: string
  desc?: string
}

/** A row from the game config catalog (hero/weapon/skill/equip), for pickers. */
export interface GameCatalogEntry {
  id?: number
  name?: string
  iconKey?: string
  desc?: string
  rarity?: number
}

/** 客户端模拟器默认参数。 */
export interface SimOptions {
  loginAddr?: string
  gateAddr?: string
}

/** 模拟客户端登录/注册参数。 */
export interface SimLoginPayload {
  mode: 'login' | 'register'
  username: string
  password: string
  loginAddr?: string
  gateAddr?: string
}

/** 登录/注册结果，request/response 为展示用报文。 */
export interface SimLoginResult {
  sessionId: string
  playerId: number
  token: string
  gateAddr: string
  takenMs: number
  request?: unknown
  response?: unknown
}

/** 连接网关结果。 */
export interface SimConnectResult {
  gateAddr: string
  remoteAddr?: string
  localAddr?: string
  takenMs: number
}

/** 首包验证/进入/退出游戏结果。 */
export interface SimStepResult {
  step: string
  route: number
  seq: number
  request?: unknown
  response?: unknown
  takenMs: number
}

/** 客户端版本（安装包清单）。 */
export interface ClientVersion {
  versionId?: number
  version?: string
  /** android / ios / windows */
  platform?: string
  downloadUrl?: string
  forceUpdate?: boolean
  /** '1' disabled, '2' normal. */
  status?: string
  releaseNote?: string
  createdAt?: string
  updatedAt?: string
}

export interface ClientVersionQuery {
  version?: string
  platform?: string
  status?: string
}
