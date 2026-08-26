## ADDED Requirements

### Requirement: 管理端预览效果图上传
工作流编辑页 MUST 提供预览效果图上传能力：支持图片（png/jpeg/webp/gif）与视频（mp4/webm）直传；上传 MUST 经过管理端 `/api/v1/media` 接口并持久化到用户文件存储（blob 存储、非公网），`Case.preview` 保存为媒体引用；上传失败 MUST 展示可诊断错误且不写入数据。

#### Scenario: 上传图片成功
- **WHEN** 用户在预览效果图字段选择一张本地图片并触发上传
- **THEN** 系统上传到 `/api/v1/media` 并持久化到 blob 存储，返回可预览的媒体引用

#### Scenario: 上传视频成功
- **WHEN** 用户在预览效果图字段选择一段本地视频并触发上传
- **THEN** 系统上传到 `/api/v1/media` 并持久化到 blob 存储，返回可预览的媒体引用

#### Scenario: 非法类型被拒绝
- **WHEN** 用户上传超出图片/视频白名单的文件类型
- **THEN** 上传被拒绝并展示明确错误，不写入 blob 存储

#### Scenario: 超过体积上限被拒绝
- **WHEN** 用户上传超过体积上限的文件
- **THEN** 上传被拒绝并展示明确错误

### Requirement: 非公网鉴权媒体拉取
预览媒体 MUST 通过管理端鉴权拉取（`GET /api/v1/media/{key}`，位于统一鉴权 gate 之下），MUST NOT 以公网可直接访问的静态 URL 暴露；响应 MUST 携带正确 `Content-Type` 并以流式方式返回。

#### Scenario: 鉴权拉取图片预览
- **WHEN** 管理端浏览器携带合法凭证请求上传后的图片 key
- **THEN** 系统流式返回该图片，`Content-Type` 与上传时一致

#### Scenario: 未授权访问被拒绝
- **WHEN** 无有效凭证请求媒体 key
- **THEN** 请求被统一鉴权 gate 拒绝，不返回二进制内容

### Requirement: 编辑页内嵌媒体预览
基础信息「预览效果图」字段 MUST 以内嵌媒体预览呈现已上传内容：图片渲染缩略/原图，视频渲染播放器；并 MUST 支持替换与移除后重新上传；「Case 说明」保持为独立文本字段。

#### Scenario: 已上传图片显示预览
- **WHEN** 预览效果图字段已绑定一个图片媒体
- **THEN** 编辑表单内嵌显示该图片预览

#### Scenario: 已上传视频显示预览
- **WHEN** 预览效果图字段已绑定一个视频媒体
- **THEN** 编辑表单内嵌显示可播放的视频预览

#### Scenario: 替换与移除预览
- **WHEN** 用户对已上传预览执行替换或移除
- **THEN** 预览随之更新或清空，最终随表单保存持久化；Case 说明文本不受影响

### Requirement: TG preview 步骤投递预览媒体
`open_case` 能力 MUST NOT 再把 `Case.preview` 当纯文本拼进提示；MUST 在 preview 步骤把预览媒体真实发送给机器人用户（图片发送为 photo、视频发送为 video），标题/文本沿用 Case 名称与 Case 说明；预览媒体缺失或不可投递时 MUST 回退为文本提示且不阻断流程。

#### Scenario: 投递图片预览给 TG 用户
- **WHEN** 用户在 TG 触发某 Case 的 preview 步骤且该 Case 预览为图片
- **THEN** bot 将图片作为 photo 发送给用户，并附带 Case 名称与 Case 说明文本

#### Scenario: 投递视频预览给 TG 用户
- **WHEN** 用户在 TG 触发某 Case 的 preview 步骤且该 Case 预览为视频
- **THEN** bot 将视频作为 video 发送给用户，并附带 Case 名称与 Case 说明文本

#### Scenario: 预览媒体缺失回退文本
- **WHEN** Case 没有可投递的预览媒体（未上传或读取失败）
- **THEN** bot 以文本形式给出 Case 名称、说明与原预览提示，流程正常继续
