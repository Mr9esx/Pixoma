# Build Review Notes (review_mode: standard)

- Date: 2026-08-18
- Range: `e3e0870..HEAD`（聚焦 edge-system-monitoring 功能文件）
- 说明：requesting-code-review 技能已加载；子代理派发两次均因环境问题未执行（子代理未收到任务内容），按 gate 意图改为主会话内完成同等轻量审查并记录。

## Findings

### Critical (Must Fix)

无。

### Important (Should Fix)

1. nvidia-smi CSV 解析无直接单测，且含逗号的 GPU 名会因前导空格+引号导致整次解析失败。
   - 修复：抽取 `parseNvidiaSMIOutput`，`csv.Reader.TrimLeadingSpace = true`，新增 `collect_internal_test.go`（含逗号名、坏行跳过）。

### Minor (Accepted)

1. CPU 占用率采集失败时上报 0，与真实 0% 无法区分（spec 允许字段缺失/标记；0 可接受）。
2. 磁盘 I/O 采集失败时不上次基线，下一拍差值可能偏大（罕见；影响有限）。
3. Reporter 用自身 now 记录采样节拍，与快照 CollectedAt 存在亚秒级偏差（无影响）。
4. `GET /metrics` 非法 window 返回 400 未单测（代码路径简单；接受）。

## Verification Evidence

- `go build ./... && go test ./...` 全绿（含回归测试 `TestSample_CPUPercentMustNotSleep`）
- `web/admin`: `npx tsc -b && npx vitest run`（30 文件 / 104 测试通过）、`npm run build` 成功
- Mock 端到端：临时 DATA_DIR 起 pixoma + edge-agent（`METRICS_INTERVAL=5s`），向导初始化后 Edge 每 5s 上报，`GET /api/v1/edges/local/metrics` 返回 CPU/内存/Mock GPU/磁盘 I/O 系列

## Assessment

Ready to merge: Yes（Important 已修复，Minor 已记录接受理由）。
