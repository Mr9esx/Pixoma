### Task 2: identity — 管理端 User List

**Files:**
- Modify: `internal/identity/domain/repository.go`
- Modify: `internal/identity/infrastructure/persistence/gorm_user.go`
- Modify: `internal/identity/infrastructure/persistence/gorm_user_test.go`

**Interfaces:**
- Produces:
  - `domain.ListQuery`：`Q string`；`TgUserID *int64`；`CreatedFrom, CreatedTo *time.Time`；`Limit, Offset int`
  - `Repository.List(ctx context.Context, q ListQuery) ([]*User, error)`
  - 过滤：`tg_user_id` 精确；`Q` 模糊匹配 `id`/`username`/`first_name`/`last_name`（参数化 `LIKE`）

- [ ] **Step 1: 写失败测试**

```go
func TestUserListFilters(t *testing.T) {
	// open migrate users table; Upsert two users with distinct username/tg ids
	// List with TgUserID exact → 1
	// List with Q matching username → 1
	// List with Limit 1 Offset 0 → len==1
}
```

具体断言用既有 `NewUserRepository` + `UpsertByTgUserID` 造数。

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/identity/infrastructure/persistence/ -run TestUserListFilters -count=1`

Expected: FAIL（`List` undefined）

- [ ] **Step 3: 最小实现**

```go
// domain/repository.go
type ListQuery struct {
	Q           string
	TgUserID    *int64
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	Limit       int
	Offset      int
}

type Repository interface {
	UpsertByTgUserID(ctx context.Context, in UpsertFrom) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	List(ctx context.Context, q ListQuery) ([]*User, error)
}
```

GORM：`Order("created_at desc")`；`TgUserID != nil` → `Where("tg_user_id = ?", *q.TgUserID)`；`Q` → `id/username/first_name/last_name LIKE ?`。

- [ ] **Step 4: 测试通过**

Run: `go test ./internal/identity/... -count=1`

- [ ] **Step 5: Commit**

```bash
git add internal/identity/
git commit -m "$(cat <<'EOF'
feat(identity): add admin List query with filters

EOF
)"
```

---

