# Pixoma 全量错误返回审计表

> 由脚本从 `internal/httpapi/**` 抽取，共 **372** 行明细。
> 「建议 code」与「error_detail」为规则草案，**需逐条复核后定稿**。
> 已知字符串码已按方案 3.4.1 映射表覆盖。
>
> **抽取口径**：本表只覆盖 `writeErr` / `writeErrCode` 调用点（并剔除 19 个 helper 函数体自身）。
> `writeError(` 的 **52 处**（`media` 8 处 + `studio` 44 处走错误分类路由器）**不在本表内**，
> 但已完整收录进文案表。
>
> **code 与最终文案的权威来源是 [`docs/api-error-messages.md`](./api-error-messages.md)**：
> 本表的「建议 code」是**粗粒度规则草案**（77 个码，同一码下可能有多条不同文案）；
> 文案表已细化为 **204 个码 / 438 个错误返回点，做到 code ↔ 文案 1:1**。定稿以文案表为准。

## 1. 总览

| 指标 | 数量 |
| --- | --- |
| 错误返回点总数 | 372 |
| 现状：吞错 | 219 |
| 现状：泄原始错误 | 144 |
| 现状：动态 | 8 |
| 现状：业务态混入成功体 | 1 |

### error_detail 需求分布

| 需求 | 数量 |
| --- | --- |
| — | 174 |
| **必须** | 165 |
| 禁止(安全) | 25 |
| 需要(依赖) | 8 |

### 建议明细码分布

| 明细码 | 含义 | 数量 |
| --- | --- | --- |
| `01` | 资源不存在 | 42 |
| `02` | 参数校验失败 | 96 |
| `03` | 冲突/状态不允许 | 17 |
| `04` | 需要确认(ack) | 3 |
| `05` | 依赖不可用 | 35 |
| `06` | 内部错误 | 130 |
| `07` | 认证失效 | 18 |
| `08` | 无权限 | 6 |
| `09` | 请求体格式错误 | 23 |
| `11` | 需要重启生效 | 1 |
| `12` | 存储桶不存在 | 1 |

### 各包分布

| 包 | 业务域 | 错误点 |
| --- | --- | --- |
| `setup` | `01` | 75 |
| `edges` | `07` | 48 |
| `channels` | `08` | 47 |
| `studio` | `12` | 42 |
| `cases` | `06` | 30 |
| `agent` | `15` | 24 |
| `adminusers` | `03` | 22 |
| `topics` | `09` | 22 |
| `users` | `02` | 21 |
| `sessions` | `04` | 12 |
| `stats` | `11` | 12 |
| `tasks` | `05` | 12 |
| `linkhealth` | `14` | 2 |
| `common` | `00` | 2 |
| `routing` | `10` | 1 |

## 2. 逐包明细

### setup（业务域 `01`，75 处）

