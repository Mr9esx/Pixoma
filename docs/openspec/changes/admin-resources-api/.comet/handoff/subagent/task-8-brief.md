### Task 8: httpapi/tasks — Task 列表/详情/取消

**Files:**
- Create: `internal/httpapi/tasks/handler.go`
- Create: `internal/httpapi/tasks/handler_test.go`

**Interfaces:**
- Consumes: `runtimedomain.TaskRepository`；取消端口：

```go
type Canceller interface {
	RequestCancel(ctx context.Context, taskID sharedkernel.TaskID) error
}
```

- Produces: `Handler{Tasks TaskRepository; Cancel Canceller}`；`Mount`：
  - `GET /`
  - `GET /{id}`
  - `POST /{id}/cancel`
- Query：`q, status, instance_id, chat_id, session_id, case_id, created_from, created_to, limit, offset`
- **禁止** 任何 ConfirmRun / `POST /` 创建 Task
- 取消映射：`ErrTaskNotFound`→404；`ErrCancelNotAllowed`→409；成功后 `Get` 再返回 DTO（200）

- [ ] **Step 1: 写失败测试**

```go
func TestTasksHandler_ListGetCancel(t *testing.T) {
	tasks := runtimedomain.NewMemoryTaskRepository()
	orch := orchestrator.New(tasks, nil, nil, notify.Nop{})
	h := &tasksapi.Handler{Tasks: tasks, Cancel: orch}
	// create pending + succeeded
	// GET list ?status=pending
	// POST /{pending}/cancel → 200 status=cancelled
	// POST /{succeeded}/cancel → 409
	// POST /missing/cancel → 404
	// 确认没有 POST / 创建路由（405/404）
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/httpapi/tasks/ -count=1`

- [ ] **Step 3: 实现**

DTO 对齐 comfyinstances 的 `taskDTO`，并加上 `chat_id`（若有）。取消**只**调用 `h.Cancel.RequestCancel`，不要在 handler 里直接 `MarkCancelled`。

- [ ] **Step 4: 测试通过**

Run: `go test ./internal/httpapi/tasks/ -count=1`

- [ ] **Step 5: Commit**

```bash
git add internal/httpapi/tasks/
git commit -m "$(cat <<'EOF'
feat(httpapi): add tasks list/get and RequestCancel endpoint

EOF
)"
```

---

