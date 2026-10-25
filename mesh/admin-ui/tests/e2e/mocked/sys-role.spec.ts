import { test, expect } from '@playwright/test'
import { authenticate, installApiMocks } from './fixtures'

/**
 * The role page: two dialogs on one screen, which is what the composable layer
 * was built for and what the Options API version could not express.
 */
test.describe('sys-role', () => {
  test.beforeEach(async({ context }) => {
    await authenticate(context)
  })

  test('lists roles', async({ page }) => {
    await installApiMocks(page)

    await page.goto('/#/admin/sys-role')
    await page.waitForSelector('.el-table')

    await expect(page.locator('.el-table__row')).toHaveCount(2)
    await expect(page.getByRole('cell', { name: '系统管理员' })).toBeVisible()
  })

  test('sends paging keys with the list request', async({ page }) => {
    const { calls } = await installApiMocks(page)

    await page.goto('/#/admin/sys-role')
    await page.waitForSelector('.el-table')

    expect(calls.role.list).toBeGreaterThan(0)
    const sent = calls.role.listQueries.at(-1) ?? ''
    expect(sent).toContain('pageIndex=1')
    expect(sent).toContain('pageSize=')
  })

  test('deleting a row asks first, then reloads', async({ page }) => {
    const { calls } = await installApiMocks(page)

    await page.goto('/#/admin/sys-role')
    await page.waitForSelector('.el-table')
    const before = calls.role.list

    await page.getByRole('row', { name: /普通角色/ }).getByRole('button', { name: '删除' }).click()
    await page.locator('.el-message-box').getByRole('button', { name: '确定' }).click()

    await expect.poll(() => calls.role.remove).toBe(1)
    await expect.poll(() => calls.role.list).toBeGreaterThan(before)
  })

  test('saving an edit reaches the update endpoint', async({ page }) => {
    const { calls } = await installApiMocks(page)

    await page.goto('/#/admin/sys-role')
    await page.waitForSelector('.el-table')

    await page.getByRole('row', { name: /普通角色/ }).getByRole('button', { name: '修改' }).click()
    const dialog = page.getByRole('dialog').filter({ hasText: '修改角色' })
    await expect(dialog).toBeVisible()
    await dialog.getByRole('button', { name: '确 定' }).click()

    await expect.poll(() => calls.role.update).toBe(1)
    await expect(dialog).toBeHidden()
  })

  test('the create dialog opens empty', async({ page }) => {
    await installApiMocks(page)

    await page.goto('/#/admin/sys-role')
    await page.waitForSelector('.el-table')

    await page.locator('.pro-table__toolbar').getByRole('button', { name: '新增' }).click()
    const dialog = page.getByRole('dialog').filter({ hasText: '添加角色' })
    await expect(dialog).toBeVisible()
    await expect(dialog.getByPlaceholder('请输入角色名称')).toHaveValue('')
  })

  test('editing loads the record and locks the identity fields', async({ page }) => {
    await installApiMocks(page)

    await page.goto('/#/admin/sys-role')
    await page.waitForSelector('.el-table')

    await page.getByRole('row', { name: /普通角色/ }).getByRole('button', { name: '修改' }).click()

    const dialog = page.getByRole('dialog').filter({ hasText: '修改角色' })
    await expect(dialog).toBeVisible()
    await expect(dialog.getByPlaceholder('请输入角色名称')).toHaveValue('普通角色')
    // Renaming a role or its key would orphan the permissions already granted
    await expect(dialog.getByPlaceholder('请输入角色名称')).toBeDisabled()
    await expect(dialog.getByPlaceholder('请输入权限字符')).toBeDisabled()
  })

  test('the super admin is told there is nothing to assign', async({ page }) => {
    await installApiMocks(page)

    await page.goto('/#/admin/sys-role')
    await page.waitForSelector('.el-table')

    // roleKey 'admin' holds everything implicitly
    await page.getByRole('row', { name: /系统管理员/ }).getByRole('button', { name: '修改' }).click()

    const dialog = page.getByRole('dialog').filter({ hasText: '修改角色' })
    await expect(dialog).toContainText('系统超级管理员无需此操作')
  })

  test('toggling status asks first and reverts on cancel', async({ page }) => {
    await installApiMocks(page)

    await page.goto('/#/admin/sys-role')
    await page.waitForSelector('.el-table')

    const row = page.getByRole('row', { name: /普通角色/ })
    await row.locator('.el-switch').click()

    const confirm = page.locator('.el-message-box')
    await expect(confirm).toBeVisible()
    await confirm.getByRole('button', { name: '取消' }).click()

    // Put back, so the switch matches what the server still holds
    await expect(row.locator('.el-switch')).not.toHaveClass(/is-checked/)
  })
})
