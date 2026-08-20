---
design-doc: docs/superpowers/specs/2026-08-17-compute-nodes-design.md
---

# 计算节点（Edge）改名、详情页与机器规格 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 管理后台把「实例」一次改成「计算节点」（表/接口/环境变量都叫 edge），详情页按 Shadcnblocks 订阅详情抄完整 class，并由 Edge 用 ghw + Comfy 上报机器规格。

**Architecture:** 库表 `comfy_instances` → `edges`，任务列 `instance_id` → `edge_id`。控制面 Admin `/api/v1/edges`，Agent JSON/query 一律 `edge_id`。Edge 启动第一次 presence 带 `hardware`；库空才写入。页面点「从机器更新」举手，下一拍心跳覆盖。Admin 详情一页滚动：头 + 两列 dl + 三张数字卡 + 系统/队列/任务表。

**Tech Stack:** Go + GORM Migrator 改名、`jaypipes/ghw`、现有 `comfyui.Client.SystemStats`、chi、TanStack Router 文件路由、Vitest 合同测试锁 class。

## Global Constraints

- 界面名：计算节点。代码/表/接口/环境变量：edge（不是 node / compute_node）
- 一次切断：不认 `INSTANCE_ID`、`/instances`、`/api/v1/comfy-instances`、表名 `comfy_instances`
- Edge 只读 `EDGE_ID`；yaml 默认节点 id 用 `default_edge_id`，不要复用进程环境变量 `EDGE_ID`
- 调度 topic 仍是 `dispatch.<id>`
- 规格：ghw 采 CPU/内存/显卡型号；显存来自 Comfy `raw.devices[].vram_total`；不上 NVML
- 第一次启动且 `hardware_json` 为空才自动写；普通重启不覆盖；「从机器更新」会覆盖手改
- 详情样式抄参考页 class，不用现有 Badge variant、不用三列观测栅格、标题不用 `font-bold`
- 只有计算节点页左栏 `280px`；其它 Master–Detail 仍 `minmax(280px,360px)`
- 不搬参考页右侧 300px 侧栏；不改 `Pool.Probe` / 调度
- Mock 必须走 `comfyui.NewClient`；`SystemStats` 至少一张假 GPU
- 界面只留字段名/按钮/校验；页面副标题固定：「登记能跑画图的电脑，看它在不在线、Comfy 开没开。」
- 未经用户明确要求不要 `git commit`（下面 Commit 步默认跳过）

## 文件地图

| 文件 | 职责 |
|---|---|
| `internal/sharedkernel/ids.go`、`events.go` | `EdgeID`；JSON `edge_id` |
| `internal/platform/db/rename_legacy.go` | 旧表/旧列改名 |
| `internal/platform/instance` → `internal/platform/edge` | 节点领域 + 规格 |
| `internal/httpapi/comfyinstances` → `internal/httpapi/edges` | Admin `/api/v1/edges` |
| `internal/httpapi/agent/handler.go` | presence/claim 用 `edge_id` + hardware |
| `internal/runtime/infrastructure/persistence/gorm_task.go` | 列 `edge_id` |
| `apps/edge-agent/**` | `EDGE_ID`、采集、presence 握手 |
| `internal/runtime/infrastructure/comfyui/client.go` | Mock 假显卡 |
| `web/admin/src/features/instances` → `features/edges` | 页面 |
| `web/admin/src/routes/_app/instances` → `_app/edges` | 路由 `/edges` |
| `docs/architecture/*`、`README.md` | 架构与运维入口 |

---

### Task 1: 标识与表一次切断（后端能编译）

**Files:**
- Modify: `internal/sharedkernel/ids.go`、`events.go`、所有引用 `InstanceID` / `instance_id` JSON 的 Go 文件（不含 `docs/openspec/changes/archive`）
- Create: `internal/platform/db/rename_legacy.go`、`rename_legacy_test.go`
- Move: `internal/platform/instance` → `internal/platform/edge`（包名 `edge`）
- Move: `internal/httpapi/comfyinstances` → `internal/httpapi/edges`（包名 `edges`）
- Modify: `internal/platform/edge/persistence/gorm_instance.go` 表名 `edges`，类型可改名 `EdgeRow`
- Modify: `internal/runtime/infrastructure/persistence/gorm_task.go` 列 `edge_id`
- Modify: `internal/httpapi/adminhost/server.go` 挂 `/api/v1/edges`
- Modify: `internal/platform/botconfig/config.go` yaml `default_edge_id` / `edges`
- Modify: `internal/platform/appboot/boot.go` 先 `RenameLegacy` 再 AutoMigrate
- Modify: `apps/edge-agent/cmd/edge-agent/main.go` 只读 `EDGE_ID`
- Modify: `apps/edge-agent/internal/pull/client.go` query/JSON `edge_id`

**Interfaces:**
- Consumes: 现有 GORM、chi
- Produces:

