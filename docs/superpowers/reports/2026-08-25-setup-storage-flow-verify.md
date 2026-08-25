# Verify Report: setup-storage-flow

- 验证模式：full（13 任务、2 个 delta spec capability、40 变更文件）
- 语言：zh-CN
- 日期：2026-08-25

## 验证命令证据

| 命令 | 结果 |
|---|---|
| `go build ./... && go test ./...` | PASS（70 包 ok，无 FAIL） |
| `cd web/admin && pnpm tsc -b && pnpm vitest run` | PASS（57 文件 / 362 测试通过） |

## Completeness

| 检查项 | 状态 |
|---|---|
| tasks.md 全部勾选 | 13/13 `[x]`，build guard 已确认 |
| 计划文件步骤全部勾选 | 28/28 `[x]`，build guard 已确认 |
| delta spec 需求均有实现 | 见 Correctness |

## Correctness（需求 → 实现证据）

| 需求/场景 | 实现证据 |
|---|---|
| setup-wizard：移除「出图机器在哪」步骤，直接对象存储配置 | `setup-steps.ts`（3 步序列）、`setup-wizard.tsx`（无 placement/RadioGroup，`draft()` 按 `blob_driver` 推断 placement） |
| setup-wizard：对象存储连通性测试（localfs 目录可写） | `blob/localfs.Check`（MkdirAll + 探测文件）+ `TestCheck_Writable` |
| setup-wizard：S3/TOS 连通性测试 | `blob/s3.Check`、`blob/tos.Check`（HeadBucket）+ `TestCheckAndEnsureBucket`（gofakes3） |
| setup-wizard：bucket 不存在提示自动创建 | `blob-test` 返回 `bucket_not_found`（`TestBlobTest_BucketNotFoundAndCreate`）；前端「bucket `xxx` 不存在，帮你创建？」确认后 `auto_create_bucket:true` 重试 |
| setup-wizard：常见错误中文提示与详情 | `blob-error.ts`（4 类映射 + 通用回退，4 单测）复用 `AlertCopy` |
| shared-blob-store：连通性检查能力 | `blob.Store.Check` 加入接口；`factory.Check`（显式凭据，不读环境变量） |
| shared-blob-store：bucket 不存在可确认自动创建 | `blob.ErrBucketNotFound` + `s3/tos.EnsureBucket`（CreateBucket/CreateBucketV2 + 复检）+ `factory.EnsureBucket` + handler `auto_create_bucket` 分支 |
| shared-blob-store：建桶权限不足可诊断 | `EnsureBucket` 错误透传为 `400 blob check failed: ...`，前端 destructive Alert 展示 |

## Coherence

- design.md D1（移除 placement、由 blob 驱动推断）：实现一致。
- design.md D2（驱动 `Check`/`EnsureBucket`、factory 显式凭据、`blob-test` 端点）：实现一致。
- design.md D3（前端复用「连通性测试 + 继续」与 Alert、`blob-error.ts`、bucket 创建提示）：实现一致。
- design.md D4（步骤序列与测试更新）：实现一致。
- Design Doc 与 delta spec 无矛盾；`blob.Store` 接口新增 `Check` 无其它实现受影响。
- 安全检查：Secret Key 仅走同源 API（与数据库密码同理）；无硬编码凭据（测试用假密钥/gofakes3）。

## 代码审查

- build 阶段（review_mode: standard）内联轻量审查 `ac0568e..HEAD`：无 Critical/Important；两条 Minor（Validate 用占位 ComfyMock、请求携带 Secret Key）已记录接受原因于 tasks.md。
- 说明：reviewer subagent 派发通道本会话多次失败，采用内联审查并记录。

## 结论

全部检查项通过，无 CRITICAL / IMPORTANT 问题。Ready for archive。

补充（归档前重开 0686c13，文案与提示）：存储步骤 Title「文件存储配置」/ Desc「决定了生成的图和视频存放的位置。」；`localfs` 选中时展示 Alert「这个配置只适合 ComfyUI 和后台在同一台机器上使用。」。前端 57 文件 / 363 测试、`pnpm tsc -b`、`go build ./... && go test ./...`（70 包 ok）全部通过；合同测试 `setup-pages.contract.test.ts` 锁定。

补充（归档前重开 4d6c6d5，warn 变体与文案）：`alert.tsx` 新增 `warn` 变体（amber 系）；localfs 提示改为 `Alert variant="warn"`，文案补全「…无法使用远程节点。」。前端 57 文件 / 363 测试、`pnpm tsc -b`、`go build ./... && go test ./...`（70 包 ok）全部通过。

补充（归档前重开 ce4a8ac，换行与 S3 标签）：localfs warn 文案两行展示（`<br />` 分隔）；驱动标签「S3 兼容」改「S3」（向导 + 设置页 i18n zh/en 同步）。前端 57 文件 / 364 测试、`pnpm tsc -b`、`go build ./... && go test ./...`（70 包 ok）全部通过。

补充（归档前重开 275aa7d，warn 图标）：localfs warn Alert 增加 `CircleAlert` 图标（`aria-hidden`）。前端 57 文件 / 364 测试、`pnpm tsc -b`、`go build ./... && go test ./...`（70 包 ok）全部通过。

补充（归档前重开 e8c41d0，结构修正）：localfs warn Alert 改为 Title「注意！」+ Description 两行文案（`<br />` 分隔），避免 `AlertTitle` 的 `line-clamp-1` 截断。前端 57 文件 / 364 测试、`pnpm tsc -b`、`go build ./... && go test ./...`（70 包 ok）全部通过。

补充（归档前重开 5c8d47c，自然换行）：移除 `<br />` 强制断行，Description 用连续文案随容器自然换行（截断根因 `line-clamp-1` 已在结构修正时避开）。前端 57 文件 / 364 测试、`pnpm tsc -b`、`go build ./... && go test ./...`（70 包 ok）全部通过。

补充（归档前重开 e283067，404 误报修复）：`blob-error.ts` 移除裸 `/404/` 映射（只认 `NoSuchBucket`/`NoSuchKey`），「Request failed (404)」走通用文案，含回归测试（RED→GREEN 已验证）。前端 57 文件 / 365 测试、`pnpm tsc -b`、`go build ./... && go test ./...`（70 包 ok）全部通过。

补充（归档前重开 587c21d，TOS 默认值）：选择火山 TOS 预填 endpoint `https://tos-cn-beijing.volces.com`、region `cn-beijing`、bucket `pixoma`（空值才填，不覆盖手输）。前端 57 文件 / 366 测试、`pnpm tsc -b`、`go build ./... && go test ./...`（70 包 ok）全部通过。

补充（归档前重开 cef3a03，info 变体）：`alert.tsx` 新增 `info` 变体（sky 蓝）；bucket 创建提示改用 `Alert variant="info"`。前端 57 文件 / 366 测试、`pnpm tsc -b`、`go build ./... && go test ./...`（70 包 ok）全部通过。
