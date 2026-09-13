## Purpose

落地页中的交互式 App UI 演示与动效，用静态 mockup 数据渲染 Pixoma 工作台界面，并配合 Framer Motion 与 reactbits 提供流畅、非阻塞的浏览体验。

## ADDED Requirements

### Requirement: 演示使用静态 mockup 数据
The interactive app UI demos SHALL render from static mockup data and SHALL NOT depend on a live backend, login, or real data fetch.

#### Scenario: 离线即可渲染演示
- **WHEN** 用户在无后端环境下打开落地页并浏览演示区
- **THEN** 演示区正常渲染，无网络请求或后端依赖，也不出现数据加载错误

### Requirement: 演示区具备交互动效
The demo surfaces SHALL support interactive behaviors (e.g. typing-like bot 对话、可切换 Case、可滚动内容) driven by Framer Motion and/or reactbits effects.

#### Scenario: 交互演示可操作且带动效
- **WHEN** 用户点击或滚动到演示区并触发交互
- **THEN** 演示产生相应动画/反馈，且不导致页面卡顿或布局跳变

### Requirement: Scroll-reveal 入场动画
Sections and demo surfaces SHALL reveal or animate in response to scroll position via a scroll-triggered mechanism, while remaining performant.

#### Scenario: 滚动触发入场动画
- **WHEN** 用户滚动使某个区块进入视口
- **THEN** 该区块触发入场动画（如淡入 / 位移动效），且多次往返滚动不累积冲突

### Requirement: 动效尊重减弱动态偏好
Animation behaviors SHALL respect the user's reduced-motion preference by disabling or simplifying non-critical animations when requested.

#### Scenario: 减弱动态时简化动画
- **WHEN** 用户开启了系统级「减弱动态效果」
- **THEN** 页面仍完整可读可操作，但跳过或弱化装饰性动画
