# 进程拆分修正（Scheduler vs Actuator）

## 角色（按职责，不是按「bot/worker」粗切）

| 角色 | 基数 | 职责 |
|------|------|------|
| **Bot** | 可多副本（无状态或粘会话在 DB） | TG Adapter + App + Session；建 Task；读状态通知用户 |
| **Scheduler** | **通常 1 个**（或主备） | 看 pending Task；路由 Topic/实例；投递 dispatch；queued |
| **Actuator** | **跟 ComfyUI 实例走**（1 实例 ↔ 1 Actuator，或同机 sidecar） | 只消费「自己实例」的 Topic；调本机/本侧 ComfyUI；写 OSS；回传状态 |

```text
Bot(s) ──建 Task(pending)──► DB/Blob
                │
                ▼
         Scheduler（单）
                │  按路由 Publish → topic: comfy.dispatch.<instance_id>
                ▼
    ┌───────────┼───────────┐
    ▼           ▼           ▼
 Actuator@A  Actuator@B  Actuator@C
    │           │           │
 ComfyUI-A   ComfyUI-B   ComfyUI-C
    │           │           │
    └───────────┴───────────┴──► Blob outputs + status → Bot 通知
```

## 一期（单 ComfyUI）怎么落

仍可 **一个进程 all-in-one**，里面三个角色都是 goroutine：

- Bot + **一个** Scheduler + **一个** Actuator（绑唯一 ComfyUI）

此时 Memory Queue 的 topic 可以先写成 `comfy.dispatch.default`。

## 二期拆进程时正确切法

```text
1) bot          — 可水平扩
2) scheduler    — 单副本（或 leader election）
3) actuator     — 每 ComfyUI 实例部署一份（配置里写 instance_id + comfyui base URL + subscribe topic）
```

不要做成「一个大 worker = Scheduler+所有 Actuator」——除非只有一台 ComfyUI。

## Worker 这个词

若还用 worker 统称：更准确是 **Actuator worker**；Scheduler 单独叫 scheduler，避免和「跟实例走的执行器」混在一个可无限扩的池子里抢同一队列却没有实例亲和。

## Queue 含义（配合此模型）

- `comfy.dispatch.<instance_id>`：Scheduler → 指定 Actuator
- `comfy.status`（或带 task_id）：Actuator → 控制面/Bot 侧消费者
- Phase1 只有一个 instance_id 时仍保留 topic 形状，二期加实例不改端口