```go
package sharedkernel
type EdgeID string
func TopicDispatch(id EdgeID) string // 仍返回 "dispatch." + string(id)

// DispatchCommand / TaskStatusEvent 字段名 EdgeID，tag json:"edge_id"
```

```go
package db
func RenameLegacy(gdb *gorm.DB) error
// 若有表 comfy_instances 且无 edges：RenameTable
// 若 tasks 有列 instance_id 且无 edge_id：RenameColumn
```

```go
package edge
func (EdgeRow) TableName() string { return "edges" }
```

Agent claim：`r.URL.Query().Get("edge_id")`。presence/heartbeat/status JSON：`edge_id`。任务列表 query：`edge_id`。

Go 结构体字段凡表示这台机器 id 的，从 `InstanceID` 改成 `EdgeID`（含 `domain.Task.EdgeID`、`presence.Store` 的 map key）。错误字符串里的 `instance:` 前缀改成 `edge:`。

- [x] **Step 1.1: 写失败测试（类型 + 旧表改名）**

`internal/sharedkernel/events_jobref_test.go` 里断言：

```go
got := TopicDispatch("local")
if got != "dispatch.local" {
    t.Fatalf("topic=%s", got)
}
ev := TaskStatusEvent{EdgeID: "gpu-1"}
raw, _ := json.Marshal(ev)
if !bytes.Contains(raw, []byte(`"edge_id":"gpu-1"`)) {
    t.Fatalf("json=%s", raw)
}
if bytes.Contains(raw, []byte("instance_id")) {
    t.Fatalf("old json key still present: %s", raw)
}
```

`internal/platform/db/rename_legacy_test.go`：

```go
func TestRenameLegacy_ComfyInstancesAndTaskColumn(t *testing.T) {
    gdb := openMem(t) // 与现有 sqlite 测试同样方式打开
    if err := gdb.Exec(`CREATE TABLE comfy_instances (id TEXT PRIMARY KEY, base_url TEXT)`).Error; err != nil {
        t.Fatal(err)
    }
    if err := gdb.Exec(`INSERT INTO comfy_instances (id, base_url) VALUES ('local', 'http://127.0.0.1:8188')`).Error; err != nil {
        t.Fatal(err)
    }
    if err := gdb.Exec(`CREATE TABLE tasks (id TEXT PRIMARY KEY, instance_id TEXT)`).Error; err != nil {
        t.Fatal(err)
    }
    if err := gdb.Exec(`INSERT INTO tasks (id, instance_id) VALUES ('t1', 'local')`).Error; err != nil {
        t.Fatal(err)
    }
    if err := RenameLegacy(gdb); err != nil {
        t.Fatal(err)
    }
    if gdb.Migrator().HasTable("comfy_instances") {
        t.Fatal("old table still exists")
    }
    if !gdb.Migrator().HasTable("edges") {
        t.Fatal("edges missing")
    }
    if gdb.Migrator().HasColumn("tasks", "instance_id") {
        t.Fatal("old column still exists")
    }
    if !gdb.Migrator().HasColumn("tasks", "edge_id") {
        t.Fatal("edge_id missing")
    }
    var url string
    if err := gdb.Raw(`SELECT base_url FROM edges WHERE id = ?`, "local").Scan(&url).Error; err != nil || url != "http://127.0.0.1:8188" {
        t.Fatalf("row migrated: url=%s err=%v", url, err)
    }
}
```

`openMem` 抄 `internal/platform/instance/persistence` 测试里现成的 sqlite 打开方式。

- [x] **Step 1.2: 跑测试，确认失败**

Run: `go test ./internal/sharedkernel ./internal/platform/db -count=1`

Expected: FAIL（还没有 `EdgeID` / `RenameLegacy`，或 JSON 仍是 `instance_id`）

- [x] **Step 1.3: 实现改名**

1. `InstanceID` → `EdgeID`；全仓库 Go 替换字段与参数（`rg 'InstanceID|instance_id' --glob '*.go'` 清零，测试夹具里的 id 值如 `"gpu-1"` 可留）
2. `git mv` 两个包目录，改 `package` 名与 import
3. `TableName` 返回 `"edges"`；`TaskRow`：`EdgeID string \`gorm:"column:edge_id;size:128;index"\``
4. `RenameLegacy` 用 `gdb.Migrator().HasTable/HasColumn/RenameTable/RenameColumn`
5. `appboot.Bootstrap`：`db.RenameLegacy(gdb)` 发生在 `AutoMigrate` 之前
6. adminhost：`r.Route("/api/v1/edges", ...)`
7. Edge main：`envOr("EDGE_ID", "local")`，不要读 `INSTANCE_ID`
8. pull client：`q.Set("edge_id", c.EdgeID)`，body `"edge_id"`
9. botconfig：`DefaultEdgeID string \`yaml:"default_edge_id"\``，`Edges []... \`yaml:"edges"\``；`os.Getenv("INSTANCE_ID")` 那行删掉（那是控制面种子，不是工人身份）

