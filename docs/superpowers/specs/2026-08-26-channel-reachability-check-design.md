---
change: channel-reachability-check
role: technical-design
canonical_spec: openspec
---

# 渠道 Telegram 可达性检测 设计

## 1. 目标

进入渠道详情页时，对 Telegram 做一次连通检测（getMe）。用一个「连通状态」元素承载结果，网络不可达时引导到设置页「网络」tab 配置代理，错误不再只进日志。

## 2. 现状与问题

- 创建渠道只落库，不做 getMe 校验。后台 assembler 每 5s 调 `bot.New` → getMe，被墙时报 `context deadline exceeded`，只写日志。
- 前端没有任何连接状态展示，也没有到代理配置的入口。
- 设置页有「网络」tab（`proxy_kind` / `proxy_host` / `proxy_port`），但 Tabs 默认在 account，无法深链。

## 3. 技术决策

### 3.1 后端：一次性检测接口

- `channelapp.Service.CheckReachability(ctx, id)`：
  1. `Store.Get(id)`；
  2. `domain.DecryptCredential(s.Key, ...)` 解密 token；
  3. `GET https://api.telegram.org/bot<token>/getMe`，`http.Client` 默认 transport（继承进程 `HTTPS_PROXY`，即设置页代理），context 5s 超时；
  4. 分类错误返回 `ReachabilityResult`。
- 错误分类纯函数 `classifyGetMeError(err, statusCode)` → `kind`: `ok` / `network` / `auth` / `other`，带面向用户的中文 `message`：
  - HTTP 200 → `ok`
  - HTTP 401 / 403 → `auth`
  - 网络错误（`url.Error`：context deadline exceeded / connection refused / EOF / TLS handshake / no such host / i/o timeout / network unreachable / proxyconnect）→ `network`
  - 其他 → `other`
- 新端点 `POST /api/v1/channels/{id}/check`，返回 `{ ok, kind, message }`；渠道不存在返回 404。
- 不缓存，每次进详情页触发一次。

### 3.2 前端：title 旁的连通状态 tag（不新增独立 Alert）

- `api/channels.ts` 加 `checkChannelReachability(id)`。
- `ChannelDetailPanel` 挂载时（id 变化）用 `useQuery` 调一次，title 旁单个 tag 承载全部状态：
  - 检测中 → 灰色小 tag「检测中…」
  - `ok` → 绿色 tag「连接正常」
  - `network` → amber tag「无法连接 Telegram」，可点直达 `/settings?tab=network`
  - `auth` → 红色 tag「Token 无效」
  - `other` → 灰色 tag「连接失败」，`title` 展示后端 message

### 3.3 设置页深链

- `/_app/settings` 路由支持 search 参数 `?tab=account|storage|network`；`SettingsPage` 的 Tabs 默认值读 search，无参数时默认 account。

## 4. 文案（pixoma-voice）

- 检测中：「检测中…」
- 正常：「连接正常」
- 被墙：「无法连接 Telegram」（可点，aria-label「去设置代理」）
- Token 无效：「Token 无效」
- 其他错误：「连接失败」

## 5. 测试

- Go：分类函数表驱动单测；handler 单测覆盖 200 / 401 / 网络错误 / 404。
- 前端：契约测试覆盖详情页调用 `checkChannelReachability` 与 network 分支的「去设置代理」链接；设置页深链测试。
- `go test ./...`、vitest、`tsc --noEmit` 全绿。
