# Pixoma 接口统一响应格式改造方案

> 状态：待评审（本阶段只出方案，不改代码）
> 范围：`/api/v1/*`（16 个包，约 340 条路由）、`/agent/v1/*`（18 条路由）

## 1. 背景与目标

### 1.1 现状

项目全部 HTTP 接口采用**扁平裸数据**响应：业务数据直接作为响应体，HTTP 状态码只出现在状态行，没有统一封装。

### 1.2 目标

所有 JSON 接口统一为如下结构。

成功：

```json
{ "message": "success", "code": 2000000, "data": { "接口数据": "..." } }
```

失败：

```json
{ "message": "资产内容不存在", "code": 4041201, "data": null }
```

失败（有必要时附加脱敏后的真实原因）：

```json
{
  "message": "模型连接测试失败",
  "code": 5021206,
  "data": null,
  "error_detail": "dial tcp 10.0.0.5:443: connect: connection refused"
}
```

## 2. 现状问题清单

> 配套文档：
>
> - [`docs/api-error-audit.md`](./api-error-audit.md) — 现状审计，372 行明细（回答「现在错在哪」）
> - [`docs/api-error-messages.md`](./api-error-messages.md) — **最终错误文案表**，204 条词条 / 438 个错误返回点（回答「改成什么」）

| # | 问题 | 证据 |
| --- | --- | --- |
| 1 | 无统一封装，业务数据裸返回 | 15 个包各自 `json.NewEncoder(w).Encode(value)` |
| 2 | `writeJSON` 重复定义 15 次 | `internal/httpapi/*/handler.go` |
| 3 | `Content-Type` 两种写法 | `studio` 用 `application/json; charset=utf-8`，其余用 `application/json` |
| 4 | 错误体 3 种形状并存 | `{"error"}` / `{"error","code"}` / 纯文本 |
| 5 | RBAC 401/403 返回**纯文本** | `adminhost/rbac.go:37,43` 使用 `http.Error(w, "unauthorized", 401)` |
| 6 | 错误 `code` 是字符串业务码 | `{"code":"case_delete_needs_ack"}`，与数字码体系冲突 |
| 7 | 成功体形状不一 | `{"ok":true}` / `{"deleted":true}` / 裸对象 / 裸数组 |
| 8 | 204 响应无 body | 10 处 `StatusNoContent`，无法承载封装 |
| 9 | `/healthz` 返回纯文本 | `adminhost/server.go:57` 写入 `"ok"` |
| 10 | 两个消费端与扁平格式强耦合 | 前端 `apiFetch`、`edge-agent` controlplane client |
| 11 | **错误细节被大面积吞掉** | 审计到的 372 个错误返回点中 219 处为静态文案，无任何真实原因 |
| 12 | **原始错误直接塞进 message** | 144 处把 `err.Error()` 拼进 message：不可本地化、未统一脱敏 |
| 13 | **错误文案中英混杂** | 173 条静态英文（`"channel not found"`）vs 2 条中文 |
| 14 | 无错误码注册表 | 码值分散在各 handler，无唯一事实源、无重复校验 |

### 2.1 各包写响应函数分布

| 包 | `writeJSON` | `writeErr` | `writeErrCode` | `writeError` | 调用点 |
| --- | --- | --- | --- | --- | --- |
| studio | ✓（含 charset） | — | — | ✓（`err error`） | 119 |
| setup | ✓ | ✓ | — | — | 92 |
| edges | ✓ | ✓ | ✓ | — | 60 |
| channels | ✓ | ✓ | — | — | 58 |
| cases | ✓ | ✓ | ✓ | — | 39 |
| topics | ✓ | ✓ | ✓ | — | 29 |
| adminusers | ✓ | ✓ | — | — | 27 |
| agent | ✓ | ✓ | — | — | 27 |
| users | ✓ | ✓ | — | — | 27 |
| stats | ✓ | ✓ | — | — | 18 |
| tasks | ✓ | ✓ | — | — | 18 |
| sessions | ✓ | ✓ | — | — | 17 |
| media | ✓ | — | — | ✓（`msg string`） | 10 |
| linkhealth | ✓ | ✓ | — | — | 4 |
| routing | ✓ | — | — | — | 2 |
| **合计** | **15** | **12** | **3** | **2** | **547** |

> 注：`adminhost` 无业务 handler，但其中间件（RBAC）需要纳入改造。

### 2.2 现有字符串 `code` 业务码（需重新映射为数字码）

| 现值 | 位置 | 前端消费方式 |
| --- | --- | --- |
| `case_delete_needs_ack` | `cases/handler.go:240` | i18n 映射 → `cases.deleteNeedsAck` |
| `edge_delete_needs_ack` | `edges/handler.go:393` | 未见前端消费 |
| `topic_default_protected` | `topics/handler.go:188` | i18n 映射 → `topics.deleteDefaultProtected` |
| `topic_delete_needs_ack` | `topics/handler.go:192` | i18n 映射 → `topics.deleteNeedsAck` |
| `not_initialized` | `setup/gate.go:88` | **行为分支**：引导进入初始化向导 |
| `restart_required` | `setup/gate.go:95` | **行为分支**：提示重启 |
| `bucket_not_found` | `setup/handler.go:450` | **行为分支**：提示创建存储桶 |

