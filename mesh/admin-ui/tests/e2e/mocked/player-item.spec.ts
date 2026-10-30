import { test, expect } from '@playwright/test'
import { authenticate, userInfo } from './fixtures'
import { captureBodies, json } from './support/crud'

/**
 * Player item management: the container grid plus the Luban catalog.
 *
 * The backend serves three mutating endpoints and answers each of them with the
 * updated container, so the grid is asserted from the response rather than from
 * a follow-up GET -- a page that ignored the response would still pass a test
 * that only counted requests.
 */

const menuTree = {
  code: 200,
  data: [
    {
      path: '/player',
      component: 'Layout',
      visible: '0',
      menuName: 'Player',
      title: 'Player',
      icon: 'user',
      noCache: false,
      children: [
        {
          path: 'item/index',
          component: '/admin/player-item/index',
          visible: '0',
          menuName: 'PlayerItemManage',
          title: 'PlayerItemManage',
          icon: 'star',
          noCache: false,
          children: null
        }
      ]
    }
  ]
}

const catalogRows = [
  { id: 1, name: '巡界币', width: 1, height: 1, rarity: 2, maxCount: 99 },
  { id: 2, name: '步枪', width: 4, height: 2, rarity: 3, maxCount: 1 }
]

const coin = {
  itemId: 11, itemCfgId: 1, count: 5, gridX: 0, gridY: 0, rotated: false, name: '巡界币', width: 1, height: 1, rarity: 2, maxCount: 99
}

const rifle = {
  itemId: 12, itemCfgId: 2, count: 1, gridX: 2, gridY: 2, rotated: false, name: '步枪', width: 4, height: 2, rarity: 3, maxCount: 1
}

const container = (items: unknown[], version: number) => ({
  playerId: 10001,
  itemType: 1,
  gridW: 8,
  gridH: 12,
  version,
  items
})

/** Stubs the shell, the catalog and the four container endpoints. */
async function installItemMocks(page: import('@playwright/test').Page) {
  const calls = { list: 0, container: 0, add: 0, move: 0, sort: 0 }

  // Registered first: Playwright matches routes in reverse registration order.
  await page.route('**/api/v1/**', async route => {
    await route.fulfill(json({ code: 200, data: {}}))
  })

  await page.route('**/api/v1/getinfo*', async route => {
    await route.fulfill(json({
      ...userInfo,
      data: {
        ...userInfo.data,
        permissions: [...userInfo.data.permissions, 'game:player:item:list']
      }
    }))
  })

  await page.route('**/api/v1/menurole*', async route => {
    await route.fulfill(json(menuTree))
  })

  await page.route(/\/api\/v1\/item(\?|$)/, async route => {
    calls.list++
    await route.fulfill(json({ code: 200, data: { list: catalogRows, count: catalogRows.length }}))
  })

  await page.route(/\/api\/v1\/player\/container(\?|$)/, async route => {
    calls.container++
    await route.fulfill(json({ code: 200, data: container([coin, rifle], 1) }))
  })

  await page.route(/\/api\/v1\/player\/container\/sort/, async route => {
    calls.sort++
    await route.fulfill(json({
      code: 200,
      data: container([{ ...coin, gridX: 0, gridY: 0 }, { ...rifle, gridX: 0, gridY: 1 }], 2)
    }))
  })

  await page.route(/\/api\/v1\/player\/container\/item$/, async route => {
    if (route.request().method() === 'POST') {
      calls.add++
      await route.fulfill(json({
        code: 200,
        data: container([coin, rifle, { ...coin, itemId: 13, count: 1, gridX: 6, gridY: 10 }], 2)
      }))
      return
    }
    calls.move++
    await route.fulfill(json({
      code: 200,
      data: container([{ ...coin, gridX: 1, gridY: 1, rotated: true }, rifle], 2)
    }))
  })

  return calls
}

/** Opens the page and loads player 10001's backpack. */
async function openContainer(page: import('@playwright/test').Page) {
  await page.goto('/#/player/item/index')
  await page.waitForSelector('.el-table')

  await expect(page.getByText('输入玩家ID后点击搜索查看容器')).toBeVisible()

  await page.getByPlaceholder('请输入玩家ID').fill('10001')
  // Scoped: the catalog panel below has its own search button.
  await page.locator('.container-card').getByRole('button', { name: '搜索' }).click()

  await expect(page.locator('.grid-item')).toHaveCount(2)
}

test.describe('player item management', () => {
  test.beforeEach(async({ context }) => {
    await authenticate(context)
  })

  test('renders the grid, its items and the Luban catalog', async({ page }) => {
    await installItemMocks(page)
    await openContainer(page)

    await expect(page.locator('.grid-cell')).toHaveCount(96)
    await expect(page.locator('.el-table__row')).toHaveCount(2)
    await expect(page.getByRole('cell', { name: '步枪', exact: true })).toBeVisible()
  })

  test('sorts the container and draws the response', async({ page }) => {
    const calls = await installItemMocks(page)
    const bodies = await captureBodies(page, /\/api\/v1\/player\/container\/sort/, 'POST')

    await openContainer(page)
    await page.locator('.container-card').getByRole('button', { name: '整理' }).click()

    await expect.poll(() => calls.sort).toBe(1)
    expect(JSON.parse(bodies.at(-1) || '{}')).toMatchObject({ playerId: 10001, itemType: 1 })

    // The stub moved the rifle to the first column; the page must have taken
    // the response instead of keeping the pre-sort layout.
    await expect(page.locator('.grid-item').nth(1)).toHaveCSS('left', '0px')
  })

  test('adds an item chosen from the catalog', async({ page }) => {
    const calls = await installItemMocks(page)
    const bodies = await captureBodies(page, /\/api\/v1\/player\/container\/item$/, 'POST')

    await openContainer(page)
    await page.getByRole('row', { name: /步枪/ }).getByRole('button', { name: '添加' }).click()

    const dialog = page.getByRole('dialog').filter({ hasText: '添加物品' })
    await expect(dialog).toBeVisible()
    await dialog.getByRole('button', { name: '确 定' }).click()

    await expect.poll(() => calls.add).toBe(1)
    expect(JSON.parse(bodies.at(-1) || '{}')).toMatchObject({
      playerId: 10001,
      itemType: 1,
      itemCfgId: 2,
      count: 1
    })
    await expect(page.locator('.grid-item')).toHaveCount(3)
  })

  test('moves and rotates an item from the grid', async({ page }) => {
    const calls = await installItemMocks(page)
    const bodies = await captureBodies(page, /\/api\/v1\/player\/container\/item$/, 'PUT')

    await openContainer(page)
    await page.locator('.grid-item').first().click()

    const dialog = page.getByRole('dialog').filter({ hasText: '移动物品' })
    await expect(dialog).toBeVisible()
    await dialog.locator('.el-switch').click()
    await dialog.getByRole('button', { name: '确 定' }).click()

    await expect.poll(() => calls.move).toBe(1)
    expect(JSON.parse(bodies.at(-1) || '{}')).toMatchObject({
      playerId: 10001,
      itemType: 1,
      itemId: 11,
      rotated: true
    })
  })
})
