## 1. 仓储与应用服务补齐

- [x] 1.1 确认/补齐 User、Session 的 List/Get 仓储（含过滤 ListQuery）
- [x] 1.2 确认 Task Cancel（`RequestCancel`）可被 admin-api 复用；扩展管理端 Task List
- [x] 1.3 Case：管理命令/查询/校验可调用；补齐 Enable；扩展 List 过滤

## 2. HTTP 接口

- [x] 2.1 实现 `/api/v1/cases` 列表/详情/创建/更新/禁用/启用
- [x] 2.2 实现 `/api/v1/users` 列表/详情（只读）
- [x] 2.3 实现 `/api/v1/sessions` 列表/详情（只读）
- [x] 2.4 实现 `/api/v1/tasks` 列表/详情与 `POST .../cancel`
- [x] 2.5 统一错误响应与分页/过滤参数风格，并补充 handler 测试

## 3. 宿主整理

- [x] 3.1 admin-api：拆分 router / middleware，挂载四类 httpapi
- [x] 3.2 bot：抽出 `apps/bot/internal/server`（healthz + 中间件），不恢复管理 CRUD

## 4. 文档与验收

- [x] 4.1 更新 admin-api README：四类资源 curl 示例与无鉴权警示
- [x] 4.2 本地联调：对四类资源走通主路径验收场景