> 注意：这些字符串码**不只是文案映射，还驱动 UI 行为分支**，因此不能简单丢弃；必须映射为稳定的数字明细码。

### 2.3 现有状态码用量（非测试）

| 状态 | 次数 |
| --- | --- |
| 500 | 157 |
| 400 | 126 |
| 200 | 101 |
| 404 | 45 |
| 409 | 20 |
| 401 | 18 |
| 503 | 18 |
| 201 | 16 |
| 204 | 10 |
| 403 | 7 |
| 202 | 3 |
| 429 | 2 |
| 413 | 2 |
| 408 / 410 / 502 | 各 1 |

### 2.4 `code` 语义冲突（重点）

`/api/v1/setup/blob-test` 存在语义冲突：它在 **HTTP 200 成功响应**中携带业务失败标记：

```go
// setup/handler.go:449
writeJSON(w, http.StatusOK, map[string]any{
    "ok":     false,
    "code":   "bucket_not_found",
    "bucket": body.BlobBucket,
})
```

统一后 `code` 字段被数字码占用，此处的 `code` 必须让位。三种处理方式：

| 方案 | 形态 | 评价 |
| --- | --- | --- |
| A. 改为标准错误 | HTTP 404 + `{"message":"存储桶不存在","code":4040112,"data":{"bucket":"..."}}` | 语义最干净；前端需改为 `catch (ApiError)` 分支 |
| B. 业务态留在 `data` | HTTP 200 + `{"message":"success","code":2000000,"data":{"ok":false,"reason":"bucket_not_found","bucket":"..."}}` | 保持 HTTP 200；但 `code:2000000` 与 `ok:false` 语义矛盾 |
| C. 保留 `ok` 语义但换字段名 | HTTP 200 + `data.reason` | 同 B 的矛盾 |

本方案**推荐 A**：把它当作真正的「资源不存在」错误处理，前端用 `err.code === 4040112` 判定。

### 2.5 其它同类「成功体内含业务态」的接口

`setup/gate.go` 的 `not_initialized` / `restart_required` 使用 HTTP 403，属正常错误路径，无冲突，仅需替换码值。

## 3. 目标规范

### 3.1 统一封装结构

后端类型定义：

```go
// internal/httpapi/response/response.go
type Envelope struct {
    Message     string `json:"message"`
    Code        int    `json:"code"`
    Data        any    `json:"data"`                   // 不加 omitempty：失败时必须显式输出 null
    ErrorDetail string `json:"error_detail,omitempty"` // 仅错误响应、且有必要时出现
}
```

字段约定：

| 字段 | 成功 | 失败 |
| --- | --- | --- |
| `message` | 固定 `"success"` | 稳定的兜底文案（供 edge-agent / curl / 日志用） |
| `code` | `2000000`（或 201/202 对应码） | 7 位业务码 |
| `data` | 业务数据 | `null` |
| `error_detail` | **不出现** | 仅在有必要时出现，承载脱敏后的真实错误 |

补充约定：

- 列表为空时 `data` 返回 `[]`，不得返回 `null`。
- 失败时 `data` 必须是 `null`，且字段不可省略。
- `error_detail` 是**错误响应专属**字段，成功响应不得出现。
- `error_detail` 承载动态内容，`message` 必须是**纯模板文案**（可被前端 i18n 完整替换）。

### 3.1.1 error_detail 分级策略

判定标准：**用户能否据此自助解决问题**。

| 类别 | message | error_detail | 数量（见审计表） |
| --- | --- | --- | --- |
| 4xx 参数校验 / 冲突 | 具体、可操作 | 通常省略 | 174 |
| 4xx 依赖问题（存储桶、模型、数据库） | 具体 | **带上** | 8 |
| 5xx / 502 / 503 | 通用可读短语 | **必须带上真实原因** | 165 |
| 401 / 403 | 通用，不泄露资源是否存在 | **禁止**（安全） | 25 |

示例（模型可达性检查，即你举的场景）：

```json
{
  "message": "模型连接测试失败",
  "code": 5021206,
  "data": null,
  "error_detail": "dial tcp 10.0.0.5:443: connect: connection refused"
}
```

对照现状：`internal/studio/application/model_config.go:261` 的 `sanitizeModelTestError` 已经把真实错误拼进 `message`（`"model connection test failed: dial tcp..."`）。改造后该函数保留脱敏与截断逻辑，但产出物改为 `error_detail`，`message` 变为可翻译的 `"模型连接测试失败"`。

### 3.1.2 error_detail 脱敏（安全要求）

**原始 Go error 会泄露**：DSN（含密码）、内网 IP/端口、SQL 片段、函数调用记录、上游密钥。当前 144 处直接把 `err.Error()` 放进 message，说明**今天已经在泄露**。