| 位置 | HTTP | 现有文案 | 现状 | 建议 code | error_detail |
| --- | --- | --- | --- | --- | --- |
| setup/gate.go:59 | 401 | `"unauthorized"` | 吞错 | `4010107` | 禁止(安全) |
| setup/gate.go:63 | 401 | `"unauthorized"` | 吞错 | `4010107` | 禁止(安全) |
| setup/gate.go:74 | 401 | `"unauthorized"` | 吞错 | `4010107` | 禁止(安全) |
| setup/gate.go:78 | 401 | `"unauthorized"` | 吞错 | `4010107` | 禁止(安全) |
| setup/gate.go:87 | 403 | `"platform not initialized"`（code: `not_initialized`） | 动态 | `4030110` | 禁止(安全) |
| setup/gate.go:94 | 403 | `"settings saved; restart pixoma for them to take effect"` | 吞错 | `4030111` | 禁止(安全) |
| setup/gate.go:101 | 401 | `"unauthorized"` | 吞错 | `4010107` | 禁止(安全) |
| setup/gate.go:106 | 401 | `"unauthorized"` | 吞错 | `4010107` | 禁止(安全) |
| setup/gate.go:110 | 403 | `"forbidden"` | 吞错 | `4030108` | 禁止(安全) |
| setup/gate.go:115 | 403 | `"forbidden"` | 吞错 | `4030108` | 禁止(安全) |
| setup/handler.go:89 | 401 | `"unauthorized"` | 吞错 | `4010107` | 禁止(安全) |
| setup/handler.go:146 | 400 | `"invalid json"` | 吞错 | `4000109` | — |
| setup/handler.go:152 | 429 | `"too many login attempts"` | 吞错 | `4290102` | — |
| setup/handler.go:162 | 401 | `"invalid credentials"` | 吞错 | `4010107` | 禁止(安全) |
| setup/handler.go:167 | 403 | `"account disabled"` | 吞错 | `4030108` | 禁止(安全) |
| setup/handler.go:172 | 401 | `"invalid credentials"` | 吞错 | `4010107` | 禁止(安全) |
| setup/handler.go:183 | 500 | `err.Error(` | 泄原始错误 | `5000106` | **必须** |
| setup/handler.go:190 | 500 | `err.Error(` | 泄原始错误 | `5000106` | **必须** |
| setup/handler.go:195 | 401 | `"invalid credentials"` | 吞错 | `4010107` | 禁止(安全) |
| setup/handler.go:201 | 500 | `err.Error(` | 泄原始错误 | `5000106` | **必须** |
| setup/handler.go:232 | 400 | `"invalid json"` | 吞错 | `4000109` | — |
| setup/handler.go:249 | 401 | `err.Error(` | 泄原始错误 | `4010107` | 禁止(安全) |
| setup/handler.go:253 | 400 | `"password must be at least 8 characters"` | 吞错 | `4000102` | — |
| setup/handler.go:257 | 400 | `"password already set"` | 吞错 | `4000103` | — |
| setup/handler.go:260 | 500 | `err.Error(` | 泄原始错误 | `5000106` | **必须** |
| setup/handler.go:290 | 400 | `"invalid json"` | 吞错 | `4000109` | — |
| setup/handler.go:299 | 400 | `"dsn required"` | 吞错 | `4000102` | 需要(依赖) |
| setup/handler.go:304 | 400 | `err.Error(` | 泄原始错误 | `4000102` | — |
| setup/handler.go:310 | 400 | `"database unreachable: "+err.Error(` | 泄原始错误 | `4000102` | 需要(依赖) |
| setup/handler.go:315 | 400 | `err.Error(` | 泄原始错误 | `4000102` | — |
| setup/handler.go:320 | 400 | `"database ping failed: "+err.Error(` | 泄原始错误 | `4000102` | 需要(依赖) |
| setup/handler.go:325 | 500 | `err.Error(` | 泄原始错误 | `5000106` | **必须** |
| setup/handler.go:338 | 400 | `"invalid json"` | 吞错 | `4000109` | — |
| setup/handler.go:343 | 500 | `err.Error(` | 泄原始错误 | `5000106` | **必须** |
| setup/handler.go:353 | 400 | `"configure database first"` | 吞错 | `4000102` | 需要(依赖) |
| setup/handler.go:357 | 400 | `"database unreachable: "+err.Error(` | 泄原始错误 | `4000102` | 需要(依赖) |
| setup/handler.go:361 | 500 | `err.Error(` | 泄原始错误 | `5000106` | **必须** |
| setup/handler.go:373 | 400 | `err.Error(` | 泄原始错误 | `4000102` | — |
| setup/handler.go:378 | 400 | `err.Error(` | 泄原始错误 | `4000102` | — |
| setup/handler.go:383 | 400 | `err.Error(` | 泄原始错误 | `4000102` | — |
| setup/handler.go:412 | 400 | `"invalid json"` | 吞错 | `4000109` | — |
| setup/handler.go:434 | 400 | `err.Error(` | 泄原始错误 | `4000102` | — |
| setup/handler.go:449 | 200 | `"bucket_not_found"（HTTP 200 成功体）` | 业务态混入成功体 | `4040112` | — |
| setup/handler.go:459 | 400 | `"blob check failed: "+err.Error(` | 泄原始错误 | `4000102` | 需要(依赖) |
| setup/handler.go:476 | 400 | `err.Error(` | 泄原始错误 | `4000102` | — |
| setup/handler.go:497 | 400 | `"finalize setup first"` | 吞错 | `4000102` | — |
| setup/handler.go:502 | 400 | `"invalid json"` | 吞错 | `4000109` | — |
| setup/handler.go:507 | 400 | `"configure database first"` | 吞错 | `4000102` | 需要(依赖) |
| setup/handler.go:512 | 400 | `err.Error(` | 泄原始错误 | `4000102` | — |
| setup/handler.go:518 | 400 | `"save settings first"` | 吞错 | `4000102` | — |
| setup/handler.go:529 | 400 | `err.Error(` | 泄原始错误 | `4000102` | — |
| setup/handler.go:533 | 400 | `err.Error(` | 泄原始错误 | `4000102` | — |
| setup/handler.go:537 | 500 | `err.Error(` | 泄原始错误 | `5000106` | **必须** |
| setup/handler.go:585 | 400 | `"change default password first"` | 吞错 | `4000102` | — |
| setup/handler.go:590 | 400 | `"configure database first"` | 吞错 | `4000102` | 需要(依赖) |
| setup/handler.go:595 | 400 | `err.Error(` | 泄原始错误 | `4000102` | — |
| setup/handler.go:601 | 400 | `"save settings first"` | 吞错 | `4000102` | — |
| setup/handler.go:605 | 400 | `err.Error(` | 泄原始错误 | `4000102` | — |
| setup/handler.go:609 | 500 | `err.Error(` | 泄原始错误 | `5000106` | **必须** |
| setup/handler.go:613 | 500 | `err.Error(` | 泄原始错误 | `5000106` | **必须** |
| setup/handler.go:649 | 401 | `"unauthorized"` | 吞错 | `4010107` | 禁止(安全) |
| setup/handler.go:756 | 400 | `"invalid json"` | 吞错 | `4000109` | — |
| setup/handler.go:761 | 400 | `"invalid email"` | 吞错 | `4000102` | — |
| setup/handler.go:765 | 500 | `err.Error(` | 泄原始错误 | `5000106` | **必须** |
| setup/handler.go:790 | 409 | `"registration disabled"` | 吞错 | `4090103` | — |
| setup/handler.go:800 | 400 | `"invalid json"` | 吞错 | `4000109` | — |
| setup/handler.go:806 | 429 | `"too many registrations"` | 吞错 | `4290102` | — |
| setup/handler.go:813 | 400 | `"账号名不能为空"` | 吞错 | `4000102` | — |
| setup/handler.go:817 | 400 | `"password must be at least 8 characters"` | 吞错 | `4000102` | — |
| setup/handler.go:821 | 400 | `"invalid email"` | 吞错 | `4000102` | — |
| setup/handler.go:828 | 500 | `"failed to hash password"` | 吞错 | `5000106` | **必须** |
| setup/handler.go:841 | 409 | `"username or email already taken"` | 吞错 | `4090103` | — |
| setup/handler.go:844 | 500 | `"create failed"` | 吞错 | `5000106` | **必须** |
| setup/handler.go:848 | 500 | `"unavailable"` | 吞错 | `5000105` | **必须** |
| setup/handler.go:853 | 500 | `err.Error(` | 泄原始错误 | `5000106` | **必须** |

