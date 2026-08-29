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

const withRules = {
  hasDefaultRoute: false,
  selectedNodeSelected: true,
} as const

describe('computeReadiness', () => {
  it('规则完整且 Topic 全部在线、有投放时为四项就绪', () => {
    expect(
      computeReadiness({
        workflow: 'ready',
        rules: [{ topic: 'a' }, { topic: 'b' }],
        enabledTopics: ['a', 'b'],
        boundTopics: ['a', 'b'],
        onlineTopics: ['a', 'b'],
        placements: [{}],
        ...withRules,
      })
    ).toEqual({
      workflow: 'ready',
      processing: 'ready',
      placements: 'ready',
      node: 'ready',
    })
  })

  it('默认路由（无规则）默认 Topic 已绑定且在线时为就绪', () => {
    expect(
      computeReadiness({
        workflow: 'ready',
        rules: [],
        enabledTopics: ['default'],
        boundTopics: ['default'],
        onlineTopics: ['default'],
        placements: [{}],
        hasDefaultRoute: true,
        selectedNodeSelected: true,
      }).processing
    ).toBe('ready')
  })

  it('默认路由（无规则）默认 Topic 未绑定时处理流程为缺口', () => {
    expect(
      computeReadiness({
        workflow: 'ready',
        rules: [],
        enabledTopics: ['default'],
        boundTopics: [],
        onlineTopics: [],
        placements: [{}],
        hasDefaultRoute: true,
        selectedNodeSelected: true,
      }).processing
    ).toBe('gap')
  })

  it('默认路由（无规则）默认 Topic 绑定但无在线节点时为警告', () => {
    expect(
      computeReadiness({
        workflow: 'ready',
        rules: [],
        enabledTopics: ['default'],
        boundTopics: ['default'],
        onlineTopics: [],
        placements: [{}],
        hasDefaultRoute: true,
        selectedNodeSelected: true,
      }).processing
    ).toBe('warn')
  })

  it('无规则且非默认路由时处理流程为缺口', () => {
    expect(
      computeReadiness({
        workflow: 'ready',
        rules: [],
        enabledTopics: ['default'],
        boundTopics: ['default'],
        onlineTopics: ['default'],
        placements: [{}],
        hasDefaultRoute: false,
        selectedNodeSelected: true,
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
        ...withRules,
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
        ...withRules,
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
        ...withRules,
      }).processing
    ).toBe('warn')
  })

  it('没有消息平台投放时投放为缺口', () => {
    expect(
      computeReadiness({
        workflow: 'ready',
        rules: [{ topic: 'a' }],
        enabledTopics: ['a'],
        boundTopics: ['a'],
        onlineTopics: ['a'],
        placements: [],
        ...withRules,
      }).placements
    ).toBe('gap')
  })

  it('运行节点未选时 node 就绪为缺口', () => {
    expect(
      computeReadiness({
        workflow: 'ready',
        rules: [],
        enabledTopics: ['default'],
        boundTopics: ['default'],
        onlineTopics: ['default'],
        placements: [{}],
        hasDefaultRoute: true,
        selectedNodeSelected: false,
      }).node
    ).toBe('gap')
  })

  it('运行节点已选时 node 就绪为 ready（离线不阻塞，由处理流程在线态承载）', () => {
    expect(
      computeReadiness({
        workflow: 'ready',
        rules: [],
        enabledTopics: ['default'],
        boundTopics: ['default'],
        onlineTopics: [],
        placements: [{}],
        hasDefaultRoute: true,
        selectedNodeSelected: true,
      }).node
    ).toBe('ready')
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
        ...withRules,
      }).workflow
    ).toBe('gap')
  })
})
