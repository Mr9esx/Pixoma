export function nextStudioActivity(
  activity: string | null | undefined,
  event: {
    type: string
    name?: unknown
    toolCallName?: unknown
    value?: unknown
  }
): string | null | undefined {
  if (event.type === 'RUN_STARTED') return '正在准备回复'
  if (
    event.type === 'RUN_FINISHED' ||
    event.type === 'RUN_ERROR' ||
    event.type === 'RUN_CANCELLED'
  )
    return null
  if (
    event.type === 'TEXT_MESSAGE_START' ||
    event.type === 'TEXT_MESSAGE_CONTENT'
  )
    return '正在生成回复'
  if (
    event.type === 'REASONING_MESSAGE_START' ||
    event.type === 'REASONING_MESSAGE_CONTENT'
  )
    return '正在思考'
  if (
    event.type === 'TEXT_MESSAGE_END' ||
    event.type === 'REASONING_MESSAGE_END'
  )
    return '正在继续处理'
  if (event.type === 'TOOL_CALL_RESULT') return '正在处理工具结果'
  if (event.type === 'TOOL_CALL_ARGS') return activity ?? '正在准备工具调用'
  if (event.type === 'TOOL_CALL_END') return '正在继续处理'
  if (event.type === 'TOOL_CALL_START') {
    switch (event.toolCallName) {
      case 'list_session_assets':
        return '正在查看资产'
      case 'read_asset':
        return '正在读取资产'
      case 'create_text_asset':
        return '正在创建资产'
      case 'update_text_asset':
        return '正在更新资产'
      case 'load_skill':
        return '正在读取技能'
      case 'edit_session_flow':
        return '正在编辑流程'
      default:
        return '正在调用工具'
    }
  }
  if (event.type === 'CUSTOM') {
    if (
      event.name === 'pixoma.model_request_started' ||
      event.name === 'pixoma.model_first_token'
    )
      return event.name === 'pixoma.model_first_token' ? '正在生成回复' : '正在思考'
    if (event.name === 'pixoma.model_request_finished') return '正在继续处理'
    if (event.name === 'pixoma.model_request_failed') return null
    if (event.name === 'pixoma.approval_required') return '等待你批准操作'
    if (event.name === 'pixoma.clarification_required') {
      const value = event.value
      const hasWorkflow =
        typeof value === 'object' && value !== null && 'workflow' in value
      return hasWorkflow ? '等待你填写工作流参数' : '等待你补充信息'
    }
  }
  return activity ?? '正在处理请求'
}

export function studioActivityLabel(
  activity: string | null | undefined,
  runStatus: string | undefined,
  waitingFor?: 'approval' | 'clarification' | 'workflow'
): string {
  if (waitingFor === 'approval' || runStatus === 'waiting_approval')
    return '等待你批准操作'
  if (waitingFor === 'workflow') return '等待你填写工作流参数'
  if (waitingFor === 'clarification' || runStatus === 'waiting_clarification')
    return '等待你补充信息'
  if (runStatus === 'queued') return '正在排队'
  return activity ?? '正在处理请求'
}

export function shouldShowStudioShimmer({
  runActive,
  activity,
  sendingImages = false,
  hasError = false,
}: {
  runActive: boolean
  activity: string | null | undefined
  sendingImages?: boolean
  hasError?: boolean
}): boolean {
  return !hasError && (sendingImages || (runActive && activity !== null))
}