`presence.Store` 的 map key 改为 `sharedkernel.EdgeID`。

- [x] **Step 1.4: 跑测试确认通过**

Run: `go test ./internal/sharedkernel ./internal/platform/db ./internal/platform/edge/... ./internal/httpapi/edges ./internal/httpapi/agent ./internal/httpapi/adminhost ./apps/edge-agent/... -count=1`

Expected: PASS

再跑：`go test ./... -count=1`

Expected: PASS（改漏的 import 会在这里爆）

收尾检查：

```bash
rg -n 'INSTANCE_ID|/api/v1/comfy-instances|comfy_instances|json:"instance_id"' --glob '!docs/openspec/changes/archive/**' --glob '!docs/superpowers/**'
```

业务代码不应再命中（测试名/注释里的旧词也清掉）。

- [x] **Step 1.5: Commit**（默认跳过）

```bash
git add -A
git commit -m "refactor: cut instance identifiers over to edge"
```

---

### Task 2: 规格模型、合并显存、写入策略、Mock 假卡

**Files:**
- Create: `internal/platform/edge/hardware.go`、`hardware_test.go`
- Modify: `internal/platform/edge/record.go` 增加 `Hardware json.RawMessage` 或 `Hardware Hardware` + `HardwareRefreshRequested bool`
- Modify: `internal/platform/edge/persistence` 列 `hardware_json`、`hardware_refresh_requested`
- Modify: `internal/runtime/infrastructure/comfyui/client.go` Mock `SystemStats`
- Test: `internal/runtime/infrastructure/comfyui/stats_test.go`

**Interfaces:**
- Consumes: Task 1 的 `edge` 包
- Produces:

```go
package edge

type GPU struct {
    Name      string `json:"name"`
    VRAMBytes uint64 `json:"vram_bytes,omitempty"`
}

type Hardware struct {
    CPUModel     string    `json:"cpu_model,omitempty"`
    CPUCores     int       `json:"cpu_cores,omitempty"`
    RAMBytes     uint64    `json:"ram_bytes,omitempty"`
    GPUs         []GPU     `json:"gpus,omitempty"`
    CollectedAt  time.Time `json:"collected_at"`
}

func HardwareEmpty(h Hardware) bool // 全空（忽略 CollectedAt）

func ShouldWriteHardware(existing Hardware, refreshRequested bool) bool
// true: existing 空，或 refreshRequested

type HostInfo struct {
    CPUModel string
    CPUCores int
    RAMBytes uint64
    GPUNames []string
}

func MergeHardware(host HostInfo, stats *comfyui.SystemStats, now time.Time) Hardware
// ghw 名字打底；Comfy raw.devices[].name + vram_total（数字，字节）按名字补显存；
// 对不上名字的 Comfy 设备 append 成额外 GPU
```

`Record` 增加 `Hardware Hardware`、`HardwareRefreshRequested bool`。

Mock：

```go
"devices": []any{
    map[string]any{
        "name":       "Mock GPU",
        "vram_total": float64(8 << 30),
        "vram_free":  float64(8 << 30),
    },
},
```

- [x] **Step 2.1: 写失败测试**

```go
func TestShouldWriteHardware(t *testing.T) {
    if !ShouldWriteHardware(Hardware{}, false) {
        t.Fatal("empty should write")
    }
    existing := Hardware{CPUModel: "X"}
    if ShouldWriteHardware(existing, false) {
        t.Fatal("filled without refresh must not write")
    }
    if !ShouldWriteHardware(existing, true) {
        t.Fatal("refresh should write")
    }
}

func TestMergeHardware_FillsVRAMByName(t *testing.T) {
    host := HostInfo{CPUModel: "Intel", CPUCores: 8, RAMBytes: 16 << 30, GPUNames: []string{"NVIDIA GeForce RTX 4090"}}
    st := &comfyui.SystemStats{Reachable: true, Raw: map[string]any{
        "devices": []any{map[string]any{"name": "NVIDIA GeForce RTX 4090", "vram_total": float64(24 << 30)}},
    }}
    h := MergeHardware(host, st, time.Unix(0, 0).UTC())
    if h.CPUModel != "Intel" || h.CPUCores != 8 || h.RAMBytes != 16<<30 {
        t.Fatalf("cpu/ram: %+v", h)
    }
    if len(h.GPUs) != 1 || h.GPUs[0].VRAMBytes != 24<<30 {
        t.Fatalf("gpus: %+v", h.GPUs)
    }
}

func TestMergeHardware_AppendsUnknownComfyDevice(t *testing.T) {
    host := HostInfo{GPUNames: []string{"Card-A"}}
    st := &comfyui.SystemStats{Reachable: true, Raw: map[string]any{
        "devices": []any{map[string]any{"name": "Card-B", "vram_total": float64(1 << 30)}},
    }}
    h := MergeHardware(host, st, time.Time{})
    if len(h.GPUs) != 2 {
        t.Fatalf("len=%d", len(h.GPUs))
    }
}
```

