### Task 7: httpapi/sessions — Session 只读 HTTP

**Files:**
- Create: `internal/httpapi/sessions/handler.go`
- Create: `internal/httpapi/sessions/handler_test.go`

**Interfaces:**
- Consumes: `conversation/domain.Repository`
- Produces: `Handler{Repo}`；`Mount`：`GET /`、`GET /{id}`
- Query：`q, user_id, chat_id, status, created_from, created_to, limit, offset`
- DTO：含 `id, user_id, chat_id, case_id, status, current_input_index, input_keys, draft, created_at, updated_at`（排障所需）
- **禁止** 通用写路由

- [ ] **Step 1: 写失败测试**

造两条 Session（不同 status/user）；列表过滤；详情含 draft；缺失 → 404（`ErrNotFound`）。

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/httpapi/sessions/ -count=1`

- [ ] **Step 3: 实现**

与 users 同风格；`status` 原样传入 `domain.Status`。

- [ ] **Step 4: 测试通过**

Run: `go test ./internal/httpapi/sessions/ -count=1`

- [ ] **Step 5: Commit**

```bash
git add internal/httpapi/sessions/
git commit -m "$(cat <<'EOF'
feat(httpapi): add read-only sessions admin handlers

EOF
)"
```

---