脱敏必须放在**序列化边界**（`response.Fail` 内部），而不是各 handler —— 否则 438 个调用点必然漏。规则：

1. 已知密钥模式替换为 `[REDACTED]`（沿用 `sanitizeModelTestError` 的做法并泛化）。
2. 长度上限 512 字符，超出截断并追加 `…`。
3. `500` 类不透传完整 error chain，只取最外层可读信息。
4. 401 / 403 永不产出 `error_detail`。

### 3.2 code 编码规则（7 位）

```text
[HTTP 状态: 3 位][业务域: 2 位][明细: 2 位]
```

- 前 3 位与 HTTP 状态码严格一致。
- **HTTP 状态码保持语义**（404 仍是 404，401 仍是 401），不采用「一律 200」方案。这样 `code` 前缀与状态行始终自洽，且现有鉴权中间件、CORS、代理、前端 401 跳转逻辑不受影响。
- 中间 2 位标识业务域，末 2 位标识该域内的具体原因。

示例：

| 场景 | HTTP | code |
| --- | --- | --- |
| 成功 | 200 | `2000000` |
| 创建成功 | 201 | `2010000` |
| 已接受（异步） | 202 | `2020000` |
| 参数校验失败 | 400 | `4000002` |
| 请求体格式错误 | 400 | `4000009` |
| 未认证 | 401 | `4010007` |
| 无权限 | 403 | `4030008` |
| 用户不存在 | 404 | `4040201` |
| 资产内容不存在 | 404 | `4041201` |
| 删除需确认 | 409 | `4090604` |
| 默认话题不可删 | 409 | `4090903` |
| 依赖不可用 | 503 | `5030005` |
| 内部错误 | 500 | `5000006` |

### 3.3 业务域码表（第 4–5 位）

| 码 | 业务域 | 对应包 |
| --- | --- | --- |
| `00` | 通用 / 跨域 | `sharedkernel`、中间件、未知域 |
| `01` | 认证与初始化 | `setup`（含 `auth`） |
| `02` | 系统账号 | `users` |
| `03` | 管理员账号 | `adminusers` |
| `04` | 会话 | `sessions` |
| `05` | 任务 | `tasks` |
| `06` | 用例 | `cases` |
| `07` | 实例 / 边缘 | `edges` |
| `08` | 渠道 | `channels`（含 menu / text / mcp_users 子资源） |
| `09` | 话题 | `topics` |
| `10` | 路由 | `routing` |
| `11` | 统计 | `stats` |
| `12` | 工作台 | `studio` |
| `13` | 媒体 | `media` |
| `14` | 链路健康 | `linkhealth` |
| `15` | 边缘 Agent | `agent`（`/agent/v1`） |
| `16`–`99` | 预留 | — |

### 3.4 明细码表（第 6–7 位）

通用明细（各域可复用）：

| 码 | 含义 |
| --- | --- |
| `00` | 通用 / 成功 |
| `01` | 资源不存在 |
| `02` | 参数校验失败 |
| `03` | 冲突 / 当前状态不允许 |
| `04` | 需要用户确认（ack） |
| `05` | 依赖不可用 |
| `06` | 内部错误 |
| `07` | 认证失效 |
| `08` | 无权限 |
| `09` | 请求体格式错误 / 过大 |
| `10` | 平台未初始化 |
| `11` | 需要重启生效 |
| `12` | 存储桶不存在 |
| `13`–`99` | 各业务域自定义 |

### 3.4.1 现有字符串码 → 数字码映射表

| 原字符串码 | 新 code | HTTP |
| --- | --- | --- |
| `case_delete_needs_ack` | `4090604` | 409 |
| `edge_delete_needs_ack` | `4090704` | 409 |
| `topic_default_protected` | `4090903` | 409 |
| `topic_delete_needs_ack` | `4090904` | 409 |
| `not_initialized` | `4030110` | 403 |
| `restart_required` | `4030111` | 403 |
| `bucket_not_found` | `4040112` | 404（见 2.4 节） |

### 3.5 7 位是否够用：结论

**够用，且与你最初的 `2000000` 完全一致。**

- 业务域 2 位 → 100 个模块，当前实际使用 16 个。
- 明细 2 位 → 每模块 100 个明细原因。
- 组合上限 10,000 个错误码，足够覆盖本项目规模。
- 若未来需要更细粒度，可在明细位内部再划分（如 `01`–`49` 通用、`50`–`99` 模块专属），无需扩位。

> 说明：你先前提到「后五位业务域」按 8 位计算，但最初的 `2000000` 本身是 7 位（`200` + `0000`）。本方案统一采用 **7 位**，成功码保持 `2000000` 不变。

### 3.6 特殊接口处理

| 接口 | 处理方式 | 理由 |
| --- | --- | --- |
| `/healthz` | **改为 JSON 封装** | 按你的选择统一 |
| AGUI SSE（`studio/agui.go`） | 保持 `text/event-stream` | 流式协议要求，不能封装 |
| media preview（二进制图片） | 保持原样 | 二进制响应，不能封装 |
| CORS `OPTIONS` 预检 | 保持 204 无 body | 浏览器协议要求，非业务响应 |
| 其余 10 处 `204` | **改为 200 + 封装** | 204 按规范不能带 body |

