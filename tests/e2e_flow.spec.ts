import { test, expect } from '@playwright/test'

const BASE = process.env.DASHBOARD_URL || 'http://127.0.0.1:28371'

test('dashboard renders sonar deck and kv console', async ({ page }) => {
  await page.goto(BASE)
  await expect(page.getByText('GORAFT')).toBeVisible()
  await expect(page.getByText('声纳指挥甲板')).toBeVisible()
  await expect(page.getByLabel('配置键')).toBeVisible()
  await page.getByLabel('配置键').fill('e2e/key')
  await page.getByLabel('配置值').fill('ok')
  await page.getByRole('button', { name: '写入并追踪' }).click()
  await expect(page.getByText(/写入成功|not_leader|请改打/)).toBeVisible({ timeout: 15000 })
})

test('chaos dialog is custom not native', async ({ page }) => {
  await page.goto(BASE)
  await page.getByRole('button', { name: 'Kill Leader 目标' }).click()
  await expect(page.getByText('确认 Kill')).toBeVisible()
  await page.getByRole('button', { name: '取消' }).click()
})
