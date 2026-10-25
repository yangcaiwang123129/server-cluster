import { Page, BrowserContext } from '@playwright/test'
import { mockCrud, json } from './support/crud'

/**
 * Fixtures for the mock-driven e2e suite.
 *
 * The pre-existing specs under tests/e2e require a live Go backend plus a
 * hand-obtained JWT, which means they cannot run unattended and never guarded
 * anything in CI. These fixtures stub the API at the network layer instead, so
 * the suite exercises the real router, the real dynamic-route generation and
 * the real keep-alive wiring while depending on nothing outside the repo.
 *
 * What is verified here is front-end behaviour under a known-good API contract
 * -- exactly what the Composition API / Pinia migration is at risk of breaking.
 */

export const ADMIN_TOKEN = 'e2e-fake-token'

/** Shape returned by GET /api/v1/getinfo */
export const userInfo = {
  code: 200,
  data: {
    roles: ['admin'],
    name: 'e2e-admin',
    avatar: 'http://localhost/avatar.png',
    introduction: 'end to end test user',
    permissions: [
      'admin:sysUser:add',
      'admin:sysUser:edit',
      // 'admin:sysUser:remove' is intentionally absent: it proves that
      // v-permisaction actually hides the delete button.
      'admin:sysUser:resetPassword',
      'admin:sysRole:add',
      'admin:sysRole:edit',
      'admin:sysRole:remove',
      'admin:sysConfig:add',
      'admin:sysConfig:edit',
      'admin:sysConfig:remove',
      'admin:sysMenu:add',
      'admin:sysMenu:edit',
      'admin:sysMenu:remove',
      'admin:sysLoginLog:remove',
      'admin:sysOperLog:query',
      'admin:sysOperLog:remove',
      // Without this the export button is not rendered at all, which is why
      // nothing had ever exercised this page's export. sysRole and sysConfig
      // export without a permission check, so they need no equivalent here.
      'admin:sysOperLog:export',
      'admin:sysDictType:add',
      'admin:sysDictType:edit',
      'admin:sysDictType:remove',
      'admin:sysDictData:add',
      'admin:sysDictData:edit',
      'admin:sysDictData:remove',
      'admin:sysApi:edit',
      'admin:sysConfigSet:list'
    ]
  }
}

/**
 * Shape returned by GET /api/v1/menurole.
 *
 * `visible: '0'` means shown -- the flag is inverted and compared loosely in
 * generaMenu, so anything else hides the entry.
 */
export const menuTree = {
  code: 200,
  data: [
    {
      // Test-only entry whose title has no language-pack key, standing in for
      // a menu an operator created in the admin UI.
      path: '/custom-page',
      component: 'Layout',
      visible: '0',
      menuName: 'CustomRoot',
      title: 'Custom',
      icon: 'star',
      noCache: false,
      children: [
        {
          path: 'index',
          component: '/admin/sys-user/index',
          visible: '0',
          menuName: 'CustomPage',
          title: 'CustomPage',
          icon: 'star',
          noCache: false,
          children: null
        }
      ]
    },
    {
      path: '/admin',
      component: 'Layout',
      visible: '0',
      menuName: 'AdminRoot',
      title: 'Admin',
      icon: 'star',
      noCache: false,
      children: [
        {
          path: 'sys-user',
          component: '/admin/sys-user/index',
          visible: '0',
          menuName: 'SysUserManage',
          title: 'User',
          icon: 'star',
          noCache: false,
          children: null
        },
        {
          // Test-only duplicate of sys-user: its menuName deliberately differs
          // from the component name, the way a menu renamed in the admin UI
          // does. Only loadView()'s runtime override keeps it cacheable.
          path: 'sys-user-renamed',
          component: '/admin/sys-user/index',
          visible: '1',
          menuName: 'SysUserRenamed',
          title: 'UserRenamed',
          icon: 'star',
          noCache: false,
          children: null
        },
        {
          path: 'sys-role',
          component: '/admin/sys-role/index',
          visible: '0',
          menuName: 'SysRoleManage',
          title: 'Role',
          icon: 'star',
          noCache: false,
          children: null
        },
        {
          path: 'sys-config',
          component: '/admin/sys-config/index',
          visible: '0',
          menuName: 'SysConfigManage',
          title: 'Config',
          icon: 'star',
          noCache: false,
          children: null
        },
        {
          path: 'sys-menu',
          component: '/admin/sys-menu/index',
          visible: '0',
          menuName: 'SysMenuManage',
          title: 'Menu',
          icon: 'star',
          noCache: false,
          children: null
        },
        {
          path: 'sys-config/set',
          component: '/admin/sys-config/set',
          visible: '0',
          menuName: 'SysConfigSet',
          title: 'ConfigSet',
          icon: 'star',
          noCache: false,
          children: null
        },
        {
          path: 'sys-api',
          component: '/admin/sys-api/index',
          visible: '0',
          menuName: 'SysApiManage',
          title: 'Api',
          icon: 'star',
          noCache: false,
          children: null
        },
        {
          path: 'dict',
          component: '/admin/dict/index',
          visible: '0',
          menuName: 'Dict',
          title: 'Dict',
          icon: 'star',
          noCache: false,
          children: null
        },
        {
          path: 'dict/data/:dictId',
          component: '/admin/dict/data',
          visible: '1',
          menuName: 'SysDictDataManage',
          title: 'DictData',
          icon: 'star',
          noCache: false,
          children: null
        },
        {
          path: 'sys-login-log',
          component: '/admin/sys-login-log/index',
          visible: '0',
          menuName: 'SysLoginLogManage',
          title: 'LoginLog',
          icon: 'star',
          noCache: false,
          children: null
        },
        {
          path: 'sys-oper-log',
          component: '/admin/sys-oper-log/index',
          visible: '0',
          menuName: 'OperLog',
          title: 'OperLog',
          icon: 'star',
          noCache: false,
          children: null
        }
      ]
    }
  ]
}

