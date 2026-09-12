## 1. 执行门面

- [x] 1.1 测试：无 IM 填表会话提交合法输入 → Task 创建成功；`comfy_mock` 下后续能读到图片 blob
- [x] 1.2 实现 `RunCase`（或等价）校验 schema、创建 Task，不经 admin ConfirmRun
- [x] 1.3 缺必填失败；MCP 用户写入归属，不写 TG chat id；非允许权限不建任务

## 2. MCP 主进程传输与协议

- [x] 2.1 主进程挂载 Streamable HTTP `/mcp` 与旧 SSE `/sse`；initialize 声明 tools/resources/prompts/logging/completions/elicitation，不声明 sampling
- [x] 2.2 tools：list_workflows / get_workflow / run_workflow（发起即返回 task_id）/ list_tasks / get_task；列表不含创建或修改 Case；resources 可读 Case 与产物；list/get 仅本 Bearer 用户
- [x] 2.3 无 Telegram 通道时，启用 MCP 平台 + 允许用户仍能创建 Task

## 3. MCP 消息平台、用户与鉴权

- [x] 3.1 平台类型 `mcp`：创建不填 IM 凭证；不启动 IM 适配器；停用后该渠道 Bearer 全部 401
- [x] 3.2 渠道详情生成用户（显示名、默认允许）+ 每用户 token 密文；复制 / 详情 / 重新生成 / 删除；viewer 无明文
- [x] 3.3 `/mcp` `/sse` 始终 Bearer → 唯一用户；无匹配、仅 cookie、渠道停用、权限非允许 → 拒绝或（run）不建任务；无 yaml 总密钥

## 4. 文档

- [x] 4.1 更新 `docs/architecture/overview.md`、`runtime.md`、`data-model.md`：MCP 入口、mcp 平台、每用户 token、notify 跳过
