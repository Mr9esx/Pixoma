import { describe, expect, it } from 'vitest'
import type { Card } from '@/lib/api/channel-menu'
import { describeAction, type WorkflowRef } from './node-view'

const wl: WorkflowRef[] = [
  { id: 10, name: '写实' },
  { id: 20, name: '二次元' },
]
const cards: Card[] = [
  { id: 'c1', name: '开始生成', text: '选风格', media: [], buttons: [] },
]

describe('describeAction', () => {
  it('open_workflow 展示绑定工作流名', () => {
    const v = describeAction(
      { type: 'open_workflow', workflow_id: '10' },
      cards,
      wl
    )
    expect(v.kind).toBe('workflow')
    expect(v.summary).toBe('打开工作流「写实」')
    expect(v.params[0].value).toBe('写实')
  })

  it('open_card 展示目标卡片名', () => {
    const v = describeAction({ type: 'open_card', card_id: 'c1' }, cards, wl)
    expect(v.kind).toBe('card')
    expect(v.summary).toBe('打开卡片「开始生成」')
  })

  it('send_text 展示文本内容', () => {
    const v = describeAction({ type: 'send_text', text: '欢迎使用' }, cards, wl)
    expect(v.kind).toBe('text')
    expect(v.summary).toBe('发文字「欢迎使用」')
  })

  it('copy_text 归为 text', () => {
    const v = describeAction({ type: 'copy_text', text: '复制我' }, cards, wl)
    expect(v.kind).toBe('text')
    expect(v.summary).toContain('复制文本')
  })

  it('send_media 展示媒体数量与链接', () => {
    const v = describeAction(
      {
        type: 'send_media',
        media: [{ kind: 'image', url: 'https://a/b.png' }],
      },
      cards,
      wl
    )
    expect(v.kind).toBe('media')
    expect(v.summary).toBe('发媒体（1 个）')
    expect(v.params[0].value).toBe('https://a/b.png')
  })

  it('open_url 展示链接', () => {
    const v = describeAction(
      { type: 'open_url', url: 'https://example.com' },
      cards,
      wl
    )
    expect(v.kind).toBe('url')
    expect(v.summary).toBe('跳转链接「https://example.com」')
  })
})
