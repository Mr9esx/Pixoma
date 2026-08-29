## Purpose

管理端消息平台菜单/卡片编辑器的 UI 行为规范：把「动作」按能力来源清晰分成「工作流能力」与「TG 平台能力」两类，把「菜单项 / 卡片 / 卡片按钮」三层用大纲视图统一导航，左 pane 提供「大纲」与「卡片库」双视图，让操作人员能扫读全部动作分布、精准定位当前编辑对象、卡片管理职责分明，并按动作类型只看到必要的参数；同时彻底清理 `open_workflow` 的 list 模式死代码（`workflow_ids[]` 数组、`mode` 字段、`open_case.list` 分支）。

## ADDED Requirements

### Requirement: 左侧双视图切换（大纲 / 卡片库）
编辑器左侧 pane MUST 在顶部提供 tab 切换两个视图：「大纲」与「卡片库」；默认进入「大纲」；tab 上的数字徽章分别显示大纲节点数 / 卡片总数。

#### Scenario: 默认进入大纲
- **WHEN** 操作人员进入消息平台的菜单编辑器
- **THEN** 左侧 pane 默认显示「大纲」tab 内容，tab 上的数字徽章显示当前菜单项数

#### Scenario: 切换到卡片库
- **WHEN** 操作人员点击「卡片库」tab
- **THEN** 左侧 pane 切换到平铺的卡片网格视图，数字徽章更新为当前卡片总数

#### Scenario: 顶部任意视图都能新建卡片
- **WHEN** 操作人员点击 tab 区右侧的「+ 新建卡片」按钮
- **THEN** 弹出新建卡片对话框，无论当前在大纲还是卡片库都可用

### Requirement: 大纲 = 纯树形结构（不混孤儿卡片）
大纲视图 MUST 只显示被引用的树形结构：菜单项下挂载其 `open_card` 引用的卡片，卡片下挂载其按钮；未挂载的卡片 MUST NOT 出现在大纲里；大纲里没有「未挂载的卡片」之类的并列分组。

#### Scenario: 选中菜单项时右侧展示菜单项编辑器
- **WHEN** 操作人员在大纲中点击某个菜单项
- **THEN** 右侧编辑面板只展示菜单项的 label 与动作选择；卡片与按钮的字段不出现

#### Scenario: 选中按钮时右侧展示按钮编辑器
- **WHEN** 操作人员在大纲中点击某个卡片下的按钮
- **THEN** 右侧编辑面板只展示该按钮的 label 与动作选择；菜单项与卡片的其他字段不出现

#### Scenario: 选中卡片时右侧展示卡片编辑器
- **WHEN** 操作人员在大纲中点击某个被引用的卡片
- **THEN** 右侧编辑面板展示卡片编辑（name / text / media / 按钮列表 + 添加按钮 + 删除卡片）

#### Scenario: 大纲项标注能力来源与层级
- **WHEN** 大纲显示任意节点
- **THEN** 菜单项与按钮节点旁出现能力来源 Badge（工作流 / TG 卡片 / TG 文字 / TG 媒体 / TG 链接 / TG 复制）；卡片节点旁出现「卡片」Badge 区分数据结构层级

### Requirement: 卡片库 = 平铺所有卡片
卡片库视图 MUST 2 列网格展示所有卡片（含未挂载的）；每张卡片 MUST 显示「name / 正文摘要 / 按钮数 / 引用状态（已引用 ● / 未挂载 ○）」；点选任一卡片进入卡片编辑视图。

#### Scenario: 查看所有卡片
- **WHEN** 操作人员切到「卡片库」tab
- **THEN** 看到所有卡片的网格；被引用的卡片标「已引用 ●」并显示被哪个菜单项引用；未挂载的卡片标「未挂载 ○」

#### Scenario: 从卡片库进入编辑
- **WHEN** 操作人员在卡片库点击某张卡片
- **THEN** 右侧编辑面板切到卡片编辑视图（name / text / media / 按钮列表）

#### Scenario: 卡片库新建/编辑/删除
- **WHEN** 操作人员在卡片库点「+ 新建卡片」或点击已有卡片
- **THEN** 弹出新建对话框或进入编辑视图；删除卡片时若该卡被菜单项引用则提示并降级引用

### Requirement: 动作按能力来源分组（单一 Select + optgroup）
动作类型选择器 MUST 用单一 `Select`，内部用 `<optgroup label="工作流能力">` / `<optgroup label="TG 平台能力">` 分组；当前项 MUST 同步显示左侧 9px 圆点（绿/蓝）+ 右侧 `src-tag` 徽章（工作流能力 / TG 平台能力）；编辑器 MUST NOT 使用常驻 Tab 切大类。

#### Scenario: 切换大类只展示该类下的动作
- **WHEN** 操作人员切换到「工作流能力」分组
- **THEN** 动作下拉只显示 `open_workflow`；切换到「TG 平台能力」分组时只显示 5 种 TG 动作

