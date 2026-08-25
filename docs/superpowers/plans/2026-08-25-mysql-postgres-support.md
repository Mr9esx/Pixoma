# mysql-postgres-support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 Pixoma 业务库完整支持 SQLite/MySQL/Postgres：Setup 向导支持配置业务库，设置页只读展示业务库信息，MySQL/Postgres 下启动装配与核心读写可用，并用集成测试锁定。

**Architecture:** 业务库连接只在 Setup 向导数据库步骤配置（现状已具备三驱动下拉 + DSN + 测连通，`POST /api/v1/setup/database` 校验并写引导态）；本期把 Task 仓储 `lease_until`/`requeue_at` 的零值 `time.Time{}` 改为写 `NULL` 并修正 `lease_until IS NOT NULL` 条件，以兼容 MySQL 严格模式；`drivers_integration_test.go` 在 env 门控下对 MySQL/Postgres 跑全模型迁移与核心链路 roundtrip；Setup 向导补 DSN 占位；设置页以合同测试锁定只读展示。

**Tech Stack:** Go 1.25 + GORM（sqlite/mysql/postgres dialector）、chi；React 18 + TanStack Query + react-i18next + Vitest（node 合同测试）、pnpm。

## Global Constraints

- 不新增第三方依赖（Go 与 npm 均不新增）。
- 不做设置页换库、不做后端换库 API；`PUT /api/v1/setup/settings` 维持「DB 只读」语义。
- 业务库连接只在 Setup 向导配置；bootstrap 引导库保持本地 SQLite。
- 不为控制面新增 `DB_DRIVER/DATABASE_DSN` 环境变量覆盖（保持 backfill 专用）。
- 换库/跨引擎数据迁移为本期非目标（迁移走数据迁移 + 重跑 Setup），文档需写明。
- MySQL 建议 8.0+；`RenameLegacy` 的 `RENAME COLUMN` 只在旧 SQLite schema 上触发，新 MySQL/PG 库为 no-op。
- 集成测试以 `PIXOMA_MYSQL_DSN` / `PIXOMA_POSTGRES_DSN` 门控，缺省 skip。
- 每个任务独立可测交付 + 单独 commit。

---

### Task 1: Task 仓储零值时间写 NULL（MySQL 严格模式兼容）

**Files:**
- Modify: `internal/runtime/infrastructure/persistence/gorm_task.go`（`PrepareForClaim`、`RequeueExpiredLeases`、`MigrateLegacyTasks` 三处）
- Test: `internal/runtime/infrastructure/persistence/gorm_task_test.go`（新增 2 个测试）
- Test: `internal/runtime/infrastructure/persistence/gorm_migrate_test.go`（追加 NULL 断言）

**Interfaces:**
- Consumes: 无（不改变公开签名）。
- Produces: `TaskRepository.PrepareForClaim`、`TaskRepository.RequeueExpiredLeases`、`persistence.MigrateLegacyTasks` 行为变化——`lease_until`/`requeue_at` 落库为 NULL（而非零值时间），`RequeueExpiredLeases` 的 WHERE 使用 `lease_until IS NOT NULL AND lease_until < ?`。

- [x] **Step 1: 写失败测试（PrepareForClaim 写 NULL）**

在 `gorm_task_test.go` 末尾追加：

```go
func TestGormTask_NullTimestampsAfterPrepareForClaim(t *testing.T) {
	gdb := openTestDB(t)
	seedSession(t, gdb, "s-null", sharedkernel.ChatID("tg:11"))
	tasks := persistence.NewTaskRepository(gdb)
	ctx := context.Background()
	now := time.Unix(600, 0).UTC()
	if err := tasks.Create(ctx, domain.NewPending("t-null", "s-null", sharedkernel.CaseID(1), "inputs/t-null", now)); err != nil {
		t.Fatal(err)
	}
	ref := sharedkernel.BlobRef{Key: "jobs/t-null/job.json"}
	ok, err := tasks.PrepareForClaim(ctx, "t-null", "default", ref, now)
	if err != nil || !ok {
		t.Fatalf("prepare ok=%v err=%v", ok, err)
	}
	var raw struct {
		LeaseUntil *time.Time
		RequeueAt  *time.Time
	}
	if err := gdb.Model(&persistence.TaskRow{}).Where("id = ?", "t-null").Scan(&raw).Error; err != nil {
		t.Fatal(err)
	}
	if raw.LeaseUntil != nil || raw.RequeueAt != nil {
		t.Fatalf("want NULL lease/requeue after prepare, got %v / %v", raw.LeaseUntil, raw.RequeueAt)
	}
}
```

