# Admin Resource Platform Context Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

```yaml
---
change: admin-resource-platform-context
design-doc: docs/superpowers/specs/2026-08-31-admin-resource-platform-context-design.md
base-ref: 4eb7e8ce096340490d757d017c3f7882dc1bcec6
---
```

**Goal:** 让后台 Task、Session、User 列表直接展示消息平台来源，并让 Task 展示可跳转的用户与会话上下文。

**Architecture:** 后端新增管理端专用投影，通过既有 `tasks → sessions → channel_users → channel_user_external_identities → channels` 关联返回上下文；Session 与 User 暴露平台无关字段。前端仅消费 API 返回的数据，在现有 DataTable 中新增或替换列，并用路由链接跳转。

**Tech Stack:** Go 1.25、GORM、chi、React 19、TanStack Table/Router、shadcn/ui、i18next、Go testing、Vitest。

## Global Constraints

- 只改 admin 投影与展示，不改任务执行、会话状态机或用户 upsert 主路径。
- 不新增数据库迁移，不把消息平台写死为 Telegram。
- User API 和前端不得出现 `tg_user_id` 字段、列或专用筛选。
- `external_user_id`、`channel_id` 使用字符串。
- 字段缺失显示空态，不根据 ID 前缀猜测平台。
- 只精确暂存当前 change 文件；保留工作区中无关改动。

---

### Task 1: Session 上下文契约

**Files:**
- Modify: `internal/conversation/domain/session.go`
- Modify: `internal/conversation/infrastructure/persistence/gorm_session.go`
- Modify: `internal/httpapi/sessions/handler.go`
- Test: `internal/httpapi/sessions/handler_test.go`

**Interfaces:**
- Consumes: `SessionRow.ChannelID` 和 `channels.name`。
- Produces: `Session.ChannelID string`；Session API list/detail 返回 `channel_id`、`channel_name`。

- [ ] **Step 1: Write the failing test**

在现有 GORM session handler 测试中创建 `channels` 行，保存 `SessionRow{ChannelID: "tg-default", ...}`，然后请求 list/detail，断言：

```go
if got["channel_id"] != "tg-default" || got["channel_name"] != "Default Telegram" {
    t.Fatalf("context: %+v", got)
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/httpapi/sessions -run TestSessionsHandler_ListGetReadOnly`
Expected: FAIL，`channel_id` 缺失。

- [ ] **Step 3: Implement minimal projection**

给 `Session` 增加 `ChannelID string`；`fromRow` 填充该字段。Session handler 建立可注入的 `ChannelNames map[string]string` 或轻量投影方法，DTO 输出 `ChannelID` 和 `ChannelName`，JSON 名分别为 `channel_id`、`channel_name`。

- [ ] **Step 4: Run tests**

