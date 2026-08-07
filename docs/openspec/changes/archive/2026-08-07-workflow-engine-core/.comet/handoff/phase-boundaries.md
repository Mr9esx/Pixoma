# 一期 / 二期边界（架构手稿）

## 一期（建议）：可演示的垂直切片

目标：私聊 Bot 能走通「选 Case → 填表上锁 → 提交 Task → 出结果」，异步骨架在，但部署形态可单机简化。

| 模块 | 一期范围 |
|------|----------|
| Case Protocol + Registry | 完整：schema/校验/tags/price/绑定；GORM+SQLite |
| Dialog Session | 完整：锁、collecting/confirming、Exit、温和无关 |
| Application API | Menu/Categories/Cases/Session/ConfirmRun/Tasks |
| TG Adapter | 菜单/列表/详情/填表/拦截/进度与结果消息 |
| Task | 状态机 + 温和取消（pending/queued） |
| Object Store | 接口 + **本地 FS** 实现 |
| Scheduler + MQ + Actuator | **单机闭环**：可用进程内队列或本地 MQ；**单 Actuator + 单 ComfyUI** |
| 扣费 / 多实例路由 / 多租户部署 | 不做 |

一期验收：用户在 TG 走完至少一个 text2img Case，收到图片；中途开新 Case 被锁；Exit 后可重开；提交后可并行再开 Case。

## 二期

- 生产 OSS、生产 MQ
- Scheduler 按 Topic/实例路由；多 Actuator
- 积分/收费（与取消、succeeded 对齐）
- Case 管理后台、多租户 Bot 部署
- 可选：TG WebApp 卡片 UI；running 强取消（若收费模型需要再议）

## 明确不做（更远期）

- 把 Domain 绑死 Telegram 类型
- 一期上强取消 / 真实支付
