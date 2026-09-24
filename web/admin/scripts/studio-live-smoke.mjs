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
  const normalizeReply = (value) =>
    String(value)
      .normalize('NFKC')
      .replace(/[^\p{L}\p{N}]/gu, '')
  let sessionID
  const waitForAsset = async (name, timeout, minimumAssistantCount) => {
    const deadline = Date.now() + timeout
    let lastSummary
    while (Date.now() < deadline) {
      const detailResponse = await context.request.get(
        `/api/v1/studio/sessions/${sessionID}`
      )
      const detail = (await detailResponse.json()).data
      const run = detail?.session?.latest_run
      const runEvents = detail?.transcript?.events?.filter(
        (event) => event.runId === run?.id
      )
      const assistantReply = detail?.transcript?.messages
        ?.filter(
          (message) =>
            message.runId === run?.id &&
            message.role === 'assistant' &&
            message.content?.trim()
        )
        .at(-1)
      lastSummary = {
        session_id: sessionID,
        run,
        assets: detail?.assets?.map((asset) => asset.name),
        events: runEvents?.map((event) => event.type),
        has_assistant_reply: Boolean(assistantReply),
      }
      if (
        run?.status === 'succeeded' &&
        detail?.assets?.some((asset) => asset.name === name) &&
        runEvents?.some((event) => event.type === 'TOOL_CALL_START') &&
        runEvents?.some((event) => event.type === 'TOOL_CALL_END') &&
        assistantReply
      ) {
        await page
          .getByRole('tabpanel', { name: /Session 资产/ })
          .getByText(name, { exact: true })
          .first()
          .waitFor({ state: 'visible', timeout: 15_000 })
        const chatMessages = page.locator('.is-assistant')
        const chatDeadline = Date.now() + 15_000
        while (
          (await chatMessages.count()) < minimumAssistantCount &&
          Date.now() < chatDeadline
        ) {
          await new Promise((resolve) => setTimeout(resolve, 100))
        }
        const messageCount = await chatMessages.count()
        assert.ok(messageCount >= minimumAssistantCount)
        const chatReply = chatMessages.last()
        await chatReply.waitFor({ state: 'visible', timeout: 15_000 })
        const expected = normalizeReply(assistantReply.content).slice(0, 24)
        const rendered = normalizeReply(await chatReply.innerText())
        assert.ok(
          expected && rendered.includes(expected),
          `the current Run's assistant reply is missing from Chat: ${JSON.stringify({ expected, rendered })}`
        )
        return messageCount
      }
      if (run?.status === 'failed') {
        throw new Error(
          `Studio live run failed: ${JSON.stringify(lastSummary)}`
        )
      }
      await new Promise((resolve) => setTimeout(resolve, 500))
    }
    throw new Error(
      `Timed out waiting for Studio asset ${name}: ${JSON.stringify(lastSummary)}`
    )
  }
  await page.goto('/studio')
  const composer = page.getByPlaceholder(
    '描述你想创作的内容，或让 Agent 调用工作流…'
  )
  await composer.waitFor({ state: 'visible' })
  const createdResponsePromise = page.waitForResponse(
    (response) =>
      response.request().method() === 'POST' &&
      new URL(response.url()).pathname === '/api/v1/studio/sessions'
  )
  await page.getByRole('button', { name: '新建对话' }).click()
  const createdResponse = await createdResponsePromise
  assert.equal(createdResponse.status(), 201)
  sessionID = (await createdResponse.json()).data?.id
  assert.ok(sessionID, 'new Studio session ID must be returned')
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
  let assistantCount = await waitForAsset('browser-story.md', 90_000, 1)
  await page.getByRole('tab', { name: '资产路线' }).click()
  await page
    .getByRole('tabpanel', { name: '资产路线' })
    .getByText('browser-story.md', { exact: true })
    .first()
    .waitFor({ state: 'visible', timeout: 15_000 })

  await page.reload()
  await page.getByRole('tab', { name: /Session 资产/ }).click()
  await page
    .getByRole('tabpanel', { name: /Session 资产/ })
    .getByText('browser-story.md', { exact: true })
    .first()
    .waitFor({ state: 'visible', timeout: 15_000 })
  assistantCount = await page.locator('.is-assistant').count()

  await composer.fill(
    '再调用 create_text_asset 工具创建 background-story.md，内容以「# 后台运行」开头，然后告诉我已创建。'
  )
  await page.getByRole('button', { name: '发送消息' }).click()
  await page.getByRole('button', { name: '资产库', exact: true }).click()
  await page.getByRole('button', { name: '对话', exact: true }).click()
  await page.getByRole('tab', { name: /Session 资产/ }).click()
  await waitForAsset('background-story.md', 90_000, assistantCount + 1)
  console.log(
    'Studio live browser smoke passed: Chat → asset → Flow, refresh and background navigation'
  )
} finally {
  await browser.close()
}
