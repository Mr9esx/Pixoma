## ADDED Requirements

### Requirement: Session 草稿支持图片 Blob 值
对话会话 MUST 允许将某个 input 键的 Draft 存为媒体 Blob 引用（与文本/数值/布尔并存）。跳过规则对允许跳过的非必填图片字段仍然有效；必填图片缺少 Blob 时不得进入可成功 Confirm 的状态。

#### Scenario: 提交图片草稿后索引前进
- **WHEN** 会话处于 collecting，当前键类型语义为图片，且提交带 Blob 的 Draft
- **THEN** 该键被记录，会话前进到下一输入或 confirming

#### Scenario: 跳过可选图片字段
- **WHEN** 当前键为非必填且允许跳过的图片字段，用户选择跳过
- **THEN** 会话不要求该 Blob，并前进到下一输入或 confirming
