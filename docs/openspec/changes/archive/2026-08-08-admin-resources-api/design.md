## Context

参见 `proposal.md`。依赖 `admin-api-foundation` 的宿主、CORS、无鉴权与实例 API 先例。User 仓储目前偏 TG upsert；Session/Task 写路径敏感，管理面默认只读 + 有限运维动作。

## Goals / Non-Goals

**Goals:**
- Case 完整管理写路径（校验后持久化 + 禁用）
- User/Session 运维查询
- Task 查询 + 取消
- 路由与错误风格与实例 API 对齐

**Non-Goals:**
- 鉴权；前端；ConfirmRun；任意篡改 Session 态机；多租户

## Decisions

1. **资源路径前缀** `/api/v1/{cases|users|sessions|tasks}`，与实例 API 同版本前缀。  
2. **Case**：走 catalog Repository/校验；禁用用既有 `Disable` 语义。  
3. **User**：先 List/Get；若仓储缺 List，本 change 补仓储查询，不引入“管理创建用户为主路径”。  
4. **Session**：只读 List/Get；不提供通用 Update。  
5. **Task**：List/Get + Cancel（复用 runtime 取消应用服务，若已有）。  
6. **分页**：Limit/Offset 或等价查询参数，与现有 ListQuery 风格一致。

## Risks / Trade-offs

- [仓储缺 List 需扩展] → 任务中显式补齐，避免 handler 直查未导出方法。  
- [取消与编排竞态] → 复用领域取消规则，失败返回明确错误。  
- [Case 大 JSON 编辑易错] → 校验失败返回可读错误；深度编辑 UX 留给前端 change。

## Migration Plan

1. 在 admin-api 增加路由与测试。  
2. 文档补充 curl 示例。  
3. 无 bot 路由回迁需求。

## Open Questions

- User List 的排序/过滤字段最小集（实现期按表结构选定，不改变“可列表/详情”规格）。
