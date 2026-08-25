# setup-storage-flow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Setup 向导去掉「出图机器在哪」步骤，直接进入对象存储配置；对象存储像数据库一样支持连通性测试，bucket 不存在时经用户确认自动创建（S3/TOS）。

**Architecture:** 前端 `SETUP_STEPS` 变 密码 → 数据库 → 对象存储，`placement` 由 blob 驱动推断（localfs→local，s3/tos→remote）；`blob/localfs`、`blob/s3`、`blob/tos` 各新增 `Check(ctx)`，`s3`/`tos` 新增 `EnsureBucket(ctx)`，blob 包新增 `ErrBucketNotFound`；`blob/factory` 提供带显式凭据的 `Check`/`EnsureBucket`；Setup 新增 `POST /api/v1/setup/blob-test`（bucket 不存在返回 `bucket_not_found`，`auto_create_bucket:true` 时创建并复检）；前端对象存储步骤复用「连通性测试 + 继续」与 Alert 模式。

**Tech Stack:** Go 1.25 + AWS SDK v2（s3）/ 火山 TOS SDK / gofakes3（测试）；React 18 + TanStack Query + Vitest（合同测试）、pnpm。

## Global Constraints

- 不新增 OSS/Azure 驱动（OSS 另行开 change）。
- 不自动建 bucket（仅用户确认后创建）；建桶需账号权限，失败给可诊断错误。
- `placement` 由 blob 驱动推断；API 层 `settings.Validate` 的远程禁止 localfs 约束保留。
- 不新增第三方依赖（s3/tos SDK、gofakes3 均已在 go.mod）。
- 前端文案沿用数据库步骤既有风格（中文友好标题 + 实际详情）。
- 每个任务独立可测交付 + 单独 commit。

---

### Task 1: 移除 placement 步骤（前端）

**Files:**
- Modify: `web/admin/src/features/setup/setup-steps.ts`
- Modify: `web/admin/src/features/setup/setup-wizard.tsx`
- Test: `web/admin/src/features/setup/setup-steps.test.ts`
- Test: `web/admin/src/features/setup/setup-pages.contract.test.ts`

**Interfaces:**
- Consumes: 无。
- Produces: `SETUP_STEPS = ['password','database','storage']`；`draft()` 的 `placement` 由 `blob_driver` 推断；对象存储驱动下拉恒显示三驱动。

- [x] **Step 1: 先写失败测试**

`setup-steps.test.ts`：`setupStepsFor(false)` 长度 3 且首步为 `database`；`previousSetupStep(withoutPassword, 'storage') === 'database'`。

`setup-pages.contract.test.ts` 新增：

```ts
it('removes the deployment placement step', () => {
  const steps = read('src/features/setup/setup-steps.ts')
  const wizard = read('src/features/setup/setup-wizard.tsx')
  expect(steps).not.toMatch(/placement/)
  expect(wizard).not.toMatch(/出图机器在哪/)
  expect(wizard).not.toMatch(/RadioGroup/)
  expect(wizard).toMatch(/const placement = blobDriver === 'localfs' \? 'local' : 'remote'/)
})
```

- [x] **Step 2: 运行测试确认失败**

Run: `cd web/admin && pnpm vitest run src/features/setup/`
Expected: FAIL（现有 4 步与 placement 文案仍在）。

- [x] **Step 3: 实现**

`setup-steps.ts`：

```ts
export const SETUP_STEPS = ['password', 'database', 'storage'] as const
```

删除 `placement` 的 `SETUP_STEP_COPY` 条目；`initialSetupStep` 保持 `edge → storage` 兼容映射。

`setup-wizard.tsx`：

- 删除 `placement` state、placement RadioGroup 表单块、`onPlacementChange`。
- `draft()` 内：`placement: blobDriver === 'localfs' ? 'local' : 'remote'`。
- 对象存储驱动下拉去掉 `{placement === 'local' ? <SelectItem value='localfs'>…</SelectItem> : null}` 条件，恒显示三驱动。

- [x] **Step 4: 运行测试确认通过**

