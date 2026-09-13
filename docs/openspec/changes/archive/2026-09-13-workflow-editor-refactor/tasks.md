## 1. 后端媒体上传与服务

- [x] 1.1 在 `internal/httpapi/adminhost` 新增 `POST /api/v1/media` 上传接口：multipart 解析、校验 MIME 白名单（png/jpeg/webp/gif、mp4/webm）与体积上限，写入 `blob.Store` 的 `previews/<uuid>.<ext>` 前缀并返回 `{ key, url }`
- [x] 1.2 新增 `GET /api/v1/media/{key}` 接口：从 `blob.Store` 流式读取并返回，携带正确 `Content-Type`，位于统一鉴权 gate 下
- [x] 1.3 为媒体接口补充服务端单元测试（非法类型/超上限被拒、鉴权拉取、流式返回）

## 2. 前端预览效果图上传组件

- [x] 2.1 实现 `MediaPreviewField` 组件：文件选择/拖拽上传、上传中状态、图片缩略与原图预览、视频播放器预览、替换与移除
- [x] 2.2 接入 `lib/api` 媒体上传与拉取客户端（拼接 `VITE_ADMIN_API_BASE`），上传失败展示可诊断错误
- [x] 2.3 将基础信息「预览效果图」字段替换为该组件，`Case.preview` 存媒体引用（blob key）；「Case 说明」保留文本字段
- [x] 2.4 补充组件 contract/行为测试（上传成功、替换、移除、非法类型提示）

## 3. TG 预览媒体投递

- [x] 3.1 调整 `internal/channel/capability/open_case.go` 的 preview 步骤：读取 `Case.preview` 媒体并通过现有 media bridge 发给用户（图片 send photo、视频 send video），文本用 Case 名称与 Case 说明
- [x] 3.2 预览媒体缺失/读取失败时回退文本提示且不阻断流程
- [x] 3.3 补充 TG 投递单元测试（图片/视频投递、缺失回退）

## 4. 统一工作流编辑组件

- [x] 4.1 抽取 `WorkflowEditor` 组件，承载导入 + 基础信息 + 输入 + 输出，`splitPane` 作为布局 prop
- [x] 4.2 让独立编辑页（`CaseForm` / 详情面板）消费 `WorkflowEditor`
- [x] 4.3 让 Quick Config 第一步（`step1-workflow.tsx`）改用 `WorkflowEditor`，删除整包重复用法

## 5. 输入/输出交互优化

- [x] 5.1 实现「一键生成」：导入后按图中字面量参数生成输入草稿、按输出槽位生成输出草稿，类型自动推断
- [x] 5.2 实现宽屏表格化批量编辑（key/类型/必填/描述/绑定列）与新增/删除行；窄屏降级为现有卡片布局
- [x] 5.3 实现内联绑定：点行内「绑定」在侧栏列出节点可绑参数/输出，点选即完成并能回填类型

## 6. 校验、文案与验证

- [x] 6.1 校验提交前输入字段全部绑定节点与参数，未绑定定位到对应字段
- [x] 6.2 补充 `workflow-editor` 相关 i18n 文案（zh/en）与缺失键
- [ ] 6.3 运行前端测试与构建；验证编辑页与向导第一步两端行为一致、TG preview 投递可用
