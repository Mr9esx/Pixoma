## Context

工作流（Case）创建/编辑有两条入口：独立编辑页走 `CaseForm`（`web/admin/src/features/cases/case-form.tsx`），Quick Config 向导第一步也复用 `CaseForm`（`web/admin/src/features/quick-config/step1-workflow.tsx`，以 `splitPane` 布局）。两者共用同一表单，但「工作流配置」由分散 section（basics / workflow-import / field-cards）组合，缺少一个可整体复用、语义内聚的编辑组件。

输入/输出是逐字段卡片（`field-cards.tsx`）：每字段手工填 key + 类型，再打开 `BindNodeDialog` 在图上点节点→点参数完成绑定，字段越多越繁琐。预览字段 `preview`（`sections/basics.tsx`）目前是普通文本输入框；该文本被 TG 侧 `open_case` 的 preview 步骤拼成「预览说明：…」展示给机器人用户（`internal/channel/capability/open_case.go`）。

管理后台目前没有文件上传端点（`internal/httpapi/adminhost` 只有资源 CRUD）；控制面进程已有 `blob.Store`（用于运行时输入/任务包/产物流转）和 `tgMediaBridge.Upload`（向 TG 发送媒体）。控制面所有 `/api/v1` 在统一鉴权 gate 下；TG 投递走不受公网影响的控制面出站。

## Goals / Non-Goals

**Goals:**
- 预览效果图支持上传图片/视频、内嵌预览、替换与移除；媒体持久化到用户文件存储（blob 存储，非公网）。
- 在 TG `open_case` 的 preview 步骤把预览媒体真实投递给用户（图片报图、视频报视频），文本沿用 Case 名称与 Case 说明。
- 把「工作流配置」收敛为单一统一编辑组件，独立编辑页与 Quick Config 第一步共用同一实现（改一次两端生效）。
- 优化输入/输出创建与绑定：从图自动生成字段、表格化批量编辑、更短的内联绑定路径。

**Non-Goals:**
- 不改任务执行、路由规则（处理流程）编辑、消息平台/节点管理页。
- 不做公网可直接访问的媒体；预览媒体经管理端鉴权，投递走控制面出站。
- 不引入新的第三方上传/编辑器依赖；复用现有 blob 存储、传媒与统一鉴权。

## Decisions

### 1. 预览媒体持久化并单一引用
上传写入与运行时相同的 `blob.Store`，前缀 `previews/<uuid>.<ext>`；`Case.preview` 存该 blob key。同一引用同时供：管理端预览（`GET /api/v1/media/{key}`）与 TG 投递（open_case 读取）。**备选**（放弃）：把媒体塞进 `preview` 的 dataURL——体积大、无法被 TG media bridge 复用、与"跟随用户文件存储"不符。

### 2. 管理端上传/拉取接口：`POST /api/v1/media` 与 `GET /api/v1/media/{key:.*}`
- `POST /api/v1/media`：multipart 上传，校验 MIME 白名单（png/jpeg/webp/gif、mp4/webm）与体积上限，写 `previews/<uuid>.<ext>`，返回 `{ key, url }`。
- `GET /api/v1/media/{key}`：从 blob 流式读取并返回，携带正确 `Content-Type`，位于统一鉴权 gate 下，不对公网开放。
- `Case.preview` 存 blob key（相对定位），创建/更新/详情接口契约不变。

### 3. TG preview 步骤投递预览媒体
`open_case` 的 preview 步骤 MUST 改变：不再把 `doc.Preview` 当文本拼进提示，而是用现有 `App` 可用的 media bridge + blob 读取把预览媒体发给用户（图片→send photo、视频→send video），文本为 Case 名称与 Case 说明；媒体缺失时回退为当前文本提示。
- 依赖：控制面已有 `blob.Store` 与 `tgMediaBridge.Upload`（腹中现成），无需新通道。

### 4. 统一编辑组件：抽取 `WorkflowEditor`
把 basics、workflow-import、inputs、outputs 组织为可复用 `WorkflowEditor`（`features/cases/workflow-editor.tsx`），由独立编辑页（`CaseForm`）与 `step1-workflow.tsx` 公共消费；`splitPane` 作为布局 prop。**备选**（放弃）：Quick Config 继续整包 `CaseForm`——props 混入布局与保存语义，职责不清。

### 5. 输入/输出创建与绑定优化
- 「一键生成」：导入后把图中全部字面量参数生成输入草稿、全部输出槽位生成输出草稿，类型自动推断。
- 表格化批量编辑：宽屏每字段一行（key/类型/必填/描述/绑定），行内增删。
- 内联绑定：行内点「绑定」在侧栏列出节点可绑参数/输出，点选即绑定并回填类型。

## Risks / Trade-offs

- [上传体积大、类型不可控] → 管理端 MIME 白名单 + 体积上限；前端预检并展示错误。
- [大视频内存占用] → 拉取时从 blob 流式 `io.Copy` 到响应，不整读进内存。
- [TG 投递失败] → preview 步骤对媒体投递失败时回退文本提示并记录日志，不阻断流程。
- [统一组件波及两端入口] → 保留编辑页与向导各自 contract test；先编辑页验证再切向导。
- [预览 key 泄露未鉴权访问] → 位于统一 gate 下；TG 投递走控制面出站、不依赖公网路由。
- [表格窄屏拥挤] → 窄屏降级为字段卡片布局（保留现有卡片）。

## Migration Plan

- 后端先合 media 接口与 TG 投递（向后兼容），前端再切换 preview 字段与统一组件。
- 既有 `doc.Preview` 文本值：作为旧数据保留，若无法识别为 blob key 则不投递媒体、回退文本。
- 回滚：关闭 media 接口与 TG 投递改动后，前端退化文本输入，不破坏已存数据。

## Open Questions

- 媒体体积上限建议 ≤25MB，视频时长限制可随实现确认。
- 既有文本型 `preview` 老数据是否要迁移为 blob key：默认不迁移，保留兼容展示。
