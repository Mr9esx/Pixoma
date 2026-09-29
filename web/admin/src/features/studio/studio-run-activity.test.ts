import { describe, expect, it } from 'vitest'
import {
  nextStudioActivity,
  shouldShowStudioShimmer,
  studioActivityLabel,
} from './studio-run-activity'

describe('nextStudioActivity', () => {
  it('shows progress while a model prepares a tool call without visible text', () => {
    let activity = nextStudioActivity(undefined, { type: 'RUN_STARTED' })
    expect(activity).toBe('正在准备回复')

    activity = nextStudioActivity(activity, {
      type: 'CUSTOM',
      name: 'pixoma.model_request_started',
    })
    expect(activity).toBe('正在思考')

    activity = nextStudioActivity(activity, {
      type: 'TOOL_CALL_START',
      toolCallName: 'read_asset',
    })
    expect(activity).toBe('正在读取资产')

    activity = nextStudioActivity(activity, { type: 'TOOL_CALL_RESULT' })
    expect(activity).toBe('正在处理工具结果')
  })

  it('shows the approval phase and clears progress when the run ends', () => {
    let activity = nextStudioActivity('正在思考', {
      type: 'CUSTOM',
      name: 'pixoma.approval_required',
    })
    expect(activity).toBe('等待你批准操作')

    activity = nextStudioActivity(activity, { type: 'RUN_FINISHED' })
    expect(activity).toBeNull()
    expect(nextStudioActivity('正在思考', { type: 'RUN_ERROR' })).toBeNull()
    expect(nextStudioActivity('正在思考', { type: 'RUN_CANCELLED' })).toBeNull()
  })

  it('keeps progress visible while assistant text or reasoning is streaming', () => {
    expect(
      nextStudioActivity('正在思考', { type: 'TEXT_MESSAGE_CONTENT' })
    ).toBe('正在生成回复')
    expect(
      nextStudioActivity('正在思考', { type: 'REASONING_MESSAGE_CONTENT' })
    ).toBe('正在思考')
  })

  it('reports model and tool lifecycle transitions', () => {
    expect(
      nextStudioActivity('正在思考', {
        type: 'CUSTOM',
        name: 'pixoma.model_first_token',
      })
    ).toBe('正在生成回复')
    expect(
      nextStudioActivity('正在生成回复', {
        type: 'CUSTOM',
        name: 'pixoma.model_request_finished',
      })
    ).toBe('正在继续处理')
    expect(
      nextStudioActivity('正在思考', {
        type: 'CUSTOM',
        name: 'pixoma.model_request_failed',
      })
    ).toBeNull()
    expect(
      nextStudioActivity(undefined, { type: 'TOOL_CALL_ARGS' })
    ).toBe('正在准备工具调用')
    expect(
      nextStudioActivity('正在处理工具结果', { type: 'TOOL_CALL_END' })
    ).toBe('正在继续处理')
  })

  it('shows shimmer phases while waiting for user input', () => {
    expect(
      nextStudioActivity('正在思考', {
        type: 'CUSTOM',
        name: 'pixoma.approval_required',
      })
    ).toBe('等待你批准操作')
    expect(
      nextStudioActivity('正在思考', {
        type: 'CUSTOM',
        name: 'pixoma.clarification_required',
        value: { question: '需要补充信息' },
      })
    ).toBe('等待你补充信息')
    expect(
      nextStudioActivity('正在思考', {
        type: 'CUSTOM',
        name: 'pixoma.clarification_required',
        value: { workflow: { name: '分镜工作流' } },
      })
    ).toBe('等待你填写工作流参数')
  })

  it('provides a phase before run events arrive or after reconnecting', () => {
    expect(studioActivityLabel(undefined, 'queued')).toBe('正在排队')
    expect(studioActivityLabel(undefined, 'running')).toBe('正在处理请求')
    expect(studioActivityLabel(undefined, 'waiting_approval')).toBe(
      '等待你批准操作'
    )
    expect(studioActivityLabel(undefined, 'waiting_clarification')).toBe(
      '等待你补充信息'
    )
    expect(
      studioActivityLabel(undefined, 'waiting_clarification', 'workflow')
    ).toBe('等待你填写工作流参数')
    expect(studioActivityLabel('正在思考', 'waiting_approval')).toBe(
      '等待你批准操作'
    )
    expect(
      studioActivityLabel('正在思考', 'waiting_clarification', 'workflow')
    ).toBe('等待你填写工作流参数')
  })

  it('shows shimmer for every active phase and image upload, then hides it after completion', () => {
    for (const activity of [undefined, '正在生成回复', '等待你批准操作']) {
      expect(
        shouldShowStudioShimmer({ runActive: true, activity })
      ).toBe(true)
    }
    expect(
      shouldShowStudioShimmer({ runActive: false, activity: null })
    ).toBe(false)
    expect(
      shouldShowStudioShimmer({ runActive: false, activity: undefined })
    ).toBe(false)
    expect(
      shouldShowStudioShimmer({
        runActive: true,
        activity: '正在思考',
        hasError: true,
      })
    ).toBe(false)
    expect(
      shouldShowStudioShimmer({ runActive: false, activity: null, sendingImages: true })
    ).toBe(true)
    expect(
      shouldShowStudioShimmer({
        runActive: false,
        activity: null,
        sendingImages: true,
        hasError: true,
      })
    ).toBe(false)
  })

  it('uses a visible phase for unrecognized production events', () => {
    expect(
      nextStudioActivity(undefined, {
        type: 'CUSTOM',
        name: 'pixoma.asset_created',
      })
    ).toBe('正在处理请求')
    expect(
      nextStudioActivity('正在思考', {
        type: 'TOOL_CALL_START',
        toolCallName: 'unknown_tool',
      })
    ).toBe('正在调用工具')
  })
})
