/**
 * Dictionary text, keyed by what the backend already stores in English.
 *
 * `values` is keyed dict_type -> dict_value, `types` by dict_type. Both keys
 * are backend identifiers (sys_user_sex, DEFAULT, Y) rather than the Chinese
 * label, so a label edit in the admin UI cannot break a lookup.
 *
 * Only the dictionaries the backend seeds are listed here -- the sets must stay
 * in step with mesh/admin/app/seed.go, or a value renders untranslated.
 * No dict_value contains a dot: lang/backend.ts looks these up by plain object
 * access rather than vue-i18n's dotted paths, so a dot would be harmless, but it
 * is worth knowing the seed data is clean.
 */
export default {
  values: {
    sys_normal_disable: {
      '2': 'Normal', // 正常
      '1': 'Disabled' // 停用
    },
    sys_user_sex: {
      '1': 'Male', // 男
      '2': 'Female', // 女
      '0': 'Unknown' // 未知
    },
    sys_show_hide: {
      '0': 'Show', // 显示
      '1': 'Hide' // 隐藏
    },
    sys_yes_no: {
      Y: 'Yes', // 是
      N: 'No' // 否
    },
    sys_common_status: {
      '2': 'Normal', // 正常
      '1': 'Closed' // 关闭
    },
    sys_oper_type: {
      '1': 'Add', // 新增
      '2': 'Edit', // 修改
      '3': 'Delete' // 删除
    }
  },
  types: {
    sys_normal_disable: 'System Switch', // 系统开关
    sys_show_hide: 'Menu Visibility', // 菜单状态
    sys_user_sex: 'User Gender', // 用户性别
    sys_yes_no: 'Yes / No', // 系统是否
    sys_common_status: 'System Status', // 通用状态
    sys_oper_type: 'Operation Type' // 操作类型
  }
}
