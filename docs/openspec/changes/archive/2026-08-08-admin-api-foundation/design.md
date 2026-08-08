## Context

参见 `proposal.md`。现状：`apps/admin-api` 仅 README 占位；`internal/httpapi/comfyinstances` 已在 `apps/bot` 挂载；`web/admin` 约定只连 admin-api。本 change 只做宿主与实例 API 迁移，其他资源与前端属批次内后续 change。

## Goals / Non-Goals

**Goals:**
- 可独立运行的 admin-api 组合根（配置、DB、chi 路由、CORS、health）
- 复用现有实例仓储/Pool 观测能力，handlers 改由 admin-api 挂载
- bot 卸下管理路由，职责回到对话与编排

**Non-Goals:**
- Case/User/Session/Task 管理 API（`admin-resources-api`）
- 前端菜单与页面（`admin-web-console`）
- 鉴权、RBAC、多租户
- 改变实例调度/健康探测的核心算法（仅保证管理写路径后池可刷新的既有约定）

## Decisions

1. **独立进程而非 bot 子路由**  
   - 选择：新建 `apps/admin-api/cmd/admin-api`。  
   - 相对：继续挂在 bot。  
   - 理由：与既有 monorepo 规划一致，避免管理面与 TG 通道耦合，便于后续只暴露 admin-api。

2. **Handler 包留在 `internal/httpapi`，换挂载点**  
   - 选择：迁移挂载，尽量少改 handler 语义。  
   - 相对：复制一份 admin 专用 handler。  
   - 理由：降低双份协议漂移；admin-api 与 bot 都可依赖同一 httpapi（bot 本期不再 Mount）。

3. **共用同一 DB/配置源约定**  
   - 选择：admin-api 读取与 bot 相同的持久化（如同一 sqlite/DSN），实例变更对 bot 池可见需明确刷新策略（启动加载 + 写后刷新，或依赖既有 Probe/Reload）。  
   - 相对：独立管理库。  
   - 理由：一期运维简单；避免双写。

4. **CORS 宽松联调默认**  
   - 选择：开发配置允许本地前端源；生产收紧留给后续。  
   - 理由：本期无鉴权 + 本地联调优先。

5. **无鉴权**  
   - 选择：明确文档警示，不实现 token。  
   - 理由：用户确认本期不做鉴权。

## Risks / Trade-offs

- [无鉴权公网暴露] → README/配置注释强制警示；默认绑定本机。  
- [bot 与 admin-api 双进程写实例导致池陈旧] → 写路径触发池刷新或文档要求重启/刷新；Design Doc 阶段细化。  
- [BREAKING 运维脚本仍打 bot 端口] → README 与变更说明给出新基址与迁移步骤。

## Migration Plan

1. 落地 admin-api 并可 curl 实例 API。  
2. bot 移除管理路由并发布说明。  
3. 更新本地脚本/README 指向 admin-api。  
4. 回滚：临时恢复 bot Mount（不推荐并行长期双挂）。

## Open Questions

- 实例写后跨进程通知 bot 池刷新的最小机制（共享文件信号 / 仅 Probe 周期 / 管理 API 触发）——不改变规格可观测行为，实现期选定。
