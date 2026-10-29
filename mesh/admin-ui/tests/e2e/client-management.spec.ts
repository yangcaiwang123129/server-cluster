import { test, expect, Page } from '@playwright/test'

/**
 * 临时验证脚本（跑完即删）：真实后端 + 真实前端的「客户端管理」端到端检查。
 * 覆盖版本管理的增/改/启停/删，以及模拟器「一键进入大厅」。
 */

const ADMIN_API = 'http://188.18.23.72:8081'
const UI = 'http://localhost:9527'

async function adminToken(): Promise<string> {
  const cap: any = await (await fetch(`${ADMIN_API}/api/v1/captcha`)).json()
  const raw = String(cap.data).replace(/^data:[^,]+,/, '')
  const svg = Buffer.from(raw, 'base64').toString('utf8')
  const code = [...svg.matchAll(/>(\d)</g)].map(m => m[1]).join('')
  const res: any = await (await fetch(`${ADMIN_API}/api/v1/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: 'admin', password: '123456', code, uuid: cap.id })
  })).json()
  if (!res.token) throw new Error('admin login failed: ' + JSON.stringify(res))
  return res.token
}

/** 应用是 hash 路由，深链要带 #。 */
async function goto(page: Page, hashPath: string) {
  await page.goto(`${UI}/#${hashPath}`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(1500)
}

test('客户端管理：版本管理 CRUD + 模拟器一键进入大厅', async({ page, context }) => {
  test.setTimeout(120_000)

  const token = await adminToken()
  await context.addCookies([{
    name: 'Admin-Token', value: token, domain: 'localhost', path: '/'
  }])

  const errors: string[] = []
  page.on('pageerror', e => errors.push('[pageerror] ' + e.message))
  page.on('console', msg => {
    if (msg.type() !== 'error') return
    const text = msg.text()
    if (text.includes('Failed to load resource')) return
    errors.push('[console] ' + text)
  })

  // ---------- 版本管理 ----------
  await goto(page, '/client/version/index')
  await expect(page.locator('.el-table')).toBeVisible()
  await expect(page.getByRole('menuitem', { name: '客户端管理', exact: true })).toBeVisible()

  await page.getByRole('button', { name: '新增' }).click()
  const addDlg = page.locator('.el-dialog').filter({ hasText: '新增版本' })
  await expect(addDlg).toBeVisible()
  const version = '2.1.' + String(Date.now()).slice(-6) + '-ui'
  await addDlg.getByPlaceholder('请输入版本号').fill(version)
  await addDlg.locator('.el-select').click()
  await page.locator('.el-select-dropdown:visible .el-select-dropdown__item').filter({ hasText: 'Windows' }).click()
  await addDlg.getByPlaceholder('请输入安装包下载地址').fill('http://example.com/pkg/2.0.0.zip')
  await addDlg.getByPlaceholder('请输入更新说明').fill('ui test')
  await addDlg.locator('.el-switch').click()
  await addDlg.getByRole('button', { name: '确 定' }).click()
  await expect(addDlg).toBeHidden({ timeout: 10_000 })

  const row = page.locator('.el-table__row').filter({ hasText: version })
  await expect(row).toBeVisible()
  await expect(row.getByText('Windows')).toBeVisible()
  await expect(row.getByText('是')).toBeVisible()
  await expect(row.locator('a')).toHaveAttribute('href', 'http://example.com/pkg/2.0.0.zip')

  // 启停
  await expect(row.locator('.el-switch')).toHaveClass(/is-checked/)
  await row.locator('.el-switch').click()
  await expect(page.getByText('状态已更新')).toBeVisible()
  await row.locator('.el-switch').click()
  await expect(row.locator('.el-switch')).toHaveClass(/is-checked/)

  // 修改
  await row.getByRole('button', { name: '修改' }).click()
  const editDlg = page.locator('.el-dialog').filter({ hasText: '修改版本' })
  await expect(editDlg.getByPlaceholder('请输入版本号')).toHaveValue(version)
  const renamed = '2.2.' + String(Date.now()).slice(-6) + '-ui'
  await editDlg.getByPlaceholder('请输入版本号').fill(renamed)
  await editDlg.getByRole('button', { name: '确 定' }).click()
  await expect(editDlg).toBeHidden({ timeout: 10_000 })
  const edited = page.locator('.el-table__row').filter({ hasText: renamed })
  await expect(edited).toBeVisible()

  // ---------- 模拟器：一键进入大厅 ----------
  await goto(page, '/client/simulator/index')
  await expect(page).toHaveURL(/\/client\/simulator\/index/)
  await expect(page.getByRole('button', { name: '一键进入大厅' })).toBeVisible()
  await expect(page.getByRole('button', { name: '一键模拟完整流程' })).toBeVisible()
  await expect(page.getByRole('button', { name: '重置' })).toBeVisible()

  await page.getByText('注册新账号').click()
  const uname = 'simui' + Date.now().toString().slice(-6)
  await page.getByPlaceholder('请输入账号').fill(uname)
  await page.getByPlaceholder('请输入密码').fill('123456')
  await page.getByRole('button', { name: '一键进入大厅' }).click()

  await expect(page.getByText('已一键进入大厅')).toBeVisible({ timeout: 40_000 })

  const steps = page.locator('.step')
  for (const title of ['登录 / 注册', '连接网关', '首包验证', '进入游戏']) {
    await expect(steps.filter({ hasText: title })).toHaveClass(/step--success/)
  }
  await expect(steps.filter({ hasText: '退出游戏' })).toHaveClass(/step--idle/)
  await expect(steps.filter({ hasText: '进入游戏' }).locator('.step-io')).toContainText('ERR_CODE_OK')

  // ---------- 清理 ----------
  await goto(page, '/client/version/index')
  const target = page.locator('.el-table__row').filter({ hasText: '2.0.1-ui' })
  await target.getByRole('button', { name: '删除' }).click()
  await page.getByRole('button', { name: '确定' }).click()
  await expect(page.locator('.el-table__row').filter({ hasText: '2.0.1-ui' })).toHaveCount(0)

  expect(errors, errors.join('\n')).toEqual([])
})
