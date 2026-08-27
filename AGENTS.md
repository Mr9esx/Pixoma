# Pixoma 项目约定

## Comet build 默认配置（用户偏好，永久生效）

运行 Comet build 阶段时，默认采用以下配置，除非用户在当次任务中明确要求其他选择：

- 工作区隔离：`current`（当前分支直接工作）
- 执行方式：`executing-plans`（主会话按计划顺序执行）
- TDD 模式：`tdd`
- 代码审查模式：`standard`

## 前端开发约定（用户偏好，永久生效）

前端开发必须优先使用 shadcn/ui 已有组件，并遵循 `shadcn` skill（`.agents/skills/shadcn`，可按需用 `pnpm dlx skills add shadcn/ui` 维护）的规范：

- 优先复用 `web/admin/src/components/ui` 里已安装的 shadcn 组件与注册表组件，不重复造轮子。
- 需要新组件时用项目包管理器运行 shadcn CLI 添加（本项目用 pnpm：`pnpm dlx shadcn@latest`）。
- 遵循 shadcn 规则：用语义类名、内置 variants、`flex` + `gap`，不写裸色值/手写 `dark:`。

## 后台设计体系 skill（用户偏好，永久生效）

`pixoma-design-system` 是本项目的设计体系，已安装在 `.agents/skills/pixoma-design-system-skill`。所有后台前端、交互相关的需求，都必须先加载并遵循该 skill：

- 先读 `README.md` 与 `DESIGN.md`，锁定视觉原则与验收清单。
- 页面引入 `colors_and_type.css`，只用其中语义令牌；禁止新增裸 hex。
- 组件形状照 `ui_kits/app/components.html` 与 `ui_kits/app/components/*.css`，布局照 `ui_kits/app/surfaces.html`；组件本身仍优先复用 shadcn/ui。
- 文案照 `build/source-examples/voice-profile.md`；交付前过一遍 `DESIGN.md` 第 11 节 10 条验收。
