# comfy-multi-instance 验证报告

- Change: `comfy-multi-instance`
- 分支: `feature/20260808/comfy-multi-instance`
- 日期: 2026-08-08
- verify_mode: full
- base-ref: `01c3b2c1c302769da85b1e73c6689839d70291f0`
- HEAD: （验证时见 git）

## 结论

**PASS**（无 CRITICAL / IMPORTANT 未处理项；若干 WARNING 已记录接受）

## 证据

### 测试

```text
go test ./internal/... ./apps/bot/... ./test/integration/ -count=1
```

结果：全部 `ok` / `no test files`；exit 0（本报告写入前当次运行）。

### 构建

build 阶段已 `record-check` 同命令；verify 再次执行并通过。

### 规模

- tasks.md：全部 `[x]`
- delta specs：6 个 capability
- 相对 base-ref：约 74 files / +6k 行

## Completeness

| 项 | 结果 |
|---|---|
| tasks.md 全勾选 | OK |
| User/Session/Task/Instance 持久化落地 | OK |
| HTTP CRUD + system/queue/tasks | OK |
| 健康探测 + round-robin + 按 InstanceID 客户端 | OK |
| 去掉 Ledger | OK |
| Makefile / README | OK |

## Correctness（对照 Design Doc / delta specs）

| 场景族 | 证据 | 结果 |
|---|---|---|
| User upsert 幂等 | identity persistence 测试 | OK |
| Session 持久化 + user_id/chat_id | conversation persistence 测试 | OK |
| Task.session_id + ListByInstance | gorm_task 测试 | OK |
| ConfirmRun / notify join Session | botapp + orchestrator 测试 | OK |
| ClaimQueued CAS / Publish 回滚 | orchestrator 测试 | OK |
| Create 重复 id → 409 | httpapi handler 测试 | OK |
| system/queue Mock + 不可达 | comfyui stats_test + handler | OK |
| Round-robin / 无实例保持 pending | orchestrator 测试 | OK |
| 集成 smoke | test/integration | OK |

## Coherence

- Design Doc 与实现一致：分接口 system/queue、Task→Session→User、去 Ledger。
- Build 终审 Critical 已修（`1990a6d`）；Important I1/I2/I4/I6/I7 已在 `tasks.md` Review notes 接受，不阻塞本 change。

## WARNING（已接受，不阻塞）

见 `docs/openspec/changes/comfy-multi-instance/tasks.md` Review notes：upsert 失败 UX、notify 重试、探测定时超时可配置、部分测试缺口、活跃 Session 唯一性。

## 安全抽查

- 未发现新增硬编码密钥；HTTP API 无鉴权（设计非目标，README 已警示）。

## 手工未复验（非阻塞）

- 真实双 Comfy 轮转与局域网 system/queue（单元/Mock 已覆盖主路径）。