### edges（业务域 `07`，48 处）

| 位置 | HTTP | 现有文案 | 现状 | 建议 code | error_detail |
| --- | --- | --- | --- | --- | --- |
| edges/handler.go:151 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:167 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:187 | 400 | `"invalid json"` | 吞错 | `4000709` | — |
| edges/handler.go:195 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:201 | 400 | `"name required"` | 吞错 | `4000702` | — |
| edges/handler.go:219 | 409 | `"instance already exists"` | 吞错 | `4090703` | — |
| edges/handler.go:222 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:227 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:231 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:235 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:240 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:249 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:259 | 404 | `"instance not found"` | 吞错 | `4040701` | — |
| edges/handler.go:263 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:273 | 404 | `"instance not found"` | 吞错 | `4040701` | — |
| edges/handler.go:277 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:282 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:286 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:297 | 404 | `"instance not found"` | 吞错 | `4040701` | — |
| edges/handler.go:301 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:306 | 400 | `"invalid json"` | 吞错 | `4000709` | — |
| edges/handler.go:312 | 400 | `"name required"` | 吞错 | `4000702` | — |
| edges/handler.go:329 | 500 | `"topics repository not configured"` | 吞错 | `5000705` | **必须** |
| edges/handler.go:335 | 400 | `"unknown or disabled topic: "+key` | 动态 | `4000702` | — |
| edges/handler.go:348 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:353 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:359 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:365 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:370 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:375 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:388 | 500 | `"delete cleanup not configured"` | 吞错 | `5000705` | **必须** |
| edges/handler.go:393 | 409 | `"edge_delete_needs_ack", 			"edge has running tasks; confirm with ack_references to mark` | 吞错 | `4090704` | — |
| edges/handler.go:398 | 404 | `"instance not found"` | 吞错 | `4040701` | — |
| edges/handler.go:402 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:409 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:421 | 404 | `"instance not found"` | 吞错 | `4040701` | — |
| edges/handler.go:424 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:434 | 400 | `"invalid limit"` | 吞错 | `4000702` | — |
| edges/handler.go:442 | 400 | `"invalid offset"` | 吞错 | `4000702` | — |
| edges/handler.go:449 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:482 | 404 | `"instance not found"` | 吞错 | `4040701` | — |
| edges/handler.go:485 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:495 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:536 | 500 | `"metrics not configured"` | 吞错 | `5000705` | **必须** |
| edges/handler.go:540 | 404 | `"instance not found"` | 吞错 | `4040701` | — |
| edges/handler.go:543 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |
| edges/handler.go:548 | 400 | `err.Error(` | 泄原始错误 | `4000702` | — |
| edges/handler.go:553 | 500 | `err.Error(` | 泄原始错误 | `5000706` | **必须** |

### channels（业务域 `08`，47 处）

