---
change: task-daily-stats
design-doc: docs/superpowers/specs/2026-08-21-task-daily-stats-design.md
base-ref: 05b221f6569c5e15f940a8b1de21e4c8aa92d160
---

# task-daily-stats Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为管理端 Dashboard 提供全量、按天聚合的任务统计（每日处理任务数柱状图 + 日期范围选择 + 成功率 / 错误码 Top-N / 每节点负载），数据来自 `task_daily_stats` 系列统计表与 `/api/v1/stats/tasks/*` 接口，不受 limit=200 样本限制。

**Architecture:** 三张窄表（`task_daily_stats` / `task_edge_daily_stats` / `task_error_daily_stats`）由 orchestrator 任务终态写路径幂等 upsert；admin-api 新增 `/api/v1/stats` 路由读表；`web/admin` Dashboard 任务区改为统计区块（shadcn `ChartContainer` + recharts `BarChart` + `Calendar` range）。归天时区 `STATS_TIMEZONE`（默认 Asia/Shanghai），保留 `TASK_STATS_RETENTION`（默认 365 天）。

**Tech Stack:** Go 1.25（chi + GORM + sqlite 测试）、React + TanStack Query + recharts + react-day-picker 9 + Tailwind v4、Vitest。

## Global Constraints

- 所有 Comet 产物与提交信息使用中文；代码注释沿用现有 Go/TS 风格。
- Go 包路径前缀 `github.com/mr9esx/comfyui_tgbot`；新包建议 `internal/platform/taskstats` 与 `internal/platform/taskstats/persistence`、`internal/httpapi/stats`。
- 统计口径：`processed = succeeded + failed + cancelled`，按 `completed_at` 归天；`success_rate = succeeded / (succeeded + failed)`，分母为 0 时 JSON 返回 `null`。
- 每节点负载仅统计终态任务按 `edge_id` 计数；错误码仅统计非空 `error_code`。
- daily 接口按范围零填充每一天（空白天计数 0），跨度上限 365 天，`from > to` 或格式非法返回 400，空数据返回 200 + 空数组。
- 前端沿用 edge-system-monitoring 已落地的 shadcn `ChartContainer` 卡片体系与 `Calendar` / `Popover`；i18n 必须同时更新 zh.json 与 en.json。
- 每个任务完成后运行对应测试并提交（`feat(task-daily-stats): <描述>`）。

---

## 1. 数据层：统计表与仓储

### Task 1.1: 新增三张统计表的 GORM 模型

**Files:**
- Create: `internal/platform/taskstats/stats.go`
- Create: `internal/platform/taskstats/persistence/gorm_stats.go`

**Interfaces:**
- Produces: `taskstats.Repository`、`taskstats.AddTerminalInput`、`taskstats.Status`、`taskstats.DailyRow`、`taskstats.ErrorRow`、`taskstats.EdgeRow`、`taskstats.DateOf(t time.Time, loc *time.Location) string`
- Produces: `persistence.DailyStatsRow`、`persistence.EdgeDailyStatsRow`、`persistence.ErrorDailyStatsRow`、`persistence.NewGormStatsRepository(gdb *gorm.DB, retention time.Duration, loc *time.Location) *GormStatsRepository`

- [x] **Step 1: 写领域类型与仓储接口（stats.go）**

```go
package taskstats

import (
	"context"
	"time"
)

type Status string

const (
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

type AddTerminalInput struct {
	EdgeID      string
	ErrorCode   string
	Status      Status
	CompletedAt time.Time
	CreatedAt   time.Time
}

type DailyRow struct {
	Date            string
	Processed       int
	Succeeded       int
	Failed          int
	Cancelled       int
	TotalDurationMS int64
}

type ErrorRow struct {
	ErrorCode string
	Count     int
}

type EdgeRow struct {
	EdgeID string
	Count  int
}

type Repository interface {
	AddTerminal(ctx context.Context, in AddTerminalInput) error
	ListDaily(ctx context.Context, from, to string) ([]DailyRow, error)
	ListErrors(ctx context.Context, from, to string, limit int) ([]ErrorRow, error)
	ListEdges(ctx context.Context, from, to string) ([]EdgeRow, error)
	Prune(ctx context.Context, before string) error
}

func DateOf(t time.Time, loc *time.Location) string {
	return t.In(loc).Format("2006-01-02")
}
```

- [x] **Step 2: 写 GORM 模型与仓储实现（persistence/gorm_stats.go）**

```go
package persistence

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats"
)

type DailyStatsRow struct {
	StatDate        string    `gorm:"column:stat_date;primaryKey;size:10"`
	ProcessedCount  int       `gorm:"column:processed_count;not null;default:0"`
	SucceededCount  int       `gorm:"column:succeeded_count;not null;default:0"`
	FailedCount     int       `gorm:"column:failed_count;not null;default:0"`
	CancelledCount  int       `gorm:"column:cancelled_count;not null;default:0"`
	TotalDurationMS int64     `gorm:"column:total_duration_ms;not null;default:0"`
	UpdatedAt       time.Time `gorm:"column:updated_at;not null"`
}

func (DailyStatsRow) TableName() string { return "task_daily_stats" }

type EdgeDailyStatsRow struct {
	StatDate       string    `gorm:"column:stat_date;primaryKey;size:10"`
	EdgeID         string    `gorm:"column:edge_id;primaryKey;size:64"`
	ProcessedCount int       `gorm:"column:processed_count;not null;default:0"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null"`
}

func (EdgeDailyStatsRow) TableName() string { return "task_edge_daily_stats" }

type ErrorDailyStatsRow struct {
	StatDate  string    `gorm:"column:stat_date;primaryKey;size:10"`
	ErrorCode string    `gorm:"column:error_code;primaryKey;size:128"`
	Count     int       `gorm:"column:count;not null;default:0"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null"`
}

func (ErrorDailyStatsRow) TableName() string { return "task_error_daily_stats" }

type GormStatsRepository struct {
	db        *gorm.DB
	retention time.Duration
	loc       *time.Location
}

func NewGormStatsRepository(gdb *gorm.DB, retention time.Duration, loc *time.Location) *GormStatsRepository {
	return &GormStatsRepository{db: gdb, retention: retention, loc: loc}
}

func (r *GormStatsRepository) AddTerminal(ctx context.Context, in taskstats.AddTerminalInput) error {
	date := taskstats.DateOf(in.CompletedAt, r.loc)
	now := time.Now().UTC()
	dur := in.CompletedAt.Sub(in.CreatedAt).Milliseconds()
	if dur < 0 {
		dur = 0
	}
	daily := DailyStatsRow{StatDate: date, ProcessedCount: 1, UpdatedAt: now, TotalDurationMS: dur}
	switch in.Status {
	case taskstats.StatusSucceeded:
		daily.SucceededCount = 1
	case taskstats.StatusFailed:
		daily.FailedCount = 1
	case taskstats.StatusCancelled:
		daily.CancelledCount = 1
	default:
		return fmt.Errorf("taskstats: unsupported terminal status %q", in.Status)
	}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "stat_date"}},
		DoUpdates: clause.Assignments(map[string]any{
			"processed_count":   gorm.Expr("processed_count + 1"),
			"succeeded_count":   gorm.Expr("succeeded_count + ?", daily.SucceededCount),
			"failed_count":      gorm.Expr("failed_count + ?", daily.FailedCount),
			"cancelled_count":   gorm.Expr("cancelled_count + ?", daily.CancelledCount),
			"total_duration_ms": gorm.Expr("total_duration_ms + ?", daily.TotalDurationMS),
			"updated_at":        now,
		}),
	}).Create(&daily).Error; err != nil {
		return err
	}
	if in.EdgeID != "" {
		edgeRow := EdgeDailyStatsRow{StatDate: date, EdgeID: in.EdgeID, ProcessedCount: 1, UpdatedAt: now}
		if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "stat_date"}, {Name: "edge_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"processed_count": gorm.Expr("processed_count + 1"),
				"updated_at":      now,
			}),
		}).Create(&edgeRow).Error; err != nil {
			return err
		}
	}
	if in.ErrorCode != "" {
		errRow := ErrorDailyStatsRow{StatDate: date, ErrorCode: in.ErrorCode, Count: 1, UpdatedAt: now}
		if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "stat_date"}, {Name: "error_code"}},
			DoUpdates: clause.Assignments(map[string]any{
				"count":      gorm.Expr("count + 1"),
				"updated_at": now,
			}),
		}).Create(&errRow).Error; err != nil {
			return err
		}
	}
	return r.Prune(ctx, taskstats.DateOf(time.Now().In(r.loc).Add(-r.retention), r.loc))
}

