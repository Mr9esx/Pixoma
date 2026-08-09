## Why

后端管理 API 就绪后，仍缺少基于 shadcn-admin 的管理后台前端：无菜单信息架构、无资源页面，运维只能靠 curl。需要在 `web/admin` 落地控制台骨架、菜单模块与各资源页面，并只通过 admin-api 完成基础管理操作。

## What Changes

- 以 satnaing/shadcn-admin（React + Vite + shadcn/ui + TanStack Router）为基座初始化 `web/admin`（pnpm）
- 实现后台**菜单模块**与默认 **Dashboard**（侧栏信息架构、路由注册、与资源页映射）
- 设计并实现资源页面：实例、Case、User、Session、Task（列表/详情/表单等基础管理交互）；Case 以结构化表单为主
- 对接 admin-api（含 foundation 实例 API 与 resources API）；交付以真 API 为准，不留独立前端 mock
- 中英双语齐全 + 语言切换；本期不做登录鉴权页；前端不直连 DB；不做 TG/Menu 落库管理

## Capabilities

### New Capabilities

- `admin-web-shell`: shadcn-admin 脚手架、布局、主题与菜单模块
- `admin-resource-pages`: 五类资源页面的信息架构、交互与对接 admin-api 的基础管理能力

### Modified Capabilities

- （无）

## Impact

- 代码：`web/admin/**`（从占位 README 变为可运行 SPA）
- 依赖：`admin-api-foundation`（必需）；全资源页依赖 `admin-resources-api`
- 工具链：Node + pnpm
- 非目标：Clerk/完整鉴权、实时大屏/告警、TG/Menu 落库、前端 mock、向公网发布、Bot UI