- [x] **Step 2: 运行测试确认失败**

Run: `go test ./internal/runtime/infrastructure/persistence/ -run TestGormTask_NullTimestampsAfterPrepareForClaim -v`
Expected: FAIL——当前实现把 `time.Time{}` 写进 `lease_until`/`requeue_at`，扫描到非 nil 指针。

- [x] **Step 3: 写失败测试（RequeueExpiredLeases 清空租约并保留条件语义）**

在 `gorm_task_test.go` 末尾追加：

```go
func TestGormTask_RequeueClearsLeaseToNULL(t *testing.T) {
	gdb := openTestDB(t)
	seedSession(t, gdb, "s-req", sharedkernel.ChatID("tg:12"))
	tasks := persistence.NewTaskRepository(gdb)
	ctx := context.Background()
	now := time.Unix(700, 0).UTC()
	ref := sharedkernel.BlobRef{Key: "jobs/t-req/job.json"}
	task := domain.NewPending("t-req", "s-req", sharedkernel.CaseID(1), "inputs/t-req", now)
	_ = task.PrepareForClaim("gpu-1", ref, now)
	_ = task.ClaimWithLease("gpu-1", time.Minute, now)
	if err := tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	later := now.Add(2 * time.Minute)
	n, err := tasks.RequeueExpiredLeases(ctx, later)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("requeued=%d want 1", n)
	}
	var raw struct {
		Status     string
		LeaseUntil *time.Time
	}
	if err := gdb.Model(&persistence.TaskRow{}).Where("id = ?", "t-req").Scan(&raw).Error; err != nil {
		t.Fatal(err)
	}
	if raw.Status != string(sharedkernel.TaskQueued) || raw.LeaseUntil != nil {
		t.Fatalf("want queued with NULL lease, got %+v", raw)
	}
}
```

- [x] **Step 4: 运行测试确认失败**

Run: `go test ./internal/runtime/infrastructure/persistence/ -run 'TestGormTask_(NullTimestampsAfterPrepareForClaim|RequeueClearsLeaseToNULL)' -v`
Expected: 两个测试都 FAIL（当前写零值时间，扫描非 nil）。

- [x] **Step 5: 追加 MigrateLegacyTasks 的 NULL 断言**

在 `gorm_migrate_test.go` 的 `TestMigrateLegacyTasks` 末尾（现有 `run2` 断言之后）追加：

```go
	var raw struct {
		LeaseUntil *time.Time
		RequeueAt  *time.Time
	}
	if err := gdb.Model(&persistence.TaskRow{}).Where("id = ?", "t-stale").Scan(&raw).Error; err != nil {
		t.Fatal(err)
	}
	if raw.LeaseUntil != nil || raw.RequeueAt != nil {
		t.Fatalf("want NULL lease/requeue after migrate, got %v / %v", raw.LeaseUntil, raw.RequeueAt)
	}
```

- [x] **Step 6: 运行测试确认失败**

Run: `go test ./internal/runtime/infrastructure/persistence/ -run TestMigrateLegacyTasks -v`
Expected: FAIL（`t-stale` 的 `lease_until`/`requeue_at` 当前是零值时间而非 NULL）。

- [x] **Step 7: 实现零值时间 NULL 化**

修改 `internal/runtime/infrastructure/persistence/gorm_task.go`：

`PrepareForClaim` 的 `Updates` map（约 145-153 行）：

```go
		Updates(map[string]any{
			"status":         string(sharedkernel.TaskQueued),
			"edge_id":        "",
			"dispatch_topic": topicKey,
			"job_ref_json":   string(raw),
			"lease_until":    gorm.Expr("NULL"),
			"requeue_at":     gorm.Expr("NULL"),
			"updated_at":     now,
		})
```

`RequeueExpiredLeases`（约 251-262 行）：

```go
	res := r.db.WithContext(ctx).Model(&TaskRow{}).
		Where("status = ? AND lease_until IS NOT NULL AND lease_until < ?", string(sharedkernel.TaskRunning), now).
		Updates(map[string]any{
			"status":      string(sharedkernel.TaskQueued),
			"edge_id":     "",
			"lease_until": gorm.Expr("NULL"),
			"requeue_at":  now,
			"updated_at":  now,
		})
```

`MigrateLegacyTasks`（约 516-526 行）：