`/healthz` 目标响应：

```json
{ "message": "success", "code": 2000000, "data": { "status": "ok" } }
```

## 4. 后端改造方案

### 4.1 新增共享响应包

新建 `internal/httpapi/response/`，作为唯一响应出口，消除 15 份重复实现。

```go
package response

type Envelope struct {
    Message string `json:"message"`
    Code    int    `json:"code"`
    Data    any    `json:"data"`
}

// 业务域常量，避免裸数字
type Domain int

const (
    DomainCommon    Domain = 0
    DomainSetup     Domain = 1
    DomainUsers     Domain = 2
    DomainAdminUser Domain = 3
    DomainSessions  Domain = 4
    DomainTasks     Domain = 5
    DomainCases     Domain = 6
    DomainEdges     Domain = 7
    DomainChannels  Domain = 8
    DomainTopics    Domain = 9
    DomainRouting   Domain = 10
    DomainStats     Domain = 11
    DomainStudio    Domain = 12
    DomainMedia     Domain = 13
    DomainLinkHeal  Domain = 14
    DomainAgent     Domain = 15
)

// 通用明细常量
const (
    DetailGeneric    = 0
    DetailNotFound   = 1
    DetailValidation = 2
    DetailConflict   = 3
    DetailNeedAck    = 4
    DetailDependency = 5
    DetailInternal   = 6
    DetailAuth       = 7
    DetailForbidden  = 8
    DetailBadBody    = 9
    // 模块专属（见 3.4.1）
    DetailNotInit     = 10
    DetailRestart     = 11
    DetailBucketMiss  = 12
)

// 成功
func OK(w http.ResponseWriter, data any)          // 200 + 2000000
func Created(w http.ResponseWriter, data any)     // 201 + 2010000
func Accepted(w http.ResponseWriter, data any)    // 202 + 2020000

// 失败
func Fail(w http.ResponseWriter, status int, d Domain, detail int, msg string)
func FailErr(w http.ResponseWriter, status int, d Domain, detail int, err error)

// 编码
func Code(status int, d Domain, detail int) int
```

关键实现要求：

1. `Content-Type` 统一为 `application/json; charset=utf-8`（取 studio 的写法，修复第 3 项不一致）。
2. 成功时 `Data` 为 `nil` 需输出 `[]` 还是 `null` 由调用方决定；列表接口必须显式传非 nil 空切片。
3. `Fail` 必须同时设置 `status` 与对应 `code` 前缀，二者由同一个 `status` 参数派生，杜绝不一致。
4. 编码写入失败（`Encode` 出错）时记录日志，但此时 header 已发出，无法改状态码——保持现有 `_ =` 忽略行为或补日志。

### 4.2 替换策略

按包逐个替换，每个包的改造是机械且可验证的：

| 原调用 | 替换为 |
| --- | --- |
| `writeJSON(w, http.StatusOK, v)` | `response.OK(w, v)` |
| `writeJSON(w, http.StatusCreated, v)` | `response.Created(w, v)` |
| `writeJSON(w, http.StatusAccepted, v)` | `response.Accepted(w, v)` |
| `writeErr(w, http.StatusBadRequest, msg)` | `response.Fail(w, 400, response.DomainXxx, response.DetailValidation, msg)` |
| `writeErrCode(w, 409, "case_delete_needs_ack", msg)` | `response.Fail(w, 409, response.DomainCases, response.DetailNeedAck, msg)` |
| `writeError(w, err)`（studio） | `response.FailErr(w, status, domain, detail, err)` |
| `w.WriteHeader(http.StatusNoContent)` | `response.OK(w, nil)`（CORS 预检除外） |

替换后删除各包内的 `writeJSON` / `writeErr` / `writeErrCode` / `writeError` 定义（共 32 处）。

**关键难点**：现有 `writeErr` 调用把状态码作为参数传入，但业务域和明细需要人工判定。438 个调用点需要逐个确认归属域与原因码——这是本次改造的主要人力成本。

### 4.3 中间件与 RBAC 改造

`adminhost/rbac.go:37,43` 当前返回纯文本，需改为：

```go
response.Fail(w, 401, response.DomainCommon, response.DetailAuth, "登录已失效，请重新登录")
response.Fail(w, 403, response.DomainCommon, response.DetailForbidden, "没有操作权限")
```

### 4.4 `/healthz` 改造

`adminhost/server.go:57` 改为 `response.OK(w, map[string]string{"status": "ok"})`。

### 4.5 错误码统一管理（`internal/apierr`）

**决策：码定义在 `internal/apierr`，HTTP 层负责把 `error` 映射为码。**

`apierr` 是全项目错误码的**唯一事实源**，并串联起本方案的其余各点（码 → 业务域、明细、HTTP 状态、i18n key、默认文案）。

