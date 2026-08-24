## ADDED Requirements

### Requirement: Case 编辑页可配置处理流程
Case 编辑/详情页 MUST 提供「处理流程」配置节，复用任务分流编辑器编辑 routing 规则（条件 → Topic），并支持保存到 Case（`PATCH /cases/{id}`）。

#### Scenario: 编辑并保存路由规则
- **WHEN** 运维在 Case 编辑页打开「处理流程」并完成规则配置
- **THEN** 规则保存在该 Case 上，保存后回读一致

#### Scenario: 未导入 workflow 也可编辑路由
- **WHEN** Case 尚未导入合法 workflow JSON
- **THEN** 路由编辑仍可用（规则与 workflow 图相互独立），保存仅写 routing

### Requirement: 主键盘树管理页
控制台 MUST 提供主键盘树管理页：编辑树形菜单项（文件夹、子项、挂载 Case、placeholder、reply_media），保存调用菜单 API；失败展示错误且不假装成功。

#### Scenario: 编辑文件夹并挂载 Case 后保存
- **WHEN** 运维配置文件夹及其 Case 关联并保存成功
- **THEN** 页面提示成功，刷新后树与挂载仍在

#### Scenario: 保存失败展示错误
- **WHEN** 菜单 API 返回校验或网络错误
- **THEN** 页面展示错误信息，不进入「已保存」误导态
