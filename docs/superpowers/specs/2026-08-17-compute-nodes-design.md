---
role: technical-design
status: accepted
---

# 计算节点（Edge）改名、详情页与机器规格

## 1. 目标与边界

管理后台里现在叫「实例」的东西，本质是跑在画图电脑上的 Edge。界面改叫 **计算节点**，底下的表、接口、环境变量、代码标识一次切到 **edge**，不留旧名别名。

详情页对照 Shadcnblocks Admin Kit **订阅详情**那一页（`data-layout="fixed"` 起、到 Transactions 表为止）：**不是只抄区块顺序，连 class、字号、间距、颜色、图标尺寸都按参考抄**。去掉三个 Tab。每台节点记下 CPU / 内存 / 显卡规格：Edge 第一次启动写入，之后启动不覆盖；页面可手改，也可点「从机器更新」。

**非目标：**

- 不改调度怎么挑机器、不改 `Pool.Probe` / 健康检查语义
- 不上 NVIDIA NVML，不为 AMD 单独接 ROCm
- 不恢复 Redis `edge:online:*`
- 不把用户/任务/会话等其它 Master–Detail 列表改成 280px
- 不搬参考页右侧 300px 订阅侧栏（发票/客户那一列）
- 不保留 `INSTANCE_ID`、`/instances`、`/api/v1/comfy-instances`、表名 `comfy_instances` 的兼容读法

## 2. 已确认决定

| 项 | 选择 |
|---|---|
| 界面名 | 计算节点 |
| 代码 / 表 / 接口 / 环境变量 | edge（不是 node / compute_node） |
| 切法 | 一次切干净，不双写、不认旧环境变量 |
| 环境变量 | 只认 `EDGE_ID` |
| 列表 | 固定 280px；顶上按名称搜；不要列表标题栏 |
| 新建 | 页面标题右侧 Action；弹窗，不占详情区 |
| 详情 | 一页滚动；样式与参考页像素级一致（抄 class，不「看起来差不多」） |
| 规格位置 | 只在详情右列，列表仍是名字 + 启用 + 在线/Comfy 标签 |
| 多卡 | 每张显卡一张卡（名字 + 显存） |
| 采集 | `jaypipes/ghw` 采 CPU / 内存 / 显卡型号；显存问本机 Comfy `SystemStats` |

## 3. 日常差别

| 你看到 | 含义 |
|---|---|
| 侧栏「计算节点」 | 以前的「实例」，就是登记能跑画图的电脑 |
| 节点在线 / 掉线 | Edge 最近 15 秒有没有向控制面报到 |
| Comfy运行中 / 未启动 | 这台电脑上的画图程序开没开 |
| 右列规格 | 这台电脑的 CPU、内存、显卡；不是此刻还剩多少显存 |
| 「从机器更新」 | 让正在跑的 Edge 再采一次并写进库，会盖掉你手改的 |

远程那台机器必须把 `INSTANCE_ID` 改成 `EDGE_ID`，否则连不上。

## 4. 命名一次切断

| 现在 | 改成 |
|---|---|
| 界面「实例」 | 计算节点 |
| 路由 `/instances` | `/edges`（参数 `$edgeId`） |
| Admin `GET/POST /api/v1/comfy-instances` | `/api/v1/edges` |
| 表 `comfy_instances` | `edges` |
| 任务列 `instance_id` | `edge_id` |
| JSON `instance_id` | `edge_id`（含 presence、task、claim） |
| 环境变量 `INSTANCE_ID` | `EDGE_ID`（不读旧名） |
| 部署命令里的 `INSTANCE_ID=` | `EDGE_ID=` |
| Go `sharedkernel.InstanceID` | `EdgeID` |
| 包 `internal/platform/instance` | `internal/platform/edge` |
| HTTP `internal/httpapi/comfyinstances` | `internal/httpapi/edges` |
| 前端 `features/instances`、`queryKeys.instances` | `features/edges`、`queryKeys.edges` |
| yaml `default_instance_id` / `comfy_instances` | `default_edge_id` / `edges` |

调度 topic 仍是 `dispatch.<id>`，`<id>` 就是计算节点 id，值不变。

启动迁移（SQLite / MySQL / Postgres 都要能过）：

1. 若存在 `comfy_instances` 且尚无 `edges`：把表改名为 `edges`
2. 若 `tasks` 仍有 `instance_id` 且尚无 `edge_id`：列改名为 `edge_id`
3. 再 AutoMigrate 补规格列

