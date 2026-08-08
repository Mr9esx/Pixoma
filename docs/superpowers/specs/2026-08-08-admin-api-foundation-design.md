---
comet_change: admin-api-foundation
role: technical-design
canonical_spec: openspec
---

# admin-api-foundation 技术设计

## 1. 目标与边界

落地可独立运行的 `apps/admin-api`，把 Comfy 实例管理/观测 HTTP 从 bot 迁出；抽出 bot 与 admin-api 共用的启动能力（开库、迁移、种子）。本期无鉴权、无新表。

非目标：Case/User/Session/Task 管理 API、管理前端、登录鉴权、改调度算法本身。

OpenSpec 能力：`admin-api-host`、`comfy-instance-admin-api`（canonical）。

## 2. 数据：继续用现有表

| 项 | 决定 |
|---|---|
| 库 | bot 与 admin-api **同一** `database_dsn` / 库文件（如 `data/app.db`） |
| 表 | 现有 `comfy_instances`（`InstanceRow`）：id、base_url、enabled、capabilities_json、时间戳 |
| Schema | **本期不新增表、不改列**；公共启动只做 AutoMigrate 现有模型 |

## 3. 架构

```
┌──────────────────────────────┐
│ internal/... 公共启动（抽出）   │
│  Open DB → AutoMigrate       │
│  → 可选种子 comfy_instances   │
└──────────────┬───────────────┘
               │
     ┌─────────┴─────────┐
     ▼                   ▼
 apps/bot            apps/admin-api
 TG + 编排生成         :8081（可配）
 探活循环内 Refresh    health + CORS
 卸管理路由            挂 comfyinstances Handler
 可留 /healthz         写后本进程 Refresh
```

- Handler 包留在 `internal/httpapi/comfyinstances`，换挂载进程，协议路径保持 `/api/v1/comfy-instances*`。
- admin-api **禁止**依赖 `channel/tg`。

## 4. 公共启动包（方案 B）

抽出「两边都要」的开机步骤，建议落在既有 platform 附近（如 `internal/platform/appboot` 或等价命名），职责仅限：

1. 按 DSN 打开 DB  
2. AutoMigrate 现有 row 模型（含 `InstanceRow` 等 bot 已用模型；admin-api 至少 migrate 实例相关）  
3. 可选：按配置种子 upsert `comfy_instances`  

**克制**：不把 ConfirmRun、TG、Orchestrator 打进公共包。bot 业务接线仍留在 `apps/bot`。

## 5. 配置与端口

| 进程 | 配置 | 建议默认 |
|---|---|---|
| bot | 现有 `configs/bot.yaml` | `http_addr: ":8080"`，`database_dsn` 指向共享库 |
| admin-api | 新增 `configs/admin-api.yaml` + example | `http_addr: ":8081"`，**相同** `database_dsn`，CORS 允许本地前端源，`comfy_mock` 与观测一致 |

文档强制：DSN 必须一致；无鉴权勿对公网暴露。

## 6. 跨进程名单可见

1. admin-api：实例写成功后对**本进程** `Pool.Refresh`（现有 handler 行为保留）。  
2. bot：在现有 `health_probe_interval` 探活循环中，**顺带** `Pool.Refresh` 从 DB 重建客户端图。  
3. 可观测约定：管理端持久化后，bot **不要求瞬时**，但在一个探活周期内必须能读到新名单并用于调度过滤。

不引入跨进程 RPC/文件通知（本期）。

## 7. BREAKING 与迁移步骤

1. 实现公共启动 + admin-api 可启动。  
2. 挂载实例 API；curl 验收。  
3. bot 移除管理路由挂载；更新 README/curl 指向 admin-api。  
4. 运维脚本改基址；回滚仅紧急时临时恢复 bot Mount（不推荐长期双挂）。

## 8. 测试策略

| 层 | 内容 |
|---|---|
| 单测 | comfyinstances handler 回归；公共启动开库+migrate；bot 探活循环触发 Refresh（可用短间隔/注入时钟或直接测 Refresh 调用路径） |
| 手工 | admin-api CRUD+观测；bot 旧管理路径不可用；同库变更后等待 ≤ 一个 probe 间隔，bot 侧名单更新可观察 |
| 文档 | 端口、同库、无鉴权警示、`make`/启动命令 |

## 9. 实现任务对齐

对应 open 阶段 `tasks.md`：骨架与配置 → 挂载迁移与 bot 卸路由 → bot 周期 Refresh → 验收与 README。公共启动抽取作为骨架任务的前置子步骤落地。
