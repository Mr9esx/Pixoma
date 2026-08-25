## 1. 向导移除「出图机器在哪」步骤

- [x] 1.1 `setup-steps.ts` 删除 `placement` 步骤与 copy，步骤序列变为 密码 → 数据库 → 对象存储；`initialSetupStep`/`previousSetupStep`/`setupStepIndex` 更新
- [x] 1.2 `setup-wizard.tsx` 删除 placement RadioGroup；`draft()` 的 `placement` 改为按 `blob_driver` 推断（localfs→local，s3/tos→remote）
- [x] 1.3 更新 `setup-steps.test.ts` 与合同测试：3 步流程、无「出图机器在哪」文案

## 2. 对象存储连通性测试（后端）

- [x] 2.1 `blob/localfs`、`blob/s3`、`blob/tos` 新增 `Check(ctx)`：目录可写探测 / `HeadBucket`（S3 与 TOS），含单元测试
- [x] 2.2 `blob/factory` 新增 `Check(ctx, CheckOptions)`（显式凭据，不依赖环境变量回退）
- [x] 2.3 `s3`/`tos` 新增 `EnsureBucket(ctx)`（HeadBucket 探测 → CreateBucket/CreateBucketV2 → 复检），`blob` 包新增 `ErrBucketNotFound`，含单元测试
- [x] 2.4 `internal/httpapi/setup` 新增 `POST /api/v1/setup/blob-test`：校验 blob 配置 → `factory.Check` → ok / `bucket_not_found` / 可诊断错误；带 `auto_create_bucket` 时先 `EnsureBucket`，含单测

## 3. 前端对象存储步骤连通性测试

- [x] 3.1 `setup-wizard.tsx` 对象存储步骤增加「连通性测试 + 继续」按钮与 success/destructive Alert（复用数据库步骤模式）；S3/TOS 显示 endpoint/region/bucket/密钥字段
- [x] 3.2 新增 `blob-error.ts`：常见对象存储错误映射中文友好标题 + 实际详情，含单元测试
- [x] 3.3 bucket 不存在时前端提示「是否帮你创建」并支持确认后自动创建（带 `auto_create_bucket` 重试），含合同测试
- [x] 3.4 合同测试：对象存储测试按钮、Alert 展示、无「出图机器在哪」

## 4. 文档与验证

- [x] 4.1 README 更新：向导步骤（密码/数据库/对象存储）、对象存储连通性测试、bucket 不存在可确认自动创建（需建桶权限）
- [x] 4.2 `go build ./...` + `go test ./...`；`pnpm tsc -b` + `pnpm vitest run`（web/admin）
