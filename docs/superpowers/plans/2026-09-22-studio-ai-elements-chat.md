# Studio Chat 采用 AI Elements

> 范围仅限 `StudioWorkspace` 中 `main#main-content` 的 `StudioChat` 对话区域；右侧资产、画布和会话资产流程不变。

1. 保留现有 AG-UI WebSocket agent、线程历史和 Studio 运行配置，把 assistant-ui 限定为运行时桥接层。
2. 使用 AI Elements 的 `Conversation`、`Message`、`MessageResponse`、`Reasoning`、`Tool`、`PromptInput`、`Suggestion` 和 `ModelSelector` 重建对话呈现与交互。
3. 让 AI Elements 的 Markdown/流式、推理和工具调用组件承接相应消息 part；保留模型、Skill、资产、权限选择器及其原有业务逻辑。
4. 更新 Studio 对话契约测试，验证 AI Elements 呈现层和 AG-UI 运行时桥接共存，并运行针对性测试及类型检查。