旧数据的 id、`base_url`、token 原样留下。

控制面种子配置里的默认节点 id 不要复用进程环境变量 `EDGE_ID`（那是工人身份）。yaml 用 `default_edge_id`。

## 5. 页面

样式真相源：本机 `Shadcnblocks Admin Kit.mhtml` 订阅详情（从 `data-layout="fixed"` 到 Transactions 表）。实现时 **原样使用参考里的 Tailwind class**，不要用现有 `Badge variant="secondary"`、三列观测栅格、或 `font-bold` 标题去凑。合同测试锁关键 class 字符串。参考页按钮上的 `shadow-xs` / `shadow-sm` 在这一页保留（其它后台页的去阴影约定不动）。

### 5.1 整页壳

```
[计算节点]                                      [新建]
[登记能跑画图的电脑，看它在不在线、Comfy 开没开。]

[搜索名称] | 详情
  列表     |
  280px    |
```

- 页面标题右侧单独 Action 区，只放「新建」
- 只有这一页把 Master–Detail 左栏写成固定 `280px`（`md:grid-cols-[280px_1fr]`）。用户/任务/会话/Case 列表宽度不动
- 列表不要 Header、不要 Header 底边线；顶上一个 Input，只按 **节点名称** 过滤（不搜 id、地址、描述）
- 列表行：名称 + 启用 + 节点在线/掉线 + Comfy运行中/未启动。不要第二行描述/URL
- 未选中时右侧仍是「请选择」
- 文案只留标题、字段名、按钮、校验/错误；不要写「请去某页」之类说明。页面副标题这一句是产品要的简介，保留上面那句白话
- 页面标题行也用参考 header 的排法：`flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between`；标题 `truncate text-2xl leading-tight font-semibold tracking-tight`（不是 `font-bold`）；「新建」用参考页 Edit 那颗主按钮（见 5.2）

### 5.2 详情头（抄参考页 header）

详情滚动容器用参考 `section`：

`mx-auto flex w-full max-w-7xl flex-col gap-7 px-6 py-7 md:px-8`

头：

`header.flex.flex-col.gap-4.lg:flex-row.lg:items-center.lg:justify-between`

左块 `flex min-w-0 flex-col gap-4`：

1. 名称行 `flex min-w-0 flex-wrap items-center gap-3`
   - 名称：`h1.truncate.text-2xl.leading-tight.font-semibold.tracking-tight`
   - Tag 紧跟名称右侧，**不要用现有 Badge variant**。真状态抄参考 Active：

     `inline-flex items-center border py-0.5 h-6 rounded-md border-emerald-600/20 bg-emerald-50 px-2 text-xs font-medium text-emerald-700 shadow-none dark:border-emerald-400/20 dark:bg-emerald-900/30 dark:text-emerald-400`

     假状态抄参考表里的 Posted/zinc：

     `inline-flex items-center border py-0.5 h-6 rounded-md border-zinc-300 bg-zinc-50 px-2 text-xs font-medium text-zinc-700 shadow-none dark:border-zinc-700 dark:bg-zinc-900/60 dark:text-zinc-300`

     三个 Tag：启用、节点在线/掉线、Comfy运行中/未启动（启用 / 在线 / 运行中 = emerald；未启用 / 掉线 / 未启动 = zinc）

2. 描述行（没有描述就不渲染）：`text-muted-foreground flex max-w-full flex-wrap items-center gap-x-4 gap-y-2 text-sm`

右块 `flex items-center gap-2`，两颗按钮，图标 `size-3.5`：

| 按钮 | 对参考 | class 要点 | 图标 |
|---|---|---|---|
| 部署 | Pause | `border border-input bg-background shadow-xs hover:bg-accent rounded-md text-xs h-8 gap-1.5 px-3` | `terminal` |
| 编辑 | Edit | `bg-primary text-primary-foreground shadow-sm hover:bg-primary/90 rounded-md text-xs h-8 gap-1.5 px-3` | `pen-line` |

- 编辑 → Modal：名称、描述、地址、端口、分类、启用；可改规格；底部删除
- 部署 → Modal：TOKEN + 部署命令（`EDGE_ID`），可复制、可重新生成 token
- 新建 → 同一套表单 Modal（无规格、无删除）；成功后关掉弹窗并选中该节点

