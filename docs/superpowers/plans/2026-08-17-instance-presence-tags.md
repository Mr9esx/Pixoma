---
design-doc: docs/superpowers/specs/2026-08-17-instance-presence-tags-design.md
---

# 实例节点 / Comfy 状态 Tag Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 列表「启用」右侧两个 Tag（节点在线/掉线、Comfy运行中/未启动）每 5 秒刷新；状态由 Edge 本机探 Comfy 后报到控制面，详情观测 Badge 用同一套。

**Architecture:** Edge 每 5 秒用 `comfyui.Client.SystemStats` 问本机画图机，再 `POST /agent/v1/presence`。控制面内存记 `last_seen` 和 `comfy_running`。后台 `GET /api/v1/comfy-instances/presence` 按「15 秒内报到过 = 在线」算出两个布尔值。列表只轮询这条接口，不整表重拉。

**Tech Stack:** Go 内存 `sync.Mutex` store、chi Agent/Admin HTTP、`comfyui.NewClient`、Edge ticker、React Query `refetchInterval: 5000`、Vitest 合同测试

## Global Constraints

- 文案固定：`节点在线` / `节点掉线` / `Comfy运行中` / `Comfy未启动`（英文：`Node online` / `Node offline` / `Comfy running` / `Comfy not started`）
- 掉线窗口：`now - last_seen >= 15s` → `edge_online=false`，同时界面 Comfy 也显示未启动（不沿用过期的运行中）
- Edge 探活间隔 5 秒；管理页 presence 轮询 5 秒
- Comfy 是否在跑只信 presence 体里的 `comfy_running`；claim / 任务 heartbeat 只允许刷新 `last_seen`
- Mock 必须走 `comfyui.NewClient(Options{Mock: true})`，`SystemStats.Reachable == true`
- 不恢复 Redis `edge:online:*`，不改 `Pool.Probe` / `ListHealthy`，不上报 GPU/显存，不改启用开关
- presence 不落库；控制面重启后全部先掉线，直到下次报到
- JSON 字段名：`instance_id`、`comfy_running`、`edge_online`、`id`
- 未经用户明确要求不要 `git commit`

## 文件地图

| 文件 | 职责 |
|---|---|
| `internal/platform/presence/store.go` | 内存 last_seen + comfy_running；算出 Snapshot |
| `internal/httpapi/agent/handler.go` | `POST /agent/v1/presence`；claim/heartbeat Touch |
| `internal/httpapi/comfyinstances/handler.go` | `GET /api/v1/comfy-instances/presence`（必须注册在 `/{id}` 之前） |
| `apps/pixoma/cmd/pixoma/main.go` | 同一个 Store 注入 Agent 与 Admin |
| `apps/edge-agent/internal/pull/client.go` | `ReportPresence` HTTP 客户端 |
| `apps/edge-agent/internal/presence/reporter.go` | 每 5 秒本机探 Comfy 再上报 |
| `apps/edge-agent/cmd/edge-agent/main.go` | `NewClient` + 启动 reporter |
| `web/admin/src/lib/api/{types,instances,query-keys}.ts` | presence 类型与 GET |
| `web/admin/src/features/instances/{presence-tags,list-panel,observation-panel,detail-panel}.tsx` | 两个 Tag；列表不整表轮询 |
| `web/admin/src/lib/i18n/locales/{zh,en}.json` | 四条文案 |
| `docs/architecture/runtime.md` | 管理列表「通不通」改为 Edge presence |

---

### Task 1: 内存 presence store

**Files:**
- Create: `internal/platform/presence/store.go`
- Test: `internal/platform/presence/store_test.go`

**Interfaces:**
- Consumes: `sharedkernel.InstanceID`
- Produces:

```go
package presence

const OnlineWindow = 15 * time.Second

type Store struct{} // 构造：presence.NewStore()；测试可设 Now

func NewStore() *Store
func (s *Store) Report(id sharedkernel.InstanceID, comfyRunning bool)
func (s *Store) Touch(id sharedkernel.InstanceID) // 只刷新 last_seen，不改 comfy_running
func (s *Store) Snapshot(id sharedkernel.InstanceID) Snapshot

type Snapshot struct {
    ID           string `json:"id"`
    EdgeOnline   bool   `json:"edge_online"`
    ComfyRunning bool   `json:"comfy_running"`
}
```

- `Snapshot`：无记录或超时 → `EdgeOnline=false` 且 `ComfyRunning=false`（即使上次报过运行中）
- `Touch` 对从未 `Report` 过的 id：记 last_seen，`comfy_running` 保持 false
- 用 `sync.Mutex` 保护 map；`Now func() time.Time` 可注入（nil 则 `time.Now().UTC()`）

- [ ] **Step 1.1: 写失败测试**

```go
package presence_test

import (
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/presence"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestStore_ReportOnlineThenTimeoutClearsComfy(t *testing.T) {
	now := time.Unix(1_000, 0).UTC()
	s := presence.NewStore()
	s.Now = func() time.Time { return now }

	s.Report(sharedkernel.InstanceID("gpu-1"), true)
	got := s.Snapshot("gpu-1")
	if !got.EdgeOnline || !got.ComfyRunning || got.ID != "gpu-1" {
		t.Fatalf("fresh report: %+v", got)
	}

	now = now.Add(15 * time.Second)
	got = s.Snapshot("gpu-1")
	if got.EdgeOnline || got.ComfyRunning {
		t.Fatalf("at 15s must be offline and comfy down: %+v", got)
	}
}

func TestStore_TouchDoesNotSetComfyRunning(t *testing.T) {
	now := time.Unix(1_000, 0).UTC()
	s := presence.NewStore()
	s.Now = func() time.Time { return now }

	s.Touch("gpu-1")
	got := s.Snapshot("gpu-1")
	if !got.EdgeOnline || got.ComfyRunning {
		t.Fatalf("touch-only: %+v", got)
	}

	s.Report("gpu-1", true)
	now = now.Add(10 * time.Second)
	s.Touch("gpu-1")
	now = now.Add(10 * time.Second) // 距 Report 已 20s，但距 Touch 10s
	got = s.Snapshot("gpu-1")
	if !got.EdgeOnline || !got.ComfyRunning {
		t.Fatalf("touch must keep last comfy_running while still online: %+v", got)
	}
}

func TestStore_UnknownInstanceOffline(t *testing.T) {
	s := presence.NewStore()
	got := s.Snapshot("missing")
	if got.ID != "missing" || got.EdgeOnline || got.ComfyRunning {
		t.Fatalf("%+v", got)
	}
}
```

说明：`Now` 若不能从测试包赋值，把 `Now` 做成导出字段，或提供 `func NewStoreWithNow(func() time.Time) *Store`。推荐导出字段 `Now func() time.Time`，与 `agent.Handler.Now` 同一风格。

- [ ] **Step 1.2: 跑测试，确认失败**

Run: `go test ./internal/platform/presence/ -count=1`

Expected: FAIL，包不存在或符号未定义

- [ ] **Step 1.3: 最小实现**

`store.go` 要点：

```go
type record struct {
	lastSeen     time.Time
	comfyRunning bool
}

func (s *Store) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now().UTC()
}

func (s *Store) Snapshot(id sharedkernel.InstanceID) Snapshot {
	view := Snapshot{ID: string(id)}
	s.mu.Lock()
	rec, ok := s.records[id]
	s.mu.Unlock()
	if !ok {
		return view
	}
	online := s.now().Sub(rec.lastSeen) < OnlineWindow
	view.EdgeOnline = online
	view.ComfyRunning = online && rec.comfyRunning
	return view
}
```

空 id 的 `Report`/`Touch` 直接 return，不写 map。

超时比较用 **`< OnlineWindow`**（满 15s 算掉线），与规格「超过约 15 秒」一致。

- [ ] **Step 1.4: 再跑测试**