Run: `cd web/admin && pnpm vitest run src/features/setup/ && pnpm tsc -b`
Expected: PASS。

- [x] **Step 5: 提交**

```bash
git add web/admin/src/features/setup/setup-steps.ts web/admin/src/features/setup/setup-wizard.tsx web/admin/src/features/setup/setup-steps.test.ts web/admin/src/features/setup/setup-pages.contract.test.ts
git commit -m "feat(setup): remove deployment placement step"
```

---

### Task 2: blob 驱动 Check / EnsureBucket（后端）

**Files:**
- Modify: `internal/platform/blob/port.go`
- Modify: `internal/platform/blob/localfs/localfs.go`
- Modify: `internal/platform/blob/s3/s3.go`
- Modify: `internal/platform/blob/tos/tos.go`
- Test: `internal/platform/blob/localfs/localfs_test.go`、`internal/platform/blob/s3/s3_test.go`、`internal/platform/blob/tos/tos_test.go`

**Interfaces:**
- Consumes: 各驱动现有 `New`/`client`/`bucket`。
- Produces: `blob.ErrBucketNotFound`；`Store.Check(ctx) error`；`Store.EnsureBucket(ctx) error`（localfs 无 EnsureBucket）。

- [x] **Step 1: 先写失败测试**

`port.go` 增加哨兵错误：

```go
var ErrBucketNotFound = errors.New("blob: bucket not found")
```

`s3_test.go` 追加（gofakes3）：

```go
func TestCheckAndEnsureBucket(t *testing.T) {
	backend := s3mem.New()
	faker := gofakes3.New(backend)
	srv := httptest.NewServer(faker.Server())
	t.Cleanup(srv.Close)
	store, err := blobs3.New(blobs3.Options{
		Endpoint: srv.URL, Region: "us-east-1", Bucket: "pixoma-missing",
		AccessKeyID: "AKIA_TEST", SecretAccessKey: "testsecret", UsePathStyle: true,
	})
	if err != nil { t.Fatal(err) }
	ctx := context.Background()
	if err := store.Check(ctx); !errors.Is(err, blob.ErrBucketNotFound) {
		t.Fatalf("want ErrBucketNotFound, got %v", err)
	}
	if err := store.EnsureBucket(ctx); err != nil { t.Fatalf("ensure: %v", err) }
	if err := store.Check(ctx); err != nil { t.Fatalf("check after ensure: %v", err) }
}
```

`localfs_test.go` 追加：

```go
func TestCheck_Writable(t *testing.T) {
	store, err := localfs.New(t.TempDir())
	if err != nil { t.Fatal(err) }
	if err := store.Check(context.Background()); err != nil { t.Fatalf("check: %v", err) }
}
```

- [x] **Step 2: 运行测试确认失败**

Run: `go test ./internal/platform/blob/...`
Expected: FAIL（`Check`/`EnsureBucket` 未定义、`ErrBucketNotFound` 未定义）。

- [x] **Step 3: 实现**

`s3.go`：

```go
func (s *Store) Check(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "NoSuchBucket") {
			return fmt.Errorf("blob/s3: %w", blob.ErrBucketNotFound)
		}
		return fmt.Errorf("blob/s3: check: %w", err)
	}
	return nil
}

func (s *Store) EnsureBucket(ctx context.Context) error {
	if err := s.Check(ctx); err == nil {
		return nil
	} else if !errors.Is(err, blob.ErrBucketNotFound) {
		return err
	}
	in := &s3.CreateBucketInput{Bucket: aws.String(s.bucket)}
	if region := strings.TrimSpace(s.region); region != "" && region != "us-east-1" {
		in.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(region),
		}
	}
	if _, err := s.client.CreateBucket(ctx, in); err != nil {
		return fmt.Errorf("blob/s3: create bucket: %w", err)
	}
	return s.Check(ctx)
}
```

`Store` 增加 `region string` 字段（`New` 里赋值）。imports 增加 `smithy.APIError`（`github.com/aws/smithy-go`）与 `types`。

`tos.go`：