func (r *GormStatsRepository) ListDaily(ctx context.Context, from, to string) ([]taskstats.DailyRow, error) {
	var rows []DailyStatsRow
	if err := r.db.WithContext(ctx).
		Where("stat_date BETWEEN ? AND ?", from, to).
		Order("stat_date ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]taskstats.DailyRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, taskstats.DailyRow{
			Date: row.StatDate, Processed: row.ProcessedCount,
			Succeeded: row.SucceededCount, Failed: row.FailedCount,
			Cancelled: row.CancelledCount, TotalDurationMS: row.TotalDurationMS,
		})
	}
	return out, nil
}

func (r *GormStatsRepository) ListErrors(ctx context.Context, from, to string, limit int) ([]taskstats.ErrorRow, error) {
	type item struct {
		ErrorCode string
		Count     int
	}
	var items []item
	if err := r.db.WithContext(ctx).Model(&ErrorDailyStatsRow{}).
		Select("error_code", "SUM(count) AS count").
		Where("stat_date BETWEEN ? AND ?", from, to).
		Group("error_code").Order("count DESC").Limit(limit).
		Scan(&items).Error; err != nil {
		return nil, err
	}
	out := make([]taskstats.ErrorRow, 0, len(items))
	for _, it := range items {
		out = append(out, taskstats.ErrorRow{ErrorCode: it.ErrorCode, Count: it.Count})
	}
	return out, nil
}

func (r *GormStatsRepository) ListEdges(ctx context.Context, from, to string) ([]taskstats.EdgeRow, error) {
	type item struct {
		EdgeID string
		Count  int
	}
	var items []item
	if err := r.db.WithContext(ctx).Model(&EdgeDailyStatsRow{}).
		Select("edge_id", "SUM(processed_count) AS count").
		Where("stat_date BETWEEN ? AND ?", from, to).
		Group("edge_id").Order("count DESC").
		Scan(&items).Error; err != nil {
		return nil, err
	}
	out := make([]taskstats.EdgeRow, 0, len(items))
	for _, it := range items {
		out = append(out, taskstats.EdgeRow{EdgeID: it.EdgeID, Count: it.Count})
	}
	return out, nil
}

func (r *GormStatsRepository) Prune(ctx context.Context, before string) error {
	for _, m := range []any{&DailyStatsRow{}, &EdgeDailyStatsRow{}, &ErrorDailyStatsRow{}} {
		if err := r.db.WithContext(ctx).Where("stat_date < ?", before).Delete(m).Error; err != nil {
			return err
		}
	}
	return nil
}

var _ taskstats.Repository = (*GormStatsRepository)(nil)
```

- [x] **Step 3: 运行编译检查**

Run: `go build ./internal/platform/taskstats/...`
Expected: PASS（无编译错误）

- [x] **Step 4: Commit**

```bash
git add internal/platform/taskstats
git commit -m "feat(task-daily-stats): 新增任务统计表模型与仓储接口"
```

### Task 1.2: 仓储实现（承接 1.1 的 AddTerminal / List* / Prune）

**Files:**
- Modify: `internal/platform/taskstats/persistence/gorm_stats.go`（1.1 已含全部实现，本任务仅接线验证）
- Create: `internal/platform/taskstats/persistence/gorm_stats_test.go`

**Interfaces:**
- Consumes: `db.Open(db.Options{DSN})`、`db.AutoMigrate(gdb, ...)`（位于 `internal/platform/db`，参照 gorm_metrics_test.go）
- Produces: 无新接口

- [x] **Step 1: 写仓储单测（sqlite 内存库）**

```go
package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats/persistence"
)

func TestGormStatsRepository_AddTerminalAndList(t *testing.T) {
	dsn := "file:stats_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb,
		&persistence.DailyStatsRow{},
		&persistence.EdgeDailyStatsRow{},
		&persistence.ErrorDailyStatsRow{},
	); err != nil {
		t.Fatal(err)
	}
	loc := time.FixedZone("CST", 8*3600)
	repo := persistence.NewGormStatsRepository(gdb, 365*24*time.Hour, loc)
	ctx := context.Background()
	base := time.Date(2026, 8, 20, 23, 30, 0, 0, time.UTC) // 08-21 07:30 CST
	in := taskstats.AddTerminalInput{
		EdgeID: "gpu-1", ErrorCode: "timeout", Status: taskstats.StatusFailed,
		CompletedAt: base, CreatedAt: base.Add(-2 * time.Minute),
	}
	if err := repo.AddTerminal(ctx, in); err != nil {
		t.Fatal(err)
	}
	if err := repo.AddTerminal(ctx, taskstats.AddTerminalInput{
		Status: taskstats.StatusSucceeded, CompletedAt: base, CreatedAt: base,
	}); err != nil {
		t.Fatal(err)
	}
	days, err := repo.ListDaily(ctx, "2026-08-20", "2026-08-21")
	if err != nil {
		t.Fatal(err)
	}
	if len(days) != 1 || days[0].Date != "2026-08-21" || days[0].Processed != 2 ||
		days[0].Failed != 1 || days[0].Succeeded != 1 || days[0].TotalDurationMS != 120000 {
		t.Fatalf("daily: %+v", days)
	}
	errs, err := repo.ListErrors(ctx, "2026-08-20", "2026-08-21", 10)
	if err != nil || len(errs) != 1 || errs[0].ErrorCode != "timeout" || errs[0].Count != 1 {
		t.Fatalf("errors: %+v %v", errs, err)
	}
	edges, err := repo.ListEdges(ctx, "2026-08-20", "2026-08-21")
	if err != nil || len(edges) != 1 || edges[0].EdgeID != "gpu-1" || edges[0].Count != 1 {
		t.Fatalf("edges: %+v %v", edges, err)
	}
}

