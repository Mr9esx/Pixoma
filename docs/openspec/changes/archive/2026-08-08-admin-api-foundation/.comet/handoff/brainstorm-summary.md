# Brainstorm Summary

- Change: admin-api-foundation
- Date: 2026-08-08

## 确认的技术方案

- 路线 B：抽出公共启动（开库 / 迁移 / 种子），bot 与 admin-api 共用
- 独立 admin-api：health、CORS、无鉴权；建议端口 :8081（可配置）
- bot 与 admin-api 共用同一 `database_dsn` / 库文件；继续使用现有 `comfy_instances`，本期无新表、无 schema 变更
- 复用 `internal/httpapi/comfyinstances`，挂到 admin-api；bot 卸下管理 CRUD/观测路由（可保留 `/healthz`）
- admin-api 写实例后立即 `Pool.Refresh` 本进程；bot 在现有探活周期（`health_probe_interval`）内顺带从 DB `Refresh`，接受秒级延迟

## 关键取舍与风险

- B 比「只搬家」多抽取工作，但后续 resources-api 少重复接线；抽取须克制，不吞 bot 业务
- 双进程 DSN 不一致会导致「改了看不见」→ 配置与文档强制同库
- bot 名单最多延迟一个探活间隔
- 无鉴权 → 仅本机/可信内网，文档警示

## 测试策略

- 单测：handler 回归；公共启动开库迁移；bot 周期 Refresh
- 手工：admin-api curl CRUD/观测；bot 旧管理路径不可用；同库变更后一轮探活内 bot 可见
- 文档：新端口、同库、无鉴权

## Spec Patch

- 在 `comfy-instance-admin-api` 增加：管理端持久化变更后，bot 在约定刷新周期内可见（不要求瞬时）
