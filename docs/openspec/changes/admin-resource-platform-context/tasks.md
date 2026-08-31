## 1. API 契约测试

- [ ] 1.1 扩展 Task admin-api 测试，断言列表和详情返回 `channel_id`、`user_id`、`session_id`
- [ ] 1.2 扩展 Session admin-api 测试，断言列表和详情返回 `channel_id`
- [ ] 1.3 扩展 User admin-api 测试，断言返回 `channel_id`、`external_user_id` 和用户资料，且平台无关 `q` 搜索可命中标识与资料字段

## 2. 后端投影与查询

- [ ] 2.1 为 Session 领域投影补充 `channel_id` 并完成持久化映射
- [ ] 2.2 为 Task admin 列表和详情通过 Session 关联返回消息平台与用户标识
- [ ] 2.3 为 User admin 列表和详情补充消息平台身份投影
- [ ] 2.4 将 User 搜索契约调整为覆盖内部 ID、外部用户 ID、用户名和姓名

## 3. 前端资源上下文

- [ ] 3.1 更新 Task、Session、User 的 API 类型和查询参数
- [ ] 3.2 在 Task 列表和详情展示消息平台、关联用户与关联 Session
- [ ] 3.3 在 Session 列表和详情展示消息平台
- [ ] 3.4 在 User 列表和详情展示消息平台与统一「用户信息」列，并移除 `tg_user_id` 展示与筛选
- [ ] 3.5 同步中英文 i18n 文案并保留空态展示

## 4. 验证

- [ ] 4.1 运行受影响后端测试
- [ ] 4.2 运行受影响前端契约测试和静态检查
- [ ] 4.3 运行 OpenSpec 校验并核对三个列表的核心验收场景
