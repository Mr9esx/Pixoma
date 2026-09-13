## 1. 多平台管道

- [ ] 1.1 测试：非 telegram 快照不得构造 TG Bot、不得请求 getMe
- [ ] 1.2 工厂按 `platform` 分支；凭证结构支持飞书 App ID/Secret
- [ ] 1.3 探测按平台；飞书走开放平台身份校验
- [ ] 1.4 后台创建表单可选飞书并提交双字段；非法平台/空凭证 4xx

## 2. 飞书适配器

- [ ] 2.1 接入官方 SDK 长连接；启用热挂、停用断开
- [ ] 2.2 实现 Outbound/MediaBridge：文本、卡片按钮、上传+发图
- [ ] 2.3 入站翻译为 `open_case`；卡片新版回调；3 秒内返回
- [ ] 2.4 私聊与群 @ 出图；身份用飞书用户 id，投递用 chat id

## 3. 验收与文档

- [ ] 3.1 `comfy_mock` 下飞书主路径出图
- [ ] 3.2 更新 `docs/architecture/overview.md`、`runtime.md`、`bounded-contexts.md`、`data-model.md`