| 位置 | HTTP | 现有文案 | 现状 | 建议 code | error_detail |
| --- | --- | --- | --- | --- | --- |
| channels/handler.go:112 | 500 | `"channel service not configured"` | 吞错 | `5000805` | **必须** |
| channels/handler.go:117 | 400 | `"invalid json"` | 吞错 | `4000809` | — |
| channels/handler.go:132 | 400 | `err.Error(` | 泄原始错误 | `4000802` | — |
| channels/handler.go:142 | 500 | `err.Error(` | 泄原始错误 | `5000806` | **必须** |
| channels/handler.go:150 | 500 | `"channel service not configured"` | 吞错 | `5000805` | **必须** |
| channels/handler.go:155 | 500 | `err.Error(` | 泄原始错误 | `5000806` | **必须** |
| channels/handler.go:162 | 500 | `err.Error(` | 泄原始错误 | `5000806` | **必须** |
| channels/handler.go:174 | 404 | `"channel not found"` | 吞错 | `4040801` | — |
| channels/handler.go:178 | 500 | `err.Error(` | 泄原始错误 | `5000806` | **必须** |
| channels/handler.go:183 | 500 | `err.Error(` | 泄原始错误 | `5000806` | **必须** |
| channels/handler.go:199 | 400 | `"invalid json"` | 吞错 | `4000809` | — |
| channels/handler.go:204 | 404 | `"channel not found"` | 吞错 | `4040801` | — |
| channels/handler.go:208 | 500 | `err.Error(` | 泄原始错误 | `5000806` | **必须** |
| channels/handler.go:229 | 404 | `"channel not found"` | 吞错 | `4040801` | — |
| channels/handler.go:233 | 500 | `err.Error(` | 泄原始错误 | `5000806` | **必须** |
| channels/handler.go:238 | 500 | `err.Error(` | 泄原始错误 | `5000806` | **必须** |
| channels/handler.go:246 | 404 | `"channel not found"` | 吞错 | `4040801` | — |
| channels/handler.go:249 | 500 | `err.Error(` | 泄原始错误 | `5000806` | **必须** |
| channels/handler.go:257 | 404 | `"channel not found"` | 吞错 | `4040801` | — |
| channels/handler.go:260 | 500 | `err.Error(` | 泄原始错误 | `5000806` | **必须** |
| channels/handler.go:268 | 500 | `"channel service not configured"` | 吞错 | `5000805` | **必须** |
| channels/handler.go:288 | 404 | `"channel not found"` | 吞错 | `4040801` | — |
| channels/handler.go:292 | 500 | `err.Error(` | 泄原始错误 | `5000806` | **必须** |
| channels/mcp_users.go:22 | 500 | `"mcp users not configured"` | 吞错 | `5000805` | **必须** |
| channels/mcp_users.go:28 | 404 | `"channel not found"` | 吞错 | `4040801` | — |
| channels/mcp_users.go:32 | 500 | `err.Error(` | 泄原始错误 | `5000806` | **必须** |
| channels/mcp_users.go:36 | 400 | `"channel is not mcp"` | 吞错 | `4000802` | — |
| channels/mcp_users.go:43 | 400 | `"invalid json"` | 吞错 | `4000809` | — |
| channels/mcp_users.go:48 | 400 | `"name required"` | 吞错 | `4000802` | — |
| channels/mcp_users.go:59 | 500 | `err.Error(` | 泄原始错误 | `5000806` | **必须** |
| channels/mcp_users.go:64 | 500 | `err.Error(` | 泄原始错误 | `5000806` | **必须** |
| channels/mcp_users.go:70 | 500 | `err.Error(` | 泄原始错误 | `5000806` | **必须** |
| channels/mcp_users.go:81 | 500 | `"mcp users not configured"` | 吞错 | `5000805` | **必须** |
| channels/mcp_users.go:87 | 404 | `"channel not found"` | 吞错 | `4040801` | — |
| channels/mcp_users.go:91 | 500 | `err.Error(` | 泄原始错误 | `5000806` | **必须** |
| channels/mcp_users.go:95 | 400 | `"channel is not mcp"` | 吞错 | `4000802` | — |
| channels/mcp_users.go:100 | 500 | `err.Error(` | 泄原始错误 | `5000806` | **必须** |
| channels/mcp_users.go:115 | 500 | `"mcp users not configured"` | 吞错 | `5000805` | **必须** |
| channels/mcp_users.go:122 | 404 | `"channel not found"` | 吞错 | `4040801` | — |
| channels/mcp_users.go:126 | 500 | `err.Error(` | 泄原始错误 | `5000806` | **必须** |
| channels/mcp_users.go:130 | 400 | `"channel is not mcp"` | 吞错 | `4000802` | — |
| channels/mcp_users.go:135 | 404 | `"user not found"` | 吞错 | `4040801` | — |
| channels/mcp_users.go:139 | 500 | `err.Error(` | 泄原始错误 | `5000806` | **必须** |
| channels/mcp_users.go:143 | 404 | `"user not found"` | 吞错 | `4040801` | — |
| channels/mcp_users.go:147 | 500 | `err.Error(` | 泄原始错误 | `5000806` | **必须** |
| channels/mcp_users.go:152 | 404 | `"user not found"` | 吞错 | `4040801` | — |
| channels/mcp_users.go:155 | 500 | `err.Error(` | 泄原始错误 | `5000806` | **必须** |

### studio（业务域 `12`，42 处）

| 位置 | HTTP | 现有文案 | 现状 | 建议 code | error_detail |
| --- | --- | --- | --- | --- | --- |
| studio/agui.go:78 | 400 | `"请求内容格式不正确"` | 吞错 | `4001202` | — |
| studio/agui.go:85 | 400 | `"会话、运行和用户消息不能为空"` | 吞错 | `4001202` | — |
| studio/agui.go:105 | 500 | `"当前连接不支持流式响应"` | 吞错 | `5001206` | **必须** |
| studio/handler.go:83 | 503 | `"资产存储服务不可用"` | 吞错 | `5031205` | **必须** |
| studio/handler.go:92 | 400 | `"请求内容格式不正确"` | 吞错 | `4001202` | — |
| studio/handler.go:111 | 503 | `"资产存储服务不可用"` | 吞错 | `5031205` | **必须** |
| studio/handler.go:118 | 400 | `"请求内容格式不正确"` | 吞错 | `4001202` | — |
| studio/handler.go:137 | 503 | `"资产存储服务不可用"` | 吞错 | `5031205` | **必须** |
| studio/handler.go:141 | 400 | `"上传文件读取失败"` | 吞错 | `4001202` | — |
| studio/handler.go:146 | 400 | `"请选择上传文件"` | 吞错 | `4001202` | — |
| studio/handler.go:170 | 400 | `"请求内容格式不正确"` | 吞错 | `4001202` | — |
| studio/handler.go:187 | 400 | `"请求内容格式不正确"` | 吞错 | `4001202` | — |
| studio/handler.go:217 | 400 | `"请求内容格式不正确"` | 吞错 | `4001202` | — |
| studio/handler.go:451 | 400 | `"请求内容格式不正确"` | 吞错 | `4001202` | — |
| studio/handler.go:534 | 400 | `"请求内容格式不正确"` | 吞错 | `4001202` | — |
| studio/handler.go:555 | 404 | `"资产内容不存在"` | 吞错 | `4041201` | — |
| studio/handler.go:569 | 404 | `"资产版本不存在"` | 吞错 | `4041201` | — |
| studio/handler.go:594 | 400 | `"请求内容格式不正确"` | 吞错 | `4001202` | — |
| studio/handler.go:615 | 400 | `"请求内容格式不正确"` | 吞错 | `4001202` | — |
| studio/handler.go:666 | 400 | `"请求内容格式不正确"` | 吞错 | `4001202` | — |
| studio/handler.go:700 | 503 | `"模型配置服务不可用"` | 吞错 | `5031205` | **必须** |
| studio/handler.go:705 | 400 | `"请求内容格式不正确"` | 吞错 | `4001202` | — |
| studio/handler.go:723 | 503 | `"模型配置服务不可用"` | 吞错 | `5031205` | **必须** |
| studio/handler.go:746 | 503 | `"模型配置服务不可用"` | 吞错 | `5031205` | **必须** |
| studio/handler.go:751 | 400 | `"请求内容格式不正确"` | 吞错 | `4001202` | — |
| studio/handler.go:768 | 503 | `"模型配置服务不可用"` | 吞错 | `5031205` | **必须** |
| studio/handler.go:773 | 400 | `"请求内容格式不正确"` | 吞错 | `4001202` | — |
| studio/handler.go:797 | 503 | `"能力配置服务不可用"` | 吞错 | `5031205` | **必须** |
| studio/handler.go:814 | 503 | `"能力配置服务不可用"` | 吞错 | `5031205` | **必须** |
| studio/handler.go:819 | 400 | `"请求内容格式不正确"` | 吞错 | `4001202` | — |
| studio/handler.go:837 | 503 | `"能力配置服务不可用"` | 吞错 | `5031205` | **必须** |
| studio/handler.go:842 | 400 | `"请求内容格式不正确"` | 吞错 | `4001202` | — |
| studio/handler.go:861 | 503 | `"能力配置服务不可用"` | 吞错 | `5031205` | **必须** |
| studio/handler.go:878 | 503 | `"能力配置服务不可用"` | 吞错 | `5031205` | **必须** |
| studio/handler.go:883 | 400 | `"请求内容格式不正确"` | 吞错 | `4001202` | — |
| studio/handler.go:901 | 503 | `"能力配置服务不可用"` | 吞错 | `5031205` | **必须** |
| studio/handler.go:906 | 400 | `"请求内容格式不正确"` | 吞错 | `4001202` | — |
| studio/handler.go:925 | 503 | `"能力配置服务不可用"` | 吞错 | `5031205` | **必须** |
| studio/handler.go:942 | 503 | `"能力配置服务不可用"` | 吞错 | `5031205` | **必须** |
| studio/handler.go:959 | 503 | `"能力配置服务不可用"` | 吞错 | `5031205` | **必须** |
| studio/handler.go:966 | 400 | `"请求内容格式不正确"` | 吞错 | `4001202` | — |
| studio/handler.go:980 | 401 | `"登录状态已失效"` | 吞错 | `4011207` | 禁止(安全) |

