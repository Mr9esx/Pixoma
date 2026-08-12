---
comet_change: pixoma-guided-deploy
role: technical-design
canonical_spec: openspec
---

# Pixoma 引导式部署（大爆炸）技术设计

## 背景

当前部署依赖多进程、YAML/`runtime_mode`/`queue`/`blob` 矩阵，以及 Redis Streams 跨进程派发，上手成本高。本 change（用户确认不拆分、落地选 **方案 A 大爆炸**）一次性切换到：

- 唯一控制面二进制 `pixoma` + 执行面 `pixoma-edge-agent`
- 默认无 Redis、无用户可见 queue、无 allinone/split
- 派活：DB 可领取态 + Edge 长轮询
- 配置：后台向导落库；发布包内嵌管理前端

OpenSpec 事实源：`docs/openspec/changes/pixoma-guided-deploy/`。

## 已确认产品决策

| 项 | 选择 |
|---|---|
| 落地策略 | 方案 A：大爆炸，默认路径不保留 Redis 双轨 |
| Edge 鉴权 | 平台共享 Agent Token |
| 业务库 | SQLite + MySQL + Postgres |
| 管理前端 | 开发独立 Vite；发布 `go:embed` 进 `pixoma` |
| 本机 Edge | `pixoma` 自动拉起子进程 |
| 配置生效 | 向导完成后重启生效 |

## 架构

```text
pixoma（控制面）
  bootstrap DB（本机）
  业务 DB（向导所选）
  TG / Orchestrator / Blob
  管理 API +（发布）静态后台
  Agent API：claim / heartbeat / status
  本机：spawn pixoma-edge-agent
        │
        ▼  Bearer agent_token
pixoma-edge-agent
  长轮询领取 → Blob.Get(job) → Comfy → status
```

硬规则：

- 执行面只有 Edge；控制面不内嵌生产 Actuator
- 跨进程不依赖 Redis Streams / memory Bus
- 本机：localfs 共用目录；远程：s3|tos，禁止 localfs
- 同进程编排可用函数/内存通道，不暴露为 queue 产品

## 数据模型

### 引导态（bootstrap）

本机独立小库（如 `data/bootstrap.db`），至少含：

- `initialized`
- 管理员用户名与密码哈希
- 业务库 `driver` + DSN
- 向导进度
- `agent_token`（或哈希；明文仅生成时展示/写入日志策略见安全）

### 业务 settings

写入业务库，包括：部署位置（本机/远程）、blob 驱动与密文凭证、localfs 根路径、TG Token、是否自动拉起 Edge、claim `wait` / `lease` 参数等。  
**不包含**用户侧 `queue.driver` / `runtime_mode`。

### Task 领取语义

在现有 `tasks` 上扩展：

- 状态流：`pending` →（调度选定实例并写好 job 包）→ `queued`（可领取）→（claim）→ `running`（带租约）→ 终态
- 字段：`job_ref`、`lease_until`；可选 `claim_generation` 防迟报
- 租约过期且未终态：回到 `queued` 可再领（或按策略 failed）
- 原子 claim：扩展现有 `ClaimQueued` 语义为带 lease 的领取

### Edge 在线

心跳写入控制面持久化（推荐 `edge_heartbeats` 表或等价），不依赖 Redis key。

## Agent API

鉴权：`Authorization: Bearer <agent_token>`（共享 Token）。

| 方法 | 路径 | 行为 |
|---|---|---|
| GET | `/agent/v1/jobs/claim?instance_id=&wait=25s` | 长轮询领取；返回 `task_id` + `job_ref` 等；超时空气 |
| POST | `/agent/v1/jobs/{id}/heartbeat` | 续租约 + 在线 |
| POST | `/agent/v1/jobs/{id}/status` | 上报状态，进入既有幂等 `applyStatus` |

默认参数建议：`wait=25s`，`lease=90s`，心跳 ≈ `lease/3`。

## 向导与管理

步骤：

1. 强制改默认管理员密码  
2. 选库 SQLite/MySQL/Postgres → 测连通 → migrate  
3. 本机 / 远程  
4. 存储（远程禁 localfs）  
5. 执行面说明（本机自动拉起 / 远程安装命令含 URL、token、instance_id）  
6. TG Bot Token 等  
7. finalize → `initialized=true` → 提示重启  

未初始化：仅登录 + 向导 API。  
已初始化：管理 API 需管理员会话（相对今日无鉴权为 BREAKING）。

`pixoma` 一体托管原 admin-api 能力；独立 `admin-api` 不再作为新部署默认路径。

## 进程与二进制

### pixoma

启动序：读 bootstrap →（已初始化则）连业务库与 settings → 装配 blob/调度/TG → 挂管理与 Agent 路由 →（发布）embed 静态页 → 本机则 spawn Edge 子进程（父退出时回收子进程）。

### pixoma-edge-agent

最小配置：控制面 URL、agent token、instance_id、blob、comfy_mock/base_url。循环：claim → 执行 → status/heartbeat。

### 大爆炸删除/停用（默认路径）

- Redis Streams 作为派发/回传总线  
- 用户配置 `runtime_mode` / `queue.driver`  
- Bot 进程内嵌生产执行面作为默认形态  
- 新部署主路径依赖手改 `bot.yaml`

允许保留极少数 env 紧急覆盖；不提供「旧 Redis split 平滑双轨」。

## 安全

- 默认监听本机回环；公网需显式配置  
- 默认密码仅未初始化（或未改密）时出现在日志；改密后不再打印明文  
- settings 中 ak/sk、Token 等：使用引导态派生密钥加密存储  
- Agent Token 与管理员密码同等敏感；轮换需重新下发 Edge 配置  

## 测试策略

必过：

1. 空目录启动：日志含 URL + 默认账密  
2. 向导 SQLite 本机 + localfs → 重启 → mock Edge 自动拉起 → 任务 claim 成功  
3. 远程 + localfs 校验失败  
4. 错误/缺失 Agent Token → 401  
5. Edge 被杀 → lease 过期后可再领（或约定收口）  

库：CI 以 SQLite 为主；MySQL/Postgres 至少 migrate+连通（testcontainers 或 build tag）。  
真 OSS/远程：可选 `-tags` 门禁，不挡默认 `go test`。  
Comfy mock 开关在新拓扑下必须仍能端到端成功。

## 风险与缓解

| 风险 | 缓解 |
|---|---|
| 主干长时间不可用 | 独立分支；合并前本机 mock E2E 必过 |
| 旧部署一夜损坏 | 发布说明 BREAKING；无兼容期 |
| 三库方言差异 | GORM 统一；向导强制连通+migrate |
| 密钥与 bootstrap 备份 | 文档威胁模型；加密 at rest |
| 子进程泄漏 | 父退出杀子；启动前清理旧 child |

## 迁移

- 新安装：只文档 `pixoma` + 向导 + Edge  
- 旧 Redis/YAML/多进程：不保证原地升级；需按向导重建（可选数据导出不作为本版必达）  
- 回滚：旧版二进制 + 旧配置，文档标明版本边界  

## Spec 对齐说明

若实现中需补验收场景，仅回写 OpenSpec delta（`platform-bootstrap` / `setup-wizard` / `agent-pull-dispatch` 及 MODIFIED 诸能力），不在本文另起需求真源。
