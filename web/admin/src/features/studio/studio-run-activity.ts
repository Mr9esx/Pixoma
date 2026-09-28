export function nextStudioActivity(
  activity: string | null | undefined,
  event: { type: string; name?: unknown; toolCallName?: unknown }
): string | null | undefined {
  if (event.type === 'RUN_STARTED') return '正在准备回复'
  if (
    event.type === 'RUN_FINISHED' ||
    event.type === 'RUN_ERROR' ||
    event.type === 'RUN_CANCELLED'
  )
    return null
  if (
    event.type === 'TEXT_MESSAGE_CONTENT' ||
    event.type === 'REASONING_MESSAGE_CONTENT'
  )
    return null
  if (
    event.type === 'TEXT_MESSAGE_END' ||
    event.type === 'REASONING_MESSAGE_END'
  )
    return '正在继续处理'
  if (event.type === 'TOOL_CALL_RESULT') return '正在处理工具结果'
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
    if (event.name === 'pixoma.model_request_started') return '正在思考'
    if (
      event.name === 'pixoma.approval_required' ||
      event.name === 'pixoma.clarification_required'
    )
      return null
  }
  return activity
}
