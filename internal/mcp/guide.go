package mcp

const (
	GuideUnauthorized = "检查聊天的 Pixoma MCP 连接器是否指向本实例且凭据仍有效。不要向用户索要 token。配好后重试。"
	GuideUnavailable  = "检查聊天的 Pixoma MCP 连接器是否指向本实例且凭据仍有效。对应的 MCP 渠道或用户可能已停用。不要向用户索要 token。配好后重试。"
	GuideForbidden    = "连接器身份与会话不一致。让用户重连该连接器后重试。不要向用户索要 token。"

	GuideToolAccessDenied = "该 MCP 用户当前不能跑工作流。请管理员在 MCP 渠道用户里改为允许后再试。"
	GuideToolWorkflowGone = "工作流不存在或已停用。先 list_workflows 换已启用的 id。"
	GuideToolMissingInput = "缺少必填：%s。先 get_workflow 再带齐 inputs 重试。"
	GuideToolTaskHidden   = "只能查当前连接器用户的任务。核对 task_id 或先 list_tasks。"
)
