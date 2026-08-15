# pixoma-guided-deploy 验证报告

- change: `pixoma-guided-deploy`
- 日期: 2026-08-15
- verify_mode: full
- review_mode: standard
- 语言: zh-CN
- base-ref: `3a0518564cee9bf9d9305caf6c44c190717af8d2`
- HEAD: `d99e131`

## 结论

**PASS**

零配置控制面、向导落库、可领取派发、本机自动 Edge、三库与重启门闩均已落地。审查中发现的 Telegram 未装配、S3 向导字段未灌入运行时、非持有者迟报覆盖任务，已在 `d99e131` 修复。`make build` 与 `make test`（`go test ./...`）退出码 0。

## 检查项

| 项 | 结果 | 证据 |
|---|---|---|
| tasks.md 全部 `[x]` | OK | 21/21 OpenSpec 任务；plan Task 1–6 已勾选 |
| 实现匹配 OpenSpec design.md | OK | 控制面+Edge、DB claim、向导落库、无默认 Redis |
| 实现匹配 Design Doc | OK | `docs/superpowers/specs/2026-08-12-pixoma-guided-deploy-design.md` |
| 能力场景可追踪 | OK | 见下方规格覆盖 |
| proposal 目标 | OK | 引导启动、配置落库、Edge 拉取、BREAKING 去掉 queue 用户路径 |
| delta spec 与 design 无矛盾 | OK | `claim_generation` / `edge_heartbeats` 在设计中为可选，未做成硬需求 |
| Design Doc 可定位 | OK | 上列路径存在且 `comet_change: pixoma-guided-deploy` |
| 构建 | OK | `make build` exit 0（recorded 2026-08-15T02:28:17Z） |
| 测试 | OK | `make test` / `go test ./...` exit 0 |
| 安全抽查 | OK | 默认 `127.0.0.1:8080`；密码 bcrypt；settings AES-GCM；Agent Token 文件 0600 且启动时哈希校验 |
| standard 审查 | OK | 全量 diff 审查后修复 CRITICAL/IMPORTANT 主路径问题 |

## 规格覆盖

| 能力 | 场景 | 实现/测试 |
|---|---|---|
| platform-bootstrap | 空目录启动打 URL/账密 | `apps/pixoma/internal/app` banner 测试；`pixoma` 入口 |
| platform-bootstrap | 未初始化拒业务 API | `internal/httpapi/setup` gate 测试 |
| platform-bootstrap | 改密后不再打明文 | banner 空密码分支 |
| setup-wizard | SQLite 本机可读回 | setup handler 测试 |
| setup-wizard | 远程禁 localfs | settings.Validate |
| setup-wizard | 保存后重启装配 | restart_required 门闩；`ApplyBlobEnv` |
| agent-pull-dispatch | claim / 空等待 / 租约再领 / status | agent + gorm_task + domain 测试 |
| task-orchestrator | 可领取、不 Publish Redis | orchestrator 测试 `Dispatch==nil` |
| task-orchestrator | 无候选保持 pending | Online/enabled 测试 |
| edge-agent | 拉取循环 mock | `apps/edge-agent/internal/pull` |
| cross-process-queue | 向导无 queue.driver | setup wizard UI/API |
| shared-blob-store | 本机 localfs；远程 s3/tos 校验 | settings + factory S3Options |
| admin-api-host | 一体托管 + /healthz | adminhost + pixoma mount |
| admin-web-shell | 未初始化进向导；restart 停在 /setup | setup-guard 测试 |

## 构建与测试证据

```
make build   # pixoma, pixoma-edge-agent, admin-api  exit 0
make test    # go test ./...                      exit 0
```

相关包抽测（审查修复后）：`factory`、`orchestrator`、`agent`、`apps/pixoma/internal/app`、`setup`、`bootstrap`、`settings`、`test/integration`。

## 审查与修复

首次 standard 审查（范围 `3a05185...c76b902`）结论为 **With fixes**。已修复并提交 `d99e131`：

- 控制面装配 Telegram 入站/出站、同进程 `task.created`、pending 扫描
- 向导 S3 endpoint/region/bucket 写入 env 与 factory
- status 拒绝非当前持有者
- 可领取调度走 `ListEnabled`（不要求控制面探通 Comfy）
- Agent Token 文件与 bootstrap 哈希交叉校验
- claim 乐观锁失败后同事务改领下一单

## 接受的偏差（WARNING，不挡 PASS）

1. **登录无节流 / SQLite 路径未限制在 DATA_DIR**  
   默认只绑回环；公网需显式 `HTTP_ADDR`。影响：若运维把端口暴露到公网，默认账密可被刷。不改本期行为。

2. **未实现 `claim_generation`**  
   Design 标明可选。已用 instance_id 持有者校验挡住跨实例迟报。同实例租约过期再领后的旧成功上报仍可能收敛，单机一台 Edge 可接受。

3. **无独立 `edge_heartbeats` 表**  
   在线语义目前落在任务 lease/心跳；远程多节点「空闲在线」不单独持久化。不影响本机 mock 主路径。

4. **独立 `admin-api` 仍可编译**  
   过渡期允许；新部署默认是 `pixoma`。

5. **手工本机 E2E（空目录 → 向导 → TG 发图 → 回图）**  
   由单元/集成测试覆盖主契约；真 Telegram 轮询需 token，未在 CI 实打。Comfy mock 路径有 pull/orchestrator/smoke 测试。

## 架构文档

`README.md`、`docs/architecture/overview.md`、`runtime.md`、`data-model.md`、`diagrams/system.html` 已随实现改为控制面 + Edge 叙事（不再把 allinone/split 当用户默认矩阵）。

## 工作区说明

未纳入本 change：`docs/openspec/changes/queue-nats-adapter/`（另一需求）、`.comet/current-change.json`（选择文件）。
