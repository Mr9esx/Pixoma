import { describe, expect, it } from 'vitest'
import { computeReadiness, workflowStatus } from './readiness'

describe('workflowStatus', () => {
  it('bindings.workflow 为空时为缺口', () => {
    expect(
      workflowStatus({
        bindings: { workflow: {} },
        inputs: [{ key: 'prompt' }],
        input_schema: { type: 'object' },
      })
    ).toBe('gap')
  })

  it('workflow / inputs / input_schema 齐全时为就绪', () => {
    expect(
      workflowStatus({
        bindings: { workflow: { 1: { class_type: 'LoadImage', inputs: {} } } },
        inputs: [{ key: 'image' }],
        input_schema: { type: 'object' },
      })
    ).toBe('ready')
  })

  it('没有输入字段时为缺口', () => {
    expect(
      workflowStatus({
        bindings: { workflow: { 1: { class_type: 'LoadImage', inputs: {} } } },
        inputs: [],
        input_schema: { type: 'object' },
      })
    ).toBe('gap')
  })
})

describe('computeReadiness', () => {
  it('规则完整且 Topic 全部在线、有投放时为三项就绪', () => {
    expect(
      computeReadiness({
        workflow: 'ready',
        rules: [{ topic: 'a' }, { topic: 'b' }],
        enabledTopics: ['a', 'b'],
        boundTopics: ['a', 'b'],
        onlineTopics: ['a', 'b'],
        placements: [{}],
      })
    ).toEqual({ workflow: 'ready', processing: 'ready', placements: 'ready' })
  })

  it('无任何规则时处理流程为缺口', () => {
    expect(
      computeReadiness({
        workflow: 'ready',
        rules: [],
        enabledTopics: ['a'],
        boundTopics: ['a'],
        onlineTopics: ['a'],
        placements: [{}],
      }).processing
    ).toBe('gap')
  })

  it('规则未连接 Topic 时处理流程为缺口', () => {
    expect(
      computeReadiness({
        workflow: 'ready',
        rules: [{ topic: 'a' }, { topic: undefined }],
        enabledTopics: ['a'],
        boundTopics: ['a'],
        onlineTopics: ['a'],
        placements: [{}],
      }).processing
    ).toBe('gap')
  })

  it('使用中的 Topic 未绑定节点时处理流程为缺口', () => {
    expect(
      computeReadiness({
        workflow: 'ready',
        rules: [{ topic: 'a' }, { topic: 'b' }],
        enabledTopics: ['a', 'b'],
        boundTopics: ['a'],
        onlineTopics: ['a'],
        placements: [{}],
      }).processing
    ).toBe('gap')
  })

  it('Topic 全部绑定但无在线节点时为警告（不阻塞发布）', () => {
    expect(
      computeReadiness({
        workflow: 'ready',
        rules: [{ topic: 'a' }],
        enabledTopics: ['a'],
        boundTopics: ['a'],
        onlineTopics: [],
        placements: [{}],
      }).processing
    ).toBe('warn')
  })

  it('没有渠道投放时投放为缺口', () => {
    expect(
      computeReadiness({
        workflow: 'ready',
        rules: [{ topic: 'a' }],
        enabledTopics: ['a'],
        boundTopics: ['a'],
        onlineTopics: ['a'],
        placements: [],
      }).placements
    ).toBe('gap')
  })

  it('工作流缺口会传导到整体结果', () => {
    expect(
      computeReadiness({
        workflow: 'gap',
        rules: [{ topic: 'a' }],
        enabledTopics: ['a'],
        boundTopics: ['a'],
        onlineTopics: ['a'],
        placements: [{}],
      }).workflow
    ).toBe('gap')
  })
})
