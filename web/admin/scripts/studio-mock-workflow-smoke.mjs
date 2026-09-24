import assert from 'node:assert/strict'
import { chromium } from 'playwright'

const frontendURL = process.env.PIXOMA_STUDIO_MOCK_FRONTEND_URL
if (!frontendURL) {
  throw new Error('Set PIXOMA_STUDIO_MOCK_FRONTEND_URL to the isolated Studio test server.')
}

const browser = await chromium.launch({ headless: true })
try {
  const context = await browser.newContext({
    baseURL: frontendURL,
    viewport: { width: 1440, height: 900 },
  })
  const page = await context.newPage()
  const pageErrors = []
  const receivedEvents = []
  page.on('pageerror', (error) => pageErrors.push(error.message))
  page.on('websocket', (socket) => {
    if (!socket.url().endsWith('/api/v1/studio/agui/ws')) return
    socket.on('framereceived', ({ payload }) => {
      try {
        receivedEvents.push(JSON.parse(String(payload)))
      } catch {
        // Non-JSON frames are irrelevant to this AG-UI assertion.
      }
    })
  })
  await page.goto('/studio')
  const composer = page.getByPlaceholder(
    '描述你想创作的内容，或让 Agent 调用工作流…'
  )
  await composer.waitFor({ state: 'visible' })

  const sessionsResponse = await context.request.get('/api/v1/studio/sessions')
  assert.equal(sessionsResponse.status(), 200)
  const sessions = (await sessionsResponse.json()).data
  assert.equal(sessions.length, 1)
  const sessionID = sessions[0].id
  const getDetail = async () => {
    const response = await context.request.get(
      `/api/v1/studio/sessions/${sessionID}`
    )
    assert.equal(response.status(), 200)
    return (await response.json()).data
  }
  const waitForDetail = async (predicate, description) => {
    const deadline = Date.now() + 20_000
    while (Date.now() < deadline) {
      const detail = await getDetail()
      if (predicate(detail)) return detail
      await new Promise((resolve) => setTimeout(resolve, 100))
    }
    throw new Error(`Timed out waiting for ${description}`)
  }

  await page.getByRole('button', { name: /Agent 操作权限：/ }).click()
  await page
    .getByRole('menuitemradio', { name: '完全访问 · 自动执行所有操作' })
    .click()
  await composer.fill('把雨夜侦探的故事做成四格漫画分镜')
  await page.getByRole('button', { name: '发送消息' }).click()
  await waitForDetail(
    (detail) =>
      detail.session.latest_run?.status === 'running' &&
      detail.assets.length === 0,
    'a running Agent before assets exist'
  )

  // Leave Chat while the server-side run is still active. The Agent must
  // finish without its WebSocket consumer and restore the assets on return.
  await page.getByRole('button', { name: '资产库', exact: true }).click()
  await page.getByRole('heading', { name: '资产库', exact: true }).waitFor({
    state: 'visible',
  })
  const whileAway = await getDetail()
  assert.equal(whileAway.session.latest_run?.status, 'running')
  assert.equal(whileAway.assets.length, 0)
  const released = await context.request.post('/__test__/release-mock-run')
  assert.equal(released.status(), 200)
  const completed = await waitForDetail(
    (detail) =>
      detail.session.latest_run?.status === 'succeeded' &&
      detail.assets.length === 2,
    'the workflow output after leaving Chat'
  )
  assert.deepEqual(
    completed.assets.map(({ name, origin }) => [name, origin]).sort(),
    [
      ['故事大纲.md', 'agent'],
      ['雨夜侦探-分镜预览.svg', 'workflow'],
    ]
  )
  assert.equal(completed.flow.nodes.length, 4)
  assert.equal(completed.flow.edges.length, 3)
  assert.equal(completed.messages.some((message) => message.role === 'user'), true)
  assert.equal(
    completed.messages.some((message) => message.role === 'assistant'),
    true
  )

  const output = completed.assets.find((asset) => asset.origin === 'workflow')
  const imageResponse = await context.request.get(
    output.versions[0].content_url
  )
  assert.equal(imageResponse.status(), 200)
  assert.match(await imageResponse.text(), /<svg[\s>]/)

  await page.getByRole('button', { name: '对话', exact: true }).click()
  await page.getByRole('tab', { name: '资产路线' }).click()
  await page.getByText('分镜预览', { exact: true }).waitFor({
    state: 'visible',
    timeout: 15_000,
  })
  await page.getByRole('tab', { name: /Session 资产/ }).click()
  await page.getByText('故事大纲.md', { exact: true }).waitFor({
    state: 'visible',
  })
  await page.getByText('雨夜侦探-分镜预览.svg', { exact: true }).waitFor({
    state: 'visible',
  })

  await page.reload()
  await page.getByRole('tab', { name: /Session 资产/ }).click()
  await page.getByText('雨夜侦探-分镜预览.svg', { exact: true }).waitFor({
    state: 'visible',
  })
  const sessionsAfterReload = (
    await (await context.request.get('/api/v1/studio/sessions')).json()
  ).data
  assert.equal(sessionsAfterReload.length, 1)

  const eventsBeforeLiveTurn = receivedEvents.length
  await composer.fill('再给雨夜侦探画一个备选分镜')
  await page.getByRole('button', { name: '发送消息' }).click()
  const liveDeadline = Date.now() + 20_000
  while (Date.now() < liveDeadline) {
    const liveEvents = receivedEvents.slice(eventsBeforeLiveTurn)
    if (
      liveEvents.some((event) => event.type === 'TEXT_MESSAGE_CONTENT') &&
      liveEvents.some((event) => event.type === 'TOOL_CALL_START') &&
      liveEvents.some((event) => event.type === 'RUN_FINISHED')
    ) {
      break
    }
    await new Promise((resolve) => setTimeout(resolve, 100))
  }
  const liveEvents = receivedEvents.slice(eventsBeforeLiveTurn)
  assert.equal(
    liveEvents.some((event) => event.type === 'TEXT_MESSAGE_CONTENT'),
    true,
    'the Chat must receive assistant text over AG-UI'
  )
  assert.equal(
    liveEvents.some((event) => event.type === 'TOOL_CALL_START'),
    true,
    'the Chat must receive the workflow tool call over AG-UI'
  )
  assert.equal(
    liveEvents.some((event) => event.type === 'RUN_FINISHED'),
    true,
    'the Chat must receive the AG-UI completion event'
  )
  const assistantText =
    '我先整理故事大纲，再调用分镜工作流生成预览图。你可以在右侧 Flow 中继续调整这条创作路线。'
  await page.getByText(assistantText, { exact: true }).nth(1).waitFor({
    state: 'visible',
  })
  for (const width of [920, 375]) {
    await page.setViewportSize({ width, height: 900 })
    const layout = await page.evaluate(() => ({
      viewport: document.documentElement.clientWidth,
      content: document.documentElement.scrollWidth,
    }))
    assert.ok(
      layout.content <= layout.viewport + 1,
      `Studio overflows horizontally at ${width}px: ${JSON.stringify(layout)}`
    )
  }
  for (const name of [
    '打开 Studio 菜单',
    '打开创作工作台',
    '选择 Skills',
    '选择资产',
    'Agent 操作权限：完全访问',
    'Browser Test Model',
    '发送消息',
  ]) {
    const target = await page.getByRole('button', { name }).boundingBox()
    assert.ok(target, `the mobile ${name} button must be visible`)
    assert.ok(
      target.width >= 44 && target.height >= 44,
      `the mobile ${name} target is smaller than 44px: ${JSON.stringify(target)}`
    )
    assert.ok(
      target.x >= 0 && target.x + target.width <= 376,
      `the mobile ${name} button is clipped: ${JSON.stringify(target)}`
    )
  }
  await page.getByRole('button', { name: '打开创作工作台' }).click()
  await page.getByRole('dialog', { name: '创作工作台' }).waitFor({
    state: 'visible',
  })
  await page.getByRole('dialog').getByRole('tab', { name: /Session 资产/ }).click()
  await page
    .getByRole('dialog')
    .getByText('雨夜侦探-分镜预览.svg', { exact: true })
    .first()
    .waitFor({ state: 'visible' })
  assert.deepEqual(pageErrors, [])
  console.log(
    'Studio mock browser journey passed: Chat → background workflow → assets and Flow → reload'
  )
} finally {
  await browser.close()
}