```go
func (s *Store) Check(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &volctos.HeadBucketInput{Bucket: s.bucket})
	if err != nil {
		var serr *volctos.TosServerError
		if errors.As(err, &serr) && serr.StatusCode == http.StatusNotFound {
			return fmt.Errorf("blob/tos: %w", blob.ErrBucketNotFound)
		}
		return fmt.Errorf("blob/tos: check: %w", err)
	}
	return nil
}

func (s *Store) EnsureBucket(ctx context.Context) error {
	if err := s.Check(ctx); err == nil {
		return nil
	} else if !errors.Is(err, blob.ErrBucketNotFound) {
		return err
	}
	if _, err := s.client.CreateBucketV2(ctx, &volctos.CreateBucketV2Input{Bucket: s.bucket}); err != nil {
		return fmt.Errorf("blob/tos: create bucket: %w", err)
	}
	return s.Check(ctx)
}
```

`localfs.go`：

```go
func (s *Store) Check(ctx context.Context) error {
	if err := ctx.Err(); err != nil { return err }
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return fmt.Errorf("blob/localfs: check mkdir: %w", err)
	}
	probe := filepath.Join(s.root, ".pixoma-probe")
	if err := os.WriteFile(probe, []byte("ok"), 0o600); err != nil {
		return fmt.Errorf("blob/localfs: check write: %w", err)
	}
	_ = os.Remove(probe)
	return nil
}
```

`tos_test.go` 追加构造错误断言（空 bucket/endpoint/region 已覆盖）与 404 判定辅助函数单测（若 `TosServerError` 可构造）。

- [x] **Step 4: 运行测试确认通过**

Run: `go test ./internal/platform/blob/...`
Expected: PASS。

- [x] **Step 5: 提交**

```bash
git add internal/platform/blob/port.go internal/platform/blob/localfs/localfs.go internal/platform/blob/s3/s3.go internal/platform/blob/tos/tos.go internal/platform/blob/localfs/localfs_test.go internal/platform/blob/s3/s3_test.go internal/platform/blob/tos/tos_test.go
git commit -m "feat(blob): connectivity check and bucket ensure for localfs/s3/tos"
```

---

### Task 3: blob/factory 显式凭据检查入口

**Files:**
- Create: `internal/platform/blob/factory/check.go`
- Test: `internal/platform/blob/factory/check_test.go`

**Interfaces:**
- Consumes: Task 2 的各驱动 `Check`/`EnsureBucket`。
- Produces: `CheckOptions`、`Check(ctx, opts)`、`EnsureBucket(ctx, opts)`（显式凭据，不读环境变量）。

- [x] **Step 1: 先写失败测试**

```go
func TestCheck_LocalFS(t *testing.T) {
	if err := factory.Check(context.Background(), factory.CheckOptions{Driver: botconfig.BlobDriverLocalFS, LocalRoot: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
}

func TestCheck_UnknownDriver(t *testing.T) {
	err := factory.Check(context.Background(), factory.CheckOptions{Driver: "oss"})
	if err == nil || !strings.Contains(err.Error(), "unknown driver") {
		t.Fatalf("got %v", err)
	}
}
```

- [x] **Step 2: 运行测试确认失败**

Run: `go test ./internal/platform/blob/factory/`
Expected: FAIL（`Check` 未定义）。

- [x] **Step 3: 实现**

```go
package factory

type CheckOptions struct {
	Driver      string
	LocalRoot   string
	Endpoint    string
	Region      string
	Bucket      string
	AccessKey   string
	SecretKey   string
}

func Check(ctx context.Context, opts CheckOptions) error {
	store, err := storeForCheck(opts)
	if err != nil { return err }
	return store.Check(ctx)
}

func EnsureBucket(ctx context.Context, opts CheckOptions) error {
	switch strings.TrimSpace(opts.Driver) {
	case botconfig.BlobDriverLocalFS:
		store, err := localfs.New(opts.LocalRoot)
		if err != nil { return err }
		return store.Check(ctx)
	case botconfig.BlobDriverS3:
		store, err := blobs3.New(blobs3.Options{
			Endpoint: opts.Endpoint, Region: opts.Region, Bucket: opts.Bucket,
			AccessKeyID: opts.AccessKey, SecretAccessKey: opts.SecretKey, UsePathStyle: true,
		})
		if err != nil { return err }
		return store.EnsureBucket(ctx)
	case botconfig.BlobDriverTOS:
		store, err := blobtos.New(blobtos.Options{
			Endpoint: opts.Endpoint, Region: opts.Region, Bucket: opts.Bucket,
			AccessKeyID: opts.AccessKey, SecretAccessKey: opts.SecretKey,
		})
		if err != nil { return err }
		return store.EnsureBucket(ctx)
	default:
		return fmt.Errorf("blob/factory: unknown driver %q", opts.Driver)
	}
}

func storeForCheck(opts CheckOptions) (blob.Store, error) { /* 同上，s3/tos 走 Check */ }
```

