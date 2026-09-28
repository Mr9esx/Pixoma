import { describe, expect, it } from 'vitest'
import { nextStudioActivity } from './studio-run-activity'

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

  it('gives the approval card priority and clears progress when the run ends', () => {
    let activity = nextStudioActivity('正在思考', {
      type: 'CUSTOM',
      name: 'pixoma.approval_required',
    })
    expect(activity).toBeNull()

    activity = nextStudioActivity(activity, { type: 'RUN_FINISHED' })
    expect(activity).toBeNull()
  })

  it('hides progress while visible text or reasoning is streaming', () => {
    expect(
      nextStudioActivity('正在思考', { type: 'TEXT_MESSAGE_CONTENT' })
    ).toBeNull()
    expect(
      nextStudioActivity('正在思考', { type: 'REASONING_MESSAGE_CONTENT' })
    ).toBeNull()
  })

  it('continues with a generic phase after an unrecognized event', () => {
    expect(
      nextStudioActivity('正在思考', {
        type: 'CUSTOM',
        name: 'pixoma.asset_created',
      })
    ).toBe('正在思考')
    expect(
      nextStudioActivity('正在思考', {
        type: 'TOOL_CALL_START',
        toolCallName: 'unknown_tool',
      })
    ).toBe('正在调用工具')
  })
})