```go
	res := gdb.WithContext(ctx).Model(&TaskRow{}).
		Where("status = ? AND edge_id != '' AND lease_until < ?", string(sharedkernel.TaskQueued), now).
		Updates(map[string]any{
			"edge_id":        "",
			"lease_until":    gorm.Expr("NULL"),
			"requeue_at":     gorm.Expr("NULL"),
			"dispatch_topic": "default",
			"updated_at":     now,
		})
```

`gorm` 已在文件 import 中（`"gorm.io/gorm"`），无需新增 import。

- [x] **Step 8: 运行全部任务相关测试确认通过**

Run: `go test ./internal/runtime/infrastructure/persistence/`
Expected: PASS（含既有 `TestGormTask_ClaimNextWithLeaseAndExpire` 等回归）。

- [x] **Step 9: 提交**

```bash
git add internal/runtime/infrastructure/persistence/gorm_task.go internal/runtime/infrastructure/persistence/gorm_task_test.go internal/runtime/infrastructure/persistence/gorm_migrate_test.go
git commit -m "fix(tasks): write NULL lease/requeue timestamps for MySQL strict mode"
```

---

### Task 2: Setup 向导数据库步骤 DSN 占位/示例

**Files:**
- Modify: `web/admin/src/features/setup/setup-wizard.tsx`
- Test: `web/admin/src/features/setup/setup-pages.contract.test.ts`

**Interfaces:**
- Consumes: 无。
- Produces: 向导数据库步骤 DSN 输入按 driver 展示占位；新增模块级常量 `DB_DSN_PLACEHOLDER: Record<string, string>`（sqlite/mysql/postgres）。

- [x] **Step 1: 写失败合同测试**

在 `setup-pages.contract.test.ts` 的 `describe` 内追加：

```ts
  it('offers per-driver DSN placeholders in the wizard database step', () => {
    const wizard = read('src/features/setup/setup-wizard.tsx')
    expect(wizard).toMatch(/DB_DSN_PLACEHOLDER/)
    expect(wizard).toMatch(/placeholder=\{DB_DSN_PLACEHOLDER\[driver\]/)
    expect(wizard).toMatch(/mysql:/)
    expect(wizard).toMatch(/postgres:/)
    expect(wizard).toMatch(/SelectItem value='sqlite'/)
    expect(wizard).toMatch(/SelectItem value='mysql'/)
    expect(wizard).toMatch(/SelectItem value='postgres'/)
  })
```

- [x] **Step 2: 运行测试确认失败**

Run: `cd web/admin && pnpm vitest run src/features/setup/setup-pages.contract.test.ts`
Expected: FAIL——`setup-wizard.tsx` 尚无 `DB_DSN_PLACEHOLDER`。

- [x] **Step 3: 实现 DSN 占位**

在 `setup-wizard.tsx` 的 import 之后、`SetupWizard` 组件之前添加：

```tsx
const DB_DSN_PLACEHOLDER: Record<string, string> = {
  sqlite: 'data/app.db',
  mysql:
    'user:password@tcp(127.0.0.1:3306)/pixoma?charset=utf8mb4&parseTime=True&loc=Local',
  postgres:
    'host=127.0.0.1 port=5432 user=pixoma password=... dbname=pixoma sslmode=disable',
}
```

把数据库步骤的 DSN Input（现有 `htmlFor='db-dsn'` 的 `Input`）改为：

```tsx
          <Field label='连接' htmlFor='db-dsn'>
            <Input
              id='db-dsn'
              value={dsn}
              onChange={(e) => setDsn(e.target.value)}
              placeholder={DB_DSN_PLACEHOLDER[driver] ?? DB_DSN_PLACEHOLDER.sqlite}
            />
          </Field>
```

- [x] **Step 4: 运行测试确认通过**

Run: `cd web/admin && pnpm vitest run src/features/setup/setup-pages.contract.test.ts`
Expected: PASS。

- [x] **Step 5: 提交**

```bash
git add web/admin/src/features/setup/setup-wizard.tsx web/admin/src/features/setup/setup-pages.contract.test.ts
git commit -m "feat(setup): per-driver DSN placeholders in wizard"
```

---

### Task 3: 设置页业务库信息只读展示合同测试

**Files:**
- Test: `web/admin/src/features/settings/settings-page.contract.test.ts`

**Interfaces:**
- Consumes: 设置页现状——account tab 用 `<p>` 展示 `initial.db_driver` / `initial.db_dsn`。
- Produces: 锁定「设置页只读展示业务库信息、无编辑/切换控件」的合同断言。

