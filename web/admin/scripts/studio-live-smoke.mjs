import assert from 'node:assert/strict'
import { chromium } from 'playwright'

const frontendURL = process.env.PIXOMA_STUDIO_FRONTEND_URL
const username = process.env.PIXOMA_STUDIO_SMOKE_USER
const password = process.env.PIXOMA_STUDIO_SMOKE_PASSWORD

if (!frontendURL || !username || !password) {
  throw new Error(
    'Set PIXOMA_STUDIO_FRONTEND_URL, PIXOMA_STUDIO_SMOKE_USER and PIXOMA_STUDIO_SMOKE_PASSWORD for an isolated Pixoma instance.'
  )
}

// Functional browser smoke only. Do not capture screenshots or page snapshots.
const browser = await chromium.launch({ headless: true })
try {
  const context = await browser.newContext({
    baseURL: frontendURL,
    viewport: { width: 1440, height: 900 },
  })
  const login = await context.request.post('/api/v1/setup/login', {
    data: { username, password },
  })
  assert.equal(login.status(), 200, 'the isolated admin login must succeed')

  const page = await context.newPage()
  await page.goto('/studio')
  await page.getByRole('button', { name: '新建对话' }).click()
  const composer = page.getByPlaceholder(
    '描述你想创作的内容，或让 Agent 调用工作流…'
  )
  await composer.waitFor({ state: 'visible' })
  await page.getByRole('button', { name: /Agent 操作权限：/ }).click()
  await page
    .getByRole('menuitemradio', { name: '完全访问 · 自动执行所有操作' })
    .click()
  await composer.fill(
    '调用 create_text_asset 工具，在当前对话创建名为 browser-story.md 的 Markdown 资产，内容以「# 雨夜侦探」开头，然后告诉我已创建。'
  )
  await page.getByRole('button', { name: '发送消息' }).click()

  await page.getByRole('tab', { name: /Session 资产/ }).click()
  await page.getByText('browser-story.md', { exact: true }).waitFor({
    state: 'visible',
    timeout: 90_000,
  })
  await page.getByRole('tab', { name: '资产路线' }).click()
  await page.getByText('browser-story.md', { exact: true }).waitFor({
    state: 'visible',
    timeout: 15_000,
  })

  await page.reload()
  await page.getByRole('tab', { name: /Session 资产/ }).click()
  await page.getByText('browser-story.md', { exact: true }).waitFor({
    state: 'visible',
    timeout: 15_000,
  })

  await composer.fill(
    '再调用 create_text_asset 工具创建 background-story.md，内容以「# 后台运行」开头，然后告诉我已创建。'
  )
  await page.getByRole('button', { name: '发送消息' }).click()
  await page.getByRole('button', { name: '资产库', exact: true }).click()
  await page.getByRole('button', { name: '对话', exact: true }).click()
  await page.getByRole('tab', { name: /Session 资产/ }).click()
  await page.getByText('background-story.md', { exact: true }).waitFor({
    state: 'visible',
    timeout: 90_000,
  })
  console.log(
    'Studio live browser smoke passed: Chat → asset → Flow, refresh and background navigation'
  )
} finally {
  await browser.close()
}
