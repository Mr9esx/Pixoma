### Task 4: runtime — Admin List + cancel 可被 admin 复用

**Files:**
- Modify: `internal/runtime/domain/task_repository.go`
- Modify: `internal/runtime/infrastructure/persistence/gorm_task.go`
- Modify: `internal/runtime/infrastructure/persistence/gorm_task_test.go`
- Create: `internal/platform/notify/nop.go`
- Modify: `internal/runtime/application/orchestrator/service_test.go`（若需覆盖 RequestCancel + Nop；可复用既有 cancel 测试）

**Interfaces:**
- Produces:
  - `domain.AdminListQuery`：
    - `Q string`
    - `Status sharedkernel.TaskStatus`
    - `InstanceID sharedkernel.InstanceID`
    - `ChatID sharedkernel.ChatID`（0 = 未过滤）
    - `SessionID sharedkernel.SessionID`
    - `CaseID sharedkernel.CaseID`
    - `CreatedFrom, CreatedTo *time.Time`
    - `Limit, Offset int`
  - `TaskRepository.List(ctx context.Context, q AdminListQuery) ([]*Task, error)`
  - `notify.Nop`：`type Nop struct{}`；`func (Nop) Publish(context.Context, sharedkernel.UserNotify) error { return nil }`
  - Cancel：**不新写取消逻辑**；admin 侧将注入 `orchestrator.Service{Tasks, Notify: notify.Nop{}, Sessions: sessionRepo, Now: ...}` 并调用 `RequestCancel`
  - HTTP 映射约定（本任务在仓储/通知就绪；handler 任务实现）：
    - `ErrTaskNotFound` → 404
    - `ErrCancelNotAllowed` → 409
    - 成功 → 200 + task DTO

**chat_id 过滤：** 与既有 `ListByChat` 相同，`JOIN sessions ON tasks.session_id = sessions.id` + `sessions.chat_id = ?`。其它过滤在 `tasks` 列上。

- [ ] **Step 1: 写失败测试**

在 `gorm_task_test.go`：

```go
func TestTaskAdminListFilters(t *testing.T) {
	// migrate TaskRow + SessionRow；插入 session(chat=42) + tasks
	// List AdminListQuery{SessionID: ...} 
	// List AdminListQuery{ChatID: 42} 经 JOIN
	// List AdminListQuery{Status: pending, CaseID: ...}
	// List AdminListQuery{Q: 前缀}
}
```

另加/确认：

```go
func TestRequestCancelViaOrchestrator(t *testing.T) {
	tasks := domain.NewMemoryTaskRepository()
	// create pending task
	orch := orchestrator.New(tasks, nil, nil, notify.Nop{})
	if err := orch.RequestCancel(ctx, "t1"); err != nil {
		t.Fatal(err)
	}
	// create succeeded task → errors.Is(ErrCancelNotAllowed)
}
```

（若 `New` 对 nil Registry 不安全，测试里传最小假对象；`RequestCancel` 路径不碰 Instances/Dispatch。）

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/runtime/infrastructure/persistence/ -run TestTaskAdminListFilters -count=1`

Expected: FAIL

- [ ] **Step 3: 最小实现**

1. 接口加 `List`；`MemoryTaskRepository.List` 内存过滤（含 ChatID 精确匹配 `t.ChatID`，无 session JOIN——内存测可设 `ChatID`）。
2. GORM `List`：有 `ChatID != 0` 时用 Table+Joins；否则 `Model(&TaskRow{})`；全部条件用 `?`。
3. 新增 `notify.Nop`。

- [ ] **Step 4: 测试通过**

Run: `go test ./internal/runtime/... ./internal/platform/notify/ -count=1`

- [ ] **Step 5: Commit**

```bash
git add internal/runtime/ internal/platform/notify/
git commit -m "$(cat <<'EOF'
feat(runtime): add admin Task List and notify.Nop for cancel wiring

EOF
)"
```

---