- [x] **Step 1: 写合同测试（锁定既有行为，预期直接 PASS）**

在 `settings-page.contract.test.ts` 的 `describe('settings page', ...)` 内追加：

```ts
  it('shows the business database driver and DSN read-only', () => {
    const page = read('settings-page.tsx')
    expect(page).toMatch(/initial\.db_driver/)
    expect(page).toMatch(/initial\.db_dsn/)
    expect(page).not.toMatch(/htmlFor=['"]db-driver['"]/)
    expect(page).not.toMatch(/htmlFor=['"]db-dsn['"]/)
    expect(page).not.toMatch(/testDatabase\(/)
    expect(page).not.toMatch(/DB_DSN_PLACEHOLDER/)
  })
```

- [x] **Step 2: 运行测试确认通过**

Run: `cd web/admin && pnpm vitest run src/features/settings/settings-page.contract.test.ts`
Expected: PASS（无源码改动，仅锁行为；若 FAIL 说明设置页已被改成编辑表单，需停下与用户对齐）。

- [x] **Step 3: 提交**

```bash
git add web/admin/src/features/settings/settings-page.contract.test.ts
git commit -m "test(admin): lock read-only business DB info on settings page"
```

---

### Task 4: MySQL/Postgres 集成测试（全模型迁移 + 核心读写）

**Files:**
- Rewrite: `internal/platform/db/drivers_integration_test.go`（`//go:build integration`）

**Interfaces:**
- Consumes: `db.Open`/`db.AutoMigrate`/`db.RenameLegacy`；`settings.NewStore`/`Store.Save`/`Store.Load`；`topicpersist.NewTopicRepository`/`Create`/`Get`；`casepersist.NewGormRepository`/`Create`/`Get`；`runtimepersist.NewTaskRepository`/`Create`/`Get`/`ClaimNextWithLease`、`persistence.MigrateLegacyTasks`；`stats` 仓储 `AddTerminal`。
- Produces: `TestIntegration_FullMigrateAndRoundtrip`（env 门控、driver 子测试）；helper `openEnvDB`、`allBusinessModels`、`exerciseCoreRoundtrip`。

- [x] **Step 1: 写失败测试（先让新测试函数就位）**

整体替换 `drivers_integration_test.go` 内容为：