`stats_test.go` 增加：Mock `SystemStats` 的 `Raw["devices"]` 非空且含 `vram_total`。

- [x] **Step 2.2: 跑测试，确认失败**

Run: `go test ./internal/platform/edge ./internal/runtime/infrastructure/comfyui -count=1 -run 'TestShouldWriteHardware|TestMergeHardware|TestMock_SystemStats'`

Expected: FAIL（函数不存在或 Mock 没有 devices）

- [x] **Step 2.3: 最小实现**

实现 `hardware.go`；persistence 读写 JSON 列；Mock 加上假卡。`vram_total` 可能是 `float64`（JSON 数字）或 `int`，`MergeHardware` 两种都收。

- [x] **Step 2.4: 跑测试确认通过**

Run: `go test ./internal/platform/edge ./internal/runtime/infrastructure/comfyui ./internal/platform/edge/persistence -count=1`

Expected: PASS

- [x] **Step 2.5: Commit**（默认跳过）

---

### Task 3: Edge 采集 + presence 握手

**Files:**
- Create: `apps/edge-agent/internal/hardware/collect.go`、`collect_test.go`
- Modify: `apps/edge-agent/internal/pull/client.go`、`client_test.go`
- Modify: `apps/edge-agent/internal/presence/reporter.go`、`reporter_test.go`
- Modify: `internal/httpapi/agent/handler.go`、`handler_test.go`
- Modify: `internal/httpapi/edges` GET/PATCH DTO 带 `hardware`；PATCH `refresh_hardware`
- Modify: `apps/edge-agent/cmd/edge-agent/main.go` 注入 collector

**Interfaces:**
- Consumes: `edge.HostInfo`、`edge.MergeHardware`、`edge.ShouldWriteHardware`
- Produces:

```go
package hardware

type Inspect func() (edge.HostInfo, error) // 生产：InspectGHW；测试注入

func InspectGHW() (edge.HostInfo, error) // 调 ghw.CPU / Memory / GPU，失败返回零值+err

func Collect(ctx context.Context, inspect Inspect, comfy comfyui.Client) edge.Hardware
// inspect 失败则 HostInfo 为零值继续；comfy 为 nil 或 SystemStats 失败则只留下 host
```

```go
// pull.Client
func (c *Client) ReportPresence(ctx context.Context, comfyRunning bool, hw *edge.Hardware) (refresh bool, err error)
// POST { edge_id, comfy_running, hardware? }
// 200 JSON { "refresh_hardware": bool }

// Reporter
type Reporter struct {
    Client *pull.Client
    Comfy  comfyui.Client
    Collect func(ctx context.Context) edge.Hardware // nil 则不带 hardware
    Every time.Duration
    sendHardware bool // 零值 false；main 里设 true 表示启动后第一次要带
}
```

Agent `presence` handler：

```go
type presenceReq struct {
    EdgeID       string         `json:"edge_id"`
    ComfyRunning bool           `json:"comfy_running"`
    Hardware     *edge.Hardware `json:"hardware"`
}
type presenceResp struct {
    RefreshHardware bool `json:"refresh_hardware"`
}
```

控制面：`Report` presence；若 `Hardware != nil`：`Get` 节点，`ShouldWriteHardware` 为真则写入并 `HardwareRefreshRequested=false`。响应 `refresh_hardware` = 写入后库里该标志仍为真（即本次没带 hardware 但页面已举手）。

Reporter：`sendHardware==true` 或上一拍 `refresh==true` 时调用 Collect 并放入 POST；成功后 `sendHardware=false`。

`go get github.com/jaypipes/ghw@latest` 只加在 edge-agent 用到的 collect 文件。

- [x] **Step 3.1: 写失败测试**

Agent：POST presence 带 `edge_id` + 空库 hardware → 再 GET 节点 `hardware.cpu_model` 有值；第二次带不同 CPU 且未举手 → 仍是第一次的值；PATCH `{refresh_hardware:true}` 后再 POST → 被覆盖。

Reporter：第一次 `ProbeAndReport` 的 HTTP body 含 `"hardware"`；第二次不含；若第一次响应 `refresh_hardware:true`，第二次再含。

Collect：注入 `Inspect` 返回固定 HostInfo，Comfy Mock → GPU 有 `vram_bytes`。

- [x] **Step 3.2: 跑测试，确认失败**

Run: `go test ./internal/httpapi/agent ./apps/edge-agent/internal/presence ./apps/edge-agent/internal/hardware ./apps/edge-agent/internal/pull -count=1`

Expected: FAIL

- [x] **Step 3.3: 实现**

handler 写入走 repo。PATCH 增加：

```go
RefreshHardware *bool `json:"refresh_hardware"`
Hardware        *edge.Hardware `json:"hardware"`
```

