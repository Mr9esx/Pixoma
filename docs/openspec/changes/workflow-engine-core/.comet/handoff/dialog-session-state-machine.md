# Dialog Session 状态机（架构手稿）

> 私聊 Bot：每个 `chat_id` 最多一条非终态 Session。  
> 与 Case 定义状态、Task 执行状态分离。  
> Comet design 暂停期间暂存于此；敲定后迁入 Design Doc / 正式 docs。

## 状态

| 状态 | 含义 | 是否上锁 |
|------|------|----------|
| （无记录） | 可浏览菜单/列表 | 否 |
| `collecting` | 已 StartCase，逐项收 input | **是** |
| `confirming` | 输入齐，待确认执行 | **是** |
| `submitted` | ConfirmRun 已建 Task | 终态，释放锁 |
| `exited` | 用户主动退出 | 终态，释放锁 |

不上锁浏览：不建 Session。  
可选不引入 `browsing` 状态，减少复杂度。

## 转移

```
(无 session)
    │ List*/GetMenu              → 仍无 session
    │ StartCase(case_id)         → collecting  〔Case 须 active〕
    ▼
collecting
    │ SubmitInput / SkipInput    → collecting（推进 index）
    │ 全部就绪                   → confirming
    │ ExitSession                → exited
    │ StartCase                  → SESSION_LOCKED
    ▼
confirming
    │ ConfirmRun 成功            → submitted（创建 Task=pending）
    │ 返回修改（可选）            → collecting
    │ ExitSession                → exited
    │ StartCase                  → SESSION_LOCKED
```

## 最小字段

`session_id`, `chat_id`, `case_id`, `case_snapshot`, `status`,
`current_input_index`, `draft_inputs`, `active_message_id?`, timestamps

## 与 Task

- ConfirmRun 成功 → 建 Task → Session 终态并解锁
- 有 running Task **默认不挡** 新开 Case
- 「同时只跑一个任务」属 Task 限流，不并入 Session 锁