```go
package apierr

type Code int

const (
    CodeOK                  Code = 2000000
    CodeCaseDeleteNeedsAck  Code = 4090604
    CodeTopicDefaultProtect Code = 4090903
    CodeModelConnTestFailed Code = 5021206
    CodeBucketNotFound      Code = 4040112
    // ...
)

// Meta 是注册表条目
type Meta struct {
    HTTP       int    // 前 3 位必须等于它
    Domain     int    // 第 4-5 位
    Detail     int    // 第 6-7 位
    I18nKey    string // 前端词条 key
    DefaultMsg string // message 兜底文案
}

var registry = map[Code]Meta{ /* 全量登记 */ }

func MetaOf(c Code) (Meta, bool)
func All() map[Code]Meta
```

**由测试强制的不变量**（`apierr/codes_test.go`）：

| 不变量 | 目的 |
| --- | --- |
| 每个 code 在 registry 中有且仅有一条 | 防止重复与遗漏 |
| `c/10000 == meta.HTTP` | 保证 code 前缀与 HTTP 状态自洽 |
| `meta.Domain`、`meta.Detail` 与 code 后 4 位一致 | 防止码值与登记表脱节 |
| 每个 code 的 `I18nKey` 在前端 zh.json/en.json 中存在 | **强制前后端词条同步** |
| 401/403 类 code 的 `DefaultMsg` 不含资源存在性信息 | 防侧信道泄露 |

前端侧配套：`web/admin/src/lib/api/error-messages.ts` 建立 `code → i18n key` 映射，未知 code 回退渲染后端 `message`。

## 5. 前端改造方案

### 5.1 核心解包（`web/admin/src/lib/api/client.ts`）

`apiFetch` 需要新增一层解包，对上层保持现有返回类型不变（`apiFetch<T>` 仍返回 `T`），因此**多数 api 模块无需改动**——但有 5 个字符串码消费点必须同步改（见 5.4）：

```ts
type Envelope<T> = {
  message: string
  code: number
  data: T
  error_detail?: string
}

// 成功：解包 data
if (res.ok) {
  const env = body as Envelope<T>
  return env.data
}

// 失败：优先用 code 查 i18n，回退到后端 message
const env = body as Envelope<null> | undefined
throw new ApiError(res.status, env?.message ?? `Request failed (${res.status})`, env?.code, env?.error_detail)
```

### 5.2 `ApiError` 适配

`ApiError.code` 当前类型是 `string?`，需改为 `number?`，并新增 `detail`：

```ts
export class ApiError extends Error {
  status: number
  code?: number
  detail?: string
  constructor(status: number, message: string, code?: number, detail?: string) { ... }
}
```

需同步检查 `task-errors.ts` 等对 `ApiError.code` 的使用点。

### 5.2.1 错误展示（承载 error_detail）

当前 `lib/handle-server-error.ts` 是唯一的全局错误出口，走 `toast.error(errMsg)`。toast 放不下长详情，需改为：

1. toast 显示本地化后的 `message`；
2. 当 `detail` 存在时，toast 附带「查看详情」动作 → 弹窗以等宽字体展示 `detail` + 复制按钮。

同时修掉 `handle-server-error.ts:11` 的硬编码英文兜底 `'Something went wrong!'`，改为 i18n key。

### 5.3 message 多语言：前端基于 code 映射

**决策：采用前端映射（方案 A）。**

依据：

| 维度 | 前端 | 后端 |
| --- | --- | --- |
| i18n 基建 | ✅ i18next + react-i18next，zh/en 各 945 行，带语言切换器 | ❌ 无任何 i18n 机制 |
| 现有错误码映射 | ✅ `localized-errors.ts` 已在做 | — |
| 当前错误文案 | — | 173 条静态英文 + 149 条动态，中英混杂 |

响应格式约定：

1. 后端 `message` 是**稳定兜底**，供 edge-agent / curl / 脚本 / 日志使用，不参与浏览器渲染。
2. 前端优先用 `code` 查 i18n key；**code 未知时**才回退渲染 `message`。
3. 后端新增 code 时，必须同步补前端词条，由测试强制一致。
4. 凡带动态内容的文案，动态部分一律移入 `error_detail`，保证 `message` 是纯模板。

不采用方案 B（`Accept-Language` + 后端翻译）的理由：i18n 基建只在前端存在，B 需从零造 Go 翻译管线并重复维护约 200 条双语文案，还要处理 `Vary: Accept-Language` 缓存；而 edge-agent 等机器消费方本就应 switch `code`，翻译对它无收益。

> 注：Telegram bot 有 per-user `language_code`，若将来要让 bot 说用户语言，那是后端 i18n 的独立需求，不影响本 HTTP 接口约定的选择。

### 5.4 字符串码消费点改造（必改，否则行为失效）

