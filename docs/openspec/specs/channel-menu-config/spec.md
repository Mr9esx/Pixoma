# channel-menu-config Specification

## Purpose
渠道菜单配置定义每个渠道作用域下的可配置菜单树：平台中立的目录与动作语义、Case 挂载与反查。菜单编辑入口位于渠道详情内，管理台不再存在独立的「主键盘」顶级模块。
## Requirements
### Requirement: 菜单按渠道作用域存储
系统 MUST 将菜单配置归属到渠道：每个渠道一份菜单文档，包含树形菜单项与菜单项–Case 关联；菜单读取与替换均以渠道 id 为作用域，不同渠道互不影响。

#### Scenario: 渠道各自菜单独立
- **WHEN** 存在两个渠道并分别保存菜单
- **THEN** 各渠道读取到的菜单互不串扰

#### Scenario: 不存在渠道被拒绝
- **WHEN** 对不存在的渠道 id 读取或保存菜单
- **THEN** 返回 404 且不创建隐式菜单

### Requirement: 菜单模型平台中立
菜单领域模型 MUST 使用平台中立的目录/动作语义（文件夹、打开 Case、占位提示、回复媒体），中立核心字段 MUST NOT 包含平台专有字段（如 Telegram 行/列网格、回调编码、消息长度上限）；平台差异数据 MAY 通过渠道扩展数据（extras）按渠道存放，MUST NOT 进入中立核心字段；平台渲染差异（键盘形态、按钮布局、回调数据、媒体上传）MUST 由对应渠道适配器处理，且 MUST NOT 因通用化抹平平台特色能力。

#### Scenario: 中立字段不含 TG 专有字段
- **WHEN** 通过管理 API 读取或提交菜单的中立字段
- **THEN** 中立字段中不存在 row/col、回调前缀或 TG 消息长度约束；平台差异仅出现在该渠道的 extras 数据中

#### Scenario: TG 特色能力保留
- **WHEN** 管理员为 TG 渠道配置根层网格布局等 TG 特色数据
- **THEN** 数据存入该渠道 extras，TG 适配器按 extras 渲染；其他渠道读取不到且不受影响

#### Scenario: 同一菜单可被多平台适配器渲染
- **WHEN** 同一份渠道菜单（folder + open_case + placeholder）分别由 TG 与后续平台适配器消费
- **THEN** 各适配器能按各自平台形态渲染，无需改写菜单配置

### Requirement: Case 挂载与反查
系统 MUST 支持菜单项挂载 Case 并从 Case 反查其出现的菜单路径；挂载的 Case 必须存在，否则拒绝保存。

#### Scenario: 保存文件夹并挂载 Case
- **WHEN** 管理面在渠道菜单中创建文件夹并关联已存在的 Case 后保存
- **THEN** 持久化成功，后续读取可见文件夹及其 Case 关联

#### Scenario: 绑定不存在的 Case 被拒绝
- **WHEN** 提交的关联 `case_id` 不存在
- **THEN** 系统拒绝保存并返回可理解的校验错误

#### Scenario: 查询 Case 的菜单挂载
- **WHEN** 调用方请求某 Case 的菜单挂载
- **THEN** 系统返回零条或多条路径；每条能标识所在渠道、菜单项与可读路径

### Requirement: 默认种子按渠道
系统在渠道无自定义菜单时 MUST 提供可用的默认种子菜单；种子包含「图片」文件夹入口并可挂载图片类 Case。

#### Scenario: 空配置使用默认种子
- **WHEN** 渠道尚无自定义菜单
- **THEN** 运行时仍能得到可用的根菜单项集合

### Requirement: 渠道差异数据（extras）管理
系统 MUST 支持按渠道为菜单项保存平台差异数据（extras）：每条 extras 以（channel_id, menu_item_id, extra_type）唯一标识，内容为 JSON；仅对应渠道的适配器读取，其他渠道不得受影响。无效或未知的 extras MUST 被忽略或按适配器校验规则拒绝，MUST NOT 影响中立字段语义。管理台 MUST 提供 extras 的可视化编辑入口。

#### Scenario: 按渠道保存与读取 extras
- **WHEN** 管理面为 TG 渠道某根菜单项保存网格布局 extras 后再次读取
- **THEN** 该渠道可读回该 extras，其他渠道读取不到

#### Scenario: 管理台编辑 extras
- **WHEN** 管理员在 TG 渠道菜单中编辑根层网格布局
- **THEN** 变更写入该渠道 extras，保存后按适配器刷新策略生效

#### Scenario: 无效 extras 不影响中立字段
- **WHEN** extras 内容为无效或未知类型
- **THEN** 适配器忽略或拒绝该 extras，菜单中立字段行为不受影响

