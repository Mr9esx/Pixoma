## MODIFIED Requirements

### Requirement: 布局与基础主题
系统 MUST 提供含侧栏的管理布局：侧栏包含品牌区、导航菜单与底栏工具区；内容区 MUST NOT 再提供仅承载语言/主题切换的顶栏。系统 MUST 继续支持基座自带的基础主题切换能力（若基座提供明暗主题）。语言切换与主题切换控件 MUST 放置在侧栏底栏；当侧栏收起为图标模式时，上述控件 MUST 仍可以图标形式被发现并操作。

#### Scenario: 布局稳定包裹页面
- **WHEN** 用户在资源页之间切换
- **THEN** 侧栏布局保持，内容区切换为目标页，且内容区上方不出现仅含语言/主题的顶栏

#### Scenario: 侧栏底栏可切换语言与主题
- **WHEN** 用户打开控制台并查看侧栏底栏
- **THEN** 可见语言切换与主题切换入口，且操作后界面语言或主题相应变化

#### Scenario: 侧栏收起后仍可操作底栏工具
- **WHEN** 侧栏处于收起（图标）模式
- **THEN** 用户仍可通过底栏对应图标入口切换语言或主题

## ADDED Requirements

### Requirement: 侧栏品牌为 Pixoma
侧栏品牌区 MUST 展示项目名称「Pixoma」，且 MUST NOT 展示 shadcn-admin 模板默认文案（如「Shadcn-Admin」「Vite + ShadcnUI」）。

#### Scenario: 侧栏可见 Pixoma
- **WHEN** 用户打开管理控制台
- **THEN** 侧栏顶部品牌区显示「Pixoma」，且不显示上述模板默认主/副标题文案
