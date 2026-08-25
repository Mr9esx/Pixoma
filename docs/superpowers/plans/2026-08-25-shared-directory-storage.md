# shared-directory-storage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 文件存储支持「共享目录（SMB / NFS 挂载）」驱动 `sharedfs`，覆盖同机房多设备共享文件；向导提供挂载指引与可写校验，S3 选项补充局域网（MinIO）提示。

**Architecture:** 新增 `blob.driver=sharedfs`（复用 localfs 实现）；`settings.Validate` 允许远程+sharedfs、仍拒绝远程+localfs，sharedfs 要求非空路径；`blob/factory` 与 `blob-test` 分发/推断 sharedfs→localfs 检查；前端向导新增「共享目录（SMB / NFS）」选项（挂载目录输入 + info 挂载指引 + 连通性测试/继续），S3 加局域网提示。

**Tech Stack:** Go 1.25 + GORM/settings；React 18 + Vitest（合同测试）、pnpm。

## Global Constraints

- 不新增第三方依赖；不做 WebDAV/SFTP；应用不负责挂载（只校验可写并给指引）。
- `sharedfs` 与 `localfs` 同实现但独立驱动值，文案/校验不共用「仅本机」语义。
- 每个任务独立可测交付 + 单独 commit。

---

### Task 1: 后端 sharedfs 驱动常量与 settings 校验

**Files:**
- Modify: `internal/platform/botconfig/config.go`
- Modify: `internal/platform/settings/settings.go`
- Test: `internal/platform/settings/settings_test.go`

**Interfaces:**
- Consumes: 无。
- Produces: `botconfig.BlobDriverSharedFS = "sharedfs"`；`settings.Validate` 支持 sharedfs（非空路径、remote 允许）。

- [x] **Step 1: 先写失败测试（settings_test.go）**

```go
func TestValidate_SharedFSRequiresRoot(t *testing.T) {
	s := settings.Settings{
		Placement:  settings.PlacementLocal,
		DBDriver:   settings.DriverSQLite,
		DBDSN:      "data/app.db",
		BlobDriver: botconfig.BlobDriverSharedFS,
	}
	err := s.Validate()
	if err == nil || !strings.Contains(err.Error(), "blob root") {
		t.Fatalf("want blob root error, got %v", err)
	}
}

func TestValidate_RemoteAllowsSharedFS(t *testing.T) {
	s := settings.Settings{
		Placement:  settings.PlacementRemote,
		DBDriver:   settings.DriverSQLite,
		DBDSN:      "data/app.db",
		BlobDriver: botconfig.BlobDriverSharedFS,
		BlobRoot:   "/mnt/pixoma-shared",
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("remote sharedfs should pass, got %v", err)
	}
}
```

- [x] **Step 2: 运行测试确认失败**

Run: `go test ./internal/platform/settings/`
Expected: FAIL（常量未定义/校验拒绝）。

- [x] **Step 3: 实现**

`botconfig/config.go` 增加 `BlobDriverSharedFS = "sharedfs"`。

`settings/settings.go` `Validate()`：

```go
switch b {
case botconfig.BlobDriverLocalFS, botconfig.BlobDriverSharedFS,
	botconfig.BlobDriverS3, botconfig.BlobDriverTOS:
default:
	return fmt.Errorf("settings: unknown blob.driver %q", b)
}
if (b == botconfig.BlobDriverLocalFS || b == botconfig.BlobDriverSharedFS) &&
	strings.TrimSpace(s.BlobRoot) == "" {
	return fmt.Errorf("settings: %s requires blob root", b)
}
```

`PlacementRemote` 分支：`localfs` 拒绝；允许 `s3`/`tos`/`sharedfs`。

- [x] **Step 4: 运行测试确认通过**

Run: `go test ./internal/platform/settings/`
Expected: PASS（含既有 remote+localfs 拒绝用例）。

- [x] **Step 5: 提交**

```bash
git add internal/platform/botconfig/config.go internal/platform/settings/settings.go internal/platform/settings/settings_test.go
git commit -m "feat(settings): sharedfs blob driver validation"
```

---

### Task 2: blob/factory 与 blob-test 支持 sharedfs

**Files:**
- Modify: `internal/platform/blob/factory/factory.go`、`internal/platform/blob/factory/check.go`
- Modify: `internal/httpapi/setup/handler.go`
- Test: `internal/platform/blob/factory/check_test.go`、`internal/httpapi/setup/handler_test.go`

**Interfaces:**
- Consumes: Task 1 的 `BlobDriverSharedFS`。
- Produces: `factory.NewFromConfig`/`Check`/`EnsureBucket` 把 `sharedfs` 分发到 localfs；`blob-test` 推断 sharedfs→remote。

- [x] **Step 1: 先写失败测试**

`factory/check_test.go`：

```go
func TestCheck_SharedFS(t *testing.T) {
	if err := factory.Check(context.Background(), factory.CheckOptions{
		Driver:    botconfig.BlobDriverSharedFS,
		LocalRoot: t.TempDir(),
	}); err != nil {
		t.Fatalf("sharedfs check: %v", err)
	}
}
```

