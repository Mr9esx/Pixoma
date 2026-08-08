## 1. Admin API 骨架

- [x] 1.1 创建 `apps/admin-api/cmd/admin-api` 入口与配置加载（含 `configs/admin-api.example.yaml`）
- [x] 1.2 Wire DB 与实例仓储/Pool（与 bot 共用持久化约定）
- [x] 1.3 挂载 `/healthz`、CORS 中间件与 chi 路由骨架
- [x] 1.4 补充 Makefile/README：启动方式与无鉴权公网警示

## 2. 实例管理 API 迁移

- [x] 2.1 在 admin-api 挂载 `/api/v1/comfy-instances*`（复用 `internal/httpapi/comfyinstances`）
- [x] 2.2 验证 CRUD + system/queue/tasks 观测与迁出前语义一致（含 mock 标记）
- [ ] 2.3 选定并实现实例写后对 bot 侧池可见的最小策略（Probe/刷新/文档约定之一）
- [x] 2.4 从 `apps/bot` 移除管理 CRUD/观测路由挂载并更新相关说明

## 3. 验收与回归

- [ ] 3.1 独立启动 admin-api，用 curl 走通实例列表/创建/更新/观测
- [ ] 3.2 确认 bot 原管理路径不再提供管理 API
- [ ] 3.3 更新根 README「实例管理与观测」指向 admin-api