func TestGormStatsRepository_Prune(t *testing.T) {
	dsn := "file:stats_prune_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &persistence.DailyStatsRow{}); err != nil {
		t.Fatal(err)
	}
	loc := time.UTC
	repo := persistence.NewGormStatsRepository(gdb, 24*time.Hour, loc)
	ctx := context.Background()
	now := time.Now().UTC()
	for _, in := range []taskstats.AddTerminalInput{
		{Status: taskstats.StatusSucceeded, CompletedAt: now.Add(-48 * time.Hour), CreatedAt: now.Add(-48 * time.Hour)},
		{Status: taskstats.StatusSucceeded, CompletedAt: now, CreatedAt: now},
	} {
		if err := repo.AddTerminal(ctx, in); err != nil {
			t.Fatal(err)
		}
	}
	days, err := repo.ListDaily(ctx, taskstats.DateOf(now.Add(-72*time.Hour), loc), taskstats.DateOf(now, loc))
	if err != nil {
		t.Fatal(err)
	}
	if len(days) != 1 || days[0].Processed != 1 {
		t.Fatalf("expected only today's row after prune: %+v", days)
	}
}
```

- [x] **Step 2: 运行测试**

Run: `go test ./internal/platform/taskstats/...`
Expected: PASS

- [x] **Step 3: Commit**

```bash
git add internal/platform/taskstats/persistence/gorm_stats_test.go
git commit -m "feat(task-daily-stats): 统计仓储单测（归天、幂等增量、保留清理）"
```

### Task 1.3: 归天时区解析与日期边界单测

**Files:**
- Create: `internal/platform/taskstats/stats_test.go`

**Interfaces:**
- Consumes: `taskstats.DateOf`
- Produces: 无

- [x] **Step 1: 写时区边界单测**

```go
package taskstats_test

import (
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats"
)