`handler_test.go`：`TestBlobTest_SharedFSSuccess`（driver=sharedfs + temp dir → 200 ok）。

- [x] **Step 2: 运行测试确认失败**

Run: `go test ./internal/platform/blob/factory/ ./internal/httpapi/setup/`
Expected: FAIL（unknown driver）。

- [x] **Step 3: 实现**

`factory.go` `NewFromConfig` 与 `check.go` `storeForCheck`/`EnsureBucket` 增加：

```go
case botconfig.BlobDriverSharedFS:
	return localfs.New(localRoot) // NewFromConfig
// check.go 的 storeForCheck / EnsureBucket 同理
```

`handler.go` `blobTest` placement 推断：

```go
if driver == "s3" || driver == "tos" || driver == botconfig.BlobDriverSharedFS {
	placement = settings.PlacementRemote
}
```

- [x] **Step 4: 运行测试确认通过**

Run: `go test ./internal/platform/blob/factory/ ./internal/httpapi/setup/`
Expected: PASS。

- [x] **Step 5: 提交**

```bash
git add internal/platform/blob/factory/factory.go internal/platform/blob/factory/check.go internal/platform/blob/factory/check_test.go internal/httpapi/setup/handler.go internal/httpapi/setup/handler_test.go
git commit -m "feat(blob): dispatch sharedfs to localfs in factory and blob-test"
```

---

### Task 3: 前端向导「共享目录（SMB / NFS）」

**Files:**
- Modify: `web/admin/src/features/setup/setup-wizard.tsx`
- Test: `web/admin/src/features/setup/setup-pages.contract.test.ts`

**Interfaces:**
- Consumes: Task 1/2 后端契约（`blob-test` 接受 sharedfs）。
- Produces: 驱动下拉「共享目录（SMB / NFS）」；挂载目录输入 + info 挂载指引；S3 局域网提示；`draft()` placement 推断 sharedfs→remote。

- [x] **Step 1: 先写失败合同测试**

```ts
it('offers shared directory (SMB/NFS) with mount guidance', () => {
  const wizard = read('src/features/setup/setup-wizard.tsx')
  expect(wizard).toMatch(/共享目录（SMB \/ NFS）/)
  expect(wizard).toMatch(/SelectItem value='sharedfs'/)
  expect(wizard).toMatch(/mount -t nfs/)
  expect(wizard).toMatch(/mount -t cifs/)
  expect(wizard).toMatch(/blobDriver === 'sharedfs'/)
  expect(wizard).toMatch(/MinIO/)
  expect(wizard).toMatch(/placement: blobDriver === 'localfs' \? 'local' : 'remote'/)
})
```

- [x] **Step 2: 运行测试确认失败**

Run: `cd web/admin && pnpm vitest run src/features/setup/`
Expected: FAIL。

- [x] **Step 3: 实现**

`setup-wizard.tsx`：

- 驱动下拉新增 `<SelectItem value='sharedfs'>共享目录（SMB / NFS）</SelectItem>`。
- `sharedfs` 分支（与 localfs 同表单字段 `blobRoot`，标签「挂载目录」）+ info Alert 挂载指引（NFS/CIFS 示例，`<pre>`）。
- `blobDriver === 's3'` 时 Endpoint 下加 hint（MinIO 示例）。
- `draft()`：`placement: blobDriver === 'localfs' ? 'local' : 'remote'`（sharedfs/s3/tos→remote）。

- [x] **Step 4: 运行测试确认通过**

Run: `cd web/admin && pnpm vitest run && pnpm tsc -b`
Expected: PASS。

- [x] **Step 5: 提交**

```bash
git add web/admin/src/features/setup/setup-wizard.tsx web/admin/src/features/setup/setup-pages.contract.test.ts
git commit -m "feat(setup): shared directory (SMB/NFS) option with mount guidance"
```

---

### Task 4: 文档与验证

**Files:**
- Modify: `README.md`、`docs/openspec/changes/shared-directory-storage/tasks.md`、本计划

- [x] **Step 1: README 更新**

「对象存储」段落补充 `sharedfs`：同机房多设备共享目录（SMB/NFS），先挂载再配置，NFS/CIFS 示例命令。

- [x] **Step 2: 全量验证**

Run: `go build ./... && go test ./...`；`cd web/admin && pnpm tsc -b && pnpm vitest run`
Expected: 全 PASS。

- [x] **Step 3: 勾选 tasks.md 全部任务并提交**

```bash
git add README.md docs/openspec/changes/shared-directory-storage/tasks.md docs/superpowers/plans/2026-08-25-shared-directory-storage.md
git commit -m "chore: complete shared-directory-storage build tasks"
```

---

## 自检记录（写完后由创建者核对）

- Spec 覆盖：`shared-blob-store`（sharedfs 驱动 + 校验矩阵 + 连通性检查）→ Task 1/2；`setup-wizard`（共享目录选项、挂载指引、S3 局域网提示、placement 推断）→ Task 3。
- 占位符扫描：无 TBD/TODO。
- 类型一致性：`botconfig.BlobDriverSharedFS` 常量在 Task 1 定义、Task 2 分发引用一致；`blob-test` 请求体 sharedfs 仅 driver+root。