/** Rows served by GET /api/v1/sys-user */
export const userRows = [
  {
    userId: 1,
    username: 'admin',
    nickName: '超级管理员',
    phone: '13800000001',
    email: 'admin@example.com',
    sex: '0',
    status: '2',
    createdAt: '2026-08-01T10:00:00Z'
  },
  {
    userId: 2,
    username: 'tester',
    nickName: '测试员',
    phone: '13800000002',
    email: 'tester@example.com',
    sex: '1',
    status: '1',
    createdAt: '2026-08-02T10:00:00Z'
  }
]

/**
 * Rows served by GET /api/v1/sys-api. One SYS route (infrastructure, not
 * grantable), one BUS route, and one with no title -- the page draws all three
 * differently.
 */
export const sysApiRows = [
  {
    id: 10,
    handle: 'go-admin/app/admin/apis.SysUser.GetPage-fm',
    title: '用户列表',
    path: '/api/v1/sys-user',
    action: 'GET',
    type: 'BUS',
    createdAt: '2026-08-01T10:00:00Z'
  },
  {
    id: 11,
    handle: 'go-admin/app/admin/apis.SysMenu.Delete-fm',
    title: '菜单删除',
    path: '/api/v1/menu',
    action: 'DELETE',
    type: 'SYS',
    createdAt: '2026-08-02T10:00:00Z'
  },
  {
    id: 12,
    handle: 'go-admin/app/admin/apis.Unnamed-fm',
    title: '',
    path: '/api/v1/unnamed',
    action: 'POST',
    type: 'BUS',
    createdAt: '2026-08-03T10:00:00Z'
  }
]

/** Rows served by GET /api/v1/dict/type */
export const dictTypeRows = [
  {
    id: 1,
    dictName: '系统状态',
    dictType: 'sys_common_status',
    status: 2,
    remark: '正常/关闭',
    createdAt: '2026-08-01T10:00:00Z'
  },
  {
    id: 2,
    dictName: '用户性别',
    dictType: 'sys_user_sex',
    status: 2,
    remark: '男/女',
    createdAt: '2026-08-02T10:00:00Z'
  }
]

/** Rows served by GET /api/v1/dict/data */
export const dictDataRows = [
  {
    dictCode: 11,
    dictSort: 1,
    dictLabel: '正常',
    dictValue: '2',
    dictType: 'sys_common_status',
    status: 2,
    remark: '正常状态',
    createdAt: '2026-08-01T10:00:00Z'
  },
  {
    dictCode: 12,
    dictSort: 2,
    dictLabel: '关闭',
    dictValue: '1',
    dictType: 'sys_common_status',
    status: 1,
    remark: '关闭状态',
    createdAt: '2026-08-02T10:00:00Z'
  }
]

/** Rows served by GET /api/v1/sys-login-log */
export const loginLogRows = [
  {
    id: 1,
    username: 'admin',
    ipaddr: '192.168.1.10',
    loginLocation: '内网',
    browser: 'Chrome',
    os: 'macOS',
    platform: 'Mac',
    status: '2',
    msg: '登录成功',
    loginTime: '2026-08-03T09:00:00Z',
    createdAt: '2026-08-03T09:00:00Z'
  },
  {
    id: 2,
    username: 'tester',
    ipaddr: '192.168.1.11',
    loginLocation: '内网',
    browser: 'Firefox',
    os: 'Windows',
    status: '1',
    msg: '密码错误',
    loginTime: '2026-08-02T09:00:00Z'
  }
]

