## 1. 能力注册表

- [x] 1.1 新增 `internal/channel/capability`：Capability 接口（ID/DisplayName/ParamsSchema(JSON Schema)/Invoke/Render）、Registry 与 `Register`
- [x] 1.2 参数校验：按能力 JSON Schema 校验能力调用参数，非法返回可理解错误
- [x] 1.3 内置 `open_case` 能力：包装现有 `botapp.Facade`（预览/开始/填表/确认/结果），params 支持 case 选择与 back 引用
- [x] 1.4 能力清单查询：注册表支持按渠道筛选渲染声明，供管理台与适配器读取

## 2. 菜单模型改为能力入口

- [ ] 2.1 `MenuNode` 移除业务 kind：新增 `CapabilityID` 与 `Params`；folder 语义降为纯分组（CapabilityID 空）；placeholder/reply 保留为展示字段
- [ ] 2.2 校验规则更新：未知 capability_id 拒绝；params 按 schema 校验；分组子项规则（纯分组/能力入口均可）重新定义
- [ ] 2.3 持久化：`channel_menu_items` 增加 capability_id / params_json 列；删除 kind 业务枚举
- [ ] 2.4 数据迁移：旧 kind → 能力入口（open_case→open_case 能力、placeholder/reply→展示字段、folder→分组）；`tg_root_layout` extras → open_case 的 tg 渲染声明
- [ ] 2.5 默认种子改为能力入口（open_case 挂载图片 Case）

## 3. 统一交互协议

- [x] 3.1 端口 `Action` 泛化为 `CapabilityInvoke{CapabilityID, Params, Account, Nav}`；`Result{Text/Options/Media/Error}` 渠道无关结果结构
- [x] 3.2 `AccountCtx{ChannelID, ExternalUserID, InternalUserID}`：适配器经身份解析填充，能力执行携带
- [ ] 3.3 适配器契约：UI 事件 → CapabilityInvoke → registry 执行 → Result → 渠道渲染；适配器不写业务分支

## 4. TG 适配器按协议渲染

- [ ] 4.1 主键盘：根层能力入口直达按钮 ≤6；超出进「更多」分组或消息按钮
- [ ] 4.2 分组与流程：点分组 → 消息按钮列出子项与 open_case 入口；返回语义保留；每行按钮数按渲染声明
- [ ] 4.3 open_case 行为等价回归：预览/开始/填表/确认/出图通知
- [ ] 4.4 移除适配器内业务 switch（open_folder/open_case 等专属动作），改走 registry

## 5. 管理台改版

- [ ] 5.1 菜单编辑器：入口类型改为「选择能力（来自 registry）→ 按 ParamsSchema 渲染参数表单 → 渠道展示微调」；移除 kind 选择
- [ ] 5.2 移除 extras 独立编辑入口（旧数据已迁移为渲染声明）
- [ ] 5.3 用户文案术语约束：管理台与 bot 文案禁用 inline/callback/extras/capability 等内部术语，统一「按钮/选项/功能」

## 6. 交互设计实现（用户端 + 管理端）

- [ ] 6.1 用户端交互：主键盘显式配置（≤6，平台不自动塞满）；超出入口限制提示
- [ ] 6.2 用户端交互：分组最多一层（消息按钮内只有能力入口，无嵌套分组）
- [ ] 6.3 用户端交互：流程每步提供返回/退出按钮（返回上一级或主菜单）
- [ ] 6.4 管理端预览：按当前配置模拟渲染 TG 主键盘与消息按钮流程（遵循显式配置与一层分组规则）

## 7. 测试与回归

- [ ] 7.1 单元：能力注册/参数校验/Result 结构/AccountCtx
- [ ] 7.2 领域与持久化：菜单能力入口校验、params_json 往返、迁移脚本
- [ ] 7.3 适配器：事件→能力调用翻译、主键盘 ≤6、一层分组消息按钮、返回语义
- [ ] 7.4 API 与前端：菜单 API 载荷（capability_id）、编辑器 schema 表单、预览渲染
- [ ] 7.5 全量 `go test ./...` + `pnpm test/build` + TG 手工回归（Case 流程等价 + 主键盘显式配置/一层分组/返回退出）
