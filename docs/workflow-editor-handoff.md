# 工作流编辑页 —— 原型 → 真实组件交接文档

> 目标：让 Codex（或任何实现者）以 `workflow-editor-optimized.html` 为视觉与交互来源，
> 在 Pixoma 的 `web/admin` 里落地成真实 React 组件。本文件描述了迁移映射、实现约束与验收口径。

## 1. 设计来源
- 原型文件：
  `workflow-editor-optimized.html`
  （OpenDesign 项目目录内；若在 Codex 中找不到，请读取其绝对路径，或将本文件复制到 `Pixoma/docs/` 后以 `docs/` 路径引用。）
- 原型是**单文件、自包含**实现，只承载视觉/交互演示，不直接可复用模块。

## 2. 现状梳理（真实代码，实施前请确认）
| 关注点 | 现状文件 | 关键点 |
|---|---|---|
| 表单主逻辑 | `web/admin/src/features/cases/case-form.tsx` | `CaseForm` 承载全部编排；`drafts`/`graph` state、`buildPayload()`、`onSubmit()`；`splitPane` 布局 prop |
| 基础信息 | `web/admin/src/features/cases/sections/basics.tsx` | `preview` 是纯文本 `<Input>`（需换上传组件） |
| 字段编辑 | `web/admin/src/features/cases/sections/field-cards.tsx` | `InputFieldCard`/`OutputFieldCard`（竖排卡片）+ `BindNodeDialog`（两步弹窗） |
| 工作流导入 | `web/admin/src/features/cases/sections/workflow-import.tsx` | 拖拽 dropzone + `CodeEditor` |
| 详情页封装 | `web/admin/src/features/cases/detail-panel.tsx` | `Dialog max-w-3xl` 内嵌 `CaseForm`（编辑 info / workflow 两入口） |
| 一键配置 Step1 | `web/admin/src/features/quick-config/step1-workflow.tsx` | 复用 `CaseForm` + `splitPane` |
| 解析/推导 | `web/admin/src/features/cases/lib/{workflow-parse,derive,derive.test}.ts` | `parseWorkflow`、`deriveBindings/InputSchema`、`validateEditor` |
| i18n | `web/admin/src/lib/i18n/locales/zh.json` / `en.json` | 文案键见 §4 |
| 设计 tokens | `web/admin/src/styles/theme.css` | oklch 单色 + terracotta accent；**不得改** |
| 迁移计划 | `docs/openspec/changes/workflow-editor-refactor/{design.md,tasks.md}` | 已定义分阶段目标，建议对齐 |

## 3. 原型 → 真实组件映射（一次性实现清单）
| 原型元素 / 交互 | 落点（真实文件） | 实现要点 |
|---|---|---|
| 单列布局、基础信息在顶部 | `case-form.tsx` 非 `splitPane` 分支 | 保持对话框内窄屏堆叠；`splitPane` 仅向导用 |
| 标题与提示文案 | 沿用现有 i18n 键，**不要新造重复文案** | 输入标题=`cases.inputsHeading`（用户需要提供什么），提示=`inputsHint`；输出=`outputsHeading`/`outputsHint`（用户会得到什么） |
| 宽屏表格化批量编辑 | 重构 `field-cards.tsx`（或新建 table 组件） | 每行：参数名/类型/绑定/必填/描述/删除；窄屏降级为现有卡片布局 |
| 轻量绑定浮层（锚定当前行） | 替换 `BindNodeDialog` 弹窗 | `Popover`/小浮层锚定该行按钮，点参数即绑定并自动推断类型；**不放大弹窗、不触发页面滚动** |
| 一键生成输入/输出草稿 | `case-form.tsx` / 新 hook | 导入后按图字面量参数生成输入草稿、按输出槽位生成输出草稿，类型自动推断（对齐 `derive.ts`） |
| 独立编辑页与向导共用 | 抽取统一 `WorkflowEditor` | `case-form.tsx` 与 `step1-workflow.tsx` 共享；`splitPane` 仅作 prop。新建 `workflow-editor.tsx`（当前不存在） |
| 封面效果图上传 | 新建 `MediaPreviewField`，替换 `basics.tsx` 的文本 `preview` | 拖拽/选择、上传中、缩略/视频预览、替换、移除、类型白名单与体积校验 |
| 顶部/底部单一主 CTA | `detail-panel.tsx` 对话底栏 + `case-form.tsx` | 一个主「保存/创建」，其余为 secondary/ghost；同语义入口勿重复 | 

## 4. 硬性约束（不得违反）
1. **设计系统锁定**：只用 `theme.css` 现有 tokens（oklch）；不引入新 hex 色板、不换字体栈、不新增全局样式类（需新增类先加入 `<style>`/组件层）。
2. **文案**：复用现有 i18n 键与文案；中文标题/提示照旧，不重写不重复。需新增文案时同步补 `zh.json` 与 `en.json`。
3. **交互原则**：
   - 单列布局适配对话框窄屏；不做左右分栏（页面宽度不够）。
   - 绑定交互保持轻量：锚定行、单次点击完成、自动推断类型；禁止两步大弹窗/滚动。
   - 每行未绑定要有行内可诊断提示（如「未绑定节点」），提交前校验并定位到字段（对齐 `validateEditor`）。
4. **后端改动独立**：`POST/GET /api/v1/media` + TG 预览投递（`internal/httpapi/adminhost`、`internal/channels/application/capability/open_case.go`）可单独成任务，纯前端阶段不要混入。

## 5. 分阶段实施（建议顺序）
- **阶段 A（纯前端，安全先做）**：抽取 `WorkflowEditor`；表格化批量编辑 + 轻量绑定浮层；一键生成；文案与校验。两端（独立页 + Step1）行为一致。
- **阶段 B**：`MediaPreviewField` 上传组件 + `lib/api` 媒体客户端（`VITE_ADMIN_API_BASE`），替换 `preview` 文本输入。
- **阶段 C（后端）**：`/api/v1/media` 上传/拉取接口 + TG 预览投递，再接入前端上传的后端地址。

## 6. 验收清单
- [ ] 独立编辑页与 QuickConfig Step1 共用同一 `WorkflowEditor`，行为一致
- [ ] 宽屏表格化、窄屏卡片降级；行内绑定浮层无需滚动即可完成绑定
- [ ] 输入/输出标题与提示沿用现有 i18n 文案（用户需要提供什么 / 用户会得到什么）
- [ ] 「一键生成」按字面量输入/输出槽位生成草稿并自动推断类型
- [ ] 提交前校验未绑定字段并定位；未绑定行有行内提示
- [ ] `preview` 字段为上传组件（图片/视频、替换、移除、白名单 + 体积校验）
- [ ] 现有 contract/单元测试仍通过（`workflow-editor.contract.test.ts` 等）；依赖后端时前端可回退文本输入
- [ ] 设计系统 token 未变，无新增 hex、无重复主 CTA、无页面滚动 hack

## 7. 已知缺口（实现时注意）
- `web/admin/src/features/cases/workflow-editor.tsx` 目前**不存在**，需要新建以承载统一组件。
- `sections/field-cards.tsx` 当前是卡片 + `BindNodeDialog` 两步弹窗，需重构为表格 + 浮层。
- `sections/basics.tsx` 的 `preview` 仍是普通文本输入，需替换。
- 后端 media 接口尚未实现，前端上传先按计划预留、后端就绪后接入。
