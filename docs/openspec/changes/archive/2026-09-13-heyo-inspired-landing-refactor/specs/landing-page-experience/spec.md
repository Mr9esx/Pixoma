## Purpose

定义 Pixoma 对外落地页重构后的完整访客体验，使页面以清晰的信息层级介绍产品、保留可操作演示，并在不同语言、设备和输入方式下稳定引导访客进入下载或自托管入口。

## ADDED Requirements

### Requirement: 页面按确认的信息架构呈现
落地页 SHALL 按导航、首屏、功能展示、跨设备使用、适用人群、FAQ、最终 CTA、页尾的顺序呈现内容。

#### Scenario: 访客浏览完整页面
- **WHEN** 访客打开任一受支持语言的落地页并向下浏览
- **THEN** 页面按规定顺序呈现全部八个区域

#### Scenario: 排除工具集成与价格区
- **WHEN** 访客浏览页面内容和导航入口
- **THEN** 页面不呈现「Works with your favorite tools」工具集成区、「Affordable for everyone!」价格区或指向这两个区域的导航入口

### Requirement: 页面保留 Pixoma 品牌与产品叙事
落地页 SHALL 使用 Pixoma 的品牌、产品能力、行动入口和原创双语文案，且 SHALL NOT 把 Heyo 的品牌、livechat 产品叙事、文案或素材作为 Pixoma 内容呈现。

#### Scenario: 首屏传递 Pixoma 价值
- **WHEN** 访客进入首屏
- **THEN** 访客看到 Pixoma 名称、ComfyUI 创作相关价值主张，以及下载和自托管行动入口

#### Scenario: 参考内容不被照搬
- **WHEN** 访客浏览任一内容区
- **THEN** 区块内容描述 Pixoma 的真实能力，而不是 Heyo 的 livechat 能力

### Requirement: 功能区保留三组交互演示
功能展示区 SHALL 保留 Bot、Case 目录、管理后台三组现有交互演示，并将每组演示与对应的 Pixoma 功能说明组合呈现。

#### Scenario: 三组演示均可使用
- **WHEN** 访客滚动到功能展示区
- **THEN** Bot、Case 目录、管理后台三组演示均可见且保留原有可操作行为

#### Scenario: 演示按需加载
- **WHEN** 功能展示区尚未接近可视区域
- **THEN** 页面不因提前装载全部演示而阻塞首屏内容显示

### Requirement: 页面解释跨设备使用方式
跨设备区 SHALL 说明访客可以通过 Telegram 在移动端发起创作，并通过 Pixoma 页面在桌面端管理 Case、任务和计算节点。

#### Scenario: 跨设备内容可理解
- **WHEN** 访客浏览跨设备区
- **THEN** 页面同时呈现移动端创作入口和桌面端管理能力，且不宣称尚未具备的客户端能力

### Requirement: 页面说明适用人群
适用人群区 SHALL 使用 Pixoma 的真实使用场景说明产品适合的主要人群，并至少覆盖个人创作者、小团队和拥有多个计算节点的使用者。

#### Scenario: 访客匹配使用场景
- **WHEN** 访客浏览适用人群区
- **THEN** 每类人群均有独立标题、核心收益和对应的 Pixoma 使用方式

### Requirement: FAQ 可通过鼠标、触控与键盘操作
FAQ 区 SHALL 提供可展开和收起的问答，并暴露正确的展开状态和内容关联，使鼠标、触控与键盘操作得到一致结果。

#### Scenario: 展开一个 FAQ
- **WHEN** 访客点击、轻触或用键盘激活某个 FAQ 标题
- **THEN** 对应答案展开，控件的可访问状态同步更新

#### Scenario: 收起一个 FAQ
- **WHEN** 访客再次激活已展开的 FAQ 标题
- **THEN** 对应答案收起，焦点保持在该标题控件

### Requirement: 导航与 CTA 指向有效目标
桌面导航和移动菜单 SHALL 指向页面内存在的区块；首屏与最终 CTA SHALL 沿用项目配置中的下载和自托管目标地址。

#### Scenario: 使用区块导航
- **WHEN** 访客激活任一页面内导航项
- **THEN** 页面定位到对应且实际存在的区块

#### Scenario: 使用行动入口
- **WHEN** 访客激活下载或自托管按钮
- **THEN** 浏览器打开项目配置的对应目标，且页面不使用占位链接

#### Scenario: 移动菜单操作闭合
- **WHEN** 访客在窄屏打开菜单并选择一个导航项
- **THEN** 页面定位到目标区块，菜单随即关闭

### Requirement: 中文与英文页面内容对等
落地页 SHALL 继续支持 `/cn` 与 `/en` 路由；新增导航、区块、FAQ 和 CTA 文案 MUST 在 `zh-CN` 与 `en` 中具备相同的信息覆盖范围。

#### Scenario: 切换语言
- **WHEN** 访客在任一区块切换中文或英文
- **THEN** 页面进入对应语言路由，所有新增可见文案使用目标语言，且不存在缺失 key 或回退 key 文本

#### Scenario: 非法语言路径
- **WHEN** 访客打开不受支持的语言路径
- **THEN** 页面沿用现有规则进入默认中文路径

### Requirement: 页面适配常见视口与输入方式
落地页 SHALL 在桌面、平板和手机视口保持内容可读、交互可用且无页面级横向溢出；触控目标 SHALL 不小于 44×44 CSS 像素。

#### Scenario: 手机视口浏览
- **WHEN** 访客在 320 CSS 像素宽的视口浏览页面
- **THEN** 标题、演示、卡片、FAQ 和 CTA 自适应重排，页面无横向滚动

#### Scenario: 键盘浏览
- **WHEN** 访客仅使用键盘遍历导航、菜单、演示控件、FAQ 与 CTA
- **THEN** 所有交互控件均可到达、可激活，并显示清晰的焦点状态

### Requirement: 动效尊重系统偏好
页面 MAY 使用进入视口、展开收起和界面反馈动效，但 MUST 在系统启用减少动态效果时关闭非必要移动，并保持内容和操作结果不变。

#### Scenario: 默认动态偏好
- **WHEN** 系统未要求减少动态效果
- **THEN** 页面以轻量动效呈现区块和交互状态，且不阻塞操作

#### Scenario: 减少动态效果
- **WHEN** 系统启用 `prefers-reduced-motion: reduce`
- **THEN** 非必要移动和循环动画被关闭或压缩，全部内容与控件仍可使用

### Requirement: 落地页保持静态可部署
落地页 SHALL 在不依赖 Pixoma 后端运行时的情况下构建为可静态托管的产物。

#### Scenario: 构建生产产物
- **WHEN** 对 `web/landing` 运行生产构建
- **THEN** TypeScript 检查和静态构建成功，页面内容与交互不要求后端 API 才能显示