```go
//go:build integration

package db_test

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	casepersist "github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	channelpersist "github.com/mr9esx/comfyui_tgbot/internal/channel/infrastructure/persistence"
	sesspersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	userpersist "github.com/mr9esx/comfyui_tgbot/internal/identity/infrastructure/persistence"
	mencardpersist "github.com/mr9esx/comfyui_tgbot/internal/menucard/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	instpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/edge/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/settings"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats"
	statspersist "github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/topic"
	topicpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/topic/persistence"
	taskdomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	runtimepersist "github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestIntegration_FullMigrateAndRoundtrip(t *testing.T) {
	drivers := []struct {
		driver string
		env    string
	}{
		{db.DriverMySQL, "PIXOMA_MYSQL_DSN"},
		{db.DriverPostgres, "PIXOMA_POSTGRES_DSN"},
	}
	for _, tc := range drivers {
		t.Run(tc.driver, func(t *testing.T) {
			gdb := openEnvDB(t, tc.driver, tc.env)
			if err := db.AutoMigrate(gdb, allBusinessModels()...); err != nil {
				t.Fatalf("migrate all: %v", err)
			}
			exerciseCoreRoundtrip(t, gdb)
		})
	}
}

func openEnvDB(t *testing.T, driver, env string) *gorm.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv(env))
	if dsn == "" {
		t.Skipf("set %s to run", env)
	}
	gdb, err := db.Open(db.Options{Driver: driver, DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, e := gdb.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})
	return gdb
}

func allBusinessModels() []any {
	return []any{
		&casepersist.CaseRow{},
		&userpersist.UserRow{},
		&userpersist.UserExternalIdentityRow{},
		&sesspersist.SessionRow{},
		&runtimepersist.TaskRow{},
		&statspersist.DailyStatsRow{},
		&statspersist.EdgeDailyStatsRow{},
		&statspersist.ErrorDailyStatsRow{},
		&statspersist.CaseDailyStatsRow{},
		&topicpersist.TopicRow{},
		&channelpersist.ChannelRow{},
		&mencardpersist.MainMenuRow{},
		&mencardpersist.CardRow{},
		&instpersist.EdgeRow{},
		&instpersist.MetricsRow{},
	}
}

func exerciseCoreRoundtrip(t *testing.T, gdb *gorm.DB) {
	t.Helper()
	ctx := context.Background()
	now := time.Unix(1000, 0).UTC()

	// RenameLegacy must be a no-op on fresh MySQL/Postgres databases.
	if err := db.RenameLegacy(gdb); err != nil {
		t.Fatalf("rename legacy: %v", err)
	}
	// Legacy menu tables dropped by the app at boot must be droppable.
	if err := gdb.Migrator().DropTable(
		"channel_menus",
		"channel_menu_items",
		"channel_menu_item_cases",
		"channel_menu_item_extras",
	); err != nil {
		t.Fatalf("drop legacy menu tables: %v", err)
	}

	// settings roundtrip
	key := bytes.Repeat([]byte{7}, 32)
	st, err := settings.NewStore(gdb, key)
	if err != nil {
		t.Fatalf("settings store: %v", err)
	}
	want := settings.Settings{
		Placement:      settings.PlacementLocal,
		DBDriver:       settings.DriverSQLite,
		DBDSN:          "data/app.db",
		BlobDriver:     "localfs",
		BlobRoot:       "data/blob",
		ComfyMock:      true,
		ComfyUIBaseURL: "http://127.0.0.1:8188",
	}
	if err := st.Save(want); err != nil {
		t.Fatalf("settings save: %v", err)
	}
	got, err := st.Load()
	if err != nil {
		t.Fatalf("settings load: %v", err)
	}
	if got.BlobRoot != want.BlobRoot || !got.ComfyMock {
		t.Fatalf("settings roundtrip mismatch: %+v", got)
	}

	// topic roundtrip
	tRepo := topicpersist.NewTopicRepository(gdb)
	tRow := topic.Topic{
		Key:       "integration",
		Name:      "Integration Topic",
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := tRepo.Create(ctx, tRow); err != nil {
		t.Fatalf("topic create: %v", err)
	}
	tgot, err := tRepo.Get(ctx, "integration")
	if err != nil || tgot.Name != "Integration Topic" {
		t.Fatalf("topic get: %+v err=%v", tgot, err)
	}

	// case roundtrip
	cRepo := casepersist.NewGormRepository(gdb)
	c := &catalogdomain.Case{
		Document: catalogdomain.CaseDocument{
			ID:    sharedkernel.CaseID(1),
			Name:  "Integration Case",
			Inputs: []catalogdomain.InputField{},
			Outputs: []catalogdomain.OutputField{},
			Bindings: catalogdomain.ComfyBindings{
				WorkflowJSON: map[string]any{},
				Inputs:       []catalogdomain.InputBinding{},
				Outputs:      []catalogdomain.OutputBinding{},
			},
			InputSchema: map[string]any{"type": "object"},
		},
		Enabled: true,
	}
	if err := cRepo.Create(ctx, c); err != nil {
		t.Fatalf("case create: %v", err)
	}
	cgot, err := cRepo.Get(ctx, sharedkernel.CaseID(1))
	if err != nil || cgot.Document.Name != "Integration Case" {
		t.Fatalf("case get: %+v err=%v", cgot, err)
	}

	// task roundtrip incl. PrepareForClaim (writes NULL timestamps)
	tasks := runtimepersist.NewTaskRepository(gdb)
	ref := sharedkernel.BlobRef{Key: "jobs/t-int/job.json"}
	task := taskdomain.NewPending("t-int", "s-int", sharedkernel.CaseID(1), "inputs/t-int", now)
	_ = task.PrepareForClaim("gpu-int", ref, now)
	if err := tasks.Create(ctx, task); err != nil {
		t.Fatalf("task create: %v", err)
	}
	claimed, err := tasks.ClaimNextWithLease(ctx, "gpu-int", []string{"integration"}, 90*time.Second, now)
	if err != nil || claimed == nil || claimed.ID != "t-int" {
		t.Fatalf("claim: %+v err=%v", claimed, err)
	}
	gotTask, err := tasks.Get(ctx, "t-int")
	if err != nil || gotTask.Status != sharedkernel.TaskRunning {
		t.Fatalf("task get: %+v err=%v", gotTask, err)
	}

	// stats upsert (clause.OnConflict) exercised twice on the same day
	statsRepo := statspersist.NewGormStatsRepository(gdb, 24*time.Hour, time.UTC)
	for i := 0; i < 2; i++ {
		if err := statsRepo.AddTerminal(ctx, taskstats.AddTerminalInput{
			EdgeID:          "gpu-int",
			Status:          taskstats.StatusSucceeded,
			CompletedAt:     now,
			CreatedAt:       now.Add(-time.Minute),
			CaseID:          1,
			QueueDurationMS: 10,
			ExecDurationMS:  20,
		}); err != nil {
			t.Fatalf("stats add %d: %v", i, err)
		}
	}

	// legacy task reassignment path
	stale := taskdomain.NewPending("t-stale2", "s-int", sharedkernel.CaseID(1), "inputs/t-stale2", now)
	_ = stale.PrepareForClaim("gpu-old", sharedkernel.BlobRef{Key: "jobs/t-stale2/job.json"}, now.Add(-time.Minute))
	stale.LeaseUntil = now.Add(-time.Second)
	if err := tasks.Create(ctx, stale); err != nil {
		t.Fatalf("stale create: %v", err)
	}
	n, err := runtimepersist.MigrateLegacyTasks(ctx, gdb, now)
	if err != nil {
		t.Fatalf("migrate legacy: %v", err)
	}
	if n < 1 {
		t.Fatalf("migrate legacy count=%d want >=1", n)
	}
}
```