`Check` 对 `s3`/`tos` 走对应 Store 的 `Check`；`localfs` 走 `Check`。

- [x] **Step 4: 运行测试确认通过**

Run: `go test ./internal/platform/blob/factory/`
Expected: PASS。

- [x] **Step 5: 提交**

```bash
git add internal/platform/blob/factory/check.go internal/platform/blob/factory/check_test.go
git commit -m "feat(blob): explicit-credential check entry in factory"
```

---

### Task 4: `POST /api/v1/setup/blob-test` 端点

**Files:**
- Modify: `internal/httpapi/setup/handler.go`
- Test: `internal/httpapi/setup/handler_test.go`
- Modify: `web/admin/src/lib/api/setup.ts`（前端调用函数）

**Interfaces:**
- Consumes: Task 3 的 `factory.Check`/`EnsureBucket`；`settings.Validate`。
- Produces: `POST /api/v1/setup/blob-test`（请求体见 Design Doc）；响应 `{ok:true}` / `{ok:false,code:"bucket_not_found",bucket}` / `{error}`。

- [x] **Step 1: 先写失败测试（handler_test.go）**

```go
func TestBlobTest_LocalFSSuccess(t *testing.T) {
	env := completeWizard(t)
	body, _ := json.Marshal(map[string]any{
		"blob_driver": "localfs",
		"blob_root":   t.TempDir(),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/blob-test", bytes.NewReader(body))
	env.auth(req)
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}

func TestBlobTest_BucketNotFoundAndCreate(t *testing.T) {
	backend := s3mem.New()
	faker := gofakes3.New(backend)
	srv := httptest.NewServer(faker.Server())
	t.Cleanup(srv.Close)
	env := completeWizard(t)
	cfg := map[string]any{
		"blob_driver": "s3", "blob_endpoint": srv.URL, "blob_region": "us-east-1",
		"blob_bucket": "pixoma-new", "blob_access_key": "AKIA_TEST", "blob_secret_key": "testsecret",
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/blob-test", bytes.NewReader(mustJSON(cfg)))
	env.auth(req)
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), `"bucket_not_found"`) {
		t.Fatalf("want bucket_not_found, got %d %s", rec.Code, rec.Body.String())
	}
	cfg["auto_create_bucket"] = true
	req = httptest.NewRequest(http.MethodPost, "/api/v1/setup/blob-test", bytes.NewReader(mustJSON(cfg)))
	env.auth(req)
	rec = httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}
```

- [x] **Step 2: 运行测试确认失败**

Run: `go test ./internal/httpapi/setup/`
Expected: FAIL（端点不存在）。

- [x] **Step 3: 实现**

`Mount` 增加 `r.Post("/blob-test", h.blobTest)`。handler：

