## 1. 身份与会话键

- [ ] 1.1 先写测试：群消息 `from.id` ≠ `chat.id` 时账户用 from；私聊仍用 chat.id
- [ ] 1.2 适配器 `account()` 改为操作者 user id；notify / 发送仍用聊天 ChatID
- [ ] 1.3 填表会话键：私聊按 chat，群按 chat+user；覆盖「同群两人互不抢锁」

## 2. 入站过滤与命令

- [ ] 2.1 测试：群闲聊忽略；@bot、/命令、回复 bot 进入处理；channel 忽略
- [ ] 2.2 实现群过滤；解析 `/run` 与 @ 文本到已启用 Case
- [ ] 2.3 群路径不发送 ReplyKeyboard；私聊 `/start` 键盘回归测试保持绿

## 3. 群填表与回图

- [ ] 3.1 群缺参时在该群提问，只接受该操作者对 bot 的回复或再次 @
- [ ] 3.2 ConfirmRun 后终态图发回发起群；幂等 notify 不变
- [ ] 3.3 `comfy_mock` 下用群触发跑通至少一条出图路径

## 4. 文档

- [ ] 4.1 更新 `docs/architecture/runtime.md`：Telegram 入站身份与群投递
