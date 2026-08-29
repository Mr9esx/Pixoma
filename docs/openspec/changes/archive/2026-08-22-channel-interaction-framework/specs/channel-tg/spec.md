## MODIFIED Requirements

### Requirement: 主菜单来自 Menu 配置
Telegram 适配器展示的主 ReplyKeyboard MUST 由**消息平台作用域**菜单的**能力入口**根项生成（含默认种子），根项数量 MUST 不超过 6 个直达入口；MUST NOT 再以源码常量作为唯一长期配置源；菜单读取以消息平台 id 为作用域。

#### Scenario: /start 展示配置中的主键盘
- **WHEN** 用户发送 `/start` 或打开主菜单，且该消息平台菜单配置可用
- **THEN** 用户收到与配置能力入口顺序/文案一致的 ReplyKeyboard（直达入口 ≤6）

### Requirement: 文件夹下钻浏览 Case
当用户点击分组（文件夹）菜单项时，适配器 MUST 以消息按钮展示该分组的子项与 open_case 能力入口，并提供返回上一级或主菜单的入口。点击能力入口 MUST 进入对应能力流程（open_case 进入既有 Case 预览或填表工作流）。

#### Scenario: 进入文件夹看到子项与 Case
- **WHEN** 用户点击根层分组项，且该分组配置了子分组与/或 open_case 能力入口
- **THEN** 用户收到消息按钮，其中可区分分组与 Case 入口，且可返回

#### Scenario: 进入子文件夹并可返回
- **WHEN** 用户在消息按钮中点击子分组，再点击返回
- **THEN** 用户回到上一层分组视图或主菜单（按设计的返回语义），过程 MUST NOT 崩溃

#### Scenario: 占位与回复媒体
- **WHEN** 用户点击占位或回复媒体展示项
- **THEN** 行为与既有占位提示 / 发文本与图片 URL 语义一致；单张图片失败 MUST NOT 导致进程崩溃

## ADDED Requirements

### Requirement: 能力入口与流程渲染
TG 适配器 MUST 按能力的消息平台渲染声明渲染入口与流程（消息按钮、返回、结果），MUST NOT 为具体业务编写分支；新注册能力加入菜单后，适配器无需改动即可完成事件翻译与结果渲染。

#### Scenario: 新能力入口直接可用
- **WHEN** 管理台把新注册能力加入 TG 菜单并保存
- **THEN** 主键盘或消息按钮出现该入口，点击触发对应能力，无需改适配器