#### Scenario: 当前项同步能力来源信号
- **WHEN** 操作人员将动作切到 `open_workflow` 或任一 TG 动作
- **THEN** 选择器左侧圆点颜色与右侧 `src-tag` 徽章随动作类型同步变化（绿=工作流 / 蓝=TG）

#### Scenario: 大类名称不依赖英文枚举
- **WHEN** 编辑器渲染分组标题
- **THEN** 标题使用本地化文案（中文/英文），不暴露后端 `ActionType` 原始字符串

### Requirement: 6 种动作类型各自精准字段
编辑器 MUST 只渲染当前动作类型所需的参数字段，未涉及类型对应的字段 MUST 不可见；具体要求为：`open_workflow` MUST 渲染工作流**单选**下拉与必要提示；`open_card` MUST 渲染卡片选择（含「+ 新建卡片」快捷入口）+ 卡片下拉填充卡片库全部卡片；`send_text` / `copy_text` MUST 渲染文本框；`send_media` MUST 渲染媒体 URL 输入；`open_url` MUST 渲染 URL 输入；切换动作类型时参数面板 MUST 实时换内容（动画/状态可接受同步切换）。

#### Scenario: 切到 open_workflow 只显工作流单选
- **WHEN** 操作人员将动作切到 `open_workflow`
- **THEN** 编辑器只显示工作流单选下拉（每个菜单项/按钮 1 个 case），不出现多选 Checkbox、不出现文本、媒体、URL 等无关字段

#### Scenario: 切到 send_text 只显文本
- **WHEN** 操作人员将动作切到 `send_text`
- **THEN** 编辑器只显示文本输入，不出现工作流选择、卡片选择、URL 等字段

#### Scenario: 切到 open_card 出现卡片下拉与新建入口
- **WHEN** 操作人员将动作切到 `open_card`
- **THEN** 编辑器显示卡片下拉（从卡片库全部卡片填充）+ 「+ 新建卡片」快捷入口；点新建入口跳到新建对话框

#### Scenario: 切换动作类型参数面板实时换内容
- **WHEN** 操作人员在动作类型 Select 上切换 6 种动作中的任意一种
- **THEN** 右侧参数面板内容立即换为对应动作的字段；不相关字段彻底不出现

### Requirement: 工作流永远单 case（彻底删除 list 模式）
编辑器 MUST 强制 `open_workflow` 动作为工作流单选：每个菜单项或按钮 MUST 绑定 1 个工作流（case），不存在多选 Checkbox 控件；这与 `quick-config/lib/menu-payload.ts` 的单数 `workflowId` 与 `internal/channel/capability/open_case.go` 的 `start case_id` 一致。编辑器 MUST NOT 提供「打开方式 direct/list」切换——list 模式在 `open_workflow` 上无产品语义；多工作流场景的承载点是 `Card.buttons[]`（1 个菜单项 → open_card → 1 张卡片 → N 个按钮，每个按钮 1 个 case）。

#### Scenario: 工作流选择是单选下拉
- **WHEN** 操作人员在 `open_workflow` 动作下选择工作流
- **THEN** 编辑器提供单选下拉，列出全部可用 case；任何时刻最多 1 个被选中

#### Scenario: 不再有 list 模式切换
- **WHEN** 操作人员查看 `open_workflow` 动作的参数面板
- **THEN** 没有任何「打开方式 direct/list」切换控件；点了菜单项就 `start case_id` 进入流程

### Requirement: 实时预览与现有存盘/校验行为
编辑器 MUST 保留「实时预览」能力并按选中层级切换视图：选中菜单项时展示主键盘视图；选中卡片时展示卡片视图（卡片正文 + 按钮列表）；选中按钮时按钮高亮；草稿未保存到预览时降级展示「当前编辑未保存到预览」。编辑器 MUST 不倒退既有存盘 / 校验 / 错误展示 / `unsaved changes` 提示；`validateMenuConfig` 与 `menu-editor.contract.test.ts` 中关于动作面板、卡片选择、卡片列表的关键 UI 契约 MUST 继续满足。

#### Scenario: 选中菜单项时预览主键盘
- **WHEN** 操作人员在大纲中选中某个菜单项
- **THEN** 模拟键盘视图上对应按钮被高亮，其余按钮保持普通态

#### Scenario: 选中卡片时预览卡片视图
- **WHEN** 操作人员选中某张卡片
- **THEN** 模拟键盘视图切到卡片视图：卡片正文 + 按钮列表

#### Scenario: 选中按钮时按钮高亮
- **WHEN** 操作人员选中某个按钮
- **THEN** 模拟键盘的卡片视图上对应按钮高亮

#### Scenario: 校验失败不破坏新分组
- **WHEN** 菜单缺少必填项并触发校验错误
- **THEN** 错误展示与字段定位按既有 `validateMenuConfig` 契约返回，且动作分类的 optgroup 展示不被打乱

#### Scenario: 既有动作面板 testid 保留
- **WHEN** 任何渲染动作选择的面板被打开
- **THEN** 关键 `data-testid`（`action-form`、`card-picker`、`card-list-panel` 等）继续存在以支持既有 contract 测试
