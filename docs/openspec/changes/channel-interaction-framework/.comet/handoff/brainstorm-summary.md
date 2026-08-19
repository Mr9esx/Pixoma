# Brainstorm Summary

- Change: channel-interaction-framework
- Date: 2026-08-19

## 已确认事实（open 阶段 + 用户决策）

- 能力注册表：能力 = {id、展示名、参数 schema、执行 handler、各渠道渲染声明}；Go 代码注册；内置 open_case（包装现有 Facade）
- 菜单 = 能力入口树：菜单项 {展示字段 + capability_id + params + children}；kind 业务语义移除
- 统一交互协议：UI 事件 → CapabilityInvoke → Result → 渠道渲染；适配器不感知具体业务
- 渠道账户上下文：权益挂渠道账户（channel + external_user_id），不做跨渠道合并
- TG 交互：**主键盘显式配置（≤6，不自动塞满）**、**分组最多一层（消息按钮内无嵌套）**、**流程走消息按钮且每步可返回/退出**
- 管理端：能力为中心编辑器 + **渠道形态预览**（TG 主键盘/消息按钮 mock 渲染）+ 直白文案（禁 inline/callback/extras/capability）
- 非目标：会员/计费/签到业务、飞书/企微适配器、跨渠道统一账户

## 待确认（候选）

- ~~能力参数 schema 的表达方式~~ → **已确认：JSON Schema**（管理台表单生成器 + 适配器校验驱动）
- ~~渲染声明的存储位置~~ → **已确认：能力定义内按渠道 + 菜单项可覆盖**
- ~~管理端预览的实现方式~~ → **已确认：后端渲染 DTO**（复用渠道渲染逻辑，预览与真实渲染同源）
- ~~open_case 返回语义~~ → **已确认：协议层导航上下文 NavContext**（back 锚点 root/分组 id，适配器统一渲染返回/退出，能力不感知导航）

## 深度技术方案（用户已确认 2026-08-19，先出第一版）

### 能力注册表
- `internal/channel/capability`：`Capability{ID, DisplayName, ParamsSchema(JSON Schema), Invoke(ctx, acct, params) (Result, error), Render(channel) (RenderDecl, error)}`
- 参数校验：JSON Schema 校验（管理台表单与适配器共用）
- 渲染声明：能力定义内按渠道默认 + 菜单项可覆盖；`RenderDecl{entry: root|message_button, config}`
- 内置 `open_case`：包装 `botapp.Facade`，params 支持 case 选择（case_ids）、列表锚点；back 由协议层 NavContext 承担

### 交互协议
- `CapabilityInvoke{CapabilityID, Params, Account, Nav}`；`Nav{back: root|group_id|flow_step}`
- `Result{Text, Options, Media, Error}` 渠道无关
- 适配器：UI 事件 → CapabilityInvoke → registry → Result → 按渠道渲染；返回/退出按钮由协议层统一处理
- `AccountCtx{ChannelID, ExternalUserID, InternalUserID}` 由身份解析填充

### 菜单模型
- `MenuNode{展示字段 + CapabilityID + Params + NavHint + Children}`；kind 业务语义移除；一层分组（消息按钮内无嵌套）
- 主键盘显式配置 ≤6；超出提示放入分组

### TG 渲染（交互设计落地）
- 主键盘 = 管理员显式配置的能力入口（≤6）
- 分组 = 一层，消息按钮列出能力入口 + 返回
- 流程每步有返回/退出（NavContext 驱动）
- open_case 行为等价回归

### 管理端
- 能力为中心编辑器：选能力 → JSON Schema 表单 → 渠道展示微调
- **后端渲染 DTO 预览**：后端按菜单+能力声明返回 TG 主键盘/消息按钮结构，前端绘制（预览与真实渲染同源）
- 文案术语约束（禁 inline/callback/extras/capability）

### 测试策略
- 单元：schema 校验、NavContext、AccountCtx、registry
- 适配器：事件→能力调用、主键盘 ≤6、一层分组、返回/退出、预览 DTO 与真实渲染一致性
- 回归：open_case 全流程、go test 全绿、前端 test/build