举手只改标志，不改 JSON。手改 `hardware` 时服务端设 `CollectedAt=now`。

GET 节点 DTO 增加 `hardware` 对象。

- [x] **Step 3.4: 跑测试确认通过**

Run: `go test ./internal/httpapi/agent ./internal/httpapi/edges ./apps/edge-agent/... -count=1`

Expected: PASS

- [x] **Step 3.5: Commit**（默认跳过）

---

### Task 4: 节点统计 API

**Files:**
- Modify: `internal/httpapi/edges/handler.go`、`handler_test.go`
- Modify: `internal/runtime/domain` 若 ListByInstance 需要无分页全量；不够就在 handler 里循环或加 `Limit` 足够大

**Interfaces:**
- Consumes: `Tasks.ListByInstance(ctx, id, query)`
- Produces:

```go
type statsDTO struct {
    TaskCount    int      `json:"task_count"`
    RuntimeMS    int64    `json:"runtime_ms"`
    SuccessRate  *float64 `json:"success_rate"`
}
GET /api/v1/edges/{id}/stats
```

算法：

- `task_count`：该 `edge_id` 全部任务数
- `runtime_ms`：状态 ∈ {succeeded, failed, cancelled} 的 `updated_at.Sub(created_at)` 毫秒求和，负的当 0
- `success_rate`：`succeeded / (succeeded+failed)`，分母 0 则 JSON `null`

路由挂在 `/{id}` 之前或用 `/{id}/stats`（chi 静态后缀，可放在 `/{id}` 之后）。与 `presence` 一样，`/presence` 必须在 `/{id}` 之前。

- [ ] **Step 4.1: 写失败测试**

插入 2 条 succeeded、1 条 failed、1 条 running，时长用固定 CreatedAt/UpdatedAt。断言 count=4、runtime 只加终态 3 条、rate=2/3。

- [ ] **Step 4.2: 跑测试，确认失败**

Run: `go test ./internal/httpapi/edges -count=1 -run Stats`

Expected: FAIL 404 或未定义

- [ ] **Step 4.3: 实现 GET stats**

- [ ] **Step 4.4: 跑测试确认通过**

Run: `go test ./internal/httpapi/edges -count=1`

Expected: PASS

- [ ] **Step 4.5: Commit**（默认跳过）

---

### Task 5: 管理端 API / 路由 / 文案改名

**Files:**
- Move: `web/admin/src/lib/api/instances.ts` → `edges.ts`（函数 `listEdges` 等）
- Move: `web/admin/src/features/instances` → `features/edges`
- Move: `web/admin/src/routes/_app/instances` → `_app/edges`（`$instanceId` → `$edgeId`）
- Modify: `web/admin/src/lib/api/types.ts`、`query-keys.ts`
- Modify: `web/admin/src/config/menu.ts`、`menu.test.ts`
- Modify: `web/admin/src/lib/i18n/locales/{zh,en}.json`：`menu.instances` 改 `menu.edges`，文案「计算节点」/ `Compute nodes`；namespace `instances` → `edges`
- Modify: `web/admin/src/features/edges/deploy-command.ts`：`EDGE_ID=`
- Modify: 所有仍 import `/instances` 的文件；跑 `pnpm exec tsr generate` 或 dev 生成 `routeTree.gen.ts`
- Modify: `web/admin/src/components/master-detail/master-detail.contract.test.ts` 把 instances 路径改成 edges 路由仍挂 Shell

**Interfaces:**
- Consumes: Task 1–4 的 HTTP
- Produces:

```ts
export type EdgeHardware = {
  cpu_model?: string
  cpu_cores?: number
  ram_bytes?: number
  gpus?: { name: string; vram_bytes?: number }[]
  collected_at?: string
}

export type ComfyEdge = {
  id: string
  name: string
  description?: string
  base_url: string
  enabled: boolean
  capabilities: string[]
  agent_token?: string
  hardware?: EdgeHardware
  created_at: string
  updated_at: string
}

export type EdgeStats = {
  task_count: number
  runtime_ms: number
  success_rate: number | null
}

queryKeys.edges.all / .presence / .detail(id) / .system / .queue / .tasks / .stats(id)
```

路径一律 `/api/v1/edges`。`TaskRecord.edge_id` 替换 `instance_id`。

中文：

- `edges.title`: 计算节点
- `edges.description`: 登记能跑画图的电脑，看它在不在线、Comfy 开没开。
- `menu.edges`: 计算节点

菜单 `id: 'edges', path: '/edges'`。

- [ ] **Step 5.1: 写失败测试**

`edges.ts` 的 API 测试：`listEdges` GET `/api/v1/edges`。

`deploy-command` 测试：命令含 `EDGE_ID=`，不含 `INSTANCE_ID`。

`menu.test.ts`：path `/edges`，id `edges`。

- [ ] **Step 5.2: 跑测试，确认失败**