### cases（业务域 `06`，30 处）

| 位置 | HTTP | 现有文案 | 现状 | 建议 code | error_detail |
| --- | --- | --- | --- | --- | --- |
| cases/handler.go:146 | 400 | `err.Error(` | 泄原始错误 | `4000602` | — |
| cases/handler.go:151 | 500 | `err.Error(` | 泄原始错误 | `5000606` | **必须** |
| cases/handler.go:167 | 400 | `"invalid json"` | 吞错 | `4000609` | — |
| cases/handler.go:171 | 400 | `err.Error(` | 泄原始错误 | `4000602` | — |
| cases/handler.go:180 | 409 | `"case already exists"` | 吞错 | `4090603` | — |
| cases/handler.go:183 | 500 | `err.Error(` | 泄原始错误 | `5000606` | **必须** |
| cases/handler.go:188 | 500 | `err.Error(` | 泄原始错误 | `5000606` | **必须** |
| cases/handler.go:205 | 400 | `"invalid case id"` | 吞错 | `4000602` | — |
| cases/handler.go:210 | 404 | `"case not found"` | 吞错 | `4040601` | — |
| cases/handler.go:214 | 500 | `err.Error(` | 泄原始错误 | `5000606` | **必须** |
| cases/handler.go:227 | 400 | `"invalid case id"` | 吞错 | `4000602` | — |
| cases/handler.go:235 | 500 | `"delete cleanup not configured"` | 吞错 | `5000605` | **必须** |
| cases/handler.go:240 | 409 | `"case_delete_needs_ack", 			"case is referenced by menu or card entries; confirm with ac` | 动态 | `4090604` | — |
| cases/handler.go:245 | 404 | `"case not found"` | 吞错 | `4040601` | — |
| cases/handler.go:249 | 500 | `err.Error(` | 泄原始错误 | `5000606` | **必须** |
| cases/handler.go:263 | 400 | `"invalid case id"` | 吞错 | `4000602` | — |
| cases/handler.go:268 | 404 | `"case not found"` | 吞错 | `4040601` | — |
| cases/handler.go:272 | 500 | `err.Error(` | 泄原始错误 | `5000606` | **必须** |
| cases/handler.go:277 | 400 | `"invalid json"` | 吞错 | `4000609` | — |
| cases/handler.go:286 | 400 | `err.Error(` | 泄原始错误 | `4000602` | — |
| cases/handler.go:294 | 500 | `err.Error(` | 泄原始错误 | `5000606` | **必须** |
| cases/handler.go:299 | 500 | `err.Error(` | 泄原始错误 | `5000606` | **必须** |
| cases/handler.go:308 | 400 | `"invalid case id"` | 吞错 | `4000602` | — |
| cases/handler.go:312 | 404 | `"case not found"` | 吞错 | `4040601` | — |
| cases/handler.go:315 | 500 | `err.Error(` | 泄原始错误 | `5000606` | **必须** |
| cases/handler.go:320 | 500 | `err.Error(` | 泄原始错误 | `5000606` | **必须** |
| cases/handler.go:329 | 400 | `"invalid case id"` | 吞错 | `4000602` | — |
| cases/handler.go:333 | 404 | `"case not found"` | 吞错 | `4040601` | — |
| cases/handler.go:336 | 500 | `err.Error(` | 泄原始错误 | `5000606` | **必须** |
| cases/handler.go:341 | 500 | `err.Error(` | 泄原始错误 | `5000606` | **必须** |