`settings.Store` 会自行 AutoMigrate `platform_settings`（`NewStore` 内），故无需显式加入模型列表。

- [x] **Step 2: 无 env 时验证 skip**

Run: `go test -tags integration ./internal/platform/db/ -v`
Expected: SKIP（`PIXOMA_MYSQL_DSN`/`PIXOMA_POSTGRES_DSN` 未设置）；若设置了 env，则连真实库执行并 Expected: PASS。

- [x] **Step 3: 本地回归（非 integration 包不受影响）**

Run: `go build ./...`
Expected: 编译通过（新测试文件带 build tag，正常 `go build` 不编译）。

- [x] **Step 4: 提交**

```bash
git add internal/platform/db/drivers_integration_test.go
git commit -m "test(db): full-model MySQL/Postgres integration roundtrip"
```

---

### Task 5: README 与数据模型文档更新

**Files:**
- Modify: `README.md`
- Modify: `docs/architecture/data-model.md`

**Interfaces:**
- Consumes: 无。
- Produces: 文档说明 Setup 配置入口、MySQL 8.0+ 建议、换库为非目标。

- [x] **Step 1: 更新 README**

在「常用环境变量」表格附近的 `DB_DRIVER` / `DATABASE_DSN` 行下方（或「新部署」段落）追加：

```markdown
### 业务数据库

新部署在初始化向导的「数据库配置」步骤选择 SQLite / MySQL / Postgres 并填写连接；设置页只读展示业务库驱动与 DSN。MySQL 建议 8.0+。业务库连接只在 Setup 向导配置，换库/跨引擎数据迁移不在界面内支持：需要迁移时请走数据迁移后重跑初始化向导。
```

- [x] **Step 2: 更新 data-model.md**

把第 5 行引言块从：

```markdown
> 数据库：默认 SQLite（`data/app.db`），向导可选 MySQL / Postgres；GORM AutoMigrate。  
> 引导态另存本机 `data/bootstrap.db`。业务 settings 在 `platform_settings`。
```

改为：

```markdown
> 数据库：默认 SQLite（`data/app.db`），Setup 向导可选 MySQL / Postgres（建议 MySQL 8.0+）；GORM AutoMigrate。业务库连接只在 Setup 向导配置，设置页只读展示驱动与 DSN；换库/跨引擎数据迁移为非目标（迁移走数据迁移 + 重跑 Setup）。  
> 引导态另存本机 `data/bootstrap.db`。业务 settings 在 `platform_settings`。
```

- [x] **Step 3: 提交**

```bash
git add README.md docs/architecture/data-model.md
git commit -m "docs: multi-database setup guidance and read-only settings note"
```

---

### Task 6: 全量验证与收尾

**Files:**
- Modify: `docs/openspec/changes/mysql-postgres-support/tasks.md`（勾选全部任务）

**Interfaces:**
- Consumes: Task 1-5 产物。
- Produces: 可交付的 build 阶段证据与勾选完成的 tasks.md。

- [x] **Step 1: Go 全量构建与测试**

Run: `go build ./... && go test ./...`
Expected: 全 PASS。