Run: `cd web/admin && pnpm test src/lib/api/edges.test.ts src/config/menu.test.ts src/features/edges/agent-credentials.test.ts`

Expected: FAIL 或文件不存在

- [ ] **Step 5.3: 搬家并改路径/文案**

这一步可以先让旧页面还能编译：路由改完后 `routeTree.gen.ts` 必须更新。不要留 `/instances` 重定向。

- [ ] **Step 5.4: 跑测试确认通过**

Run: `cd web/admin && pnpm test`

Expected: PASS（旧合同若仍锁 Tab，下一 Task 再改）

- [ ] **Step 5.5: Commit**（默认跳过）

---

### Task 6: 列表壳——280px、搜索、标题右侧新建

**Files:**
- Modify: `web/admin/src/components/master-detail/master-detail-shell.tsx`（保持默认 `minmax(280px,360px)`）
- Modify: `web/admin/src/routes/_app/edges/route.tsx`
- Modify: `web/admin/src/features/edges/list-panel.tsx`
- Modify: `web/admin/src/features/edges/edges.contract.test.ts`
- Modify: `web/admin/src/components/master-detail/master-detail.contract.test.ts`：默认宽度断言不动；edges 路由另锁 280px

**Interfaces:**
- Consumes: `MasterDetailShell` 已有 `className`（`cn`/`twMerge` 必须让后写的 `md:grid-cols-[280px_1fr]` 覆盖默认）
- Produces: 页面标题行参考 header：`flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between`；h1 `truncate text-2xl leading-tight font-semibold tracking-tight`；右侧「新建」按钮 class 与参考 Edit 相同（见 Task 7）；点击打开创建弹窗状态（弹窗 UI 可在 Task 8 接上，本 Task 至少把按钮放到标题行并去掉列表 Header）

列表：

- 无 Header、无 Header `border-b`
- 顶上 `Input`，只按 `name` 包含匹配（大小写不敏感）
- 行：名称 + 启用文案 + `PresenceTags`；不要第二行描述/URL

- [ ] **Step 6.1: 写失败合同测试**

```ts
it('puts create on the page title row and searches by name only', () => {
  const layout = read('../../routes/_app/edges/route.tsx')
  const list = read('list-panel.tsx')
  expect(layout).toMatch(/lg:flex-row lg:items-center lg:justify-between/)
  expect(layout).toMatch(/leading-tight font-semibold tracking-tight/)
  expect(layout).not.toMatch(/font-bold/)
  expect(layout).toMatch(/md:grid-cols-\[280px_1fr\]/)
  expect(list).not.toMatch(/border-b px-4 py-3/)
  expect(list).toMatch(/<Input/)
  expect(list).not.toMatch(/item\.base_url/)
})
```

- [ ] **Step 6.2: 跑测试，确认失败**

Run: `cd web/admin && pnpm test src/features/edges/edges.contract.test.ts`

Expected: FAIL

- [ ] **Step 6.3: 改 layout 与 list**

标题「新建」先 `setCreateOpen(true)` 或导航逻辑留给 Task 8；本 Task 按钮必须在标题行，不在列表里。

- [ ] **Step 6.4: 跑测试确认通过**

Run: `cd web/admin && pnpm test src/features/edges/edges.contract.test.ts src/components/master-detail/master-detail.contract.test.ts src/config/menu.test.ts`

Expected: PASS；其它页 Shell 仍含 `minmax(280px,360px)`

- [ ] **Step 6.5: Commit**（默认跳过）

---

### Task 7: 详情像素级（头、dl、数字卡、观测）

**Files:**
- Create: `web/admin/src/features/edges/kit-classes.ts`（把参考 class 收成常量，合同测试锁这些字符串）
- Modify: `web/admin/src/features/edges/detail-panel.tsx`
- Modify: `web/admin/src/features/edges/presence-tags.tsx`（改用 kit Tag class，不用 `Badge` variant）
- Modify: `web/admin/src/features/edges/observation-panel.tsx`
- Modify: `web/admin/src/features/edges/edges.contract.test.ts`
- Modify: `web/admin/src/lib/api/edges.ts` 增加 `getEdgeStats`
- 删除详情里的 Tabs / 内联表单 / `contentRegionClassName` 三 Tab 结构

**Interfaces:**
- Consumes: `kit-classes.ts` 常量必须与 spec §5 一致
- Produces:

```ts
export const kit = {
  pageSection: 'mx-auto flex w-full max-w-7xl flex-col gap-7 px-6 py-7 md:px-8',
  header: 'flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between',
  title: 'truncate text-2xl leading-tight font-semibold tracking-tight',
  tagOn: 'inline-flex items-center border py-0.5 h-6 rounded-md border-emerald-600/20 bg-emerald-50 px-2 text-xs font-medium text-emerald-700 shadow-none dark:border-emerald-400/20 dark:bg-emerald-900/30 dark:text-emerald-400',
  tagOff: 'inline-flex items-center border py-0.5 h-6 rounded-md border-zinc-300 bg-zinc-50 px-2 text-xs font-medium text-zinc-700 shadow-none dark:border-zinc-700 dark:bg-zinc-900/60 dark:text-zinc-300',
  desc: 'text-muted-foreground flex max-w-full flex-wrap items-center gap-x-4 gap-y-2 text-sm',
  btnGhost: 'border border-input bg-background shadow-xs hover:bg-accent rounded-md text-xs h-8 gap-1.5 px-3',
  btnPrimary: 'bg-primary text-primary-foreground shadow-sm hover:bg-primary/90 rounded-md text-xs h-8 gap-1.5 px-3',
  rule: 'bg-border shrink-0 h-[1px] w-full',
  dl: 'grid gap-x-20 gap-y-4 text-sm md:grid-cols-2',
  field: 'grid grid-cols-[9.5rem_minmax(0,1fr)] items-start gap-4',
  dt: 'text-muted-foreground flex items-center gap-3 font-medium',
  dd: 'min-w-0 truncate font-medium',
  statsWrap: 'text-card-foreground border bg-muted/15 rounded-lg p-1 shadow-none',
  statsGrid: 'grid gap-1 md:grid-cols-3',
  statsCell: 'bg-background rounded-md border px-4 py-3',
  statsLabel: 'text-muted-foreground flex items-center gap-2 text-xs font-medium',
  statsValue: 'mt-5 text-2xl font-semibold tracking-tight',
  sectionTitle: 'text-[15px] font-semibold',
  sectionDash: 'border-border min-w-0 flex-1 border-t border-dashed',
  tableWrap: 'overflow-hidden rounded-md border',
  th: 'text-muted-foreground h-10 font-medium',
  tagSmOn: 'inline-flex items-center border py-0.5 font-semibold h-5 rounded-md px-1.5 text-[11px] shadow-none border-emerald-600/20 bg-emerald-50 text-emerald-700 dark:border-emerald-400/20 dark:bg-emerald-900/30 dark:text-emerald-400',
  tagSmOff: 'inline-flex items-center border py-0.5 font-semibold h-5 rounded-md px-1.5 text-[11px] shadow-none border-zinc-300 bg-zinc-50 text-zinc-700 dark:border-zinc-700 dark:bg-zinc-900/60 dark:text-zinc-300',
  tagSmFail: 'inline-flex items-center border py-0.5 font-semibold h-5 rounded-md px-1.5 text-[11px] shadow-none border-red-600/20 bg-red-50 text-red-700 dark:border-red-400/20 dark:bg-red-900/30 dark:text-red-400',
}
```

详情结构（无 Tab）：

1. `section.kit.pageSection` 可滚动
2. header：名称 + 三枚 Tag（启用/在线/Comfy）+ 描述；右「部署」「编辑」
3. `div data-orientation="horizontal" role="none" className={kit.rule}`
4. `dl.kit.dl` 两列 div：左 ID/创建时间/地址/端口/分类/状态；右「从机器更新」+ CPU/内存/显卡
5. 再一条 rule
6. 三张数字卡（`getEdgeStats`）
7. 三节观测：系统、队列、最近任务（虚线标题 + 副标题「版本、内存、显存」/「正在跑、等待」/「状态、Case、ID、时间」）
8. 任务表用 `kit.tableWrap`；观测里不再放 PresenceTags

图标（lucide，dt `className="size-4"`，按钮图标 `size-3.5`）：`Hash`、`CalendarClock`、`Globe`、`EthernetPort`（没有则 `Unplug`）、`Tags`、`BadgeCheck`、`Cpu`、`MemoryStick`、`Gpu`（没有则 `Microchip`）、`RefreshCcw`、`Terminal`、`PenLine`、`ListTodo`、`Timer`。

地址/端口：`new URL(base_url)` 取 hostname/port；缺 port 时 https=443、http=80。

「从机器更新」：`patchEdge(id, { refresh_hardware: true })`。

- [ ] **Step 7.1: 写失败合同测试**

断言 `detail-panel.tsx`：

- 含 `kit.pageSection` / `data-orientation="horizontal"` / `grid-cols-[9.5rem_minmax(0,1fr)]`
- 不含 `tabInfo`、`TabsTrigger`、`md:grid-cols-3` 观测旧栅格（数字卡的 `md:grid-cols-3` 在 statsGrid，允许只出现在数字卡常量）
- 含 `PenLine`、`Terminal`、`RefreshCcw`
- `presence-tags.tsx` 不含 `from '@/components/ui/badge'`

- [ ] **Step 7.2: 跑测试，确认失败**

Run: `cd web/admin && pnpm test src/features/edges/edges.contract.test.ts`

Expected: FAIL

- [ ] **Step 7.3: 实现详情与观测改版**