Run: `go test ./internal/platform/presence/ -count=1`

Expected: PASS

---

### Task 2: Agent `POST /agent/v1/presence` + claim/heartbeat Touch

**Files:**
- Modify: `internal/httpapi/agent/handler.go`
- Modify: `internal/httpapi/agent/handler_test.go`
- Modify: `apps/pixoma/cmd/pixoma/main.go`（本任务先注入 Agent；Admin 在 Task 3 接上同一指针）

**Interfaces:**
- Consumes: Task 1 的 `*presence.Store`（`Handler.Presence *presence.Store`）
- Produces: `POST /agent/v1/presence` JSON `{"instance_id":"...","comfy_running":true}` → 204；未授权 401；缺 instance_id 400
- `GET /jobs/claim` 在每次进入 wait 循环（含 200ms 空转）调用 `Presence.Touch(instanceID)`
- `POST /jobs/{id}/heartbeat` 成功后续约 `Touch(instanceID)`（不改 comfy_running）
- `Presence == nil` 时 presence 路由返回 500 `"presence not configured"`；claim/heartbeat 跳过 Touch

- [ ] **Step 2.1: 写失败测试**（加在 `handler_test.go`，扩展 `mountAgent` 注入 store）

把 `mountAgent` 改成能带 `*presence.Store`。新增：

```go
func TestAgent_PresenceReportsComfy(t *testing.T) {
	store := presence.NewStore()
	now := time.Unix(1000, 0).UTC()
	store.Now = func() time.Time { return now }
	h := &agent.Handler{
		Token:    "secret-token",
		Tasks:    runtimedomain.NewMemoryTaskRepository(),
		Presence: store,
		Now:      func() time.Time { return now },
	}
	r := chi.NewRouter()
	r.Route("/agent/v1", h.Mount)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]any{
		"instance_id":   "gpu-1",
		"comfy_running": true,
	})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/agent/v1/presence", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		raw, _ := io.ReadAll(res.Body)
		t.Fatalf("status=%d body=%s", res.StatusCode, raw)
	}
	got := store.Snapshot("gpu-1")
	if !got.EdgeOnline || !got.ComfyRunning {
		t.Fatalf("%+v", got)
	}
}

func TestAgent_PresenceUnauthorized(t *testing.T) {
	// 无 Authorization 或错误 token → 401，store 仍无记录
}

func TestAgent_ClaimTouchesLastSeenWithoutComfy(t *testing.T) {
	store := presence.NewStore()
	now := time.Unix(1000, 0).UTC()
	store.Now = func() time.Time { return now }
	h := &agent.Handler{
		Token:    "secret-token",
		Tasks:    runtimedomain.NewMemoryTaskRepository(),
		Presence: store,
		Now:      func() time.Time { return now },
	}
	// GET claim wait=0s → Touch
	got := store.Snapshot("gpu-1")
	if !got.EdgeOnline || got.ComfyRunning {
		t.Fatalf("claim must only touch: %+v", got)
	}
}
```

- [ ] **Step 2.2: 跑测试，确认失败**

Run: `go test ./internal/httpapi/agent/ -count=1`

Expected: FAIL，无 `/presence` 路由

- [ ] **Step 2.3: 实现**

`Mount` 增加：

```go
r.Post("/presence", h.presence)
```

`presence` handler：decode `instance_id` + `comfy_running`，`authorize`，`h.Presence.Report(...)`，204。

`claim` 循环开头（`for` 内、查任务前）`if h.Presence != nil { h.Presence.Touch(instanceID) }`，这样 25s 长轮询期间每 200ms 也会刷新 last_seen，空闲 Edge 不会被 15s 窗口误判掉线。

`heartbeat` 在 `ok == true` 之后 Touch。

`pixoma/main.go`：`pres := presence.NewStore()`，赋给 `agentH.Presence`。先用局部变量，Task 3 再赋给 instances Handler（同一指针）。

- [ ] **Step 2.4: 再跑测试**