- [x] **Step 2: 前端类型检查与测试**

Run: `cd web/admin && pnpm tsc -b && pnpm vitest run`
Expected: 全 PASS。

- [x] **Step 3: 勾选 tasks.md 全部任务**

把 `docs/openspec/changes/mysql-postgres-support/tasks.md` 中全部 `- [ ]` 改为 `- [x]`（1.1、1.2、2.1、2.2、3.1、4.1、4.2、5.1、5.2）。

- [x] **Step 4: 提交收尾**

```bash
git add docs/openspec/changes/mysql-postgres-support/tasks.md
git commit -m "chore: complete mysql-postgres-support build tasks"
```

- [x] **Step 5: 运行 build 阶段守卫（由主会话执行，不属于本任务提交内容）**

Run: `comet guard mysql-postgres-support build --apply`
Expected: ALL PASS，phase 推进到 verify。

### Task 7: Setup 数据库步骤结构化连接表单（归档前重开）

**Files:**
- Create: `web/admin/src/features/setup/db-dsn.ts`、`web/admin/src/features/setup/db-dsn.test.ts`
- Modify: `web/admin/src/features/setup/setup-wizard.tsx`、`web/admin/src/features/setup/setup-pages.contract.test.ts`、`web/admin/vitest.config.ts`
- Docs: `docs/openspec/changes/mysql-postgres-support/specs/setup-wizard/spec.md`、`design.md`、Design Doc、`tasks.md`

**Interfaces:**
- Consumes: Task 2 的 `DB_DSN_PLACEHOLDER`。
- Produces: 纯函数 `buildSqliteDSN(path)`、`buildMySQLDSN({host,port,user,password,database,params?})`、`buildPostgresDSN({host,port,user,password,database,sslmode,params?})`；向导数据库步骤按 driver 显示结构化字段 + 「附加参数」输入框（MySQL `&a=b`、Postgres 空格分隔 `key=value`）；切驱动时端口默认值联动。

- [x] **Step 1: 新增 db-dsn 组装纯函数与单元测试（TDD RED→GREEN）**

`db-dsn.test.ts` 5 个用例覆盖 sqlite 路径、mysql 组装与 host/port 默认、postgres 组装与 sslmode/port 默认；先跑 RED（模块不存在）再实现 `db-dsn.ts` 转 GREEN。

- [x] **Step 2: 更新向导数据库步骤为结构化字段表单**

SQLite 显示数据库文件路径；MySQL 显示 Host/端口/用户/密码/数据库；Postgres 增加 SSL 模式（disable/require/prefer）；密码 `type=password`；每个服务型驱动显示「附加参数」输入框（`db-extra-params`），参数直接追加进组装结果，不提供整串 DSN 编辑入口；`dsn` 由 `useMemo` 按字段组装。

- [x] **Step 3: 切换驱动端口默认值联动**

`onDriverChange`：切到 mysql 且端口为 5432 时改 3306；切到 postgres 且端口为 3306 时改 5432；字段值保留可编辑。

- [x] **Step 4: 更新合同测试并跑全量前端测试**

合同测试改为断言结构化字段（db-driver/db-sqlite-path/db-host/db-password）、端口联动、附加参数输入框（`db-extra-params`，含 MySQL/Postgres 示例占位），并断言不存在高级 raw DSN 编辑入口；`pnpm vitest run` 全 PASS，`pnpm tsc -b` 通过。

- [x] **Step 5: 提交**

```bash
git add web/admin/src/features/setup/db-dsn.ts web/admin/src/features/setup/db-dsn.test.ts web/admin/src/features/setup/setup-wizard.tsx web/admin/src/features/setup/setup-pages.contract.test.ts web/admin/vitest.config.ts docs/openspec/changes/mysql-postgres-support/specs/setup-wizard/spec.md docs/openspec/changes/mysql-postgres-support/design.md docs/superpowers/specs/2026-08-25-mysql-postgres-support-design.md docs/openspec/changes/mysql-postgres-support/tasks.md docs/superpowers/plans/2026-08-25-mysql-postgres-support.md
git commit -m "feat(setup): structured database connection form for mysql/postgres"
```

### Task 8: 错误用 Alert 展示（中文友好标题 + 实际详情）

**Files:**
- Create: `web/admin/src/features/setup/db-error.ts`、`web/admin/src/features/setup/db-error.test.ts`
- Modify: `web/admin/src/features/setup/setup-wizard.tsx`、`web/admin/src/features/setup/setup-pages.contract.test.ts`、`web/admin/vitest.config.ts`
- Docs: `specs/setup-wizard/spec.md`、`design.md`、Design Doc、`tasks.md`、本计划

