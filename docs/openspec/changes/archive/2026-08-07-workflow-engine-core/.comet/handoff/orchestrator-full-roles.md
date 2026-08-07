# 完整系统角色与交互（不按单/多机裁剪）

## 三角色（控制面完整版）

| 角色 | 基数 | 职责边界 |
|------|------|----------|
| **Bot** | 可多副本 | 渠道：TG Session/菜单/填表；创建执行请求；**只负责对用户说话**（收通知指令后发 TG） |
| **Orchestrator**（原 Scheduler 放大） | 通常单点/主备 | **任务控制面**：准入、选实例、投递、跟踪状态、超时/失败策略、温和取消、**驱动通知** |
| **Actuator** | 每 ComfyUI 实例一份 | **执行面**：消费本实例任务、调 ComfyUI、写 Blob、回报进度/终态；**不碰 TG** |

命名：完整系统里不宜叫瘦「Scheduler」；用 **Orchestrator** 更准确。调度只是它的子集。

## 完整主路径

```text
用户
  │
  ▼
 Bot ──① 校验+Session结束+建Task(pending)+inputs→Blob
  │         写入「可编排」事实（DB）
  │
  ▼
 Orchestrator
  │  ② 发现 pending / 订阅 created
  │  ③ 选 instance（能力标签、负载、健康）
  │  ④ Publish dispatch.<instance_id> ；Task→queued
  │
  ▼
 Actuator@Inst
  │  ⑤ 消费；Task→running；拉 Blob；调 ComfyUI
  │  ⑥ 进度可选上报（running/progress）
  │  ⑦ 产物→Blob；上报终态 succeeded|failed
  │
  ▼
 Orchestrator
  │  ⑧ 收敛终态（写 DB、超时对账、失败重试策略若有）
  │  ⑨ Publish notify / 调用通知端口：请 Bot 向 chat 回消息
  │
  ▼
 Bot ──⑩ 读 Blob/结果摘要 → sendPhoto/editMessage → 用户
```

## Orchestrator 完整能力清单（不是「只投递」）

1. **编排入口**：感知新 Task（DB outbox / 队列 / 内部事件）  
2. **调度**：按 Case 标签、实例能力、负载、熔断选 `instance_id`  
3. **投递**：`dispatch.<instance_id>`；维护 queued  
4. **状态收敛**：消费 Actuator 的 status/progress；更新 Task；处理乱序与幂等  
5. **超时与失败策略**：派中超时、执行超时、可重试错误重新调度（换实例或同实例）  
6. **温和取消**：pending/queued 取消落地（勿派发或发 cancel-dispatch）；running 按策略忽略 interrupt  
7. **通知驱动**：终态（及可选关键进度）→ **Notify 端口** → Bot 发 TG（Orchestrator **不持有** Bot Token 发消息更干净）  
8. **实例注册表**：Actuator/ComfyUI 健康与能力（完整系统需要）

## Actuator 完整能力

1. 只订自己的 `dispatch.<instance_id>`  
2. 执行与 ComfyUI 会话（prompt_id、监听）  
3. 输入输出 Blob  
4. 向上回传：`accepted` / `running` / `progress?` / `succeeded` / `failed`  
5. 本地并发限额（单实例队列深度）

## Bot 完整能力

1. TG 全渠道与 Session 锁  
2. ConfirmRun → 落 Task+Blob（或丢给 Orchestrator 的 create API）  
3. **通知执行器**：消费 `notify.*`，把结果打给用户  
4. 查询我的任务（读 DB）

## 队列（完整）

| Topic | 方向 | 用途 |
|-------|------|------|
| `task.created` 或 DB 监听 | Bot/App → Orchestrator | 新任务可编排（也可用 DB 扫描） |
| `dispatch.<instance_id>` | Orchestrator → Actuator | 下发执行 |
| `status`（或 `status.<instance_id>`） | Actuator → Orchestrator | 进度与终态 |
| `notify.telegram`（或 `notify.user`） | Orchestrator → Bot | 请渠道侧回用户 |

## 依赖方向

```text
Bot → DB/Blob ；Bot ← notify
Orchestrator → DB/Blob/Queue ；Orchestrator ← status ；Orchestrator → notify
Actuator → ComfyUI/Blob/Queue(status) ；Actuator ← dispatch

禁止：Actuator → Bot/TG
禁止：Orchestrator 直接发 TG（可选严格）
```

## 和「瘦 Scheduler」的差别

| | 瘦 Scheduler | Orchestrator（完整） |
|--|--------------|----------------------|
| 选实例投递 | 有 | 有 |
| 跟踪/对账终态 | 无/弱 | **有** |
| 超时重试 | 无 | **有** |
| 驱动回用户 | 无 | **有（经 notify→Bot）** |
| 实例健康 | 无 | **有** |
