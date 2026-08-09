## ADDED Requirements

### Requirement: TG Menu 管理页
控制台 MUST 提供 TG Menu 管理页：展示当前菜单项列表，支持编辑文案、顺序、启用、动作类型，以及 Case 绑定（选择已有 Case id）、按 tag 列表、占位提示，以及回复媒体（文本与图片 URL 列表）。保存 MUST 调用 admin-api Menu 接口；失败时 MUST 展示错误且不假装成功。本期 MUST NOT 要求浏览器本地上传图片文件到对象存储（图片以 URL 配置）。

#### Scenario: 编辑并保存绑定 Case
- **WHEN** 运维将某菜单项动作设为进入 Case，选择合法 Case 并保存
- **THEN** 页面提示成功，刷新后仍显示该绑定

#### Scenario: 编辑并保存回复媒体
- **WHEN** 运维将某菜单项动作设为回复媒体，填写文本与图片 URL 并保存成功
- **THEN** 页面提示成功，刷新后仍显示该回复配置

#### Scenario: 保存失败展示错误
- **WHEN** admin-api 返回校验或网络错误
- **THEN** 页面展示错误信息，本地不进入「已保存」误导态

### Requirement: 请求仅指向 admin-api
Menu 页的读写请求 MUST 仅使用 `VITE_ADMIN_API_BASE` 指向的 admin-api；MUST NOT 引入前端 mock 作为验收路径。

#### Scenario: Network 指向 admin-api
- **WHEN** 运维在 Menu 页加载或保存
- **THEN** 浏览器请求前缀为配置的 admin-api 基址 + Menu API 路径
