---
role: technical-design
status: accepted
---

# 实例列表节点 / Comfy 状态 Tag 技术设计

## 1. 目标与边界

实例列表每一行「启用」右侧两个 Tag，每 5 秒刷新。状态来自 **Edge 本机探 Comfy，再跟着心跳报到控制面**，不是控制面去 ping 画图机。详情「实例观测」里原来的可达 Badge 改成同一套，避免两套说法。

文案固定：

| Tag | 真 | 假 |
|---|---|---|
| 工人 | 节点在线 | 节点掉线 |
| 画图机 | Comfy运行中 | Comfy未启动 |

**非目标：**

- 不恢复 Redis `edge:online:*`
- 控制面 `Pool.Probe` / 调度选实例逻辑这轮不改
- Edge 不上报完整 GPU/显存；观测页的 system/queue 明细仍走现有 `/system` `/queue`（远程可能仍打不通，只把「通不通」这两个 Tag 改准）
- 不改启用开关语义

## 2. 已确认决定

| 项 | 选择 |
|---|---|
| 两个 Tag | 节点在线/掉线 + Comfy运行中/未启动 |
| 位置 | 列表「启用」右侧；只动 Tag，不加说明句 |
| Comfy 怎么判断 | Edge 本机问 Comfy；Mock 算运行中 |
| 怎么到控制面 | Edge 心跳上报，不是控制面 ping Comfy |
| 掉线 | 上次报到超过约 15 秒 |
| 列表刷新 | 每 5 秒只拉状态，不整表重绘闪烁 |
| 详情观测 | 可达 Badge 改用同一套上报 |

## 3. 日常差别

| 你看到 | 含义 |
|---|---|
| 节点在线 | 这台机器上的工人最近 15 秒内向控制面报到过 |
| 节点掉线 | 工人没在报到（没起 Edge、进程死了、网络断了） |
| Comfy运行中 | 工人本机认为画图机活着（含 Mock） |
| Comfy未启动 | 工人说画图机没起来，或工人已经掉线（不再假装还在跑） |

远程 GPU 机器只出站时，这两个 Tag 仍然准。以前控制面自己 ping Comfy，远程会一直像没启动。

## 4. 上报与读取

```text
Edge 每 5 秒：本机探 Comfy → POST /agent/v1/presence
                 { instance_id, comfy_running }
控制面内存：last_seen、comfy_running
后台列表/详情每 5 秒：GET /api/v1/comfy-instances/presence
                 → 每条 { id, edge_online, comfy_running }
```

- `edge_online`：`now - last_seen < 15s`
- 工人掉线时：界面 Comfy 也显示「未启动」（没有新报到，不沿用过期的「运行中」）
- 领活长轮询、任务内 heartbeat 也可顺手刷新 `last_seen`（防止 presence 偶然失败时误判掉线）；Comfy 是否在跑以 presence 体为准
- 控制面重启后全部先显示掉线/未启动，直到 Edge 再次报到（最多约 5 秒）
- 未启用的实例同样显示；没人报到就是掉线 + 未启动

Edge 探 Comfy 走现有 `comfyui.Client`（`NewClient`，Mock 开则 `reachable=true` 且不必真 HTTP）。

## 5. 界面

列表 `InstanceListPanel`：「启用/未启用」右侧两个 Tag，文案如上，不要再写 Reachable 或英文 Edge。

详情观测 `ReachBadge`：去掉「可达/不可达」，改成同一对 Tag（Mock 小标可保留）。

每 5 秒只更新 Tag 用的 query，不要 `refetchInterval` 整份实例列表。

## 6. 文档

改 `docs/architecture/runtime.md`：实例在线/Comfy 是否在跑以 Edge presence 为准，不再把管理列表的「通不通」写成控制面探 Comfy。不改表结构（presence 不落库）。

## 7. 验收

- 本机 Mock：Edge 在跑时列表为「节点在线」「Comfy运行中」，停掉 Edge 后约 15 秒内变成「节点掉线」「Comfy未启动」
- 真 Comfy 关掉服务、Edge 仍在：节点在线，Comfy未启动
- 远程只有出站时，两个 Tag 仍随 Edge 报到变化
- 详情观测 Badge 与列表同一套文案和同一数据源
- Mock 与真机探活都走 `comfyui.NewClient`
