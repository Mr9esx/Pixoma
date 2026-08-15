# Brainstorm Summary

- Change: pixoma-guided-deploy
- Date: 2026-08-12

## 确认的技术方案

- 落地：**方案 A 大爆炸**（默认路径不保留 Redis/queue 双轨）
- 拓扑：`pixoma` 控制面 + `pixoma-edge-agent`；本机自动拉起 Edge
- 派活：DB 可领取 + 长轮询 claim/heartbeat/status；共享 Agent Token
- 配置：bootstrap 本机小库 + 业务库 settings；向导落库；重启生效
- 业务库：SQLite + MySQL + Postgres
- 前端：开发独立；发布 embed 进 pixoma
- 本机 localfs；远程 s3|tos（禁 localfs）

## 关键取舍与风险

- 选大爆炸换干净终点，代价是 BREAKING、无平滑双轨、分支周期长
- 密钥进库需加密；旧 Redis 部署不保证升级
- 子进程 Edge 需随父进程生命周期管理

## 测试策略

- 空目录启动账密 → 本机 SQLite+localfs+mock Edge E2E
- 远程 localfs 拒绝；Token 鉴权；lease 回收
- CI 以 SQLite 为主；MySQL/Postgres 连通+migrate；真 OSS 可选 tag

## Spec Patch

无（OpenSpec delta 已在 open 阶段覆盖；实现中若补场景再回写 delta）