| 文件 | 现状 | 改法 |
| --- | --- | --- |
| `lib/api/localized-errors.ts:4-11` | `Record<string,string>` 用字符串码做 i18n 映射 | key 改为数字码：`{ 4090604: 'cases.deleteNeedsAck' }` 等 |
| `features/setup/setup-wizard.tsx:192` | `res.code === 'bucket_not_found'` | 改为 `catch` 分支判定 `err.code === 4040112`（见 2.4 节方案 A） |
| `lib/api/setup.ts:152` | `apiFetch<{ok:boolean; code?:string; bucket?:string}>` | 移除 `code?: string`；`bucket` 移入错误 `data` 或改类型 |
| `features/setup/db-error.ts:128` | 正则匹配 `unauthorized\|not_initialized\|restart_required` | 改为按数字码判定，避免匹配自然语言文案 |
| `lib/api/client.ts:8` | `ApiError.code?: string` | 改为 `code?: number` |

> `db-error.ts` 目前靠正则匹配**错误消息文本**来判断错误类型，本身就很脆弱（文案一改就失效）。本次改造应顺带改为按 `code` 精确判定。

### 5.5 兼容性风险点

| 位置 | 风险 |
| --- | --- |
| 401 全局跳转 | 保持基于 HTTP 状态判断，不受封装影响 |
| `PUBLIC_AUTH_PATHS` | 不受影响 |
| 二进制 / SSE 调用 | 未走 `apiFetch`，不受影响；需确认 studio SSE 未复用 `apiFetch` |
| `res.text()` 空体 | 204 改 200 后不再出现空体 |

## 6. 其他消费端

`apps/edge-agent/internal/controlplane/client.go` 是**第二个独立消费端**，直接把 body 解码为 `Job` 结构：

```go
if err := json.NewDecoder(res.Body).Decode(&job); err != nil { ... }
```

改造后必须同步解包 `data`，涉及 `Claim`、`presence` 等方法。这是最容易被遗漏的破坏点，需与 `edge-agent` 一起发布，否则边缘节点拉取任务会全部失败。

## 7. 测试改造

| 类别 | 数量 | 影响 |
| --- | --- | --- |
| Go handler 测试 | 31 个文件 | 断言响应体，需加一层 `data` 解包 |
| 前端 api 测试 | 21 个文件 | mock 响应需包一层封装 |
| 接口一致性测试 | `*.contract.test.ts` | 需核对 |

建议在测试侧提供统一 helper（如 Go 的 `decodeEnvelope(t, body)`、前端的 `mockEnvelope(data)`），避免 52 个文件各写一遍解包逻辑。

## 8. 分阶段实施计划

| 阶段 | 内容 | 产出 |
| --- | --- | --- |
| P0 | 建立 `response` 包 + `apierr` 码注册表 + 脱敏层 + 测试 helper | 共享基建，可独立评审 |
| P1 | 复核 [`api-error-messages.md`](./api-error-messages.md) 204 条文案，定稿措辞与 code | **错误文案 + 码表定稿** |
| P2 | 把定稿文案写入 `apierr` 注册表 `DefaultMsg` / `I18nKey` | 后端文案单一事实源 |
| P3 | 改造 `adminhost`（中间件、RBAC、healthz、CORS） | 骨架与鉴权路径先统一 |
| P4 | 前端 `client.ts` 解包 + `ApiError` + 错误展示（含 `error_detail` 弹窗）+ zh/en 词条 | 前端就绪，等待后端 |
| P5 | 按包替换 438 个错误返回点（16 个包，可并行/分批） | 后端全量统一 |
| P6 | 同步改造 `edge-agent` controlplane client | 双端一致 |
| P7 | 全量测试修复 + 端到端验证 | 验收 |

建议 P4 按「低风险包 → 高风险包」排序：`linkhealth` / `routing` / `media` → `adminusers` / `users` / `sessions` / `tasks` / `stats` → `topics` / `cases` / `edges` / `channels` / `agent` / `setup` → `studio`。

**P1 是本方案的关键前置**：204 条文案已按 voice profile 出稿，但仍是草案，需逐条复核措辞与动作句；定稿后才能写入 `apierr` 并机械替换 P5。

**过渡期策略**：由于前端与后端必须同步切换，建议不做双格式兼容（避免长期技术债），而是在同一分支/同一次发布中原子切换。若需要灰度，可在 `apiFetch` 中按 `data` 字段是否存在做临时兼容，但应在 P6 后移除。

## 9. 风险

