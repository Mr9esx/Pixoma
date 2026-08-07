# Task 状态机（架构手稿）

## 与 Session 关系

- ConfirmRun：Session confirming → 建 Task(pending) → Session submitted（解锁）
- 同一 chat 可同时有多个非终态 Task；不阻止新 Session

## 状态

| 状态 | 谁推进 | 含义 |
|------|--------|------|
| `pending` | 控制面 | 已建 Task；输入已（或正在）物化到 OSS；待调度 |
| `queued` | Scheduler | 已按 Topic 写入 MQ，等 Actuator |
| `running` | Actuator | 已调 ComfyUI，已有 comfy_prompt_id，监听中 |
| `succeeded` | Actuator | 产物已在 OSS；output 引用齐全 |
| `failed` | 任一层 | 终态失败 |
| `cancelled` | 控制面/用户 | 在允许窗口内取消 |

## 转移

```
ConfirmRun
    → pending
         → queued          〔Scheduler 投递成功〕
         → failed          〔物化 OSS / 投递失败〕
         → cancelled
    queued
         → running         〔Actuator 开始执行〕
         → failed
         → cancelled       〔若尚未被消费，可取消〕
    running
         → succeeded
         → failed
         → cancelled       〔若 Comfy/业务支持中断；否则可仅标记取消意向〕
```

## 最小字段

`task_id`, `chat_id`, `user_id?`, `case_id`, `case_snapshot`,
`status`, `input_uris[]`, `output_uris[]`, `comfy_prompt_id?`,
`error_code?`, `error_message?`, `topic?`, timestamps

## OSS 约定（草案）

- `inputs/{task_id}/...`
- `outputs/{task_id}/...`
- 本地开发：文件系统根目录模拟 Bucket

## 取消策略（待你确认的一点）

- A（简）：仅 `pending`/`queued` 可 cancelled；`running` 只能等结束或标 failed
- B：running 也尝试取消（依赖 ComfyUI interrupt API）
