## Purpose

落地页多语言能力，按语言路由组织面向用户的中英文内容，并让新增语言时只扩展词条而无须改动页面结构。

## ADDED Requirements

### Requirement: 支持 zh-CN 与 en 两种语言
The landing page SHALL provide localized copy for `zh-CN` and `en` locales.

#### Scenario: 切换语言显示对应文案
- **WHEN** 用户在 `zh-CN` 与 `en` 之间切换
- **THEN** 页面上所有面向用户的文案随语言切换变化，且无缺失或错位的硬编码串

### Requirement: 基于路由的语言切换
The locale selection SHALL be reflected in the URL path (e.g. `/cn`, `/en`), so a given locale is shareable and bookmarkable.

#### Scenario: 语言体现在 URL 且可分享
- **WHEN** 用户访问 `/zh-CN` 或 `/en` 对应路径
- **THEN** 页面渲染对应语言内容，且该 URL 可被直接分享打开

### Requirement: 默认语言与未知语言回退
The landing page SHALL default to `zh-CN` when no locale is specified, and SHALL fall back to `zh-CN` when an unsupported or invalid locale is requested.

#### Scenario: 未指定语言时使用中文
- **WHEN** 用户访问不带语言前缀的路径或请求一个不支持的语言
- **THEN** 页面以 `zh-CN` 渲染，而非报错或空白

### Requirement: 词条可扩展
The i18n layer SHALL organize strings as a key-based catalogue so that adding a new language requires only adding new entries, without modifying page components.

#### Scenario: 新增语言只加词条
- **WHEN** 开发者需要新增一种语言
- **THEN** 只需补充对应语言词条，无需改动渲染组件
