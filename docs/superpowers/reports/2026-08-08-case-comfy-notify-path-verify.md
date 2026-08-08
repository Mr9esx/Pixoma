# 验证报告：case-comfy-notify-path

- **日期**: 2026-08-08
- **verify_mode**: full
- **分支**: `feature/20260808/case-comfy-notify-path`
- **base-ref**: `eba6a71755051501e9c96760be288af3aedbd011`
- **HEAD**: `5d7dde5c3cda8f7cb61901618fc5d652e16a27e8`
- **结果**: **PASS**

## 摘要计分卡

| 维度 | 结果 |
|------|------|
| Completeness（tasks 14/14） | OK |
| Correctness（delta specs / 场景） | OK |
| Coherence（design.md / Design Doc） | OK |
| 构建与测试 | OK |
| 安全（本 change 范围） | OK（CRITICAL/IMPORTANT 无） |
| Verify 代码审查 | APPROVED |

## 完整验证检查项

| # | 检查项 | 结果 | 证据 |
|---|--------|------|------|
| 1 | tasks.md 全部 `[x]` | OK | 14/14；OpenSpec `progress.remaining=0` |
| 2 | 实现符合 `design.md` 高层决策 | OK | CaseSnapshot 替 StaticWorkflows；Upload 后注入；TG 按类型分流；basename 发图；混合种子 + 仓库内 fixture |
| 3 | 实现符合 Design Doc | OK | 节点 10/20/60；`img-edit` 绑定；`comfy_mock` 开关；架构 A |
| 4 | 能力规格场景 | OK | 见下方场景映射 |
| 5 | proposal.md 目标 | OK | 注入闭环、Mock/真实、混合 Case、TG 采图/标量/安全文件名、冒烟 |
| 6 | delta spec 与 design doc 无矛盾 | OK | Open Question「fixture 来源」已由 Design Doc 落定为仓库内文件；非矛盾 |
| 7 | Design Doc 可定位 | OK | `docs/superpowers/specs/2026-08-08-case-comfy-notify-path-design.md` |

## 场景覆盖（抽样对照）

| Spec | 场景 | 实现/测试 |
|------|------|-----------|
| comfyui-executor | 文本/图片注入；缺映射；空 workflow；Mock 主路径 | `snapshot_test.go`；`smoke_test.go`；`upload_test.go` |
| channel-tg | Photo→Blob；图字段拒文本；number；basename | `adapter_test.go`；`bot_test.go` |
| workflow-protocol | 混合 Case；仅文本当图失败 | `validate_test.go`；`configs/cases/img-edit.json` |
| dialog-session | Blob Draft 推进 | TG 采图路径 + 冒烟 |

## 构建 / 测试证据

命令（本轮 verify 新跑）：

```text
go test ./internal/runtime/... ./internal/channel/tg/... ./internal/catalog/... ./internal/packaging/botapp/... ./test/integration/... -count=1
go build -o /dev/null ./apps/bot/cmd/comfyui-bot/
```

结果：相关包全部 `ok`；`BUILD_OK`。

改动规模（`base-ref...HEAD`）：44 files，+3627/−26（含文档与配置）。

## 安全

- Build 终审 + 复审：下载失败不泄漏 bot token URL；HTTP 下载 30s Timeout + 20MiB LimitReader — **FIXED / APPROVED**。
- Verify 审查（[Verify-scoped code review](2a77701e-9a55-44a1-bdfc-5061953b44a1)）：无新增 CRITICAL/IMPORTANT。

## 非阻塞项（WARNING / SUGGESTION，不否决）

1. **WARNING**：`HandleUserMedia` 在 Put/Submit 失败时仍可能把 `err.Error()` 发给用户（下载路径已脱敏）。建议后续统一为固定文案 + 服务端日志。
2. **WARNING**：非 `image/*` Document 静默忽略，与 Design「明确拒绝」略有差距；不影响 Photo 主验收。
3. **WARNING**：执行面读 Blob 无与下载一致的显式字节上限（主路径已受 TG 下载上限约束）。
4. **SUGGESTION**：HTTP Upload multipart 可补 Content-Type；Case 内嵌 workflow 与 `configs/workflows/` 长期防漂移。

以上不构成 CRITICAL/IMPORTANT，不触发 verify-fail；可作为后续 tweak。

## 结论

**PASS** — 可进入 archive 阶段（归档前仍须用户确认）。