| 风险 | 影响 | 缓解 |
| --- | --- | --- |
| 438 个错误返回点人工判定业务域易出错 | 错误码语义不一致 | P1 定稿文案表 + code review 抽查 |
| `edge-agent` 遗漏改造 | 边缘节点全线拉取失败 | P5 独立阶段 + 端到端联调 |
| 前端与后端切换不同步 | 全站白屏/报错 | 原子发布，不跨版本 |
| 52 个测试文件改动量大 | 改造周期拉长 | 统一 helper，机械替换 |
| 204 → 200 影响 DELETE 语义 | 客户端兼容 | 前端同步适配；无外部第三方消费方 |
| 字符串码改数字码后前端行为分支失效 | 初始化向导、重启提示、建桶提示、删除确认文案全部失灵 | 按 3.4.1 映射表逐个改造 5 个消费点，并补前端测试 |
| `blob-test` 的 `ok:false` 语义冲突未决 | 该接口无法纳入统一格式 | 按 2.4 节先定方案（推荐 A） |
| `db-error.ts` 依赖错误文案正则 | 文案一改即失效 | 顺带改为按数字码判定 |
| **`error_detail` 泄露密钥/内网信息** | 安全问题：DSN、内网 IP、上游密钥外泄 | 脱敏收敛在 `response.Fail` 单点（3.1.2），401/403 禁产出，补单测覆盖脱敏规则 |
| **`message` 改为纯模板后丢失动态信息** | 用户看不到真实原因 | 动态内容必须移入 `error_detail`，审计表逐条标注 |
| **后端新增 code 未同步前端词条** | 前端回退显示后端兜底文案 | `apierr` 测试强制校验词条存在（4.5） |

## 10. 验收标准

1. 全部 `/api/v1/*` 与 `/agent/v1/*` JSON 接口返回 `{message, code, data}`；错误响应按需附加 `error_detail`。
2. 成功响应 `message` 恒为 `"success"`，`code` 为 `2xxxxxx`，且**不含** `error_detail`。
3. 失败响应 `data` 恒为 `null`，`code` 前 3 位等于 HTTP 状态码。
4. 5xx / 502 / 503 错误**必须**带 `error_detail`；401 / 403 **不得**带。
5. 所有 `error_detail` 通过脱敏校验，不含密钥、DSN、内网地址。
6. `message` 为纯模板文案，不含动态拼接内容。
7. `apierr` 注册表不变量测试全部通过（唯一性、前缀自洽、词条存在）。
8. 每个 code 在前端 zh.json / en.json 中都有对应词条。
9. 全仓不存在 `writeJSON` / `writeErr` / `writeErrCode` / `writeError` 的包内重复定义。
10. 全仓不存在 `{"error": ...}` 形状的错误体。
11. `/healthz`、RBAC 401/403 均为 JSON 封装。
12. 空列表返回 `data: []`。
13. Go 与前端全部测试通过；`edge-agent` 联调通过。
14. `Content-Type` 全站统一为 `application/json; charset=utf-8`。
15. **文案合规**：`docs/api-error-messages.md` 204 条全部写入，无 `请 / 您 / 进行 / 完成 / 实施 / 执行` 等 voice profile 禁用词。
16. **指导性**：每条错误文案含界面上的可执行动作（重新保存 / 刷新页面 / 等待后重试 / 去某个页面配置），不出现「操作失败」这类无信息文案。
17. **不教阅读方法**：`message` 不出现「查看 error_detail」「查看日志」「定位原因」；技术原因由 `error_detail` 承载并由前端直接展示。
18. **code ↔ 文案 1:1**：不存在同一 code 对应多条文案（`apierr` 测试强制）。

## 11. 已定决策与待确认项

### 11.1 已定决策

| # | 决策 | 结论 |
| --- | --- | --- |
| 1 | code 位数 | **7 位**（`[HTTP:3][业务域:2][明细:2]`），成功码 `2000000` |
| 2 | 成功码粒度 | 固定 `2000000`（通用域 `00`） |
| 3 | 错误封装 | 与成功同构，`data` 为 `null` |
| 4 | 错误码定义位置 | `internal/apierr` 定义码，HTTP 层做映射 |
| 5 | i18n 方案 | **前端基于 code 映射**；后端 `message` 作兜底 |
| 6 | `error_detail` 策略 | 按错误类别分级（5xx 必须、401/403 禁止） |
| 7 | 脱敏位置 | 序列化边界（`response.Fail`）统一脱敏 |
| 8 | 特殊接口 | `/healthz` 封装；SSE 与二进制保持原样 |
| 9 | 审计表 | 已产出 `docs/api-error-audit.md`（372 行明细，抽取时未含 `writeError`） |
| 10 | **错误文案** | 已产出 [`docs/api-error-messages.md`](./api-error-messages.md)，**204 条词条 / 438 个错误返回点**，按 `docs/voice-profile.md` 出稿 |
| 11 | **指导性承载位置** | 放在 `message` 内（**事实句 + 动作句**两段式），不新增 `error_hint` 字段 |
| 12 | **code ↔ 文案** | **1:1**；粒度到「域 + HTTP + 文案」，同一文案不跨 code |
| 13 | **i18n key** | `apiError.<7位code>`，由 `apierr` 测试强制校验 zh/en 双语存在 |
| 14 | **实体命名** | `/api/v1/users` 用「用户」（bot 用户，profile 例外）；`/api/v1/adminusers` 用「管理员」 |

### 11.2 待确认项