```go
func (h *Handler) blobTest(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireSession(w, r); !ok { return }
	var body struct {
		BlobDriver      string `json:"blob_driver"`
		BlobRoot        string `json:"blob_root"`
		BlobEndpoint    string `json:"blob_endpoint"`
		BlobRegion      string `json:"blob_region"`
		BlobBucket      string `json:"blob_bucket"`
		BlobAccessKey   string `json:"blob_access_key"`
		BlobSecretKey   string `json:"blob_secret_key"`
		AutoCreateBucket bool `json:"auto_create_bucket"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	driver := strings.ToLower(strings.TrimSpace(body.BlobDriver))
	placement := settings.PlacementLocal
	if driver == settings.DriverSQLite { /* no-op */ }
	if driver == "s3" || driver == "tos" {
		placement = settings.PlacementRemote
	}
	cfg := settings.Settings{
		Placement:     placement,
		DBDriver:      settings.DriverSQLite,
		DBDSN:         "data/app.db",
		BlobDriver:    driver,
		BlobRoot:      body.BlobRoot,
		BlobEndpoint:  body.BlobEndpoint,
		BlobRegion:    body.BlobRegion,
		BlobBucket:    body.BlobBucket,
		BlobAccessKey: body.BlobAccessKey,
		BlobSecretKey: body.BlobSecretKey,
	}
	if err := cfg.Validate(); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	opts := factory.CheckOptions{
		Driver: driver, LocalRoot: body.BlobRoot, Endpoint: body.BlobEndpoint,
		Region: body.BlobRegion, Bucket: body.BlobBucket,
		AccessKey: body.BlobAccessKey, SecretKey: body.BlobSecretKey,
	}
	err := factory.Check(r.Context(), opts)
	if errors.Is(err, blob.ErrBucketNotFound) && !body.AutoCreateBucket {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "code": "bucket_not_found", "bucket": body.BlobBucket})
		return
	}
	if errors.Is(err, blob.ErrBucketNotFound) {
		err = factory.EnsureBucket(r.Context(), opts)
	}
	if err != nil {
		writeErr(w, http.StatusBadRequest, "blob check failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
```

`web/admin/src/lib/api/setup.ts` 增加：

```ts
export function testBlob(input: {
  blob_driver: string
  blob_root?: string
  blob_endpoint?: string
  blob_region?: string
  blob_bucket?: string
  blob_access_key?: string
  blob_secret_key?: string
  auto_create_bucket?: boolean
}) {
  return apiFetch<{ ok: boolean; code?: string; bucket?: string }>(
    '/api/v1/setup/blob-test',
    { method: 'POST', body: JSON.stringify(input) },
  )
}
```

- [x] **Step 4: 运行测试确认通过**

Run: `go test ./internal/httpapi/setup/`
Expected: PASS。

- [x] **Step 5: 提交**

```bash
git add internal/httpapi/setup/handler.go internal/httpapi/setup/handler_test.go web/admin/src/lib/api/setup.ts
git commit -m "feat(setup): blob connectivity test endpoint with bucket create"
```

---

### Task 5: 前端对象存储步骤（测试/继续 + bucket 创建提示）

**Files:**
- Modify: `web/admin/src/features/setup/setup-wizard.tsx`
- Create: `web/admin/src/features/setup/blob-error.ts`、`blob-error.test.ts`
- Modify: `web/admin/vitest.config.ts`
- Test: `web/admin/src/features/setup/setup-pages.contract.test.ts`

**Interfaces:**
- Consumes: Task 4 的 `testBlob`；Task 1 的 storage 步骤。
- Produces: 对象存储步骤「连通性测试 + 继续」；bucket 不存在提示「是否帮你创建」并确认后自动创建；`blob-error.ts` 错误映射。

- [x] **Step 1: 先写失败测试**

`blob-error.test.ts`：

```ts
import { blobErrorCopy } from './blob-error'

it('maps InvalidAccessKeyId to a credentials title', () => {
  const c = blobErrorCopy(new Error('InvalidAccessKeyId'))
  expect(c.title).toBe('对象存储密钥错误，请检查 Access Key / Secret Key')
})
it('maps NoSuchBucket to a bucket title', () => {
  const c = blobErrorCopy(new Error('NoSuchBucket: The specified bucket does not exist'))
  expect(c.title).toBe('bucket 不存在，请检查 bucket 名称')
})
it('falls back to generic copy', () => {
  const c = blobErrorCopy(new Error('boom'))
  expect(c.title).toBe('对象存储配置失败，请重试')
})
```

合同测试：

```ts
it('tests blob connectivity and offers bucket creation', () => {
  const wizard = read('src/features/setup/setup-wizard.tsx')
  expect(wizard).toMatch(/testBlob\(/)
  expect(wizard).toMatch(/bucket_not_found/)
  expect(wizard).toMatch(/帮你创建/)
  expect(wizard).toMatch(/auto_create_bucket/)
})
```

- [x] **Step 2: 运行测试确认失败**

Run: `cd web/admin && pnpm vitest run src/features/setup/`
Expected: FAIL。

- [x] **Step 3: 实现**

`blob-error.ts`：

```ts
import type { AlertCopy } from './db-error'

const BLOB_ERROR_PATTERNS: Array<{ pattern: RegExp; title: string }> = [
  { pattern: /InvalidAccessKeyId|SignatureDoesNotMatch|AccessDenied/i, title: '对象存储密钥错误，请检查 Access Key / Secret Key' },
  { pattern: /NoSuchBucket|NoSuchKey|404/i, title: 'bucket 或对象不存在，请检查 bucket 名称' },
  { pattern: /connection refused|i\/o timeout|timeout|no such host/i, title: '连不上对象存储，请检查 endpoint 和网络' },
  { pattern: /empty (endpoint|region|bucket)|missing/i, title: '对象存储配置不完整，请填全必填项' },
]

export function blobErrorCopy(err: unknown): AlertCopy {
  const detail = err instanceof Error ? err.message : String(err)
  const matched = BLOB_ERROR_PATTERNS.find(({ pattern }) => pattern.test(detail))
  return { title: matched?.title ?? '对象存储配置失败，请重试', detail }
}
```

`setup-wizard.tsx` storage 步骤：

- 增加 `blobTested`/`blobPromptCreate` state 与 `testBlob` 调用：通过 → success Alert「连接正常」；`bucket_not_found` → 显示提示「bucket `xxx` 不存在，帮你创建？」+「创建」按钮 → 带 `auto_create_bucket:true` 重试；其它错误 → `blobErrorCopy` destructive Alert。
- `StepActions` 传 `onTest`/`testPassed`（复用组件）。

`vitest.config.ts` include 增加 `src/features/setup/blob-error.test.ts`。

- [x] **Step 4: 运行测试确认通过**

Run: `cd web/admin && pnpm vitest run && pnpm tsc -b`
Expected: PASS。

- [x] **Step 5: 提交**

```bash
git add web/admin/src/features/setup/setup-wizard.tsx web/admin/src/features/setup/blob-error.ts web/admin/src/features/setup/blob-error.test.ts web/admin/src/features/setup/setup-pages.contract.test.ts web/admin/vitest.config.ts
git commit -m "feat(setup): blob connectivity test and bucket creation prompt"
```

---

### Task 6: 文档与验证

**Files:**
- Modify: `README.md`、`docs/openspec/changes/setup-storage-flow/tasks.md`、本计划

- [x] **Step 1: README 更新**

在「业务数据库」段落附近追加「对象存储」说明：向导步骤（密码/数据库/对象存储）、连通性测试、bucket 不存在可确认自动创建（需建桶权限）、localfs 跨机需共享目录（NFS/SMB）。

- [x] **Step 2: 全量验证**

Run: `go build ./... && go test ./...`；`cd web/admin && pnpm tsc -b && pnpm vitest run`
Expected: 全 PASS。

- [x] **Step 3: 勾选 tasks.md 全部任务并提交**

```bash
git add README.md docs/openspec/changes/setup-storage-flow/tasks.md docs/superpowers/plans/2026-08-25-setup-storage-flow.md
git commit -m "chore: complete setup-storage-flow build tasks"
```

### Task 7: 存储步骤文案与 localfs 提示

**Files:**
- Modify: `web/admin/src/features/setup/setup-steps.ts`、`web/admin/src/features/setup/setup-wizard.tsx`、`web/admin/src/features/setup/setup-pages.contract.test.ts`
- Docs: `design.md`、Design Doc、`tasks.md`、本计划

**Interfaces:**
- Consumes: Task 1 的 storage 步骤。
- Produces: 存储步骤 Title「文件存储配置」/ Desc「决定了生成的图和视频存放的位置。」；`blobDriver === 'localfs'` 时展示 warn Alert（`CircleAlert` 图标 + Title「注意！」+ Description 连续文案，自然换行不强制断行）；驱动标签「S3 兼容」改「S3」（向导 + 设置页 i18n）。

- [x] **Step 1: 先写失败合同测试**

断言 `title: '文件存储配置'`、`desc: '决定了生成的图和视频存放的位置。'`、wizard 含 `AlertTitle>注意！`、`AlertDescription`、`CircleAlert`、连续文案与 `Alert variant='warn'`、`SelectItem value='s3'>S3` 且无「S3 兼容」；RED 确认。

- [x] **Step 2: 实现并全量验证**

`setup-steps.ts` 更新 storage copy；`alert.tsx` 新增 `warn` 变体（amber 系）；`setup-wizard.tsx` 在 `blobDriver === 'localfs'` 时渲染 `Alert variant='warn'`（`CircleAlert` + Title「注意！」+ Description 连续文案），S3 标签改「S3」；`zh/en.json` 的 `blobS3` 同步；`pnpm vitest run`（57 文件 / 364 测试）与 `pnpm tsc -b` 通过。

- [x] **Step 3: 提交**

```bash
git add web/admin/src/components/ui/alert.tsx web/admin/src/features/setup/setup-steps.ts web/admin/src/features/setup/setup-wizard.tsx web/admin/src/features/setup/setup-pages.contract.test.ts docs/openspec/changes/setup-storage-flow/design.md docs/superpowers/specs/2026-08-25-setup-storage-flow-design.md docs/openspec/changes/setup-storage-flow/tasks.md docs/superpowers/plans/2026-08-25-setup-storage-flow.md
git commit -m "feat(setup): localfs warn alert with remote-node caveat"
```

### Task 8: 火山 TOS 默认值预填

**Files:**
- Modify: `web/admin/src/features/setup/setup-wizard.tsx`、`web/admin/src/features/setup/setup-pages.contract.test.ts`
- Docs: `design.md`、Design Doc、`tasks.md`、本计划

**Interfaces:**
- Consumes: Task 1 的对象存储驱动下拉。
- Produces: 选择 TOS 时预填 `TOS_DEFAULTS`（endpoint/region/bucket），空值才填、不覆盖手输。

- [x] **Step 1: 先写失败合同测试**

断言 `tos-cn-beijing.volces.com`、`cn-beijing`、`TOS_DEFAULTS`、`onBlobDriverChange` 存在；RED 确认。

- [x] **Step 2: 实现并全量验证**

新增 `TOS_DEFAULTS` 常量与 `onBlobDriverChange`（切到 tos 且字段为空时填充）；驱动下拉 `onValueChange={onBlobDriverChange}`；`pnpm vitest run`（57 文件 / 366 测试）与 `pnpm tsc -b` 通过。

- [x] **Step 3: 提交**

```bash
git add web/admin/src/features/setup/setup-wizard.tsx web/admin/src/features/setup/setup-pages.contract.test.ts docs/openspec/changes/setup-storage-flow/design.md docs/superpowers/specs/2026-08-25-setup-storage-flow-design.md docs/openspec/changes/setup-storage-flow/tasks.md docs/superpowers/plans/2026-08-25-setup-storage-flow.md
git commit -m "feat(setup): prefill volcengine tos defaults"
```

---

## 自检记录（写完后由创建者核对）

- Spec 覆盖：`setup-wizard`（去 placement、对象存储连通性测试、bucket 自动创建提示、localfs 校验、API 层 remote+localfs 拒绝）→ Task 1/4/5/6；`shared-blob-store`（Check + EnsureBucket + 权限失败）→ Task 2/3/4。
- 占位符扫描：无 TBD/TODO；代码步骤含真实内容。
- 类型一致性：`blob.ErrBucketNotFound` 在 port.go 定义、各驱动/工厂/handler 引用一致；`testBlob` 返回类型与 `blob-test` 响应一致；`AlertCopy` 复用。