Run: `go test ./internal/conversation/... ./internal/httpapi/sessions`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/conversation/domain/session.go internal/conversation/infrastructure/persistence/gorm_session.go internal/httpapi/sessions/handler.go internal/httpapi/sessions/handler_test.go
git commit -m "feat: expose session platform context"
```

### Task 2: User 平台身份契约与搜索

**Files:**
- Modify: `internal/identity/domain/user.go`
- Modify: `internal/identity/infrastructure/persistence/gorm_user.go`
- Modify: `internal/httpapi/users/handler.go`
- Test: `internal/httpapi/users/handler_test.go`

**Interfaces:**
- Consumes: `channel_users`、`channel_user_external_identities`。
- Produces: `User.ChannelID string`、`User.ExternalUserID string`；User DTO 返回同名 JSON 字段。

- [ ] **Step 1: Write failing tests**

扩展现有用户 handler 测试，断言 list/detail 包含 `channel_id`、`external_user_id`；用第二个用户验证 `q=9002` 命中外部 ID，`q=bob_access` 命中用户名。另加断言响应对象不包含 `tg_user_id`。

```go
for _, key := range []string{"channel_id", "external_user_id"} {
    if _, ok := row[key]; !ok {
        t.Fatalf("missing %s: %+v", key, row)
    }
}
if _, ok := row["tg_user_id"]; ok {
    t.Fatalf("telegram-specific key returned: %+v", row)
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/httpapi/users -run TestUsersHandler`
Expected: FAIL，identity 字段缺失或查询不命中外部 ID。

- [ ] **Step 3: Implement identity projection**

给 `User` 增加只读上下文字段。List 查询 join 外部身份表并填充字段；GetByID 读取用户行后按 `user_id` 补读身份。User DTO 增加两个字段并输出 JSON。`parseListQuery` 已支持 `external_user_id`，不新增 TG 专用参数；`q` 的 repository 条件增加 `external_user_id LIKE ?`。

- [ ] **Step 4: Run tests**

Run: `go test ./internal/identity/... ./internal/httpapi/users`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/identity/domain/user.go internal/identity/infrastructure/persistence/gorm_user.go internal/httpapi/users/handler.go internal/httpapi/users/handler_test.go
git commit -m "feat: expose generic user identity context"
```

### Task 3: Task 管理投影

**Files:**
- Create: `internal/runtime/infrastructure/persistence/gorm_task_admin.go`
- Modify: `internal/httpapi/tasks/handler.go`
- Modify: `apps/pixoma/cmd/pixoma/main.go`
- Modify: `apps/pixoma/cmd/pixoma/internal/livedemo/server.go`
- Test: `internal/httpapi/tasks/handler_test.go`

**Interfaces:**
- Consumes: Task repository 过滤结构、Session、User identity、channels。
- Produces: `TaskAdminProjection` 实现 `List(ctx, runtimedomain.AdminListQuery) ([]AdminTask, error)` 和 `Get(ctx, TaskID) (*AdminTask, error)`；注入到 `tasksapi.Handler.Context`。

`AdminTask` 字段包含 Task DTO 既有字段加 `ChannelID`、`ChannelName`、`UserID` 和 `User *AdminTaskUser`。`AdminTaskUser` 包含 `ID`、`ChannelID`、`ExternalUserID`、`Username`、`FirstName`、`LastName`。

- [ ] **Step 1: Write failing projection test**

用 GORM 建库并迁移 `TaskRow`、`SessionRow`、`UserRow`、`UserExternalIdentityRow`、`ChannelRow`。创建一条平台为 `tg-default` 的用户、Session 和 Task。请求 Task list/detail，断言：

```go
want := map[string]any{
    "session_id": "s1",
    "channel_id": "tg-default",
    "channel_name": "Default Telegram",
    "user_id": userID,
}
for key, value := range want {
    if got[key] != value {
        t.Fatalf("%s = %v, want %v; task=%+v", key, got[key], value, got)
    }
}
user, ok := got["user"].(map[string]any)
if !ok || user["id"] != userID || user["external_user_id"] != "9001" {
    t.Fatalf("user context: %+v", got["user"])
}
```

另建一个没有 Session 的 Task，断言 list 仍返回主行且 `channel_id == ""`。

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/httpapi/tasks -run TestTasksHandler_ContextProjection`
Expected: FAIL，projection 尚不存在。

- [ ] **Step 3: Implement projection**

在 GORM 查询中使用显式 SELECT 和 LEFT JOIN；主表别名固定为 `tasks`，排序、offset/limit 和状态过滤都使用 `tasks.` 前缀。`q` 搜索继续覆盖 task id、case id 和 session id。复用现有 `parseAdminListQuery` 语义。Handler 的 list/get 优先使用 `Context`；为空时回退旧逻辑，保证现有内存仓储测试不回归。

在两个应用装配点注入：

```go
Tasks: &tasksapi.Handler{
    Tasks: taskRepo, Cancel: orch,
    Context: taskpersist.NewTaskAdminProjection(gdb),
},
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/httpapi/tasks ./internal/runtime/...`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/runtime/infrastructure/persistence/gorm_task_admin.go internal/httpapi/tasks/handler.go internal/httpapi/tasks/handler_test.go apps/pixoma/cmd/pixoma/main.go apps/pixoma/cmd/pixoma/internal/livedemo/server.go
git commit -m "feat: add task admin platform projection"
```

### Task 4: 前端 API 类型与查询

**Files:**
- Modify: `web/admin/src/lib/api/types.ts`
- Modify: `web/admin/src/lib/api/users.ts`
- Test: `web/admin/src/lib/api/users.test.ts`

**Interfaces:**
- Consumes: 后端 `channel_id`、`channel_name`、`external_user_id`、Task 嵌套 `user`。
- Produces: `TaskRecord.user?: TaskUserContext | null`；`SessionRecord.channel_id`、`channel_name`；`UserRecord.channel_id`、`external_user_id`。

- [ ] **Step 1: Write failing type-level test**

更新 users API 测试的 mock 数据，发送 `external_user_id`，断言请求 URL 使用 `external_user_id` 而不是 `tg_user_id`，返回类型测试可用对象字面量赋给 `UserRecord`。

```ts
const data = await listUsers({ external_user_id: '9001', q: 'bob' })
expect(data[0]).toMatchObject({ channel_id: 'tg-default', external_user_id: '9001' })
```

- [ ] **Step 2: Run test to verify failure**

Run: `pnpm --dir web/admin exec vitest run src/lib/api/users.test.ts`
Expected: FAIL，字段类型或请求参数不匹配。

- [ ] **Step 3: Implement types**

在 `TaskRecord` 增加 `channel_id?: string`、`channel_name?: string`、`user_id?: string`、`user?: TaskUserContext | null`；Session 增加平台字段；User 替换 `tg_user_id: number` 为 `channel_id: string`、`external_user_id: string`。`listUsers` 参数改为 `channel_id?: string`、`external_user_id?: string`。

- [ ] **Step 4: Run tests**

Run: `pnpm --dir web/admin exec vitest run src/lib/api/users.test.ts src/lib/api/tasks.test.ts src/lib/api/sessions.test.ts`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add web/admin/src/lib/api/types.ts web/admin/src/lib/api/users.ts web/admin/src/lib/api/users.test.ts
git commit -m "feat: type admin platform context"
```

### Task 5: Task Table 与详情

**Files:**
- Modify: `web/admin/src/features/tasks/list-panel.tsx`
- Modify: `web/admin/src/features/tasks/detail-panel.tsx`
- Modify: `web/admin/src/features/tasks/list-panel.contract.test.ts`
- Modify: locale files containing `tasks.*`

**Interfaces:**
- Consumes: `TaskRecord.channel_name/channel_id`、`session_id`、`user_id`、`user`。
- Produces: 平台文本列、用户和 Session 链接列。

- [ ] **Step 1: Write failing contract assertions**

更新源码契约测试，断言平台列、用户路由 `/users/$userId`、Session 路由 `/sessions/$sessionId` 和 `tasks.fieldPlatform` 同时存在；断言空态逻辑仍在。

```ts
expect(source).toMatch(/fieldPlatform/)
expect(source).toMatch(/to='\/users\/\$userId'/)
expect(source).toMatch(/to='\/sessions\/\$sessionId'/)
```

- [ ] **Step 2: Run test to verify failure**

Run: `pnpm --dir web/admin exec vitest run src/features/tasks/list-panel.contract.test.ts`
Expected: FAIL，上下文列尚未实现。

- [ ] **Step 3: Implement columns**

在 Case 后插入平台列，内容 `channel_name || channel_id || '—'`；用户列显示 `user?.username || user?.first_name || user?.last_name || user_id || '—'`，并用 TanStack Router `Link` 跳转；Session 列显示 `session_id || '—'` 并链接。详情增加平台、用户和 Session 字段。

- [ ] **Step 4: Run tests**

Run: `pnpm --dir web/admin exec vitest run src/features/tasks/list-panel.contract.test.ts`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add web/admin/src/features/tasks/list-panel.tsx web/admin/src/features/tasks/detail-panel.tsx web/admin/src/features/tasks/list-panel.contract.test.ts web/admin/src/locales
git commit -m "feat: show task platform context"
```

### Task 6: Session Table 与详情

**Files:**
- Modify: `web/admin/src/features/sessions/list-panel.tsx`
- Modify: `web/admin/src/features/sessions/detail-panel.tsx`
- Modify: `web/admin/src/features/sessions/list-panel.contract.test.ts`
- Modify: locale files containing `sessions.*`

**Interfaces:**
- Consumes: `SessionRecord.channel_name/channel_id`。
- Produces: Session 列表和详情的平台列。

- [ ] **Step 1: Write failing contract assertion**

更新 Session 契约测试，断言 `sessions.fieldPlatform` 和 `channel_name || channel_id` 投影存在。

```ts
expect(source).toMatch(/fieldPlatform/)
expect(source).toMatch(/channel_name \|\| channel_id/)
```

- [ ] **Step 2: Run test to verify failure**

Run: `pnpm --dir web/admin exec vitest run src/features/sessions/list-panel.contract.test.ts`
Expected: FAIL，平台列缺失。

- [ ] **Step 3: Implement platform display**

在列表用户列附近插入平台列；详情在用户字段旁加 `Field`。空值统一渲染 `—`。

- [ ] **Step 4: Run tests**

Run: `pnpm --dir web/admin exec vitest run src/features/sessions/list-panel.contract.test.ts`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add web/admin/src/features/sessions/list-panel.tsx web/admin/src/features/sessions/detail-panel.tsx web/admin/src/features/sessions/list-panel.contract.test.ts web/admin/src/locales
git commit -m "feat: show session platform"
```

### Task 7: User Table、详情与搜索

**Files:**
- Modify: `web/admin/src/features/users/list-panel.tsx`
- Modify: `web/admin/src/features/users/detail-panel.tsx`
- Modify: `web/admin/src/features/users/list-panel.contract.test.ts`
- Modify: `web/admin/src/components/filters/list-filter.contract.test.ts`
- Modify: locale files containing `users.*`

**Interfaces:**
- Consumes: `UserRecord.channel_id`、`channel_name`、`external_user_id`、用户资料。
- Produces: 平台列和统一「用户信息」列。

- [ ] **Step 1: Write failing contract assertions**

更新 User 契约测试，断言 `fieldPlatform`、`fieldUserInfo`、`external_user_id` 和 `q` 存在；断言源码不包含 `tg_user_id` 和 `fieldTgUserId`。更新过滤测试路由参数为 `external_user_id`。

```ts
expect(source).toMatch(/fieldPlatform/)
expect(source).toMatch(/fieldUserInfo/)
expect(source).not.toMatch(/tg_user_id|fieldTgUserId/)
```

- [ ] **Step 2: Run test to verify failure**

Run: `pnpm --dir web/admin exec vitest run src/features/users/list-panel.contract.test.ts src/components/filters/list-filter.contract.test.ts`
Expected: FAIL，旧 TG 字段仍在且新列缺失。

- [ ] **Step 3: Implement platform and user columns**

删除 `tg_user_id` 列和详情字段。新增平台列，展示 `channel_name || channel_id || '—'`。新增「用户信息」列：主行 `username || displayName(first_name, last_name) || '—'`，次行 `external_user_id`。搜索对象保持单一 `q`。

- [ ] **Step 4: Run tests**

Run: `pnpm --dir web/admin exec vitest run src/features/users/list-panel.contract.test.ts src/components/filters/list-filter.contract.test.ts`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add web/admin/src/features/users/list-panel.tsx web/admin/src/features/users/detail-panel.tsx web/admin/src/features/users/list-panel.contract.test.ts web/admin/src/components/filters/list-filter.contract.test.ts web/admin/src/locales
git commit -m "feat: show generic user identity"
```

### Task 8: 全量受影响验证

**Files:**
- Modify: `docs/openspec/changes/admin-resource-platform-context/tasks.md`

**Interfaces:**
- Consumes: 前 7 个任务的实现。
- Produces: 勾选完成的 Comet tasks、测试证据。

- [ ] **Step 1: Run backend tests**

Run: `go test ./internal/conversation/... ./internal/identity/... ./internal/httpapi/sessions ./internal/httpapi/users ./internal/httpapi/tasks`
Expected: PASS。

- [ ] **Step 2: Run frontend tests and checks**

Run: `pnpm --dir web/admin exec vitest run src/lib/api/users.test.ts src/lib/api/tasks.test.ts src/lib/api/sessions.test.ts src/features/tasks/list-panel.contract.test.ts src/features/sessions/list-panel.contract.test.ts src/features/users/list-panel.contract.test.ts src/components/filters/list-filter.contract.test.ts && pnpm --dir web/admin exec tsc -b --pretty false`
Expected: PASS。

- [ ] **Step 3: Run OpenSpec validation**

Run: `comet classic openspec -- validate admin-resource-platform-context --type change --strict`
Expected: PASS。

- [ ] **Step 4: Update task checkboxes**

把 `tasks.md` 中已完成任务改为 `- [x]`，保持任务文本不变。

- [ ] **Step 5: Commit**

```bash
git add docs/openspec/changes/admin-resource-platform-context/tasks.md
git commit -m "test: verify admin platform context"
```