### agent（业务域 `15`，24 处）

| 位置 | HTTP | 现有文案 | 现状 | 建议 code | error_detail |
| --- | --- | --- | --- | --- | --- |
| agent/handler.go:86 | 400 | `"edge_id required"` | 吞错 | `4001502` | — |
| agent/handler.go:90 | 401 | `"unauthorized"` | 吞错 | `4011507` | 禁止(安全) |
| agent/handler.go:106 | 500 | `err.Error(` | 泄原始错误 | `5001506` | **必须** |
| agent/handler.go:124 | 408 | `"canceled"` | 吞错 | `4081502` | — |
| agent/handler.go:137 | 400 | `"invalid json"` | 吞错 | `4001509` | — |
| agent/handler.go:142 | 400 | `"edge_id required"` | 吞错 | `4001502` | — |
| agent/handler.go:146 | 401 | `"unauthorized"` | 吞错 | `4011507` | 禁止(安全) |
| agent/handler.go:151 | 500 | `err.Error(` | 泄原始错误 | `5001506` | **必须** |
| agent/handler.go:155 | 409 | `"heartbeat rejected"` | 吞错 | `4091503` | — |
| agent/handler.go:164 | 500 | `"presence not configured"` | 吞错 | `5001505` | **必须** |
| agent/handler.go:176 | 400 | `"invalid json"` | 吞错 | `4001509` | — |
| agent/handler.go:181 | 400 | `"edge_id required"` | 吞错 | `4001502` | — |
| agent/handler.go:185 | 401 | `"unauthorized"` | 吞错 | `4011507` | 禁止(安全) |
| agent/handler.go:191 | 500 | `err.Error(` | 泄原始错误 | `5001506` | **必须** |
| agent/handler.go:200 | 500 | `err.Error(` | 泄原始错误 | `5001506` | **必须** |
| agent/handler.go:206 | 500 | `err.Error(` | 泄原始错误 | `5001506` | **必须** |
| agent/handler.go:217 | 500 | `err.Error(` | 泄原始错误 | `5001506` | **必须** |
| agent/handler.go:242 | 500 | `"status not configured"` | 吞错 | `5001505` | **必须** |
| agent/handler.go:255 | 400 | `"invalid json"` | 吞错 | `4001509` | — |
| agent/handler.go:259 | 400 | `"edge_id required"` | 吞错 | `4001502` | — |
| agent/handler.go:263 | 401 | `"unauthorized"` | 吞错 | `4011507` | 禁止(安全) |
| agent/handler.go:268 | 409 | `"task not found"` | 吞错 | `4091503` | — |
| agent/handler.go:272 | 409 | `"stale holder"` | 吞错 | `4091503` | — |
| agent/handler.go:286 | 409 | `err.Error(` | 泄原始错误 | `4091503` | — |

### adminusers（业务域 `03`，22 处）

| 位置 | HTTP | 现有文案 | 现状 | 建议 code | error_detail |
| --- | --- | --- | --- | --- | --- |
| adminusers/handler.go:66 | 400 | `"invalid limit"` | 吞错 | `4000302` | — |
| adminusers/handler.go:73 | 500 | `"list failed"` | 吞错 | `5000306` | **必须** |
| adminusers/handler.go:95 | 400 | `"invalid json"` | 吞错 | `4000309` | — |
| adminusers/handler.go:101 | 400 | `"账号名不能为空"` | 吞错 | `4000302` | — |
| adminusers/handler.go:105 | 400 | `"password must be at least 8 characters"` | 吞错 | `4000302` | — |
| adminusers/handler.go:109 | 400 | `"invalid email"` | 吞错 | `4000302` | — |
| adminusers/handler.go:114 | 500 | `"failed to hash password"` | 吞错 | `5000306` | **必须** |
| adminusers/handler.go:150 | 400 | `"invalid json"` | 吞错 | `4000309` | — |
| adminusers/handler.go:154 | 400 | `"invalid email"` | 吞错 | `4000302` | — |
| adminusers/handler.go:158 | 400 | `"invalid role"` | 吞错 | `4000302` | — |
| adminusers/handler.go:166 | 500 | `"count admins failed"` | 吞错 | `5000306` | **必须** |
| adminusers/handler.go:170 | 409 | `"cannot remove the last admin"` | 吞错 | `4090303` | — |
| adminusers/handler.go:189 | 500 | `"failed to hash password"` | 吞错 | `5000306` | **必须** |
| adminusers/handler.go:212 | 500 | `"count admins failed"` | 吞错 | `5000306` | **必须** |
| adminusers/handler.go:216 | 409 | `"cannot delete the last admin"` | 吞错 | `4090303` | — |
| adminusers/handler.go:221 | 500 | `"delete failed"` | 吞错 | `5000306` | **必须** |
| adminusers/handler.go:242 | 404 | `"user not found"` | 吞错 | `4040301` | — |
| adminusers/handler.go:245 | 500 | `"load failed"` | 吞错 | `5000306` | **必须** |
| adminusers/handler.go:250 | 409 | `"username or email already taken"` | 吞错 | `4090303` | — |
| adminusers/handler.go:253 | 500 | `"create failed"` | 吞错 | `5000306` | **必须** |
| adminusers/handler.go:258 | 409 | `"username or email already taken"` | 吞错 | `4090303` | — |
| adminusers/handler.go:261 | 500 | `"update failed"` | 吞错 | `5000306` | **必须** |

### topics（业务域 `09`，22 处）

