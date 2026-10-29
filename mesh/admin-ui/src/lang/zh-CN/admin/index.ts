import sysUser from './sys-user'
import sysRole from './sys-role'
import sysMenu from './sys-menu'
import sysConfig from './sys-config'
import dict from './dict'
import sysApi from './sys-api'
import sysOperLog from './sys-oper-log'
import sysLoginLog from './sys-login-log'
import player from './player'
import gameContainer from './game-container'
import gameHero from './game-hero'
import gameWeapon from './game-weapon'
import gameSkill from './game-skill'
import gameEquip from './game-equip'
import simulator from './simulator'
import clientVersion from './client-version'

/**
 * The admin section, one module per page under src/views/admin.
 *
 * The file names match those directories -- sys-user.ts for sys-user/ -- so
 * where a page's text lives needs no decision. The keys are camelCase because
 * they are read as `t('admin.sysUser.addTitle')`. A directory holding two pages
 * still gets one file: sys-config.ts carries the list page at its top level and
 * the settings form under `set`, dict.ts carries the types under `type` and the
 * entries under `data`.
 */
export default {
  sysUser,
  sysRole,
  sysMenu,
  sysConfig,
  dict,
  sysApi,
  sysOperLog,
  sysLoginLog,
  player,
  gameContainer,
  gameHero,
  gameWeapon,
  gameSkill,
  gameEquip,
  simulator,
  clientVersion
}
