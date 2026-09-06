---
comet_change: link-health-single-source
role: technical-design
canonical_spec: openspec
---

# 链路健康唯一主人 深度技术设计

## 1. 目标

配置链路上的绿黄只由控制面组装器计算。后台列表、详情、拓扑只渲染。用活路不变式防止「某一跳坏了只修看见它的那一页」。通道临时去看 `adapter_state` 不是本 change 的交付物。

Canonical 行为见 OpenSpec delta。高层见该 change 的 `design.md`。图见 `docs/openspec/changes/link-health-single-source/diagrams/link-health.html`。

## 2. 现状（会再漏页的结构）

| 位置 | 怎么算 | 结构问题 |
|---|---|---|
| 工作流列表/详情 | `caseReferences`（已临时看通道进程） | 只堵住这一格 |
| 队列列表/详情 | `topicReferences`（Case + Edge + presence） | 完全不看平台 |
| 节点列表 | `listHealthTone`（自己启停/在线/Comfy） | 不看图 |
| 节点详情 | `edgeReferences` | 不看平台 |
| 通道列表 | `enabled && adapter_state` | 与详情不同源 |
| 通道详情 | `channelReferences(reachability)` | 探测只活在本页 query |
| 拓扑 | 平台用探测缓存，工作流用 `caseReferences` | 同一张图两套规则 |

缺数据当可用（如找不到通道当绿灯）也属于同一结构：页面拼不齐输入时撒谎。

## 3. 模块切分

```text
internal/packaging/linkhealth/   拼图、合成起点、走活路、不变式
internal/httpapi/linkhealth/     GET /api/v1/link-health
web/admin  删除运行时 *References / listHealthTone
           只映射 DTO → StatusDot / LinkHealthSection / LinkGraph
```

不新建写侧限界上下文。CRUD 仍在 Channel / MenuCard / Catalog / Topic / Edge。

## 4. 数据流

```text
只读端口：通道(+适配器+组装器读得到的探测)
         菜单树、Case.routing、Topic、Edge、presence
        │
        ▼
packaging/linkhealth
  1. 边：平台—菜单→工作流→队列→节点
  2. 各点局部可用（组装器内合成，禁止页面另持定义）
  3. 从可用平台走到就绪节点 = 活路
  4. 节点展示 = 局部过关 ∧ 有活路穿过（平台用合成结果）
        │
        ▼
GET /api/v1/link-health
  nodes[].health / upstream / downstream
  edges[]
        │
        ├─ 四类列表圆点
        ├─ 详情状态与关联
        └─ 拓扑（高亮 path-through 仍可前端做，不算健康）
```

节点 id：`platform:{id}` / `case:{id}` / `topic:{key}` / `edge:{id}`。

## 5. 不变式与测试

固定规则，不绑死通道：

- 唯一路径上任一 hop 不可用 → 必须经过它的工作流/队列不得 `ok`。
- 另有活路 → 资源自身可 `ok`，坏 hop 只在引用里 `warn`。
- 组装器读不到所需运行态 → `pending`，不得 `ok`。

Go 表测至少三条：唯一入口不可用、双入口有活路、唯一节点不可用。再加一条输入缺失。禁止用「通道 error 则 case warn」当唯一验收。

HTTP：有会话 200；无会话拒；热路径不得调用外部探测函数。

前端合同：三处使用同一 `queryKeys.linkHealth`；源码不再调用 `caseReferences(` 等到运行路径；失败不标绿。

## 6. 运行信号可见性

探测、适配器、心跳都必须是组装器端口读得到的，而不是某一 React Query 缓存。

实现可选：通道上次探测随通道读模型提供；或读不到则全图相关点 pending。**选哪种不改变本 change 的标题。** 查询热路径禁止打 Telegram/飞书。

presence 保持控制面内存，与现网一致。

## 7. 前端删除发明权

删除（运行时）：

- `caseReferences` / `topicReferences` / `edgeReferences` / `channelReferences` / `listHealthTone` 在页面中的调用
- `useTopologySource` 的多路列表拼图（改为一次健康查询）
- 工作流列表为每个 case 打 `menu-placements` 来算圆点

保留：`LinkHealthSection`、`StatusDot`、只读 `LinkGraph`、i18n 断点文案（key 由组装器返回）。

`check` 若仍存在，成功后只 invalidate 通道与 `linkHealth`，让组装器下一拍读到新信号。

## 8. 边界

- 度数 0 的资源：查询仍返回（列表需要）；全量拓扑可继续不画孤立点。
- 健康 `bad`（实体缺失）图上按黄档。
- Bot / Orchestrator 本期不读本查询。
- 不把四类资源合成一个写 API。

## 9. 发布与回滚

先上组装器与测试，再切前端。回滚前端即可回到旧拼装；组装器接口可留。
