### Task 1: catalog — Enable + ListQuery 过滤扩展

**Files:**
- Modify: `internal/catalog/domain/repository.go`
- Modify: `internal/catalog/infrastructure/persistence/gorm_repository.go`
- Modify: `internal/catalog/infrastructure/persistence/gorm_repository_test.go`
- Modify（若有 Memory stub）: 任何实现 `domain.Repository` 的测试假对象（如 `internal/runtime/infrastructure/actuator/snapshot_test.go` 的 `memCases`）补 `Enable`

**Interfaces:**
- Produces:
  - `domain.ListQuery` 字段：`Tag, MenuKey, Category string`；`Enabled *bool`；`Q string`；`CreatedFrom, CreatedTo *time.Time`；`Limit, Offset int`
  - `Repository.Enable(ctx context.Context, id sharedkernel.CaseID) error`（不存在 → `ErrNotFound`）
  - `Repository.List` 对 `Q`：在 `id`/`name` 上参数化 `LIKE`；时间窗过滤 `created_at`

- [ ] **Step 1: 写失败测试**

在 `gorm_repository_test.go` 追加：

```go
func TestEnableAndListFilters(t *testing.T) {
	repo := openTestDB(t)
	ctx := context.Background()
	c1 := sampleCase("alpha-case", "t1")
	c1.Document.Name = "Alpha Workflow"
	c1.Document.Categories = []string{"gen"}
	_ = repo.Create(ctx, c1)
	c2 := sampleCase("beta-case", "t2")
	c2.Document.Name = "Beta Other"
	_ = repo.Create(ctx, c2)

	if err := repo.Disable(ctx, "alpha-case"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Enable(ctx, "alpha-case"); err != nil {
		t.Fatal(err)
	}
	got, _ := repo.Get(ctx, "alpha-case")
	if !got.Enabled {
		t.Fatal("expected enabled after Enable")
	}
	if err := repo.Enable(ctx, "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}

	list, err := repo.List(ctx, domain.ListQuery{Q: "Alpha", Limit: 10})
	if err != nil || len(list) != 1 || list[0].Document.ID != "alpha-case" {
		t.Fatalf("q filter: err=%v list=%+v", err, list)
	}
	en := true
	list, err = repo.List(ctx, domain.ListQuery{Enabled: &en, Category: "gen", Limit: 10})
	if err != nil || len(list) != 1 {
		t.Fatalf("enabled+category: err=%v n=%d", err, len(list))
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/catalog/infrastructure/persistence/ -run TestEnableAndListFilters -count=1`

Expected: FAIL（`Enable` undefined 和/或 `Q` 字段不存在）

- [ ] **Step 3: 最小实现**

1. `ListQuery` 增加 `Q string`、`CreatedFrom/CreatedTo *time.Time`（需 `import "time"`）。
2. 接口增加 `Enable`。
3. `Enable` 镜像 `Disable`，`Update("enabled", true)`，`RowsAffected==0` → `ErrNotFound`。
4. `List` 中：

```go
if q.Q != "" {
	like := "%" + q.Q + "%"
	tx = tx.Where("id LIKE ? OR name LIKE ?", like, like)
}
if q.CreatedFrom != nil {
	tx = tx.Where("created_at >= ?", *q.CreatedFrom)
}
if q.CreatedTo != nil {
	tx = tx.Where("created_at <= ?", *q.CreatedTo)
}
```

（既有 Tag/Category 过滤保持参数化；**不要**把原始 `q.Tag` 拼进无占位符字符串以外的危险拼接——现有 `LIKE "%\""+tag+"\"%"` 对内部 tag 可接受，但 `Q` 必须用 `?`。）

5. 所有 `Repository` 假实现补空 `Enable`。

- [ ] **Step 4: 测试通过**

Run: `go test ./internal/catalog/... -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/catalog/
git commit -m "$(cat <<'EOF'
feat(catalog): add Enable and extend ListQuery filters for admin

EOF
)"
```

---