**Interfaces:**
- Consumes: Task 7 的结构化表单。
- Produces: `setupErrorCopy(err): { title: string; detail: string }`；向导与 StepActions 的错误渲染改用 `Alert variant="destructive"`。

- [x] **Step 1: 写 db-error 映射与单元测试（TDD）**

6 个用例覆盖：拒绝连接、鉴权失败（1045/28000）、库不存在（1049）、超时、未知驱动、通用回退；先 RED（模块不存在）再实现 `db-error.ts` 转 GREEN。

- [x] **Step 2: 向导错误渲染改 Alert**

`run()` 捕获错误改为 `setError(setupErrorCopy(err))`；StepActions 与「正在重启」卡片渲染 `Alert variant="destructive"`（AlertTitle=友好标题、AlertDescription=实际详情）；删除旧的 `<p className='text-sm text-destructive'>`。

- [x] **Step 3: 合同测试与全量验证**

合同测试断言 Alert destructive / AlertTitle / AlertDescription / setupErrorCopy 且无旧错误文本；`pnpm vitest run` 全 PASS、`pnpm tsc -b` 通过。

- [x] **Step 4: 提交**

```bash
git add web/admin/src/features/setup/db-error.ts web/admin/src/features/setup/db-error.test.ts web/admin/src/features/setup/setup-wizard.tsx web/admin/src/features/setup/setup-pages.contract.test.ts web/admin/vitest.config.ts docs/openspec/changes/mysql-postgres-support/specs/setup-wizard/spec.md docs/openspec/changes/mysql-postgres-support/design.md docs/superpowers/specs/2026-08-25-mysql-postgres-support-design.md docs/openspec/changes/mysql-postgres-support/tasks.md docs/superpowers/plans/2026-08-25-mysql-postgres-support.md
git commit -m "feat(setup): alert-based error copy with friendly Chinese titles"
```

### Task 9: 错误映射边界场景扩展

**Files:**
- Modify: `web/admin/src/features/setup/db-error.ts`、`web/admin/src/features/setup/db-error.test.ts`
- Docs: `design.md`、Design Doc、`tasks.md`、本计划

**Interfaces:**
- Consumes: Task 8 的 `setupErrorCopy`。
- Produces: 映射表从 6 类扩到约 28 类（DNS/路由、连接中断、连接数满、文件锁/磁盘、死锁、角色/表缺失、SSL/TLS、登录/密码、向导步骤前置、远程 localfs、代理、未授权等），单测 19 用例。

- [x] **Step 1: 先写边界场景失败测试**

新增 13 个用例（DNS no such host、1040 连接数满、database is locked、deadlock 40P01、role/relation does not exist、TLS handshake、connection reset、invalid credentials、弱密码、configure database first、远程 localfs、unauthorized），RED 确认。

- [x] **Step 2: 扩展映射表并保证命中顺序**

按具体→通用排序：连接拒绝 → 鉴权 → 库不存在 → 角色/表缺失 → 通用 does not exist → 超时 → DNS/路由 → 连接中断 → 连接数满 → 文件锁/磁盘 → 死锁 → SSL/TLS → 未知驱动 → 缺 DSN → 登录/密码 → 步骤前置 → 存储/代理/未授权 → 回退。

- [x] **Step 3: 全量验证与提交**

`pnpm vitest run`（59 文件 / 355 测试）与 `pnpm tsc -b` 通过；提交 `feat(setup): extend error mapping edge cases`。

---

## 自检记录（写完后由创建者核对）

- Spec 覆盖：`multi-database-support`（三驱动支持 → Task 4；设置页只读 → Task 3；Setup 向导不可达不落盘 → 后端现状 + Task 4 集成验证）、`setup-wizard`（三驱动 + DSN 占位 + 可诊断错误 → Task 2/4）。换库/平台设置预置需求已从 delta spec 移除，不在任务中。
- 占位符扫描：无 TBD/TODO；所有代码步骤含真实内容。
- 类型一致性：`gorm.Expr("NULL")` 三处一致；`RequeueExpiredLeases` 签名不变；`MigrateLegacyTasks` 保持包级函数；前端 `DB_DSN_PLACEHOLDER` 常量在 Task 2 定义并被合同测试引用；集成测试中的仓储 API 与源码签名一致。