去掉「信息 / 部署 / 观测」三个 Tab，去掉详情区里的内联表单。不要参考页那个 `...` 更多菜单（删除只在编辑弹窗）。

### 5.3 两列字段（抄参考页 `dl`）

分割线必须是：

`div[data-orientation=horizontal][role=none].bg-border.shrink-0.h-[1px].w-full`

`dl`：`grid gap-x-20 gap-y-4 text-sm md:grid-cols-2`。为了「左信息、右规格」不交叉灌格，做成 **两列各自一叠**（左一列 div、右一列 div），每行仍抄参考单项：

`div.grid.grid-cols-[9.5rem_minmax(0,1fr)].items-start.gap-4`

- `dt`：`text-muted-foreground flex items-center gap-3 font-medium`，左侧 lucide 图标 `size-4`
- `dd`：`min-w-0 truncate font-medium`

**左列**

| 字段名 | 值 | 图标 |
|---|---|---|
| ID | 节点 id | `hash` |
| 创建时间 | 本地可读时间 | `calendar-clock` |
| 地址 | `base_url` 的 host | `globe` |
| 端口 | `base_url` 的 port | `ethernet-port` |
| 分类 | capabilities，空则空 | `tags` |
| 状态 | 启用 / 未启用 | `badge-check` |

库里仍存一条 `base_url`。展示和编辑拆成地址 + 端口；保存沿用原来的 http/https，新建默认 `http`。

**右列**（顶上放「从机器更新」，按钮 class 与「部署」相同，图标 `refresh-ccw` `size-3.5`）

| 字段名 | 值 | 图标 |
|---|---|---|
| CPU | 型号；有核数则带上 | `cpu` |
| 内存 | 人类可读容量 | `memory-stick` |
| 显卡 | 每张卡一行：名字 + 显存；多卡多行 `dd` | `gpu` |

从未采集时对应 `dd` 空着，不要补说明句。

### 5.4 数字卡与观测（抄参考页三卡 + Transactions）

再一条同样的 `h-[1px]` 横线。

外层不要参考页的 `xl:grid-cols-[..._300px]` 侧栏。数字卡区域：

```
div.text-card-foreground.border.bg-muted/15.rounded-lg.p-1.shadow-none
  div.p-0
    div.grid.gap-1.md:grid-cols-3
      每格：div.bg-background.rounded-md.border.px-4.py-3
        标签：div.text-muted-foreground.flex.items-center.gap-2.text-xs.font-medium + lucide size-4
        数字：p.mt-5.text-2xl.font-semibold.tracking-tight
```

| 卡 | 算法 | 图标 |
|---|---|---|
| 任务总数 | 该 `edge_id` 下全部任务条数（含进行中） | `list-todo` |
| 总运行时长 | 终态（成功/失败/取消）`updated_at - created_at` 之和 | `timer` |
| 成功率 | 成功 / (成功+失败)；没有成功或失败时显示「—」 | `badge-check` |

`GET /api/v1/edges/{id}/stats` 返回这三个数。

观测不要旧的三列栅格。改成参考 `section.flex.flex-col.gap-4` 连用三节（系统、队列、最近任务），节标题组件与 Transactions 相同：

```
div.flex.flex-col.gap-1
  div.flex.items-center.gap-3
    h2.text-[15px].font-semibold
    span.border-border.min-w-0.flex-1.border-t.border-dashed
  p.text-muted-foreground.text-xs
```

| 节 | 标题 | 副标题（字段名，不是说明句） | 内容 |
|---|---|---|---|
| 系统 | 系统 | 版本、内存、显存 | 字段行，class 与 5.3 的 `dt`/`dd` 相同 |
| 队列 | 队列 | 正在跑、等待 | 同上 |
| 最近任务 | 最近任务 | 状态、Case、ID、时间 | 表 |

最近任务表抄 Transactions：

- 容器：`overflow-hidden rounded-md border`
- 表：`w-full caption-bottom text-sm`
- 表头行：`border-b bg-muted/25`，`th`：`text-muted-foreground h-10 font-medium`，首列 `px-4`
- 单元格：`px-4 py-3` / `py-3`；状态用表内小 Tag：`h-5 rounded-md px-1.5 text-[11px] font-semibold shadow-none`，成功 emerald、失败 `border-red-600/20` 红系、其它 zinc

头上已经有 presence Tag，观测里不要再贴一套。系统栏是 Comfy **此刻** 的版本/空闲显存，和右列「机器规格」不是同一份数据。

