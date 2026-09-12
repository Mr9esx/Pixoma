# mcp-caller-skill 验证报告

- Change: mcp-caller-skill
- Date: 2026-09-12
- Language: zh-CN
- verify_mode: full
- base-ref: 99891849405c1e26d618ad03cbdb6d73b9a45c33
- HEAD: 当前分支 `feature/20260910/mcp-call-workflow`

## 证据

- `go test ./internal/mcp/ -count=1` → `ok`（本轮 verify 新跑）
- `git diff --stat 9989184...HEAD`：17 文件，与 tasks 一致（auth/tools/guide、skill、README、OpenSpec/计划）

## 完整验证清单

| # | 项 | 结果 |
|---|---|---|
| 1 | tasks.md 全部 `[x]` | PASS（7/7） |
| 2 | 符合 change `design.md`（方案 A、错误在 MCP 边界、skill 路径） | PASS |
| 3 | 符合 Design Doc | PASS（固定句、Flush、401 干净正文） |
| 4 | delta spec 场景 | PASS，见下表 |
| 5 | proposal 目标 | PASS：skill + 提示词 + 指导失败文案 |
| 6 | spec 与 design doc | PASS，无矛盾 |
| 7 | Design Doc 可定位 | PASS：`docs/superpowers/specs/2026-09-12-mcp-caller-skill-design.md` |

## Spec 场景

| Requirement | 证据 |
|---|---|
| 一句提示词即可安装 | README 整句 + `TestPixomaMCPSkill_InstallPromptInREADME` |
| skill 教调用顺序，鉴权从略 | `docs/skills/pixoma-mcp/SKILL.md` |
| 找不到工具提醒配连接器 | skill 错误表第一行 |
| MCP 失败文案指导 AI | `Guide*` + auth/tools 测试 |
| skill 错误对照 | skill 表与 Guide 同义 |
| 不进开发技能树 | `TestPixomaMCPSkill_ProductAttachment` |

## 代码审查

Build 阶段 standard 审查：无 Critical。Important（401 后缀 `: invalid token`）已在 verify 前修复，`TestMCP_DisabledChannelBodyMentionsUnavailable` 断言无 `invalid token`。

Minor（README 章节标题、io.Writer n）接受：不影响正确性/安全。原因：提示词整句已在 README；改写层只服务 `http.Error`。

## 结论

**PASS**。无 CRITICAL / IMPORTANT 未修复项。