Run: `go test ./internal/httpapi/agent/ -count=1`

Expected: PASS

---

### Task 3: Admin `GET /api/v1/comfy-instances/presence`

**Files:**
- Modify: `internal/httpapi/comfyinstances/handler.go`
- Modify: `internal/httpapi/comfyinstances/handler_test.go`
- Modify: `apps/pixoma/cmd/pixoma/main.go`

**Interfaces:**
- Consumes: 同一 `*presence.Store`；`Handler.Repo` 列出全部实例
- Produces: `GET /api/v1/comfy-instances/presence` → 200 JSON 数组，每个已存在实例一条 `{id, edge_online, comfy_running}`；从未报到的实例两项都为 false
- 路由必须写在 `r.Get("/{id}", ...)` **之前**，否则 chi 会把 `presence` 当 id

- [ ] **Step 3.1: 写失败测试**

在现有 `TestHandler_CreateListAndTasksFilter` 同文件新增独立测试，复用内存 DB + create 一台实例：

```go
func TestHandler_PresenceListsAllInstances(t *testing.T) {
	// 建库 + 创建 gpu-2（抄现有 create 测试的 Open/migrate/repo）
	store := presence.NewStore()
	now := time.Unix(1000, 0).UTC()
	store.Now = func() time.Time { return now }
	store.Report("gpu-2", true)

	h := &comfyinstances.Handler{Repo: repo, Pool: pool, Tasks: tasks, Mock: true, EncKey: testEncKey(), Presence: store}
	r := chi.NewRouter()
	r.Route("/api/v1/comfy-instances", h.Mount)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	res, err := http.Get(srv.URL + "/api/v1/comfy-instances/presence")
	// 200
	var body []map[string]any
	// 找到 id==gpu-2，edge_online==true，comfy_running==true
}
```

再测：不 Report、以及 `now+16s` 后两项为 false。再 `GET /api/v1/comfy-instances/presence` 不得 404（被 `/{id}` 吃掉）。

- [ ] **Step 3.2: 跑测试，确认失败**

Run: `go test ./internal/httpapi/comfyinstances/ -count=1 -run Presence`

Expected: FAIL，404 或无 Presence 字段

- [ ] **Step 3.3: 实现**

`Handler` 增加 `Presence *presence.Store`。

`Mount` 最前面（`/` 之后即可，但必须早于 `/{id}`）：

```go
r.Get("/presence", h.listPresence)
```

```go
func (h *Handler) listPresence(w http.ResponseWriter, r *http.Request) {
	recs, err := h.Repo.List(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]presence.Snapshot, 0, len(recs))
	for _, rec := range recs {
		if h.Presence == nil {
			out = append(out, presence.Snapshot{ID: string(rec.ID)})
			continue
		}
		out = append(out, h.Presence.Snapshot(rec.ID))
	}
	writeJSON(w, http.StatusOK, out)
}
```

若本包没有 `writeJSON`，复用文件里已有的 JSON 写法。`Presence == nil` 时全部 false，不要 500（管理页仍能渲染 Tag）。

`pixoma/main.go`：把 Task 2 的 `pres` 赋给 `&comfyinstances.Handler{..., Presence: pres}`。

- [ ] **Step 3.4: 再跑测试**

Run: `go test ./internal/httpapi/comfyinstances/ ./internal/httpapi/agent/ -count=1`

Expected: PASS

---

### Task 4: Edge 本机探 Comfy 并上报

**Files:**
- Modify: `apps/edge-agent/internal/pull/client.go`
- Modify: `apps/edge-agent/internal/pull/client_test.go`
- Create: `apps/edge-agent/internal/presence/reporter.go`
- Test: `apps/edge-agent/internal/presence/reporter_test.go`
- Modify: `apps/edge-agent/cmd/edge-agent/main.go`

**Interfaces:**
- Consumes: `pull.Client`；`comfyui.Client`（必须 `comfyui.NewClient`）
- Produces:

```go
func (c *Client) ReportPresence(ctx context.Context, comfyRunning bool) error
// POST {BaseURL}/agent/v1/presence
// body: {"instance_id": c.InstanceID, "comfy_running": comfyRunning}
// 401 → fmt.Errorf("pull: unauthorized")
// 非 204/200 → 带 body 的 status error

package edgepresence // 目录 apps/edge-agent/internal/presence，包名 presence 会与 platform/presence 冲突时：包名用 `edgepresence` 或目录仍 presence、包名 `edgereport`

type Reporter struct {
	Client *pull.Client
	Comfy  comfyui.Client
	Every  time.Duration // <=0 则 5s
}

func (r *Reporter) ProbeAndReport(ctx context.Context) error
func (r *Reporter) Run(ctx context.Context) error // 先立刻 probe 一次，再 ticker；ctx 取消退出
```

`ProbeAndReport`：`context.WithTimeout(ctx, 2*time.Second)` 调 `Comfy.SystemStats`；`err==nil && st != nil && st.Reachable` 才 `comfyRunning=true`；再 `ReportPresence`。Comfy 为 nil 或探失败都报 `false`（工人仍在线）。

`main.go`：`comfy, err = comfyui.NewClient(comfyui.Options{Mock: comfyMock, BaseURL: comfyURL})`（删掉直接 `&comfyui.Mock{}`）。`go reporter.Run(ctx)` 后 `loop.Run(ctx)`。

- [ ] **Step 4.1: 写失败测试**

`client_test.go`：

```go
func TestClient_ReportPresence(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/agent/v1/presence" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	c := pull.NewClient(srv.URL, "tok", "gpu-1")
	if err := c.ReportPresence(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if got["instance_id"] != "gpu-1" || got["comfy_running"] != true {
		t.Fatalf("%v", got)
	}
}
```

`reporter_test.go`：

```go
func TestReporter_MockComfyReportsRunning(t *testing.T) {
	var running any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		running = body["comfy_running"]
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	comfy, err := comfyui.NewClient(comfyui.Options{Mock: true})
	if err != nil {
		t.Fatal(err)
	}
	r := &edgepresence.Reporter{
		Client: pull.NewClient(srv.URL, "tok", "gpu-1"),
		Comfy:  comfy,
	}
	if err := r.ProbeAndReport(context.Background()); err != nil {
		t.Fatal(err)
	}
	if running != true {
		t.Fatalf("mock must report comfy_running=true, got %v", running)
	}
}

func TestReporter_UnreachableComfyReportsFalse(t *testing.T) {
	// httptest 立刻关闭，或 comfyui.NewClient(Mock:false, BaseURL: srv.URL) 后关闭 srv
	// ProbeAndReport 必须报 comfy_running=false，且 err 最好为 nil（探失败也要报到）
}
```

- [ ] **Step 4.2: 跑测试，确认失败**

Run: `go test ./apps/edge-agent/internal/pull/ ./apps/edge-agent/internal/presence/ -count=1`

Expected: FAIL，无 `ReportPresence` / 无 reporter 包

- [ ] **Step 4.3: 实现**

`Reporter.Run` 用 `time.NewTicker`，`select` 必须含 `ctx.Done()`。ticker 在 defer `Stop`。探活失败只打 slog，不要让 Run 退出（工人还在领活）。

包名：目录 `apps/edge-agent/internal/presence`，包名 `presence`（与 `internal/platform/presence` 不在同一 import 路径，Go 允许；`main.go` import 时给 Edge 包别名 `edgereport`）。

- [ ] **Step 4.4: 再跑测试**

Run: `go test ./apps/edge-agent/... ./internal/httpapi/agent/ ./internal/platform/presence/ -count=1`

Expected: PASS

---

### Task 5: 管理页两个 Tag + 5 秒只拉 presence