观测字段行复用 `kit.field`/`dt`/`dd`，不要 `MetricRow` 的左右 flex。

- [ ] **Step 7.4: 跑测试确认通过**

Run: `cd web/admin && pnpm test src/features/edges`

Expected: PASS

- [ ] **Step 7.5: Commit**（默认跳过）

---

### Task 8: 新建 / 编辑 / 部署 Modal

**Files:**
- Modify: `web/admin/src/features/edges/edge-form.tsx`（由 instance-form 改名）
- Modify: `web/admin/src/features/edges/detail-panel.tsx`、`route.tsx`
- Modify: `web/admin/src/features/edges/agent-credentials.tsx`
- Modify: `web/admin/src/features/edges/edges.contract.test.ts`
- 使用现有 `components/ui/dialog.tsx`

**Interfaces:**
- Consumes: Task 7 的编辑/部署按钮；Task 6 的新建按钮
- Produces:

创建 Dialog：名称、描述、地址、端口、分类、启用；无规格、无删除。保存 `base_url = `${scheme}://${host}:${port}``，新建 scheme=`http`。成功后关闭并 `navigate({ to: '/edges/$edgeId', params: { edgeId: created.id } })`。

编辑 Dialog：同上 + CPU/核数/内存字节/显卡列表（可增删行）+ 删除确认。保存可同时 PATCH `base_url`、`hardware`。

部署 Dialog：现有 token + `<pre>` 命令 + 复制 + 重新生成。

不要 `/edges/new` 详情占位（可删 `$edgeId === 'new'` 分支）。

- [ ] **Step 8.1: 写失败合同测试**

```ts
expect(detail).toMatch(/<Dialog/)
expect(detail).toMatch(/refresh_hardware/)
expect(form).toMatch(/htmlFor=['"]edge-host['"]/)
expect(form).toMatch(/htmlFor=['"]edge-port['"]/)
expect(form).not.toMatch(/htmlFor=['"]instance-id['"]/)
expect(layout).not.toMatch(/edgeId === 'new'/)
```

- [ ] **Step 8.2: 跑测试，确认失败**

Run: `cd web/admin && pnpm test src/features/edges/edges.contract.test.ts`

Expected: FAIL

- [ ] **Step 8.3: 实现三个 Dialog**

地址解析失败时校验错误用现有错误展示，不要加说明句。

- [ ] **Step 8.4: 跑测试确认通过**

Run: `cd web/admin && pnpm test`

Expected: PASS

- [ ] **Step 8.5: Commit**（默认跳过）

---

### Task 9: 架构文档与 README

**Files:**
- Modify: `docs/architecture/data-model.md`、`runtime.md`、`overview.md`、`task-data-walkthrough.md`
- Modify: `README.md`
- Modify: `apps/admin-api/README.md` 若仍写 comfy-instances

**Interfaces:**
- Consumes: 已落地的表名/路径
- Produces: 文档与实现一致

改动要点：

- 表 `edges`（原 `comfy_instances`）+ `hardware_json` + `hardware_refresh_requested`
- `tasks.edge_id`
- Admin `/api/v1/edges`、`/presence`、`/{id}/stats`
- Agent `{ edge_id, comfy_running, hardware? }` → `{ refresh_hardware }`
- 环境变量 `EDGE_ID`
- 界面「计算节点」

- [ ] **Step 9.1: 写失败检查（文档合同可用 rg 测试脚本或手工断言）**

在 `docs/architecture/data-model.md` 增加/改段落后，用测试不够现实时：本 Task 以 `rg` 自检。

```bash
rg -n 'comfy_instances|/api/v1/comfy-instances|INSTANCE_ID' docs/architecture README.md apps/admin-api/README.md
```

Expected（实现后）：无业务旧名（历史句子若必须提及「原表名」只出现在迁移说明一句里）

- [ ] **Step 9.2: 先跑 rg，确认文档仍是旧名（红）**

- [ ] **Step 9.3: 改文档**

- [ ] **Step 9.4: 再跑 rg + `go test ./...` + `cd web/admin && pnpm test`**

Expected: 文档无旧入口；测试绿；`comfy_mock` 主路径代码仍走 `comfyui.NewClient`

- [ ] **Step 9.5: Commit**（默认跳过）

---

## Spec coverage

| Spec | Task |
|---|---|
| 界面/表/API/env 一次改 edge | 1, 5, 9 |
| 启动迁移旧表旧列 | 1 |
| ghw + Comfy 显存、首次写入、刷新、手改 | 2, 3, 8 |
| Mock 假 GPU | 2 |
| 统计三卡算法 | 4, 7 |
| 列表 280px、搜索名称、标题新建 | 6 |
| 详情像素级 class / 无 Tab | 7 |
| 编辑/部署/新建 Modal、地址端口 | 8 |
| 架构文档 | 9 |
| 不改 Probe、不搬 300px 侧栏、不上 NVML | 未做即遵守 |