1. **业务域码表**（3.3 节）是否认可？尤其 `menus` / `text-templates` 并入 `channels`（`08`）的处理。
2. **`blob-test` 语义冲突**（2.4 节）采用哪个方案？推荐 A（改为 HTTP 404 + `4040101`）。
3. **204 → 200** 是否接受（涉及 10 处，含 studio 与 agent）。
4. **是否保留 HTTP 语义状态码**：本方案保留（推荐），即失败时 HTTP 仍为 4xx/5xx，而非一律 200。
5. **是否需要灰度兼容期**：本方案建议原子切换，不做双格式兼容。
6. **字符串码是否彻底移除**：本方案彻底移除（`code` 只承载数字码），行为分支改由数字码驱动。若希望保留可读字符串，需额外引入 `reason` 字段，会破坏「三字段统一」。
7. **文案逐条复核**：`api-error-messages.md` 204 条是否直接作为 P1 基线？重点是 §6 列出的四项（动作句可执行性、配置项键名、401/403 脱敏、英文词条）。

## 12. 实施记录

本轮已完成 P0–P7 全部阶段，后端与前端同步切换，无兼容层。

### 12.1 后端

| 文件 | 内容 |
| --- | --- |
| `internal/apierr/apierr.go` | 204 个错误码注册表，由 `docs/api-error-messages.md` 生成，文件头标注「不要手改」 |
| `internal/apierr/locale_test.go` | 契约测试：每个码在 zh/en 词条里都存在，且反向无死词条；英文词条不含中文 |
| `internal/response/response.go` | `Envelope`、`OK` / `OKStatus` / `Fail` / `FailErr` / `FailCode`、`StatusOf`、`detailAllowed` |
| `internal/response/sanitize.go` | `error_detail` 脱敏：URL 凭证、键值对密钥、Bearer 令牌、500 字截断（按 UTF-8 边界） |
| `internal/httpapi/*` | 16 个包的 438 个调用点全部改写；32 个包内 `writeJSON` / `writeErr` / `textWrite*` / `menuWrite*` 辅助函数删除 |
| `internal/httpapi/apitest/apitest.go` | 测试辅助：`DataBytes` / `DataReader` 自动解包 `data` |
| `apps/edge-agent/.../envelope.go` | 边缘代理侧的解包契约，非成功码直接报错，不再分别处理 204 |

`internal/httpapi/setup/handler.go` 的 `bucket_not_found` 分支从「HTTP 200 + 字符串错误码」改为「HTTP 404 + `4040101`」，bucket 名走 `error_detail`。

`internal/httpapi/studio/handler.go` 新增 `failFromError`，把 `domain.ErrNotFound` / `ErrAlreadyExists` / `domain.ErrInvalid` 等映射到对应业务码，覆盖 44 处调用。

保留原样的三处：AGUI 的 SSE 流、媒体二进制预览、CORS `OPTIONS` 预检 204。这三者不是业务 JSON 响应。

### 12.2 前端

| 文件 | 内容 |
| --- | --- |
| `web/admin/src/lib/api/client.ts` | `apiFetch` 解包 `data`；`ApiError` 携带 `{status, code: number, detail?, message}`；无封装响应原样返回 |
| `web/admin/src/lib/api/error-copy.ts` | `apiErrorMessage`（按 `apiError.<code>` 查词条，退回后端文案）、`apiErrorDetail`、`apiErrorText` |
| `web/admin/src/lib/i18n/locales/zh.json` / `en.json` | 各 204 条 `apiError.<code>` 词条 |
| `web/admin/src/features/setup/db-error.ts` / `blob-error.ts` | 模式匹配改为看 `error_detail`（用户文案与技术原文已分离） |
| `web/admin/src/features/setup/setup-wizard.tsx` | bucket 缺失改为捕获 `ApiError.code === 4040101` |
| `web/admin/src/lib/api/localized-errors.ts` | 删除冲突分支从字符串码改为数字码（`4090604` / `4090904` / `4090913`） |

`message` 只承载「事实句 + 动作句」，技术原文由 `error_detail` 承载并由前端直接展示，文案里不出现「查看日志」「定位原因」这类阅读指引。

### 12.3 验证结果

- `go test ./internal/... ./apps/...` 全部通过。
- `web/admin` vitest 541 项通过。
- 全仓无 `writeJSON` / `writeErr` / `writeError` / `textWrite*` / `menuWrite*` 定义，无 `{"error": ...}` 形状。
- 仅剩两处直接 `WriteHeader`，即 CORS 预检与二进制预览，均为有意保留。
- 204 个码与 zh/en 词条一一对应，无缺失、无多余、无空值。

### 12.4 已知遗留

1. `internal/httpapi/config-topology/topology-card.contract.test.ts` 的「卡片位于数据面板顶部」断言在 `feat-studio` 分支上已失败，与本次改造无关（`web/admin/src/features/dashboard/workbench-data-board.tsx` 未改动）。
2. `web/admin` 的 `tsc -b` 报错集中在 `src/features/studio/studio-chat.tsx` 与 `src/components/ai-elements/prompt-input.tsx`，同属分支既有改动。
3. 生成器的明细号分配在同一（HTTP，业务域）分组内插入新条目时会整体后移。继续往已有分组添加错误码前，需要先改成稳定分配方案。
