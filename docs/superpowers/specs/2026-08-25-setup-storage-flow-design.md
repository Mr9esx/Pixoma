---
comet_change: setup-storage-flow
role: technical-design
canonical_spec: openspec
---

# setup-storage-flow 深度设计

## 背景与范围

目标（详见 `docs/openspec/changes/setup-storage-flow/proposal.md`）：Setup 去掉「出图机器在哪」步骤，数据库配置后直接进入对象存储配置；对象存储像数据库一样支持连通性测试，bucket 不存在时在用户确认后自动创建（S3/TOS）。

现状约束：

- 向导步骤为 密码 → 数据库 → placement → 对象存储；对象存储步骤无任何连通性校验。
- `settings.Settings.Validate` 按 placement 校验 blob 合法组合（远程禁止 localfs）；`blob` 各驱动只有 Put/Get。
- 数据库步骤已有「连通性测试 + 继续」与 success/destructive Alert 模式，可直接复用。
- S3 SDK 提供 `HeadBucket`/`CreateBucket`；TOS SDK 提供 `ClientV2.HeadBucket`/`CreateBucketV2`。

## 目标 / 非目标

**目标**

- 向导步骤变 密码 → 数据库 → 对象存储；`placement` 由 blob 驱动推断（localfs→本机，s3/tos→远程），API 层远程禁止 localfs 约束保留。
- 对象存储步骤支持连通性测试：localfs 目录可写；S3/TOS `HeadBucket` 校验 endpoint/region/bucket/密钥。
- bucket 不存在时返回 `bucket_not_found`，前端提示是否代为创建，用户确认后创建并复检（S3 `CreateBucket`、TOS `CreateBucketV2`）。

**非目标**

- 不新增 OSS/Azure 等对象存储驱动（OSS 另行开 change）。
- 不改数据库步骤行为、不改设置页对象存储配置、不在向导内重做 Edge 部署指引。
- localfs 无 bucket 概念，不涉及自动建桶。

## 技术方案

### 1. 移除 placement 步骤（前端）

- `web/admin/src/features/setup/setup-steps.ts`：`SETUP_STEPS` 变 `['password','database','storage']`，删除 `placement` 步骤与 copy；`initialSetupStep`/`previousSetupStep`/`setupStepIndex` 更新。
- `setup-wizard.tsx`：删除 placement RadioGroup 与状态；`draft()` 中 `placement` 由 `blob_driver` 推断：

```ts
const placement = blobDriver === 'localfs' ? 'local' : 'remote'
```

- 对象存储步骤驱动下拉始终显示 `localfs`/`s3`/`tos`（不再按 placement 隐藏 localfs）。
- 后端 `draft` 的 wizard step 映射保持兼容（local→storage，remote→edge→storage）。

### 2. blob 驱动连通性与建桶（后端）

`internal/platform/blob`：

- 新增哨兵错误：

```go
var ErrBucketNotFound = errors.New("blob: bucket not found")
```

- `blob/localfs` 新增 `Check(ctx) error`：`os.MkdirAll(root)` + 探测文件写入/删除。
- `blob/s3` 新增 `Check(ctx) error` 与 `EnsureBucket(ctx) error`：
  - `Check`：`client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})`；404/NotFound → `ErrBucketNotFound`。
  - `EnsureBucket`：`Check` 为 nil 直接返回；`ErrBucketNotFound` 时 `CreateBucket`（region 非 `us-east-1` 时带 `CreateBucketConfiguration{LocationConstraint}`）→ 再 `Check` 确认。
- `blob/tos` 新增 `Check(ctx) error` 与 `EnsureBucket(ctx) error`：
  - `Check`：`client.HeadBucket(ctx, &volctos.HeadBucketInput{Bucket: bucket})`；404 → `ErrBucketNotFound`。
  - `EnsureBucket`：`CreateBucketV2(ctx, &volctos.CreateBucketV2Input{Bucket: bucket})` → 再 `Check` 确认。

`internal/platform/blob/factory` 新增：

```go
type CheckOptions struct {
    Driver      string
    LocalRoot   string
    Endpoint    string
    Region      string
    Bucket      string
    AccessKey   string
    SecretKey   string
}

func Check(ctx context.Context, opts CheckOptions) error          // 显式凭据，不读环境变量
func EnsureBucket(ctx context.Context, opts CheckOptions) error   // s3/tos 建桶后复检
```

### 3. `POST /api/v1/setup/blob-test`（后端）

请求体：

```json
{
  "blob_driver": "s3|tos|localfs",
  "blob_root": "data/blob",
  "blob_endpoint": "...",
  "blob_region": "...",
  "blob_bucket": "...",
  "blob_access_key": "...",
  "blob_secret_key": "...",
  "auto_create_bucket": false
}
```