**Files:**
- Modify: `web/admin/src/lib/api/types.ts`
- Modify: `web/admin/src/lib/api/instances.ts`
- Modify: `web/admin/src/lib/api/instances.test.ts`
- Modify: `web/admin/src/lib/api/query-keys.ts`
- Modify: `web/admin/src/lib/i18n/locales/zh.json`
- Modify: `web/admin/src/lib/i18n/locales/en.json`
- Create: `web/admin/src/features/instances/presence-tags.tsx`
- Test: `web/admin/src/features/instances/presence-tags.test.ts`（合同：文案 key + 列表不整表 refetch）
- Modify: `web/admin/src/features/instances/list-panel.tsx`
- Modify: `web/admin/src/features/instances/observation-panel.tsx`
- Modify: `web/admin/src/features/instances/detail-panel.tsx`
- Modify: `web/admin/src/routes/_app/instances/route.tsx`
- Modify: `web/admin/src/features/instances/instances.contract.test.ts`

**Interfaces:**
- Consumes: `GET /api/v1/comfy-instances/presence` → `InstancePresence[]`
- Produces:

```ts
export type InstancePresence = {
  id: string
  edge_online: boolean
  comfy_running: boolean
}

export function listPresence() {
  return apiFetch<InstancePresence[]>('/api/v1/comfy-instances/presence')
}

queryKeys.instances.presence = ['instances', 'presence'] as const
```

i18n（`instances` 命名空间）：

| key | zh | en |
|---|---|---|
| `nodeOnline` | 节点在线 | Node online |
| `nodeOffline` | 节点掉线 | Node offline |
| `comfyRunning` | Comfy运行中 | Comfy running |
| `comfyStopped` | Comfy未启动 | Comfy not started |

`PresenceTags({ edgeOnline, comfyRunning })`：启用右侧两个 `Badge`。在线 `secondary`，掉线 `destructive`；Comfy 运行 `secondary`，未启动 `outline` 或 `destructive`。缺数据（map 里没有该 id）按掉线 + 未启动。

列表：`useQuery({ queryKey: queryKeys.instances.presence, queryFn: listPresence, refetchInterval: 5000 })` 放在 `InstancesLayout`，把 `presenceById` 传给 `InstanceListPanel`。`listInstances` 的 query **不要** 加 `refetchInterval`。

详情观测：`ReachBadge` 改为同一对 Tag（可保留 Mock 小标）。数据来自 `queryKeys.instances.presence`（同 key，React Query 去重），**不要**再用 `snap.reachable` 当「通不通」。system/queue 的错误 Alert 可保留（远程控制面 ping 仍可能失败）。

- [ ] **Step 5.1: 写失败测试**

`instances.test.ts` 追加：

```ts
it('listPresence GETs /api/v1/comfy-instances/presence', async () => {
  const fetchMock = vi.fn().mockResolvedValue(
    new Response(
      JSON.stringify([{ id: 'gpu-1', edge_online: true, comfy_running: false }]),
      { status: 200, headers: { 'Content-Type': 'application/json' } },
    ),
  )
  vi.stubGlobal('fetch', fetchMock)
  const data = await listPresence()
  expect(data[0].edge_online).toBe(true)
  expect(fetchMock).toHaveBeenCalledWith(
    expect.stringContaining('/api/v1/comfy-instances/presence'),
    expect.anything(),
  )
})
```

`instances.contract.test.ts` 追加：

```ts
it('shows presence tags to the right of enabled and polls presence every 5s', () => {
  const list = read('list-panel.tsx')
  const route = read('../routes/_app/instances/route.tsx'.replace('features/instances/', 'routes/_app/instances/'))
  // 更稳：从 here 拼到 route 与 list-panel
  expect(list).toMatch(/PresenceTags/)
  expect(list).toMatch(/instances\.enabled/)
  const layout = readFileSync(join(here, '../../../routes/_app/instances/route.tsx'), 'utf8')
  expect(layout).toMatch(/refetchInterval:\s*5000/)
  expect(layout).toMatch(/listPresence/)
  expect(layout).not.toMatch(/queryFn:\s*listInstances[\s\S]*refetchInterval/)
})
```