| 位置 | HTTP | 现有文案 | 现状 | 建议 code | error_detail |
| --- | --- | --- | --- | --- | --- |
| topics/handler.go:69 | 500 | `err.Error(` | 泄原始错误 | `5000906` | **必须** |
| topics/handler.go:87 | 400 | `"invalid json"` | 吞错 | `4000909` | — |
| topics/handler.go:92 | 400 | `"invalid topic key (lowercase letters/digits/hyphens)"` | 吞错 | `4000902` | — |
| topics/handler.go:96 | 400 | `"name required"` | 吞错 | `4000902` | — |
| topics/handler.go:108 | 409 | `"topic key already exists"` | 吞错 | `4090903` | — |
| topics/handler.go:111 | 500 | `err.Error(` | 泄原始错误 | `5000906` | **必须** |
| topics/handler.go:116 | 500 | `err.Error(` | 泄原始错误 | `5000906` | **必须** |
| topics/handler.go:127 | 404 | `"topic not found"` | 吞错 | `4040901` | — |
| topics/handler.go:130 | 500 | `err.Error(` | 泄原始错误 | `5000906` | **必须** |
| topics/handler.go:145 | 400 | `"invalid json"` | 吞错 | `4000909` | — |
| topics/handler.go:150 | 404 | `"topic not found"` | 吞错 | `4040901` | — |
| topics/handler.go:156 | 400 | `"name required"` | 吞错 | `4000902` | — |
| topics/handler.go:163 | 409 | `"default topic cannot be disabled"` | 吞错 | `4090903` | — |
| topics/handler.go:170 | 500 | `err.Error(` | 泄原始错误 | `5000906` | **必须** |
| topics/handler.go:183 | 500 | `"delete cleanup not configured"` | 吞错 | `5000905` | **必须** |
| topics/handler.go:188 | 409 | `"topic_default_protected", "default topic cannot be deleted"` | 吞错 | `4090903` | — |
| topics/handler.go:192 | 409 | `"topic_delete_needs_ack", 			"topic is referenced by cases or edges; confirm with ack_re` | 动态 | `4090904` | — |
| topics/handler.go:197 | 404 | `"topic not found"` | 吞错 | `4040901` | — |
| topics/handler.go:201 | 500 | `err.Error(` | 泄原始错误 | `5000906` | **必须** |
| topics/handler.go:245 | 404 | `"topic not found"` | 吞错 | `4040901` | — |
| topics/handler.go:248 | 500 | `err.Error(` | 泄原始错误 | `5000906` | **必须** |
| topics/handler.go:266 | 500 | `err.Error(` | 泄原始错误 | `5000906` | **必须** |

### users（业务域 `02`，21 处）

| 位置 | HTTP | 现有文案 | 现状 | 建议 code | error_detail |
| --- | --- | --- | --- | --- | --- |
| users/handler.go:79 | 400 | `err.Error(` | 泄原始错误 | `4000202` | — |
| users/handler.go:84 | 500 | `err.Error(` | 泄原始错误 | `5000206` | **必须** |
| users/handler.go:88 | 500 | `err.Error(` | 泄原始错误 | `5000206` | **必须** |
| users/handler.go:105 | 404 | `"user not found"` | 吞错 | `4040201` | — |
| users/handler.go:109 | 500 | `err.Error(` | 泄原始错误 | `5000206` | **必须** |
| users/handler.go:113 | 500 | `err.Error(` | 泄原始错误 | `5000206` | **必须** |
| users/handler.go:155 | 400 | `"invalid json"` | 吞错 | `4000209` | — |
| users/handler.go:160 | 400 | `"invalid access"` | 吞错 | `4000202` | — |
| users/handler.go:165 | 404 | `"user not found"` | 吞错 | `4040201` | — |
| users/handler.go:169 | 500 | `err.Error(` | 泄原始错误 | `5000206` | **必须** |
| users/handler.go:227 | 500 | `err.Error(` | 泄原始错误 | `5000206` | **必须** |
| users/handler.go:237 | 403 | `"forbidden"` | 吞错 | `4030208` | 禁止(安全) |
| users/handler.go:246 | 500 | `err.Error(` | 泄原始错误 | `5000206` | **必须** |
| users/handler.go:251 | 500 | `err.Error(` | 泄原始错误 | `5000206` | **必须** |
| users/handler.go:257 | 500 | `err.Error(` | 泄原始错误 | `5000206` | **必须** |
| users/handler.go:265 | 500 | `"mcp token not configured"` | 吞错 | `5000205` | **必须** |
| users/handler.go:271 | 404 | `"user not found"` | 吞错 | `4040201` | — |
| users/handler.go:275 | 500 | `err.Error(` | 泄原始错误 | `5000206` | **必须** |
| users/handler.go:280 | 404 | `"mcp token not found"` | 吞错 | `4040201` | — |
| users/handler.go:285 | 404 | `"mcp token not found"` | 吞错 | `4040201` | — |
| users/handler.go:289 | 500 | `err.Error(` | 泄原始错误 | `5000206` | **必须** |

### sessions（业务域 `04`，12 处）

