## ADDED Requirements

### Requirement: 主键盘管理页（树）
控制台 MUST 提供「主键盘」管理页：编辑树形菜单项（含文件夹、子项、挂载 Case、placeholder、reply_media）。保存 MUST 调用 admin-api 树形 Menu 接口；失败时 MUST 展示错误且不假装成功。本期 MUST NOT 要求浏览器本地上传图片。

#### Scenario: 编辑文件夹并挂载 Case 后保存
- **WHEN** 运维配置文件夹及其 Case 关联并保存成功
- **THEN** 页面提示成功，刷新后树与挂载仍在

#### Scenario: 保存失败展示错误
- **WHEN** admin-api 返回校验或网络错误
- **THEN** 页面展示错误信息，不进入「已保存」误导态

### Requirement: Case 详情展示菜单挂载
Case 详情 MUST 展示该 Case 出现在主键盘中的路径列表（只读）；数据 MUST 来自 admin-api。

#### Scenario: 已挂载 Case 可见路径
- **WHEN** 运维打开已挂到某文件夹的 Case 详情
- **THEN** 页面展示至少一条可读的主键盘路径

### Requirement: 请求仅指向 admin-api
主键盘页与 Case 挂载展示的请求 MUST 仅使用 `VITE_ADMIN_API_BASE`；MUST NOT 引入前端 mock 作为验收路径。

#### Scenario: Network 指向 admin-api
- **WHEN** 运维加载主键盘页或带挂载信息的 Case 详情
- **THEN** 浏览器请求前缀为配置的 admin-api 基址
