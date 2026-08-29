## Purpose

Pixoma 对外产品落地页，通过完整的信息架构与内容区块向终端用户传递「随时随地用自己的 ComfyUI 做 AI 艺术创作」的价值主张，并引导下载/自托管。

## ADDED Requirements

### Requirement: 落地页渲染完整信息架构
The landing page SHALL render the top-level information architecture in the following order: a hero section, multiple feature showcase sections, a use-case section, and a call-to-action (CTA) section.

#### Scenario: 打开落地页看到完整区块
- **WHEN** 用户打开落地页首页
- **THEN** 页面按顺序渲染 Hero、特性展示区、使用场景与 CTA 四个区块，且各区块内容可见

### Requirement: Hero 展示价值主张与主行动入口
The hero section SHALL display a primary headline conveying the product value proposition and at least one primary CTA that links to the download or self-hosting entry point.

#### Scenario: Hero 主标语与 CTA 可见
- **WHEN** 用户浏览首屏
- **THEN** 可见主标语（价值主张）与至少一个主 CTA 按钮，且 CTA 指向下载或自托管入口

### Requirement: 特性展示区包含交互式 App UI 演示
Each feature showcase section SHALL pair its descriptive copy with an interactive app UI demo that renders the corresponding Pixoma workbench surface (e.g. Telegram Bot 对话、Case 目录、管理后台).

#### Scenario: 特性区展示对应界面演示
- **WHEN** 用户滚动到一个特性展示区
- **THEN** 该区展示特性说明，并渲染对应的交互式 App UI 演示，而不只是静态截图

### Requirement: 落地页响应式布局
The landing page SHALL adapt its layout across desktop, tablet, and mobile viewport widths so that all sections remain usable and readable without horizontal overflow.

#### Scenario: 移动端布局可用
- **WHEN** 用户在移动端视口打开落地页
- **THEN** 页面内容自适应换行、无横向溢出，CTA 与导航可操作性正常

### Requirement: 产出可静态部署的构建结果
The landing page SHALL produce a static, deployable build output via its build pipeline, with no TypeScript errors and no runtime dependencies on a backend.

#### Scenario: 构建通过并产出静态文件
- **WHEN** 对 `web/landing` 运行构建命令
- **THEN** 构建成功（无 TS 错误），并产出可直接托管的静态文件