/** Rows served by GET /api/v1/sys-opera-log */
export const operLogRows = [
  {
    id: 1,
    // title and businessType are shown in the detail drawer and written into
    // the export, and were absent here until the export started resolving
    // businessType through a dictionary -- an absent field exports as blank,
    // which looks the same as a correct one.
    title: '用户管理',
    businessType: '1',
    operName: 'admin',
    requestMethod: 'POST',
    operUrl: '/api/v1/sys-user',
    operIp: '192.168.1.10',
    operLocation: '内网',
    latencyTime: '12ms',
    operParam: '{"username":"newbie"}',
    jsonResult: '{"code":200}',
    status: '2',
    operTime: '2026-08-03T10:00:00Z'
  },
  {
    id: 2,
    title: '用户管理',
    businessType: '3',
    operName: 'tester',
    requestMethod: 'DELETE',
    operUrl: '/api/v1/sys-user',
    operIp: '192.168.1.11',
    operLocation: '内网',
    latencyTime: '8ms',
    operParam: '{"ids":[3]}',
    jsonResult: '{"code":500}',
    status: '1',
    operTime: '2026-08-02T10:00:00Z'
  }
]

/**
 * Rows served by GET /api/v1/menu -- the whole tree, like the department one.
 *
 * `sysApi` is the expanded form of `apis`: the backend routes this menu grants.
 * The 权限标识 column shows it in a popover.
 */
export const menuRows = [
  {
    menuId: 1,
    parentId: 0,
    title: '系统管理',
    menuName: 'SystemRoot',
    menuType: 'M',
    path: '/admin',
    component: 'Layout',
    icon: 'setting',
    sort: 1,
    visible: '0',
    permission: '',
    apis: [] as number[],
    sysApi: [] as Array<Record<string, unknown>>,
    createdAt: '2026-08-01T10:00:00Z',
    children: [
      {
        menuId: 2,
        parentId: 1,
        title: '用户管理',
        menuName: 'SysUserManage',
        menuType: 'C',
        path: 'sys-user',
        component: '/admin/sys-user/index',
        icon: 'user',
        sort: 1,
        visible: '0',
        permission: 'admin:sysUser:list',
        apis: [10],
        sysApi: [{ id: 10, title: '用户列表', path: '/api/v1/sys-user', action: 'GET', type: 'BUS' }],
        createdAt: '2026-08-02T10:00:00Z',
        children: []
      },
      {
        menuId: 3,
        parentId: 2,
        title: '用户新增',
        menuName: '',
        menuType: 'F',
        path: '',
        component: '',
        icon: '',
        sort: 2,
        visible: '0',
        permission: 'admin:sysUser:add',
        apis: [] as number[],
        sysApi: [] as Array<Record<string, unknown>>,
        createdAt: '2026-08-03T10:00:00Z',
        children: []
      }
    ]
  }
]

/**
 * Rows served by GET /api/v1/config.
 *
 * The primary key is `id`, not `configId` -- worth stating, because the page's
 * export used to project the latter and wrote an empty column for it.
 */
export const configRows = [
  {
    id: 1,
    configName: '初始密码',
    configKey: 'sys_user_initPassword',
    configValue: 'Init@123',
    configType: 'Y',
    isFrontend: '2',
    remark: '新建用户的默认密码',
    createdAt: '2026-08-02T10:00:00Z'
  },
  {
    id: 2,
    configName: '应用名称',
    configKey: 'sys_app_name',
    configValue: 'go-admin',
    configType: 'N',
    isFrontend: '1',
    remark: '',
    createdAt: '2026-08-01T10:00:00Z'
  }
]

/** Rows served by GET /api/v1/role */
export const roleRows = [
  {
    roleId: 1,
    roleName: '系统管理员',
    roleKey: 'admin',
    roleSort: 1,
    status: '2',
    remark: '',
    menuIds: [1, 2],
    createdAt: '2026-08-01T10:00:00Z'
  },
  {
    roleId: 2,
    roleName: '普通角色',
    roleKey: 'common',
    roleSort: 2,
    status: '1',
    remark: '',
    menuIds: [1],
    createdAt: '2026-08-02T10:00:00Z'
  }
]

/** Injects the auth cookie so the router guard treats the session as signed in. */
export async function authenticate(context: BrowserContext) {
  await context.addCookies([{
    name: 'Admin-Token',
    value: ADMIN_TOKEN,
    domain: 'localhost',
    path: '/',
    httpOnly: false,
    secure: false,
    sameSite: 'Lax'
  }])
}

