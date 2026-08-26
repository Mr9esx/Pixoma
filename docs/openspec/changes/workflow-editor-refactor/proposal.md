## Why

工作流编辑页的「预览效果图」「用户需要提供什么（输入）」「用户会得到什么（输出）」体验割裂且低效：预览只能填文本，无法上传图片/视频给 TG 用户看 Case 效果；输入/输出每字段要手工填类型、再打开弹层在图上点节点点参数，字段一多非常繁琐；Quick Config 向导与独立编辑页虽复用表单，但编辑器不是单一可维护组件。

## What Changes

- 「预览效果图」字段升级为媒体上传组件：支持直接上传图片/视频、内嵌预览、替换与移除；该媒体将作为 Case 效果预览发送给 TG 用户。
- 新增管理端媒体上传与鉴权拉取接口，上传落盘到用户文件存储（blob 存储，非公网），`Case.preview` 持久化为该媒体引用。
- TG 侧 `open_case` 的 preview 步骤把预览媒体真实投递给用户：图片报图、视频报视频，标题/文本沿用 Case 名称与「Case 说明」。
- 将「工作流配置」（导入 + 基础信息 + 输入 + 输出）收敛为单一统一编辑组件，被独立编辑页与 Quick Config 第一步共同复用，改一次两端都生效。
- 重做输入/输出创建与绑定：从图自动生成字段、表格化批量编辑、内联绑定，明显减少点击。
- 非目标：不改任务执行、不动路由规则（处理流程）编辑、不改渠道/节点管理页；预览媒体不对公网开放，必须走鉴权访问。

## Capabilities

### New Capabilities
- `workflow-preview-media`: 预览效果图上传、非公网存储（跟随用户文件存储）与鉴权预览；`open_case` 预览步骤把该媒体真实投递给 TG 用户。
- `workflow-input-output-editor`: 统一工作流编辑组件（导入、基础信息、输入、输出），输入/输出字段自动生成、表格化批量编辑与内联绑定；被编辑页与 Quick Config 第一步共用。

### Modified Capabilities
- `quick-config-wizard`: 第一步 MUST 复用同一工作流编辑组件，其输入/输出与预览交互与独立编辑页一致。
- `admin-resource-pages`: Case 创建/编辑页使用统一工作流编辑组件，预览效果图字段支持媒体上传与预览。

## Impact

- 后端管理端：`internal/httpapi/adminhost` 新增媒体上传与鉴权拉取接口；依赖 `blob.Store`，新增 `previews/` 前缀；受统一 gate 保护。
- 后端 TG 投递：`internal/channel/capability/open_case.go` 的 preview 步骤读取预览媒体并经由现有 media bridge 发送给用户（图片/视频）。
- 前端：`web/admin/src/features/cases/`（case-form、sections/basics、field-cards、workflow-import 收敛为统一编辑组件）；`web/admin/src/features/quick-config/step1-workflow.tsx` 消费统一组件。
- 数据：`Case.preview` 字段语义从文本改为媒体引用（blob key），`Case.description` 保持文本（Case 说明）；创建/更新/详情接口契约不变。
- 依赖/i18n：新增上传与绑定相关文案；无新增第三方依赖，复用现有 media bridge 与原语。
