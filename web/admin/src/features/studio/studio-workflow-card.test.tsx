import { describe, expect, it } from 'vitest'
import { render } from 'vitest-browser-react'
import '@/styles/index.css'
import { StudioWorkflowCard } from './studio-workflow-card'

describe('StudioWorkflowCard', () => {
  it('collects required inputs and submits their typed values', async () => {
    let submitted: Record<string, unknown> | undefined
    const screen = await render(
      <StudioWorkflowCard
        workflow={{
          id: '12', name: '角色三视图', description: '生成角色设定图',
          input_schema: {
            type: 'object', required: ['prompt', 'steps'],
            properties: { prompt: { type: 'string', title: '角色描述' }, steps: { type: 'number', title: '步数' } },
          },
          input_fields: [
            { key: 'prompt', type: 'string', required: true },
            { key: 'steps', type: 'number', required: true },
          ],
        }}
        assets={[]}
        onSubmit={(inputs) => { submitted = inputs }}
        onSkip={() => {}}
      />
    )

    await expect.element(screen.getByRole('button', { name: '提交工作流' })).toBeDisabled()
    await screen.getByRole('textbox', { name: '角色描述 *' }).fill('雨夜侦探')
    await screen.getByRole('spinbutton', { name: '步数 *' }).fill('24')
    await screen.getByRole('button', { name: '提交工作流' }).click()
    expect(submitted).toEqual({ prompt: '雨夜侦探', steps: 24 })
  })

  it('lets the user skip a workflow', async () => {
    let skipped = false
    const screen = await render(
      <StudioWorkflowCard
        workflow={{ id: '12', name: '角色三视图', input_schema: { type: 'object' }, input_fields: [] }}
        assets={[]}
        onSubmit={() => {}}
        onSkip={() => { skipped = true }}
      />
    )
    await screen.getByRole('button', { name: '跳过' }).click()
    expect(skipped).toBe(true)
  })

  it('keeps workflow details visible while the input fields scroll', async () => {
    const screen = await render(
      <StudioWorkflowCard
        workflow={{
          id: '12', name: '角色三视图', description: '生成角色设定图',
          input_schema: { type: 'object' },
          input_fields: Array.from({ length: 18 }, (_, index) => ({
            key: `field_${index}`, type: 'string', required: false,
          })),
        }}
        assets={[]}
        onSubmit={() => {}}
        onSkip={() => {}}
      />
    )
    const card = document.querySelector<HTMLElement>("[data-testid='studio-workflow-card']")
    const info = document.querySelector<HTMLElement>("[data-testid='studio-workflow-info']")
    const inputs = document.querySelector<HTMLElement>("[data-testid='studio-workflow-inputs']")
    const footer = card?.querySelector<HTMLElement>("[data-slot='card-footer']")
    expect(card).not.toBeNull()
    expect(info).not.toBeNull()
    expect(inputs).not.toBeNull()
    expect(footer).not.toBeNull()
    if (!card || !info || !inputs || !footer) return

    await expect.element(screen.getByText('生成角色设定图')).toBeVisible()
    expect(info.getBoundingClientRect().right).toBeLessThanOrEqual(inputs.getBoundingClientRect().left)
    expect(card.getBoundingClientRect().height).toBeLessThan(info.getBoundingClientRect().height + footer.getBoundingClientRect().height + 60)
    expect(footer.getBoundingClientRect().top).toBeGreaterThanOrEqual(info.getBoundingClientRect().bottom)
    expect(footer.getBoundingClientRect().right).toBeGreaterThanOrEqual(inputs.getBoundingClientRect().right)
    expect(getComputedStyle(info).borderRightWidth).toBe('0px')
    expect(getComputedStyle(footer).borderTopWidth).toBe('0px')
    expect(getComputedStyle(inputs).overflowY).toBe('auto')
    expect(inputs.scrollHeight).toBeGreaterThan(inputs.clientHeight)
    const infoTop = info.getBoundingClientRect().top
    inputs.scrollTop = inputs.scrollHeight
    expect(inputs.scrollTop).toBeGreaterThan(0)
    expect(info.getBoundingClientRect().top).toBe(infoTop)
  })
})
