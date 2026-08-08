### Task 3: conversation — 管理端 Session List

**Files:**
- Modify: `internal/conversation/domain/service.go`（`Repository` 接口所在文件）
- Modify: `internal/conversation/domain/session.go`（增加 `ErrNotFound`，与 `ErrNoActiveSession` 并存）
- Modify: `internal/conversation/infrastructure/persistence/gorm_session.go`
- Modify: `internal/conversation/domain/service.go` 内 `MemoryRepository` 实现 `List`
- Create/Modify: `internal/conversation/infrastructure/persistence/gorm_session_test.go`（若无则新建）

**Interfaces:**
- Produces:
  - `domain.ListQuery`：`Q string`；`UserID string`；`ChatID *int64`（或 `sharedkernel.ChatID` 零值表示未设）；`Status Status`；`CreatedFrom, CreatedTo *time.Time`；`Limit, Offset int`
  - `Repository.List(ctx, q ListQuery) ([]*Session, error)`
  - `GetByID` 缺失时：持久化层改为返回 `domain.ErrNotFound`（新建 sentinel）；`GetActiveByChat` 仍用 `ErrNoActiveSession`。更新所有依赖 `errors.Is(..., ErrNoActiveSession)` 的 GetByID 断言

**说明：** 管理详情需要区分「无活跃会话」与「ID 不存在」。`GetByID` 用 `ErrNotFound`；活跃查询保持原语义。

- [ ] **Step 1: 写失败测试**

```go
func TestSessionListFilters(t *testing.T) {
	// migrate SessionRow; Save two sessions with不同 user_id/chat_id/status
	// List by user_id → 1
	// List by status=submitted → 匹配
	// List by chat_id → 1
	// Q 匹配 case_id 或 id 前缀
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/conversation/... -run TestSessionListFilters -count=1`

Expected: FAIL

- [ ] **Step 3: 最小实现**

GORM List：按字段 `Where`；`Q` → `id LIKE ? OR case_id LIKE ? OR user_id LIKE ?`；`Order("updated_at desc")`。

`GetByID`：

```go
if errors.Is(err, gorm.ErrRecordNotFound) {
	return nil, domain.ErrNotFound
}
```

`MemoryRepository.List`：线性过滤，供 handler 单测可选用。

- [ ] **Step 4: 测试通过**

Run: `go test ./internal/conversation/... -count=1`

- [ ] **Step 5: Commit**

```bash
git add internal/conversation/
git commit -m "$(cat <<'EOF'
feat(conversation): add admin Session List and ErrNotFound for GetByID

EOF
)"
```

---