`observation-panel` 合同：不再匹配 `instances.reachable` / `instances.unreachable` 作为 Badge 文案，改为 `PresenceTags` 或 `nodeOnline`。

- [ ] **Step 5.2: 跑测试，确认失败**

Run: `pnpm --dir web/admin test -- src/lib/api/instances.test.ts src/features/instances/instances.contract.test.ts`

Expected: FAIL

- [ ] **Step 5.3: 实现 UI**

`presence-tags.tsx` 保持小：两个 Badge + 可选 mock。列表行布局：`启用` 文本右侧紧跟两个 Tag，不要加说明句。

`InstanceListPanel` 增加 `presenceById?: Record<string, InstancePresence>`。

`route.tsx`：

```ts
const presenceQuery = useQuery({
  queryKey: queryKeys.instances.presence,
  queryFn: listPresence,
  refetchInterval: 5000,
})
const presenceById = Object.fromEntries(
  (presenceQuery.data ?? []).map((row) => [row.id, row]),
)
```

详情 `detail-panel.tsx` 同样 `useQuery` 该 key（`enabled: tab === 'observe'` 可以，但列表已在拉；为免观测页未打开列表时也能刷新，详情在 observe tab 也用 `refetchInterval: 5000` 即可，同一 queryKey 会共享）。

- [ ] **Step 5.4: 再跑测试**

Run: `pnpm --dir web/admin test -- src/lib/api/instances.test.ts src/features/instances/`

Expected: PASS

---

### Task 6: 架构文档

**Files:**
- Modify: `docs/architecture/runtime.md`（规格要求）
- Modify: `docs/architecture/data-model.md` 观测 API 表加一行 presence（内存，不落库）
- Modify: `docs/architecture/task-data-walkthrough.md` 7.5 节：删掉 Redis `edge:online:*` 作为现行方案，改为内存 presence

**Interfaces:**
- Consumes: 本计划已实现的两条 HTTP
- Produces: 文档写明管理列表「通不通」= Edge presence，不是控制面 ping Comfy；`GET .../system` 仍是硬件明细，远程可能不通

- [ ] **Step 6.1: 改 runtime.md §7 表**

增加：

| HTTP | 数据源 |
|---|---|
| `POST /agent/v1/presence` | Edge 本机 `SystemStats` 结果写入控制面内存 |
| `GET /api/v1/comfy-instances/presence` | 内存 last_seen / comfy_running；15s 无报到视为掉线 |

§5 观测 API 那句「`GET .../system` → `/system_stats`」后面补一句：列表/详情「节点 / Comfy」两个 Tag **不以** 控制面探 Comfy 为准。

- [ ] **Step 6.2: data-model 观测表加 presence 行，注明不落库**

- [ ] **Step 6.3: walkthrough 7.5 改为现行内存心跳，并写明不再用 Redis `edge:online:*` 驱动管理页**

- [ ] **Step 6.4: 全量相关测试再跑一遍**

Run:

```
go test ./internal/platform/presence/ ./internal/httpapi/agent/ ./internal/httpapi/comfyinstances/ ./apps/edge-agent/... -count=1
pnpm --dir web/admin test -- src/lib/api/instances.test.ts src/features/instances/
```

Expected: 全部 PASS

---

## 自检（对照规格）

| 规格项 | 任务 |
|---|---|
| 两个 Tag 固定文案、启用右侧 | Task 5 |
| Edge 本机探 Comfy，Mock 走 NewClient | Task 4 |
| POST presence 心跳上报，不是控制面 ping | Task 2 + 4 |
| 15s 掉线且 Comfy 一并未启动 | Task 1 |
| 列表 5s 只拉状态 | Task 5 |
| 详情观测同一数据源 | Task 5 |
| 不恢复 Redis、不改 Pool.Probe | 全局约束 / 无对应任务 |
| claim 长轮询刷新 last_seen | Task 2 |
| runtime.md | Task 6 |
| 未启用实例同样显示；没报到=掉线+未启动 | Task 3 + 5 |