/**
 * Stubs every backend call the app shell makes, plus one mock per resource.
 *
 * Resources are composed from mockCrud rather than written out here, so adding a
 * page means adding a line and its rows, not another block of routes whose glob
 * has to be checked against every earlier one.
 *
 * The counters are grouped by resource -- `calls.post.list`, not
 * `calls.postList`. Endpoints that are not CRUD (the password reset, the role's
 * data scope) keep their own entries under `calls.extra`.
 */
/**
 * The three routes every signed-in page needs, shared by both mock sets.
 *
 * The catch-all must be registered before anything else: Playwright matches
 * routes in reverse registration order, so a catch-all added later would
 * swallow every request. Callers layer their specific routes on top of this.
 *
 * Counters live on the caller's object so each spec can assert how many times
 * the profile was fetched -- the sign-in tests care about that as much as the
 * authenticated ones do.
 */
async function installSessionMocks(page: Page, counters: { getinfo: number, menurole: number }) {
  // Settings and other incidental lookups are answered with an empty payload
  // so nothing waits on the network.
  await page.route('**/api/v1/**', async route => {
    await route.fulfill(json({ code: 200, data: {}}))
  })

  await page.route('**/api/v1/getinfo*', async route => {
    counters.getinfo++
    await route.fulfill(json(userInfo))
  })

  await page.route('**/api/v1/menurole*', async route => {
    counters.menurole++
    await route.fulfill(json(menuTree))
  })
}

export async function installApiMocks(page: Page) {
  const extra = { getinfo: 0, menurole: 0, passwordReset: 0, operLogClean: 0, setConfigSaves: 0, setConfigBody: '' }

  /** Milliseconds to hold a write open, so a test can submit again mid-flight. */
  const delays = { userWrite: 0, passwordReset: 0 }

  await installSessionMocks(page, extra)

  await page.route('**/api/v1/dict-data/option-select*', async route => {
    const dictType = new URL(route.request().url()).searchParams.get('dictType')
    // Labels follow the seed data in go-admin/config/db.sql -- sys_common_status
    // is 2=正常 / 1=关闭, which is not the same as sys_normal_disable's 停用.
    const DICTS: Record<string, Array<{ label: string, value: string }>> = {
      sys_user_sex: [{ label: '男', value: '0' }, { label: '女', value: '1' }],
      sys_yes_no: [{ label: '是', value: 'Y' }, { label: '否', value: 'N' }],
      sys_show_hide: [{ label: '显示', value: '0' }, { label: '隐藏', value: '1' }],
      sys_common_status: [{ label: '正常', value: '2' }, { label: '关闭', value: '1' }],
      // All twelve, not the two the fallback below would supply: the operation
      // log exports this column, and a value the options do not cover comes out
      // of dictLabel as the code itself -- which is the exact failure the
      // export fix is about.
      sys_oper_type: [
        { label: '新增', value: '1' }, { label: '修改', value: '2' },
        { label: '删除', value: '3' }, { label: '授权', value: '4' },
        { label: '导出', value: '5' }, { label: '导入', value: '6' },
        { label: '强退', value: '7' }, { label: '生成代码', value: '8' },
        { label: '清空数据', value: '9' }, { label: '登录', value: '10' },
        { label: '退出', value: '11' }, { label: '获取验证码', value: '12' }
      ]
    }
    const data = DICTS[dictType ?? ''] ?? [{ label: '停用', value: '1' }, { label: '正常', value: '2' }]
    await route.fulfill(json({ code: 200, data }))
  })

  await page.route('**/api/v1/dict/type-option-select*', async route => {
    await route.fulfill(json({ code: 200, data: dictTypeRows }))
  })

  await page.route('**/api/v1/set-config*', async route => {
    if (route.request().method() === 'PUT') {
      extra.setConfigSaves++
      extra.setConfigBody = route.request().postData() ?? ''
      // 更新成功 is what the endpoint answers (app/admin/apis/sys_config.go),
      // not the 保存成功 the button promises.
      await route.fulfill(json({ code: 200, msg: '更新成功', data: null }))
      return
    }
    await route.fulfill(json({
      code: 200,
      data: {
        sys_app_name: 'go-admin',
        sys_app_logo: '',
        sys_user_initPassword: 'Init@123',
        sys_index_skinName: 'skin-blue'
        // sys_index_sideTheme deliberately absent
      }
    }))
  })

  await page.route('**/api/v1/configKey/**', async route => {
    await route.fulfill(json({ code: 200, data: { configValue: 'Init@123' }}))
  })

  await page.route('**/api/v1/roleMenuTreeselect/*', async route => {
    await route.fulfill(json({
      code: 200,
      data: {
        menus: [{ id: 1, label: '系统管理', children: [{ id: 2, label: '用户管理', children: [] }] }],
        checkedKeys: [2]
      }
    }))
  })

  await page.route('**/api/v1/operlog/clean*', async route => {
    extra.operLogClean++
    await route.fulfill(json({ code: 200, msg: 'ok' }))
  })

  await page.route('**/api/v1/role-status*', async route => {
    await route.fulfill(json({ code: 200, msg: 'ok' }))
  })

  await page.route('**/api/v1/user/pwd/reset*', async route => {
    extra.passwordReset++
    if (delays.passwordReset) {
      await new Promise(resolve => setTimeout(resolve, delays.passwordReset))
    }
    await route.fulfill(json({ code: 200, msg: 'ok' }))
  })

  const calls = {
    extra,
    user: await mockCrud(page, {
      base: 'sys-user',
      rows: userRows,
      idKey: 'userId',
      // The detail endpoint returns fields the list omits
      detail: row => ({ ...row, remark: '来自详情接口' }),
      delays: { write: delays.userWrite }
    }),
    role: await mockCrud(page, { base: 'role', rows: roleRows, idKey: 'roleId' }),
    config: await mockCrud(page, { base: 'config', rows: configRows, idKey: 'id' }),
    sysApi: await mockCrud(page, {
      base: 'sys-api', rows: sysApiRows, idKey: 'id'
    }),
    dictType: await mockCrud(page, {
      base: 'dict/type', rows: dictTypeRows, idKey: 'id'
    }),
    dictData: await mockCrud(page, {
      base: 'dict/data', rows: dictDataRows, idKey: 'dictCode'
    }),
    loginLog: await mockCrud(page, {
      base: 'sys-login-log', rows: loginLogRows, idKey: 'id'
    }),
    operLog: await mockCrud(page, {
      base: 'sys-opera-log', rows: operLogRows, idKey: 'id'
    }),
    menu: await mockCrud(page, {
      base: 'menu',
      rows: menuRows,
      idKey: 'menuId',
      paginated: false,
      childrenKey: 'children'
    })
  }

  return { calls, delays }
}