| 位置 | HTTP | 现有文案 | 现状 | 建议 code | error_detail |
| --- | --- | --- | --- | --- | --- |
| sessions/handler.go:111 | 400 | `err.Error(` | 泄原始错误 | `4000402` | — |
| sessions/handler.go:117 | 500 | `err.Error(` | 泄原始错误 | `5000406` | **必须** |
| sessions/handler.go:127 | 500 | `err.Error(` | 泄原始错误 | `5000406` | **必须** |
| sessions/handler.go:137 | 500 | `err.Error(` | 泄原始错误 | `5000406` | **必须** |
| sessions/handler.go:147 | 500 | `err.Error(` | 泄原始错误 | `5000406` | **必须** |
| sessions/handler.go:160 | 404 | `"session not found"` | 吞错 | `4040401` | — |
| sessions/handler.go:164 | 500 | `err.Error(` | 泄原始错误 | `5000406` | **必须** |
| sessions/handler.go:168 | 404 | `"session not found"` | 吞错 | `4040401` | — |
| sessions/handler.go:173 | 500 | `err.Error(` | 泄原始错误 | `5000406` | **必须** |
| sessions/handler.go:181 | 404 | `"session not found"` | 吞错 | `4040401` | — |
| sessions/handler.go:185 | 500 | `err.Error(` | 泄原始错误 | `5000406` | **必须** |
| sessions/handler.go:190 | 500 | `err.Error(` | 泄原始错误 | `5000406` | **必须** |

### stats（业务域 `11`，12 处）

| 位置 | HTTP | 现有文案 | 现状 | 建议 code | error_detail |
| --- | --- | --- | --- | --- | --- |
| stats/handler.go:89 | 400 | `msg` | 动态 | `4001102` | — |
| stats/handler.go:94 | 500 | `err.Error(` | 泄原始错误 | `5001106` | **必须** |
| stats/handler.go:140 | 400 | `msg` | 动态 | `4001102` | — |
| stats/handler.go:147 | 400 | `"invalid limit: must be 1..100"` | 吞错 | `4001102` | — |
| stats/handler.go:154 | 500 | `err.Error(` | 泄原始错误 | `5001106` | **必须** |
| stats/handler.go:171 | 400 | `msg` | 动态 | `4001102` | — |
| stats/handler.go:176 | 500 | `err.Error(` | 泄原始错误 | `5001106` | **必须** |
| stats/handler.go:203 | 400 | `msg` | 动态 | `4001102` | — |
| stats/handler.go:210 | 400 | `"invalid limit: must be 1..20"` | 吞错 | `4001102` | — |
| stats/handler.go:217 | 500 | `err.Error(` | 泄原始错误 | `5001106` | **必须** |
| stats/handler.go:240 | 500 | `"metrics not configured"` | 吞错 | `5001105` | **必须** |
| stats/handler.go:246 | 500 | `err.Error(` | 泄原始错误 | `5001106` | **必须** |

### tasks（业务域 `05`，12 处）

| 位置 | HTTP | 现有文案 | 现状 | 建议 code | error_detail |
| --- | --- | --- | --- | --- | --- |
| tasks/handler.go:113 | 400 | `err.Error(` | 泄原始错误 | `4000502` | — |
| tasks/handler.go:119 | 500 | `err.Error(` | 泄原始错误 | `5000506` | **必须** |
| tasks/handler.go:134 | 500 | `err.Error(` | 泄原始错误 | `5000506` | **必须** |
| tasks/handler.go:152 | 404 | `"task not found"` | 吞错 | `4040501` | — |
| tasks/handler.go:156 | 500 | `err.Error(` | 泄原始错误 | `5000506` | **必须** |
| tasks/handler.go:164 | 404 | `"task not found"` | 吞错 | `4040501` | — |
| tasks/handler.go:168 | 500 | `err.Error(` | 泄原始错误 | `5000506` | **必须** |
| tasks/handler.go:176 | 500 | `"cancel not configured"` | 吞错 | `5000505` | **必须** |
| tasks/handler.go:182 | 404 | `"task not found"` | 吞错 | `4040501` | — |
| tasks/handler.go:186 | 409 | `"cancel not allowed"` | 吞错 | `4090503` | — |
| tasks/handler.go:190 | 500 | `err.Error(` | 泄原始错误 | `5000506` | **必须** |
| tasks/handler.go:195 | 500 | `err.Error(` | 泄原始错误 | `5000506` | **必须** |

### linkhealth（业务域 `14`，2 处）

| 位置 | HTTP | 现有文案 | 现状 | 建议 code | error_detail |
| --- | --- | --- | --- | --- | --- |
| linkhealth/handler.go:17 | 500 | `"link health not configured"` | 吞错 | `5001405` | **必须** |
| linkhealth/handler.go:22 | 500 | `err.Error(` | 泄原始错误 | `5001406` | **必须** |

### common（业务域 `00`，2 处）

| 位置 | HTTP | 现有文案 | 现状 | 建议 code | error_detail |
| --- | --- | --- | --- | --- | --- |
| adminhost/rbac.go:37 | 401 | `"unauthorized"` | 吞错 | `4010007` | 禁止(安全) |
| adminhost/rbac.go:43 | 403 | `"forbidden"` | 吞错 | `4030008` | 禁止(安全) |

### routing（业务域 `10`，1 处）

| 位置 | HTTP | 现有文案 | 现状 | 建议 code | error_detail |
| --- | --- | --- | --- | --- | --- |
| routing/handler.go:24 | 500 | `"registry not configured"` | 吞错 | `5001005` | **必须** |

## 3. 复核要点

1. **业务域归属**：脚本按包名推断，`setup` 内含认证/向导/存储等多类，可能需要细分。
2. **明细码**：按文案关键词规则推断，`400` 类易把「参数错误」与「依赖不可用」混淆，需人工确认。
3. **error_detail 禁止项**：401/403 共 25 处，须确认不泄露「资源是否存在」等侧信道信息。
4. **泄原始错误**：144 处当前把 `err.Error()` 直接放进 message，改造时须分离并过脱敏。
5. **吞错**：219 处为静态文案，需按「用户能否自助解决」判定是否补 error_detail。