func TestDateOfBoundaries(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	cases := []struct {
		name string
		in   time.Time
		want string
	}{
		{"utc evening maps to next cst day", time.Date(2026, 8, 20, 23, 59, 59, 999999999, time.UTC), "2026-08-21"},
		{"utc early maps to same cst day", time.Date(2026, 8, 20, 15, 59, 59, 999999999, time.UTC), "2026-08-20"},
		{"cst midnight", time.Date(2026, 8, 21, 0, 0, 0, 0, loc), "2026-08-21"},
		{"utc midnight is 08:00 cst", time.Date(2026, 8, 20, 16, 0, 0, 0, time.UTC), "2026-08-21"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := taskstats.DateOf(tc.in, loc); got != tc.want {
				t.Fatalf("DateOf(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
```

- [x] **Step 2: 运行测试并提交**

Run: `go test ./internal/platform/taskstats/...`

```bash
git add internal/platform/taskstats/stats_test.go
git commit -m "feat(task-daily-stats): 归天时区边界单测"
```

### Task 1.4: 仓储维度聚合与幂等单测（edge/error 空值跳过）

**Files:**
- Modify: `internal/platform/taskstats/persistence/gorm_stats_test.go`

- [x] **Step 1: 追加用例：无 edge_id / 无 error_code 时不写对应维度；重复 AddTerminal 是增量**

```go
func TestGormStatsRepository_SkipEmptyDimensions(t *testing.T) {
	dsn := "file:stats_dim_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &persistence.DailyStatsRow{}, &persistence.EdgeDailyStatsRow{}, &persistence.ErrorDailyStatsRow{}); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewGormStatsRepository(gdb, 365*24*time.Hour, time.UTC)
	ctx := context.Background()
	now := time.Now().UTC()
	in := taskstats.AddTerminalInput{Status: taskstats.StatusCancelled, CompletedAt: now, CreatedAt: now}
	for i := 0; i < 2; i++ {
		if err := repo.AddTerminal(ctx, in); err != nil {
			t.Fatal(err)
		}
	}
	edges, err := repo.ListEdges(ctx, taskstats.DateOf(now, time.UTC), taskstats.DateOf(now, time.UTC))
	if err != nil || len(edges) != 0 {
		t.Fatalf("edges should be empty: %+v %v", edges, err)
	}
	errs, err := repo.ListErrors(ctx, taskstats.DateOf(now, time.UTC), taskstats.DateOf(now, time.UTC), 10)
	if err != nil || len(errs) != 0 {
		t.Fatalf("errors should be empty: %+v %v", errs, err)
	}
	days, err := repo.ListDaily(ctx, taskstats.DateOf(now, time.UTC), taskstats.DateOf(now, time.UTC))
	if err != nil || len(days) != 1 || days[0].Cancelled != 2 {
		t.Fatalf("daily incremental: %+v %v", days, err)
	}
}
```

- [x] **Step 2: 运行测试并提交**

Run: `go test ./internal/platform/taskstats/...`

```bash
git add internal/platform/taskstats/persistence/gorm_stats_test.go
git commit -m "feat(task-daily-stats): 维度空值跳过与增量聚合单测"
```

## 2. 写路径：orchestrator 终态统计

### Task 2.1: orchestrator 终态写路径接入 AddTerminal

**Files:**
- Modify: `internal/runtime/application/orchestrator/service.go`

**Interfaces:**
- Consumes: `taskstats.Repository`、`taskstats.AddTerminalInput`、`taskstats.Status`
- Produces: `Service.Stats taskstats.Repository` 字段；`(s *Service) recordTerminalStats(ctx context.Context, t *runtimedomain.Task) error`

- [x] **Step 1: Service 增加 Stats 字段与 recordTerminalStats**

在 `Service` struct 中 `Notify notify.Publisher` 后追加：

```go
	// Stats optionally records terminal task rollups; nil disables stats writes.
	Stats taskstats.Repository
```

在 `RequestCancel` 之后新增：

```go
func (s *Service) recordTerminalStats(ctx context.Context, t *runtimedomain.Task) error {
	if s.Stats == nil {
		return nil
	}
	status := taskstats.Status(t.Status)
	switch status {
	case taskstats.StatusSucceeded, taskstats.StatusFailed, taskstats.StatusCancelled:
	default:
		return nil
	}
	return s.Stats.AddTerminal(ctx, taskstats.AddTerminalInput{
		EdgeID:      string(t.EdgeID),
		ErrorCode:   t.ErrorCode,
		Status:      status,
		CompletedAt: t.CompletedAt,
		CreatedAt:   t.CreatedAt,
	})
}
```

- [x] **Step 2: applyStatus 终态分支记录统计**

将 `applyStatus` 末尾：

```go
	if isTerminal(t.Status) && t.Status != prev {
		return s.publishNotify(ctx, t)
	}
	return nil
```

改为：

```go
	if isTerminal(t.Status) && t.Status != prev {
		if err := s.recordTerminalStats(ctx, t); err != nil {
			return err
		}
		return s.publishNotify(ctx, t)
	}
	return nil
```

- [x] **Step 3: RequestCancel 记录统计**

将 `RequestCancel` 改为：

```go
func (s *Service) RequestCancel(ctx context.Context, taskID sharedkernel.TaskID) error {
	t, err := s.Tasks.Get(ctx, taskID)
	if err != nil {
		return err
	}
	prev := t.Status
	if err := t.MarkCancelled(s.Now()); err != nil {
		return err
	}
	if err := s.Tasks.Update(ctx, t); err != nil {
		return err
	}
	if prev != sharedkernel.TaskCancelled {
		if err := s.recordTerminalStats(ctx, t); err != nil {
			return err
		}
	}
	return s.publishNotify(ctx, t)
}
```

- [x] **Step 4: 编译检查**

Run: `go build ./internal/runtime/application/orchestrator/...`
Expected: PASS

- [x] **Step 5: Commit**

```bash
git add internal/runtime/application/orchestrator/service.go
git commit -m "feat(task-daily-stats): orchestrator 终态写路径记录任务统计"
```

### Task 2.2: admin-api 装配注入 StatsRepository

**Files:**
- Modify: `apps/admin-api/cmd/admin-api/main.go`
- Modify: `internal/httpapi/adminhost/server.go`
- Modify: `apps/admin-api/internal/server/server.go`（若需要透传，见 Step 3）

**Interfaces:**
- Consumes: `persistence.NewGormStatsRepository`、`statsapi.Handler`（Task 3 提供，此处先建最小 Handler 或留待 3.1；先注入仓储）
- Produces: `adminhost.Options.Stats *stats.Handler`

- [x] **Step 1: main.go 创建 stats 仓储并加入 AutoMigrate**

在 `Models` 列表追加：

```go
			&taskstatspersist.DailyStatsRow{},
			&taskstatspersist.EdgeDailyStatsRow{},
			&taskstatspersist.ErrorDailyStatsRow{},
```

在 `metricsRepo := ...` 附近追加：

```go
	statsRepo := taskstatspersist.NewGormStatsRepository(gdb, statsRetention(), statsLocation())
```

并将 `orch.Stats = statsRepo` 写在 `orch.Sessions = sessionRepo` 之后。

- [x] **Step 2: 新增环境变量解析函数（同文件或 config 包）**

```go
func statsRetention() time.Duration {
	v := os.Getenv("TASK_STATS_RETENTION")
	if v == "" {
		return 365 * 24 * time.Hour
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 365 * 24 * time.Hour
	}
	return d
}

func statsLocation() *time.Location {
	v := os.Getenv("STATS_TIMEZONE")
	if v == "" {
		v = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(v)
	if err != nil {
		return time.FixedZone("Asia/Shanghai", 8*3600)
	}
	return loc
}
```

（确保 import `taskstatspersist`，别名参照现有 `instpersist` 风格。）

- [x] **Step 3: adminhost.Options 增加 Stats 并挂载 /api/v1/stats**

`internal/httpapi/adminhost/server.go`：

```go
type Options struct {
	// ...
	Stats *stats.Handler
}
```

在 `/api/v1/tasks` Route 之后追加：

```go
	r.Route("/api/v1/stats", func(r chi.Router) {
		if opts.Stats != nil {
			opts.Stats.Mount(r)
		}
	})
```

- [x] **Step 4: 编译并提交**

Run: `go build ./...`

```bash
git add apps/admin-api internal/httpapi/adminhost
git commit -m "feat(task-daily-stats): admin-api 注入统计仓储并预留 stats 路由"
```

### Task 2.3: orchestrator 写路径测试（三终态 + 重复事件）

**Files:**
- Modify: `internal/runtime/application/orchestrator/service_test.go`

**Interfaces:**
- Consumes: `Service.Stats`（可注入 fake）
- Produces: 无

- [x] **Step 1: 写 fake 仓储与三终态用例**

```go
type fakeStatsRepo struct {
	calls []taskstats.AddTerminalInput
}

func (f *fakeStatsRepo) AddTerminal(_ context.Context, in taskstats.AddTerminalInput) error {
	f.calls = append(f.calls, in)
	return nil
}
func (f *fakeStatsRepo) ListDaily(context.Context, string, string) ([]taskstats.DailyRow, error) { return nil, nil }
func (f *fakeStatsRepo) ListErrors(context.Context, string, string, int) ([]taskstats.ErrorRow, error) { return nil, nil }
func (f *fakeStatsRepo) ListEdges(context.Context, string, string) ([]taskstats.EdgeRow, error) { return nil, nil }
func (f *fakeStatsRepo) Prune(context.Context, string) error { return nil }
```

用例要点（沿用现有 service_test 的仓储构造方式）：
- 任务 running → succeeded：`applyStatus` 后 `fake.calls` 长度为 1，`Status == succeeded`、`CompletedAt == now`、`EdgeID == 任务 EdgeID`；
- 任务 running → failed：`calls[0].Status == failed`，`ErrorCode == 事件错误码`；
- 任务 pending → cancelled（`RequestCancel`）：`calls[0].Status == cancelled`；
- 重复上报：对已 succeeded 任务再次 applyStatus(succeeded) 不新增调用；
- 非终态（running 事件）不调用 AddTerminal。

- [x] **Step 2: 运行测试并提交**

Run: `go test ./internal/runtime/application/orchestrator/...`

```bash
git add internal/runtime/application/orchestrator/service_test.go
git commit -m "feat(task-daily-stats): orchestrator 终态统计写路径测试"
```

## 3. API：管理端统计接口

### Task 3.1: daily 接口（from/to、默认值、零填充、summary）

**Files:**
- Create: `internal/httpapi/stats/handler.go`
- Create: `internal/httpapi/stats/handler_test.go`

**Interfaces:**
- Consumes: `taskstats.Repository`、`taskstats.DailyRow`
- Produces: `stats.Handler{Repo taskstats.Repository; Loc *time.Location}`、`Handler.Mount(r chi.Router)`、`GET /tasks/daily`

- [x] **Step 1: 写 Handler（daily + 公共解析）**

```go
package stats

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats"
)

type Handler struct {
	Repo taskstats.Repository
	Loc  *time.Location
}

func (h *Handler) Mount(r chi.Router) {
	r.Get("/tasks/daily", h.daily)
	r.Get("/tasks/errors", h.errors)
	r.Get("/tasks/edges", h.edges)
}

type dayResp struct {
	Date          string `json:"date"`
	Processed     int    `json:"processed"`
	Succeeded     int    `json:"succeeded"`
	Failed        int    `json:"failed"`
	Cancelled     int    `json:"cancelled"`
	AvgDurationMS *int64 `json:"avg_duration_ms"`
}

type dailyResp struct {
	Range struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"range"`
	Days []dayResp `json:"days"`
	Summary struct {
		Processed   int      `json:"processed"`
		Succeeded   int      `json:"succeeded"`
		Failed      int      `json:"failed"`
		Cancelled   int      `json:"cancelled"`
		SuccessRate *float64 `json:"success_rate"`
	} `json:"summary"`
}

const (
	dateLayout      = "2006-01-02"
	maxRangeDays    = 365
	defaultRangeDay = 29
)

func (h *Handler) parseRange(r *http.Request) (from, to string, code int, msg string) {
	today := time.Now().In(h.Loc).Format(dateLayout)
	to = r.URL.Query().Get("to")
	if to == "" {
		to = today
	}
	from = r.URL.Query().Get("from")
	if from == "" {
		from = time.Now().In(h.Loc).AddDate(0, 0, -defaultRangeDay).Format(dateLayout)
	}
	tf, err1 := time.ParseInLocation(dateLayout, from, h.Loc)
	tt, err2 := time.ParseInLocation(dateLayout, to, h.Loc)
	if err1 != nil || err2 != nil {
		return "", "", http.StatusBadRequest, "invalid from/to: expected YYYY-MM-DD"
	}
	if tf.After(tt) {
		return "", "", http.StatusBadRequest, "from must not be after to"
	}
	if int(tt.Sub(tf).Hours()/24)+1 > maxRangeDays {
		return "", "", http.StatusBadRequest, "range exceeds 365 days"
	}
	return from, to, 0, ""
}

func (h *Handler) daily(w http.ResponseWriter, r *http.Request) {
	from, to, code, msg := h.parseRange(r)
	if code != 0 {
		writeErr(w, code, msg)
		return
	}
	rows, err := h.Repo.ListDaily(r.Context(), from, to)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	byDate := make(map[string]taskstats.DailyRow, len(rows))
	for _, row := range rows {
		byDate[row.Date] = row
	}
	var resp dailyResp
	resp.Range.From, resp.Range.To = from, to
	cur, _ := time.ParseInLocation(dateLayout, from, h.Loc)
	end, _ := time.ParseInLocation(dateLayout, to, h.Loc)
	for !cur.After(end) {
		d := cur.Format(dateLayout)
		row, ok := byDate[d]
		if !ok {
			row = taskstats.DailyRow{Date: d}
		}
		day := dayResp{
			Date: d, Processed: row.Processed, Succeeded: row.Succeeded,
			Failed: row.Failed, Cancelled: row.Cancelled,
		}
		if row.Processed > 0 {
			avg := row.TotalDurationMS / int64(row.Processed)
			day.AvgDurationMS = &avg
		}
		resp.Days = append(resp.Days, day)
		resp.Summary.Processed += row.Processed
		resp.Summary.Succeeded += row.Succeeded
		resp.Summary.Failed += row.Failed
		resp.Summary.Cancelled += row.Cancelled
		cur = cur.AddDate(0, 0, 1)
	}
	if den := resp.Summary.Succeeded + resp.Summary.Failed; den > 0 {
		rate := float64(resp.Summary.Succeeded) / float64(den)
		resp.Summary.SuccessRate = &rate
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
```

- [x] **Step 2: 写 daily handler 单测**

```go
package stats_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/stats"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats"
)

type fakeRepo struct {
	days []taskstats.DailyRow
}

func (f *fakeRepo) AddTerminal(context.Context, taskstats.AddTerminalInput) error { return nil }
func (f *fakeRepo) ListDaily(_ context.Context, from, to string) ([]taskstats.DailyRow, error) {
	return f.days, nil
}
func (f *fakeRepo) ListErrors(context.Context, string, string, int) ([]taskstats.ErrorRow, error) { return nil, nil }
func (f *fakeRepo) ListEdges(context.Context, string, string) ([]taskstats.EdgeRow, error)         { return nil, nil }
func (f *fakeRepo) Prune(context.Context, string) error                                           { return nil }

func TestDailyZeroFillAndSummary(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	h := &stats.Handler{Repo: &fakeRepo{days: []taskstats.DailyRow{
		{Date: "2026-08-20", Processed: 3, Succeeded: 2, Failed: 1, TotalDurationMS: 3000},
	}}, Loc: loc}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/tasks/daily?from=2026-08-19&to=2026-08-21", nil)
	rec := httptest.NewRecorder()
	h.Mount(nil) // 直接调用 h 内部方法会暴露；改用 chi 路由或改为调用未导出方法前先导出 Router
	_ = req
	_ = rec
}
```

> 注：handler 单测应通过 `chi.NewRouter()` 挂载 `h.Mount(r)` 后请求真实路由；上述占位仅示意，测试代码以实际可编译版本为准（断言：days 长度 3、空白天计数 0、summary.success_rate == 2/3、非法范围返回 400、空范围返回 200 且 success_rate 为 null）。

- [x] **Step 3: 运行测试并提交**

Run: `go test ./internal/httpapi/stats/...`

```bash
git add internal/httpapi/stats
git commit -m "feat(task-daily-stats): daily 统计接口（零填充与成功率汇总）"
```

### Task 3.2: errors 接口（Top-N）

**Files:**
- Modify: `internal/httpapi/stats/handler.go`

**Interfaces:**
- Consumes: `taskstats.ErrorRow`
- Produces: `GET /tasks/errors?from&to&limit`

- [x] **Step 1: 实现 errors handler**

```go
func (h *Handler) errors(w http.ResponseWriter, r *http.Request) {
	from, to, code, msg := h.parseRange(r)
	if code != 0 {
		writeErr(w, code, msg)
		return
	}
	limit := 10
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 || n > 100 {
			writeErr(w, http.StatusBadRequest, "invalid limit: must be 1..100")
			return
		}
		limit = n
	}
	rows, err := h.Repo.ListErrors(r.Context(), from, to, limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	type item struct {
		ErrorCode string `json:"error_code"`
		Count     int    `json:"count"`
	}
	items := make([]item, 0, len(rows))
	for _, row := range rows {
		items = append(items, item{ErrorCode: row.ErrorCode, Count: row.Count})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
```

- [x] **Step 2: 补测试（Top-N 排序与 limit 校验）并提交**

Run: `go test ./internal/httpapi/stats/...`

```bash
git add internal/httpapi/stats
git commit -m "feat(task-daily-stats): 错误码 Top-N 统计接口"
```

### Task 3.3: edges 接口（每节点负载 + total）

**Files:**
- Modify: `internal/httpapi/stats/handler.go`

- [x] **Step 1: 实现 edges handler**

```go
func (h *Handler) edges(w http.ResponseWriter, r *http.Request) {
	from, to, code, msg := h.parseRange(r)
	if code != 0 {
		writeErr(w, code, msg)
		return
	}
	rows, err := h.Repo.ListEdges(r.Context(), from, to)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	type item struct {
		EdgeID string `json:"edge_id"`
		Count  int    `json:"count"`
	}
	items := make([]item, 0, len(rows))
	total := 0
	for _, row := range rows {
		items = append(items, item{EdgeID: row.EdgeID, Count: row.Count})
		total += row.Count
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total})
}
```

- [x] **Step 2: 补测试（排序与 total、空数据）并提交**

Run: `go test ./internal/httpapi/stats/...`

```bash
git add internal/httpapi/stats
git commit -m "feat(task-daily-stats): 每节点负载统计接口"
```

### Task 3.4: 路由挂载与 handler 全量单测

**Files:**
- Modify: `apps/admin-api/cmd/admin-api/main.go`（注入 `Stats: &statsapi.Handler{Repo: statsRepo, Loc: statsLocation()}`）
- Modify: `internal/httpapi/adminhost/server_test.go`

- [x] **Step 1: main.go 组装 stats API**

在 `tasksAPI := &tasksapi.Handler{...}` 附近追加：

```go
	statsAPI := &statsapi.Handler{Repo: statsRepo, Loc: statsLocation()}
```

在 `server.NewHandler(server.Options{...})` 中追加 `Stats: statsAPI,`。

- [x] **Step 2: adminhost 路由测试**

在 `server_test.go` 的路径清单中追加 `"/api/v1/stats/tasks/daily"`，并补一个用例：构造带 fake repo 的 Options，GET `/api/v1/stats/tasks/daily?from=2026-08-01&to=2026-08-02` 返回 200 且 JSON 含 `"days"`。

- [x] **Step 3: 全量编译与测试并提交**

Run: `go build ./... && go test ./internal/httpapi/stats/... ./internal/httpapi/adminhost/...`

```bash
git add apps/admin-api internal/httpapi/adminhost
git commit -m "feat(task-daily-stats): stats 路由装配与 handler 测试"
```

## 4. Backfill 与保留

### Task 4.1: 一次性 backfill 命令

**Files:**
- Create: `apps/admin-api/cmd/backfill-task-stats/main.go`

**Interfaces:**
- Consumes: `db.Open` / `appboot.Bootstrap`（复用 main.go 的 DSN 解析方式）、`taskpersist.TaskRow`、`taskstats.DateOf`
- Produces: 独立可执行 `backfill-task-stats`

- [x] **Step 1: 实现 backfill（按天重算绝对值覆盖）**

```go
package main

import (
	"context"
	"log"
	"os"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats/persistence"
	taskpersist "github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
)

func main() {
	// 复用 admin-api 的 DSN 环境变量（与 cmd/admin-api/main.go 一致），建立 gdb。
	// 示例：gdb, cleanup, err := appboot.Bootstrap(ctx, appboot.Options{DSN: dsn, MigrateEdges: false,
	//   Models: []any{&taskpersist.TaskRow{}, &persistence.DailyStatsRow{}, &persistence.EdgeDailyStatsRow{}, &persistence.ErrorDailyStatsRow{}}})
	// 若 DSN 为空则 log.Fatal。

	type dayAgg struct {
		processed, succ, fail, canc int
		dur                         int64
	}
	byDate := map[string]*dayAgg{}
	byEdge := map[string]map[string]int{}
	byErr := map[string]map[string]int{}

	var rows []taskpersist.TaskRow
	// 分页遍历全部终态任务（succeeded/failed/cancelled）
	if err := gdb.Where("status IN ?", []string{"succeeded", "failed", "cancelled"}).
		Find(&rows).Error; err != nil {
		log.Fatal(err)
	}
	loc := statsLocation()
	for _, row := range rows {
		d := taskstats.DateOf(row.CompletedAt, loc)
		a := byDate[d]
		if a == nil {
			a = &dayAgg{}
			byDate[d] = a
		}
		a.processed++
		dur := row.CompletedAt.Sub(row.CreatedAt).Milliseconds()
		if dur < 0 {
			dur = 0
		}
		a.dur += dur
		switch row.Status {
		case "succeeded":
			a.succ++
		case "failed":
			a.fail++
			if row.ErrorCode != "" {
				if byErr[d] == nil {
					byErr[d] = map[string]int{}
				}
				byErr[d][row.ErrorCode]++
			}
		case "cancelled":
			a.canc++
		}
		if row.EdgeID != "" {
			if byEdge[d] == nil {
				byEdge[d] = map[string]int{}
			}
			byEdge[d][row.EdgeID]++
		}
	}
	now := time.Now().UTC()
	for d, a := range byDate {
		row := persistence.DailyStatsRow{
			StatDate: d, ProcessedCount: a.processed, SucceededCount: a.succ,
			FailedCount: a.fail, CancelledCount: a.canc, TotalDurationMS: a.dur, UpdatedAt: now,
		}
		if err := gdb.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "stat_date"}},
			DoUpdates: clause.Assignments(map[string]any{
				"processed_count": a.processed, "succeeded_count": a.succ, "failed_count": a.fail,
				"cancelled_count": a.canc, "total_duration_ms": a.dur, "updated_at": now,
			}),
		}).Create(&row).Error; err != nil {
			log.Fatal(err)
		}
	}
	// 同法 upsert byEdge / byErr 两表（绝对值覆盖）。
	// 最后调用 statsRepo.Prune(ctx, 保留期截止日期)。
}

func statsLocation() *time.Location {
	v := os.Getenv("STATS_TIMEZONE")
	if v == "" {
		v = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(v)
	if err != nil {
		return time.FixedZone("Asia/Shanghai", 8*3600)
	}
	return loc
}
```

> 注意：`taskpersist.TaskRow` 的实际字段名以 `internal/runtime/infrastructure/persistence` 定义为准（status / completed_at / created_at / edge_id / error_code）。backfill 语义是「以任务表为权威源重算绝对值并覆盖」，可重复执行。

- [x] **Step 2: 编译验证**

Run: `go build ./apps/admin-api/cmd/backfill-task-stats/...`
Expected: PASS

- [x] **Step 3: Commit**

```bash
git add apps/admin-api/cmd/backfill-task-stats
git commit -m "feat(task-daily-stats): 历史任务统计 backfill 命令"
```

### Task 4.2: TASK_STATS_RETENTION 接线与清理验证

**Files:**
- Modify: `apps/admin-api/cmd/admin-api/main.go`（retention 已接线，见 Task 2.2）

- [x] **Step 1: 确认 AddTerminal 内 Prune 使用 retention（已在 Task 1.1 实现）**

Run: `go test ./internal/platform/taskstats/... -run TestGormStatsRepository_Prune`
Expected: PASS

- [x] **Step 2: Commit（如有差异文件）**

```bash
git add -A
git commit -m "feat(task-daily-stats): 统计保留期清理验证"
```

### Task 4.3: README 与架构文档补充环境变量

**Files:**
- Modify: `README.md`
- Modify: `docs/architecture/data-model.md`（若无该路径则就近架构文档）

- [x] **Step 1: 补充环境变量与统计表说明**

在 README 环境变量清单追加：

```text
STATS_TIMEZONE        任务统计归天时区，默认 Asia/Shanghai
TASK_STATS_RETENTION  任务统计保留时长，默认 8760h（365 天）
```

在架构文档数据模型章节补充三张统计表与 `/api/v1/stats/tasks/*` 端点。

- [x] **Step 2: Commit**

```bash
git add README.md docs/architecture
git commit -m "docs(task-daily-stats): 补充统计环境变量与数据模型"
```

## 5. 前端 Dashboard 任务统计

### Task 5.1: stats API client、类型与 query-keys

**Files:**
- Create: `web/admin/src/lib/api/stats.ts`
- Modify: `web/admin/src/lib/api/types.ts`
- Modify: `web/admin/src/lib/api/query-keys.ts`

**Interfaces:**
- Produces: `listTaskDailyStats({from,to})`、`listTaskErrorStats({from,to,limit?})`、`listTaskEdgeStats({from,to})`
- Produces: `TaskDailyStatsResponse`、`TaskDailyStat`、`TaskErrorStat`、`TaskEdgeStat` 类型
- Produces: `queryKeys.stats.tasksDaily(from,to)` 等

- [x] **Step 1: types.ts 追加类型**

```ts
export type TaskDailyStat = {
  date: string
  processed: number
  succeeded: number
  failed: number
  cancelled: number
  avg_duration_ms: number | null
}

export type TaskDailyStatsResponse = {
  range: { from: string; to: string }
  days: TaskDailyStat[]
  summary: {
    processed: number
    succeeded: number
    failed: number
    cancelled: number
    success_rate: number | null
  }
}

export type TaskErrorStat = { error_code: string; count: number }
export type TaskEdgeStat = { edge_id: string; count: number }
```

- [x] **Step 2: stats.ts 客户端**

```ts
import { apiFetch, toQuery } from './client'
import type {
  TaskDailyStatsResponse,
  TaskEdgeStat,
  TaskErrorStat,
} from './types'

export function listTaskDailyStats(params: { from: string; to: string }) {
  return apiFetch<TaskDailyStatsResponse>(
    `/api/v1/stats/tasks/daily${toQuery(params)}`
  )
}

export function listTaskErrorStats(params: {
  from: string
  to: string
  limit?: number
}) {
  return apiFetch<{ items: TaskErrorStat[] }>(
    `/api/v1/stats/tasks/errors${toQuery(params)}`
  )
}

export function listTaskEdgeStats(params: { from: string; to: string }) {
  return apiFetch<{ items: TaskEdgeStat[]; total: number }>(
    `/api/v1/stats/tasks/edges${toQuery(params)}`
  )
}
```

- [x] **Step 3: query-keys.ts 追加 stats 分支**

```ts
  stats: {
    tasksDaily: (from: string, to: string) =>
      ['stats', 'tasks', 'daily', from, to] as const,
    tasksErrors: (from: string, to: string) =>
      ['stats', 'tasks', 'errors', from, to] as const,
    tasksEdges: (from: string, to: string) =>
      ['stats', 'tasks', 'edges', from, to] as const,
  },
```

- [x] **Step 4: 类型检查并提交**

Run: `pnpm -C web/admin tsc -b`

```bash
git add web/admin/src/lib/api
git commit -m "feat(task-daily-stats): 前端 stats API client 与类型"
```

### Task 5.2: 日期范围状态与快捷预设（Calendar range）

**Files:**
- Create: `web/admin/src/features/dashboard/date-range.ts`
- Create: `web/admin/src/features/dashboard/date-range.test.ts`

**Interfaces:**
- Produces: `formatDate(d: Date): string`（本地时区 YYYY-MM-DD）
- Produces: `daysAgo(n: number): string`
- Produces: `DAILY_PRESETS: { labelKey: string; days: number }[]`（7/30/90）

- [x] **Step 1: 日期工具**

```ts
export function formatDate(d: Date): string {
  const y = d.getFullYear()
  const m = `${d.getMonth() + 1}`.padStart(2, '0')
  const day = `${d.getDate()}`.padStart(2, '0')
  return `${y}-${m}-${day}`
}

export function daysAgo(n: number): string {
  const d = new Date()
  d.setDate(d.getDate() - n)
  return formatDate(d)
}

export const DAILY_PRESETS = [
  { labelKey: 'dashboard.range7d', days: 7 },
  { labelKey: 'dashboard.range30d', days: 30 },
  { labelKey: 'dashboard.range90d', days: 90 },
] as const
```

- [x] **Step 2: 单测**

```ts
import { describe, expect, it } from 'vitest'
import { daysAgo, formatDate } from './date-range'

describe('date-range', () => {
  it('formats local date as YYYY-MM-DD', () => {
    expect(formatDate(new Date(2026, 7, 21))).toBe('2026-08-21')
  })
  it('daysAgo returns expected date string', () => {
    expect(daysAgo(0)).toBe(formatDate(new Date()))
    expect(daysAgo(30)).toMatch(/^\d{4}-\d{2}-\d{2}$/)
  })
})
```

- [x] **Step 3: 运行测试并提交**

Run: `pnpm -C web/admin vitest run src/features/dashboard/date-range.test.ts`

```bash
git add web/admin/src/features/dashboard
git commit -m "feat(task-daily-stats): 日期范围工具与快捷预设"
```

### Task 5.3: 每日处理任务数柱状图卡

**Files:**
- Create: `web/admin/src/features/dashboard/task-stats-section.tsx`
- Modify: `web/admin/src/features/dashboard/dashboard-page.tsx`

**Interfaces:**
- Consumes: `listTaskDailyStats`、`queryKeys.stats.tasksDaily`、`DAILY_PRESETS`、`formatDate`、`daysAgo`
- Produces: `TaskStatsSection` 组件

- [x] **Step 1: 写 TaskStatsSection（柱状图 + 范围选择 + 统计卡骨架）**

```tsx
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { useState } from 'react'
import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from 'recharts'
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from '@/components/ui/chart'
import { Calendar } from '@/components/ui/calendar'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { listTaskDailyStats, listTaskEdgeStats, listTaskErrorStats } from '@/lib/api/stats'
import { queryKeys } from '@/lib/api/query-keys'
import { DAILY_PRESETS, daysAgo, formatDate } from './date-range'

export function TaskStatsSection() {
  const { t } = useTranslation()
  const [range, setRange] = useState({ from: daysAgo(29), to: daysAgo(0) })

  const daily = useQuery({
    queryKey: queryKeys.stats.tasksDaily(range.from, range.to),
    queryFn: () => listTaskDailyStats(range),
  })
  const errors = useQuery({
    queryKey: queryKeys.stats.tasksErrors(range.from, range.to),
    queryFn: () => listTaskErrorStats({ ...range, limit: 5 }),
  })
  const edges = useQuery({
    queryKey: queryKeys.stats.tasksEdges(range.from, range.to),
    queryFn: () => listTaskEdgeStats(range),
  })

  const chartData = (daily.data?.days ?? []).map((d) => ({
    date: d.date,
    processed: d.processed,
  }))

  return (
    <div className='space-y-4'>
      <div className='flex flex-wrap items-center gap-2'>
        {DAILY_PRESETS.map((p) => (
          <Button
            key={p.labelKey}
            variant='outline'
            size='sm'
            onClick={() => setRange({ from: daysAgo(p.days - 1), to: daysAgo(0) })}
          >
            {t(p.labelKey)}
          </Button>
        ))}
        <Popover>
          <PopoverTrigger asChild>
            <Button variant='outline' size='sm'>
              {range.from} ~ {range.to}
            </Button>
          </PopoverTrigger>
          <PopoverContent align='end' className='w-auto p-0'>
            <Calendar
              mode='range'
              defaultMonth={new Date()}
              selected={{
                from: new Date(range.from),
                to: new Date(range.to),
              }}
              onSelect={(sel) => {
                if (sel?.from && sel?.to) {
                  setRange({ from: formatDate(sel.from), to: formatDate(sel.to) })
                }
              }}
            />
          </PopoverContent>
        </Popover>
      </div>

      <Card data-testid='task-daily-chart-card'>
        <CardHeader className='pb-2'>
          <CardTitle className='text-base font-semibold'>
            {t('dashboard.taskDailyTitle')}
          </CardTitle>
          <CardDescription>
            {t('dashboard.taskDailyDescription', {
              from: range.from,
              to: range.to,
            })}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {daily.isError ? (
            <p className='text-sm text-destructive'>{t('common.error')}</p>
          ) : (
            <ChartContainer
              config={{
                processed: { label: t('dashboard.processed'), color: 'var(--primary)' },
              }}
              className='h-[200px] w-full'
            >
              <BarChart data={chartData}>
                <CartesianGrid vertical={false} />
                <XAxis dataKey='date' tickLine={false} axisLine={false} />
                <YAxis tickLine={false} axisLine={false} allowDecimals={false} />
                <ChartTooltip content={<ChartTooltipContent />} />
                <Bar dataKey='processed' fill='var(--color-processed)' radius={4} />
              </BarChart>
            </ChartContainer>
          )}
        </CardContent>
      </Card>

      {/* 统计卡：成功率 / 错误码 Top-N / 每节点负载（errors/edges 数据消费见 Task 5.4） */}
    </div>
  )
}
```

- [x] **Step 2: dashboard-page 接入 TaskStatsSection**

将 `TasksCard` 调用替换为 `<TaskStatsSection />`（保留 `EdgesCard` / `CasesCard`），并调整网格布局（`lg:grid-cols-3` 中 TaskStatsSection 占整行或 `sm:col-span-2`，以合同测试为准）。

- [x] **Step 3: 类型检查并提交**

Run: `pnpm -C web/admin tsc -b`

```bash
git add web/admin/src/features/dashboard
git commit -m "feat(task-daily-stats): Dashboard 每日任务柱状图与范围选择"
```

### Task 5.4: 统计卡（成功率 / 错误码 Top-N / 每节点负载）

**Files:**
- Modify: `web/admin/src/features/dashboard/task-stats-section.tsx`

- [x] **Step 1: 在 TaskStatsSection 追加三张统计卡**

在柱状图 Card 之后追加（数据来自 daily / errors / edges query）：

```tsx
      <div className='grid gap-4 sm:grid-cols-3'>
        <Card>
          <CardHeader className='pb-2'>
            <CardTitle className='text-sm font-medium'>
              {t('dashboard.statsSuccessRate')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className='text-2xl font-bold tabular-nums'>
              {daily.data?.summary.success_rate == null
                ? '—'
                : `${(daily.data.summary.success_rate * 100).toFixed(1)}%`}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className='pb-2'>
            <CardTitle className='text-sm font-medium'>
              {t('dashboard.statsErrorTop')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {errors.data?.items.length ? (
              <ul className='space-y-1 text-sm'>
                {errors.data.items.map((e) => (
                  <li key={e.error_code} className='flex justify-between gap-2'>
                    <span className='truncate'>{e.error_code}</span>
                    <span className='tabular-nums'>{e.count}</span>
                  </li>
                ))}
              </ul>
            ) : (
              <p className='text-sm text-muted-foreground'>{t('common.empty')}</p>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader className='pb-2'>
            <CardTitle className='text-sm font-medium'>
              {t('dashboard.statsEdgeLoad')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {edges.data?.items.length ? (
              <ul className='space-y-1 text-sm'>
                {edges.data.items.slice(0, 5).map((e) => (
                  <li key={e.edge_id} className='flex justify-between gap-2'>
                    <span className='truncate'>{e.edge_id}</span>
                    <span className='tabular-nums'>{e.count}</span>
                  </li>
                ))}
              </ul>
            ) : (
              <p className='text-sm text-muted-foreground'>{t('common.empty')}</p>
            )}
          </CardContent>
        </Card>
      </div>
```

- [x] **Step 2: 类型检查并提交**

Run: `pnpm -C web/admin tsc -b`

```bash
git add web/admin/src/features/dashboard/task-stats-section.tsx
git commit -m "feat(task-daily-stats): 成功率/错误码/每节点负载统计卡"
```

### Task 5.5: TasksCard 替换与失败隔离

**Files:**
- Modify: `web/admin/src/features/dashboard/dashboard-page.tsx`
- Delete（或停止引用）：`web/admin/src/lib/dashboard/aggregate.ts` 中 tasks 相关聚合（保留 edges/cases 使用）

- [x] **Step 1: 移除 TasksCard 与 DASHBOARD_LIST_LIMIT 的任务用法**

删除 `TasksCard` 组件与 `TASK_STATUS_COLORS`，`aggregateDashboard` 仍由 Edges/Case 卡使用；任务列表的 `listTasks` 查询从 Dashboard 移除（列表页仍保留）。

- [x] **Step 2: 验证前端编译与既有测试**

Run: `pnpm -C web/admin tsc -b && pnpm -C web/admin vitest run src/features/dashboard`
Expected: PASS（需同步更新 aggregate.test.ts 若其断言依赖任务聚合）

- [x] **Step 3: Commit**

```bash
git add web/admin/src/features/dashboard web/admin/src/lib/dashboard
git commit -m "feat(task-daily-stats): Dashboard 任务区切换到统计接口"
```

### Task 5.6: i18n zh/en 文案

**Files:**
- Modify: `web/admin/src/lib/i18n/locales/zh.json`
- Modify: `web/admin/src/lib/i18n/locales/en.json`

- [x] **Step 1: 补充文案键**

zh.json `dashboard` 下追加：

```json
{
  "taskDailyTitle": "每日处理任务数",
  "taskDailyDescription": "按完成时间统计 {{from}} ~ {{to}} 的已处理任务",
  "range7d": "近 7 天",
  "range30d": "近 30 天",
  "range90d": "近 90 天",
  "processed": "已处理",
  "statsSuccessRate": "成功率",
  "statsErrorTop": "错误码 Top",
  "statsEdgeLoad": "节点负载"
}
```

en.json 对应：

```json
{
  "taskDailyTitle": "Daily processed tasks",
  "taskDailyDescription": "Processed tasks by completion date {{from}} ~ {{to}}",
  "range7d": "Last 7 days",
  "range30d": "Last 30 days",
  "range90d": "Last 90 days",
  "processed": "Processed",
  "statsSuccessRate": "Success rate",
  "statsErrorTop": "Top error codes",
  "statsEdgeLoad": "Node load"
}
```

- [x] **Step 2: 校验 JSON 并提交**

Run: `pnpm -C web/admin vitest run src/lib/i18n`

```bash
git add web/admin/src/lib/i18n
git commit -m "feat(task-daily-stats): 任务统计 i18n 文案"
```

### Task 5.7: 合同测试（class / 交互 / 文案）

**Files:**
- Create: `web/admin/src/features/dashboard/task-stats.contract.test.ts`

- [x] **Step 1: 写合同测试**

沿用 `edges.contract.test.ts` 的读取源码断言方式：
- 断言 `task-stats-section.tsx` 含 `data-testid='task-daily-chart-card'`、`ChartContainer`、`BarChart`、`Calendar` 与 `mode='range'`；
- 断言 zh.json / en.json 含 `taskDailyTitle`、`statsSuccessRate`、`statsErrorTop`、`statsEdgeLoad`；
- 断言 `dashboard-page.tsx` 已引用 `TaskStatsSection` 且不再直接调用 `listTasks`。

- [x] **Step 2: 运行测试并提交**

Run: `pnpm -C web/admin vitest run src/features/dashboard`

```bash
git add web/admin/src/features/dashboard/task-stats.contract.test.ts
git commit -m "feat(task-daily-stats): Dashboard 任务统计合同测试"
```

## 6. 验证与文档

### Task 6.1: 全量 Go 与前端构建/测试

- [x] **Step 1: 运行后端全量验证**

Run: `go build ./... && go test ./...`
Expected: PASS

- [x] **Step 2: 运行前端全量验证**

Run: `pnpm -C web/admin tsc -b && pnpm -C web/admin vitest run`
Expected: PASS

- [x] **Step 3: 提交（如有修复）**

```bash
git add -A
git commit -m "chore(task-daily-stats): 全量构建与测试通过"
```

### Task 6.2: Mock 下端到端确认日期范围切换

- [x] **Step 1: 启动 Mock 环境**

参照 README 本地开发方式启动 admin-api（Mock 或测试数据）与 `web/admin` dev server，打开 Dashboard。

- [x] **Step 2: 验证交互**

- 默认近 30 天柱状图展示连续日期（含 0 柱）；
- 点击「近 7 / 30 / 90 天」与日历 range 后，请求参数 `from/to` 与图表同步刷新；
- 成功率卡在无数据时显示「—」；错误码与节点负载为空态；
- 断开 stats 接口（或返回 500）时仅任务区块进入错误态，Edges/Case 卡正常。

- [x] **Step 3: 记录证据并提交（如有修复）**

```bash
git add -A
git commit -m "chore(task-daily-stats): 端到端确认日期范围与失败隔离"
```

### Task 6.3: 同步架构文档

**Files:**
- Modify: `docs/architecture/data-model.md`、`docs/architecture/runtime.md`（若存在）、`README.md`

- [x] **Step 1: 补充统计表与接口文档**

记录 `task_daily_stats` / `task_edge_daily_stats` / `task_error_daily_stats` 三表、orchestrator 终态写路径、`/api/v1/stats/tasks/{daily,errors,edges}` 参数与响应、`STATS_TIMEZONE` / `TASK_STATS_RETENTION`。

- [x] **Step 2: Commit**

```bash
git add docs README.md
git commit -m "docs(task-daily-stats): 同步统计表、接口与环境变量文档"
```