## 6. 机器规格

### 6.1 落库

`edges` 增加：

| 列 | 说明 |
|---|---|
| `hardware_json` | 规格快照，可空 |
| `hardware_refresh_requested` | 布尔，默认 false |

`hardware_json` 形状：

```json
{
  "cpu_model": "Intel Core i9-13900K",
  "cpu_cores": 24,
  "ram_bytes": 68719476736,
  "gpus": [
    { "name": "NVIDIA GeForce RTX 4090", "vram_bytes": 25769803776 }
  ],
  "collected_at": "2026-08-17T14:00:00Z"
}
```

Admin 读节点详情时带上这份规格。PATCH 允许手改上述字段（不含 `collected_at`，服务端写）。PATCH `{ "refresh_hardware": true }` 只举手，不立刻改规格。

### 6.2 Edge 采集

Edge 进程内用 **`github.com/jaypipes/ghw`**（不自己读 `/proc`、不解析 `nvidia-smi`）：

- CPU：型号 + 核数
- 内存：物理总量字节
- 显卡：PCI 清单里的产品名（NVIDIA / AMD / Intel 都能出现在名单里；Mac 上 ghw 能力不完整，采不到就空着）

显存：**不来自 ghw**。Edge 问本机已有的 `comfyui.Client.SystemStats`，从 `raw.devices[].name` + `vram_total` 填 `gpus[].vram_bytes`。按名字对得上就补到 ghw 那张卡上；对不上就作为额外卡附上。Comfy 没开：型号仍可有，显存空。

Mock：`comfyui.Mock.SystemStats` 必须带至少一张假设备（名字 + `vram_total`），保证 `COMFY_MOCK=true` 时页面右列能看到显卡。ghw 在开发机上仍采真实 CPU/内存。

采集失败不阻止 Edge 启动，规格里缺什么就空什么。

### 6.3 何时写入

presence 请求增加可选 `hardware`。响应增加 `refresh_hardware`。

| 情况 | Edge | 控制面 |
|---|---|---|
| 进程启动后第一次报到 | 带上刚采的 `hardware` | `hardware_json` 为空才写入；已有则丢弃（重启不覆盖） |
| 之后心跳 | 不带 `hardware`；若上一次响应 `refresh_hardware=true` 则再带一次 | 不改规格，除非下面两行 |
| 页面点「从机器更新」 | 下一拍心跳先看到 `refresh_hardware=true`，再下一拍带采集结果（最多约 10 秒） | 置 `hardware_refresh_requested`；写入后清标志 |
| 用户在编辑弹窗改规格 | — | PATCH 直接写 `hardware_json` |

手改之后，只有再点「从机器更新」（或库被清空后的下一次首次报到）才会被机器采集盖掉。普通重启 Edge 不会盖。

Agent 体字段用 `edge_id`，不再用 `instance_id`。

## 7. 文档与验收

实现时同步：

- `docs/architecture/data-model.md`：表 `edges`、任务 `edge_id`、规格列
- `docs/architecture/runtime.md`、`task-data-walkthrough.md`、`overview.md`：实例改称计算节点 / Edge，接口路径
- 根 `README.md`：`EDGE_ID`、侧栏名称

验收：

- 详情对照参考页：标题/Tag/按钮/字段行（含 dt 图标）/分割线/数字卡/虚线节标题/任务表的 class 与 §5 列出的一致；合同测试锁这些字符串。不得用现有 Badge variant 或三列观测栅格凑
- 地址展示为 URL 的 host；保存时沿用原来的 http/https，新建默认 `http`
- 菜单、标题、空状态都是「计算节点」；`/instances` 删除，不留跳转
- 列表 280px、可按名称搜索、新建在标题右侧
- 详情无 Tab；编辑/部署走 Modal；两列字段与三张数字卡按上面算法
- 新库直接建 `edges`；旧库 `comfy_instances` 迁过去后数据还在
- Edge 只认 `EDGE_ID`；部署命令不再出现 `INSTANCE_ID`
- Mock 开：首次报到能写下 CPU/内存（本机 ghw）和假显卡；重启 Edge 不覆盖手改；点「从机器更新」会覆盖
- 真 Comfy 关掉时仍能写下 CPU/内存/显卡型号，显存可空
- `comfy_mock: true` 主路径（生成并收到图）不被这次改名破坏
