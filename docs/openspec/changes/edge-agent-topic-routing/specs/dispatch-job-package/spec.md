## ADDED Requirements

### Requirement: 云上组装任务包并写入 Blob（双模式）
在发布 dispatch 之前，控制面 MUST 组装方案 A 任务包并写入当前模式配置的 Blob。任务包 MUST 包含：已注入非图片输入的 workflow；图片的节点/字段与输入 `BlobRef`；产物前缀约定。任务包 MUST NOT 内嵌图片二进制。allinone 与 split MUST 共用该语义。

#### Scenario: prep 写出 job 对象
- **WHEN** 调度准备投递某 Task 且 Case 与 staged 输入完整
- **THEN** Blob 中存在该 Task 的任务包，且 workflow 已含非图片字段值

#### Scenario: 缺少必填输入则不发布 dispatch
- **WHEN** 必填 staged 输入缺失
- **THEN** 不发布 dispatch，并记录可诊断原因

### Requirement: Dispatch 携带 job_ref
`DispatchCommand` MUST 包含 `job_ref`。执行面 MUST 仅依据 `job_ref` 与任务包获取执行材料，成功主路径 MUST NOT 依赖 `input_prefix` 回查 Case 库。

#### Scenario: 执行面凭 job_ref 拉取材料
- **WHEN** 执行面收到含有效 `job_ref` 的 dispatch
- **THEN** 能 Get 到任务包，并按 images 列表从 Blob 拉取图片

#### Scenario: job_ref 无效则失败 status
- **WHEN** `job_ref` 指向不存在对象或任务包无法解析
- **THEN** 上报 failed status，且不向 ComfyUI 假装成功提交

### Requirement: 执行面完成本机图片注入
对任务包中的每张图片，执行面 MUST 上传到本机 ComfyUI（或 Mock），将本地引用写入 workflow 后再 Submit。

#### Scenario: 图片注入后提交
- **WHEN** 任务包含一张 image 绑定
- **THEN** Submit 前该节点字段为本机 Upload 返回的可用引用