/** The answer installLoginMocks accepts; anything else is rejected. */
export const CAPTCHA_ANSWER = '1234'

/**
 * Mocks for the endpoints reachable *before* a session exists.
 *
 * Deliberately separate from installApiMocks, and used without authenticate():
 * every other spec starts by planting a cookie, which is precisely what makes
 * the sign-in path itself unreachable from them. The one page 91% of visitors
 * pass through was the one page nothing executed.
 *
 * The captcha is answered with a real 1x1 PNG data URI rather than a bare
 * string, because the page binds it to <img src> -- a placeholder would leave
 * the element broken and any assertion about it meaningless.
 */
export async function installLoginMocks(page: Page) {
  const calls = {
    captcha: 0,
    login: 0,
    getinfo: 0,
    menurole: 0,
    /** Body of the most recent sign-in attempt, as the client sent it. */
    lastLogin: null as Record<string, unknown> | null
  }

  await installSessionMocks(page, calls)

  await page.route('**/api/v1/app-config*', async route => {
    await route.fulfill(json({
      code: 200,
      data: { sys_app_name: 'go-admin', sys_app_logo: '' }
    }))
  })

  await page.route('**/api/v1/captcha*', async route => {
    calls.captcha++
    await route.fulfill(json({
      code: 200,
      // A distinct id per issue, so a test can prove the refresh actually
      // replaced the challenge instead of redrawing the same one.
      id: `e2e-captcha-${calls.captcha}`,
      data: 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=='
    }))
  })

  await page.route('**/api/v1/login', async route => {
    calls.login++
    calls.lastLogin = JSON.parse(route.request().postData() || '{}')
    const { username, password, code } = calls.lastLogin as Record<string, string>

    if (username === 'admin' && password === '123456' && code === CAPTCHA_ANSWER) {
      await route.fulfill(json({ code: 200, token: ADMIN_TOKEN, data: null }))
      return
    }
    // 400, matching gin-jwt's LoginHandler: `mw.unauthorized(c, 400, ...)` for
    // every Authenticator failure, captcha included. Not 401 -- that code is
    // reserved for token validation and makes the interceptor reload the page,
    // which is a different path entirely.
    await route.fulfill(json({ code: 400, msg: '验证码错误', data: null }))
  })

  return calls
}