处理：

1. 构造 `settings.Settings`（placement 按 driver 推断），`Validate()` 校验（远程禁止 localfs 等）；失败 400。
2. `factory.Check(ctx, CheckOptions{...})`：
   - `nil` → 200 `{"ok":true}`。
   - `errors.Is(err, blob.ErrBucketNotFound)` 且 `auto_create_bucket=false` → 200 `{"ok":false,"code":"bucket_not_found","bucket":"<name>"}`（前端提示）。
   - `errors.Is(err, blob.ErrBucketNotFound)` 且 `auto_create_bucket=true` → `factory.EnsureBucket` → 200 `{"ok":true}`；失败 400 可诊断错误。
   - 其它错误 → 400 `{"error":"blob check failed: "+err.Error()}`。

### 4. 前端对象存储步骤

- 字段：localfs → 目录；s3/tos → endpoint / region / bucket / access key / secret key（密钥 `type=password`）。
- 按钮：复用数据库步骤的「连通性测试 + 继续」；测试调 `blob-test`，通过显示 success Alert（`CircleCheck` + 连接正常），失败显示 destructive Alert。
- bucket 不存在：测试返回 `bucket_not_found` → 展示提示「bucket `xxx` 不存在，帮你创建？」（创建/取消）；确认后带 `auto_create_bucket:true` 重试，成功显示连接正常。
- 新增 `blob-error.ts`：InvalidAccessKeyId、SignatureDoesNotMatch、NoSuchBucket、AccessDenied、connection refused、timeout、空配置等 → 中文友好标题 + 实际详情（复用 Alert 形态）；`setupErrorCopy` 通用回退。

## 数据流：对象存储连通性测试

```text
用户填 endpoint/region/bucket/密钥 → 点「连通性测试」
  → POST /api/v1/setup/blob-test（auto_create_bucket=false）
  → Validate → factory.Check（HeadBucket）
  ├─ ok → success Alert「连接正常」
  ├─ bucket_not_found → 提示「是否帮你创建」
  │    └─ 用户确认 → POST blob-test（auto_create_bucket=true）
  │         → factory.EnsureBucket（CreateBucket/CreateBucketV2）→ 复检 → success Alert
  └─ 其它错误 → destructive Alert（中文标题 + 实际详情）
```

## 边界条件

- bucket 不存在但用户未确认：绝不创建，返回 `bucket_not_found`。
- 建桶权限不足：`EnsureBucket` 失败 → 400 可诊断错误，不标记连接正常。
- region 为 `us-east-1` 时 S3 `CreateBucket` 不带 LocationConstraint（MinIO/S3 兼容兼容处理）。
- localfs 目录不可写：`Check` 失败 → 可诊断错误。
- 未知 blob driver：`Validate`/`factory.Check` 返回未知驱动错误。
- 密钥为空：S3/TOS `New` 允许但 HeadBucket 会鉴权失败 → 可诊断错误。

## 测试策略

- Go 单测：
  - `blob/s3`、`blob/tos` `Check` 的 NotFound 判定与 `EnsureBucket` 流程（用 gofakes3 / SDK fake）。
  - `blob/localfs` `Check` 目录可写/不可写。
  - `factory.Check`/`EnsureBucket` 分发与未知驱动。
  - `blob-test` 端点：非法配置 400、bucket_not_found 响应、auto_create_bucket 成功路径（注入 fake checker）。
- 前端：`setup-steps.test.ts` 3 步序列；合同测试断言无「出图机器在哪」、测试按钮、success/destructive Alert、bucket 创建提示与重试；`blob-error.ts` 单测。
- 回归：`go build ./... && go test ./...`；`cd web/admin && pnpm tsc -b && pnpm vitest run`。

## 风险与缓解

- [建桶权限不足] → 创建失败报可诊断错误；README 说明需 `CreateBucket`/`PutBucket` 权限。
- [MinIO/S3 兼容 region 差异] → `us-east-1` 省略 LocationConstraint，path-style 由现有 `S3_PATH_STYLE` 约定。
- [bucket 名非法（大小写/长度）] → 建桶失败报可诊断错误，提示用户修改。

## 交付清单

- `web/admin/src/features/setup/setup-steps.ts`、`setup-wizard.tsx`：步骤序列、placement 推断、对象存储表单与测试/继续、bucket 创建提示。
- `web/admin/src/features/setup/blob-error.ts`：错误映射。
- `internal/platform/blob`：`ErrBucketNotFound`、`localfs/s3/tos` 的 `Check`/`EnsureBucket`。
- `internal/platform/blob/factory`：`Check`/`EnsureBucket`（显式凭据）。
- `internal/httpapi/setup`：`blob-test` 端点。
- 文档：README、规格（setup-wizard / shared-blob-store）、tasks.md。
