### Task 6: httpapi/users — User 只读 HTTP

**Files:**
- Create: `internal/httpapi/users/handler.go`
- Create: `internal/httpapi/users/handler_test.go`

**Interfaces:**
- Consumes: `identity/domain.Repository`
- Produces: `Handler{Repo}`；`Mount`：`GET /`、`GET /{id}`
- Query：`q, tg_user_id, created_from, created_to, limit, offset`
- **禁止** POST/PATCH/DELETE 路由
- DTO 字段：`id, tg_user_id, username, first_name, last_name, language_code, last_seen_at, created_at, updated_at`

- [ ] **Step 1: 写失败测试**

```go
func TestUsersHandler_ListGetReadOnly(t *testing.T) {
	// Upsert user; GET /api/v1/users → 含该用户
	// GET /api/v1/users/{id} → 200
	// GET missing → 404
	// 路由上 Method POST /api/v1/users → 405 或 404（未注册即可）
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/httpapi/users/ -count=1`

- [ ] **Step 3: 实现**

`tg_user_id` 用 `strconv.ParseInt`；非法 → 400。`ErrNotFound` → 404。

- [ ] **Step 4: 测试通过**

Run: `go test ./internal/httpapi/users/ -count=1`

- [ ] **Step 5: Commit**

```bash
git add internal/httpapi/users/
git commit -m "$(cat <<'EOF'
feat(httpapi): add read-only users admin handlers

EOF
)"
```

---

