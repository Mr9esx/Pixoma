# 消息平台 Telegram 可达性检测 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 进入消息平台详情页时对 Telegram 做一次 getMe 连通检测，用一个连通状态元素展示结果，网络不可达时引导到 `/settings?tab=network` 配置代理。

**Architecture:** 后端在 `channelapp.Service` 增加 `CheckReachability`（解密 token → getMe，5s 超时，默认 transport 继承进程 `HTTPS_PROXY`），错误分类为 `ok/network/auth/other`；新增 `POST /api/v1/channels/{id}/check`。前端消息平台详情页挂载时调用一次，按 `kind` 渲染单个状态元素（检测中 / 正常 / 被墙引导 / Token 无效 / 其他错误）；设置页支持 `?tab=network` 深链。

**Tech Stack:** Go 1.x (chi, gorm, go-telegram/bot), React 19 + TanStack Router/Query, Tailwind v4, vitest。

## Global Constraints

- 工作分支：`feat-init`（用户选项 current，不建 worktree）。
- 文案遵守 `docs/voice-profile.md` 与 `web/admin/src/lib/i18n/copy-quality.test.ts`：中文不出现「请」做软化词（`^请` 或 `[。！？，]请` 会被 contract test 拒绝）。
- TDD：每个任务先写失败测试，看它失败，再实现到绿。
- 后端命令：`go test ./internal/channel/application/... ./internal/httpapi/channels/...`
- 前端命令：`cd web/admin && npx vitest run src/features/channels/channel-layout.contract.test.ts src/features/settings/settings-page.contract.test.ts && npx tsc --noEmit`

---

### Task 1: 后端 reachability 分类 + 服务 + 端点

**Files:**
- Create: `internal/channel/application/reachability.go`
- Create: `internal/channel/application/reachability_test.go`
- Modify: `internal/channel/application/service.go`（`Service` 加字段与方法）
- Modify: `internal/httpapi/channels/handler.go`（路由 + handler）
- Modify: `internal/httpapi/channels/handler_test.go`（helper 返回值 + 新测试）

**Interfaces:**
- Produces: `application.ReachabilityKind`（`"ok" | "network" | "auth" | "other"`）、`application.ReachabilityResult{OK bool; Kind ReachabilityKind; Message string}`、`Service.CheckReachability(ctx context.Context, id string) (ReachabilityResult, error)`、`Service.CheckTelegram func(ctx, token) (ReachabilityResult, error)` 测试注入点。

- [ ] **Step 1: 写分类函数失败测试**

`internal/channel/application/reachability_test.go`：

```go
package application

import (
	"context"
	"errors"
	"net/url"
	"testing"
)

func TestClassifyGetMeError(t *testing.T) {
	cases := []struct {
		name     string
		status   int
		err      error
		wantOK   bool
		wantKind ReachabilityKind
	}{
		{"ok", 200, nil, true, ReachabilityOK},
		{"unauthorized", 401, nil, false, ReachabilityAuth},
		{"forbidden", 403, nil, false, ReachabilityAuth},
		{"deadline", 0, context.DeadlineExceeded, false, ReachabilityNetwork},
		{"connection refused", 0, &url.Error{Op: "Post", URL: "https://api.telegram.org", Err: errors.New("dial tcp 149.154.167.220:443: connect: connection refused")}, false, ReachabilityNetwork},
		{"eof", 0, errors.New("unexpected EOF"), false, ReachabilityNetwork},
		{"no such host", 0, &url.Error{Err: errors.New("dial tcp: lookup api.telegram.org: no such host")}, false, ReachabilityNetwork},
		{"tls timeout", 0, &url.Error{Err: errors.New("net/http: TLS handshake timeout")}, false, ReachabilityNetwork},
		{"other error", 0, errors.New("boom"), false, ReachabilityOther},
		{"server error", 500, nil, false, ReachabilityOther},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyGetMeError(tc.status, tc.err)
			if got.OK != tc.wantOK || got.Kind != tc.wantKind {
				t.Fatalf("got %+v", got)
			}
		})
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/channel/application/...`
Expected: FAIL，`classifyGetMeError undefined`。

- [ ] **Step 3: 实现分类函数与检测函数**

`internal/channel/application/reachability.go`：

```go
package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ReachabilityKind string

const (
	ReachabilityOK      ReachabilityKind = "ok"
	ReachabilityNetwork ReachabilityKind = "network"
	ReachabilityAuth    ReachabilityKind = "auth"
	ReachabilityOther   ReachabilityKind = "other"
)

type ReachabilityResult struct {
	OK      bool             `json:"ok"`
	Kind    ReachabilityKind `json:"kind"`
	Message string           `json:"message"`
}

const reachabilityTimeout = 5 * time.Second

func classifyGetMeError(statusCode int, err error) ReachabilityResult {
	switch {
	case err == nil && statusCode == http.StatusOK:
		return ReachabilityResult{OK: true, Kind: ReachabilityOK}
	case err == nil && (statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden):
		return ReachabilityResult{OK: false, Kind: ReachabilityAuth, Message: "unauthorized"}
	case err != nil && isNetworkError(err):
		return ReachabilityResult{OK: false, Kind: ReachabilityNetwork, Message: err.Error()}
	case err != nil:
		return ReachabilityResult{OK: false, Kind: ReachabilityOther, Message: err.Error()}
	default:
		return ReachabilityResult{OK: false, Kind: ReachabilityOther, Message: fmt.Sprintf("telegram api status %d", statusCode)}
	}
}

func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	var ue *url.Error
	if errors.As(err, &ue) {
		return networkMarker(ue.Error()) || networkMarker(ue.Err.Error())
	}
	return networkMarker(err.Error())
}

var networkMarkers = []string{
	"context deadline exceeded",
	"connection refused",
	"connection reset",
	"eof",
	"tls handshake timeout",
	"i/o timeout",
	"no such host",
	"network is unreachable",
	"proxyconnect tcp",
	"client.timeout",
}

func networkMarker(msg string) bool {
	m := strings.ToLower(msg)
	for _, marker := range networkMarkers {
		if strings.Contains(m, marker) {
			return true
		}
	}
	return false
}

func checkTelegramReachability(ctx context.Context, token string) (ReachabilityResult, error) {
	ctx, cancel := context.WithTimeout(ctx, reachabilityTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.telegram.org/bot"+token+"/getMe", nil)
	if err != nil {
		return ReachabilityResult{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return classifyGetMeError(0, err), nil
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	return classifyGetMeError(resp.StatusCode, nil), nil
}
```

- [ ] **Step 4: 运行确认通过**

Run: `go test ./internal/channel/application/...`
Expected: PASS。

- [ ] **Step 5: 写服务方法失败测试（经 handler）**

先改 `internal/httpapi/channels/handler_test.go` 的 helper，让它返回 `*channelapp.Service`，并同步改既有调用点：

```go
func openChannelsServer(t *testing.T) (*httptest.Server, *channelapp.Service) {
	t.Helper()
	gdb, err := db.Open(db.Options{DSN: "file:ch_http_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &channelpersist.ChannelRow{}); err != nil {
		t.Fatal(err)
	}
	svc := &channelapp.Service{
		Store: channelpersist.NewGormRepository(gdb),
		Key:   make([]byte, 32),
	}
	h := &channelsapi.Handler{Svc: svc}
	r := chi.NewRouter()
	r.Route("/api/v1/channels", func(r chi.Router) {
		h.Mount(r)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv, svc
}
```

原测试开头改为 `srv, _ := openChannelsServer(t)`。新增测试：

```go
func TestChannelsHandler_CheckReachability(t *testing.T) {
	srv, svc := openChannelsServer(t)
	res, m := post(t, srv.URL+"/api/v1/channels", map[string]any{
		"platform": "telegram", "name": "主机器人", "token": "1234567890",
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("create status=%d body=%v", res.StatusCode, m)
	}
	id := m["id"].(string)

	var gotToken string
	svc.CheckTelegram = func(_ context.Context, token string) (channelapp.ReachabilityResult, error) {
		gotToken = token
		return channelapp.ReachabilityResult{OK: false, Kind: channelapp.ReachabilityNetwork, Message: "dial timeout"}, nil
	}

	checkRes, err := http.Post(srv.URL+"/api/v1/channels/"+id+"/check", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer checkRes.Body.Close()
	var out channelapp.ReachabilityResult
	_ = json.NewDecoder(checkRes.Body).Decode(&out)
	if checkRes.StatusCode != http.StatusOK {
		t.Fatalf("check status=%d", checkRes.StatusCode)
	}
	if gotToken != "1234567890" {
		t.Fatalf("token=%q", gotToken)
	}
	if out.OK || out.Kind != channelapp.ReachabilityNetwork {
		t.Fatalf("out=%+v", out)
	}

	notFoundRes, err := http.Post(srv.URL+"/api/v1/channels/missing/check", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	notFoundRes.Body.Close()
	if notFoundRes.StatusCode != http.StatusNotFound {
		t.Fatalf("not found status=%d", notFoundRes.StatusCode)
	}
}
```

Run: `go test ./internal/httpapi/channels/...`
Expected: FAIL，`undefined: channelapp.Service.CheckTelegram`（编译失败即验证入口缺失）。

- [ ] **Step 6: 实现服务方法与端点**

`internal/channel/application/service.go` 的 `Service` struct 增加字段，并新增方法：

```go
type Service struct {
	Store Repository
	Key   []byte
	Notify notify.Publisher
	DeleteWithCleanup func(ctx context.Context, channelID string) ([]sharedkernel.ChatID, error)
	// CheckTelegram overrides the default getMe probe (tests).
	CheckTelegram func(ctx context.Context, token string) (ReachabilityResult, error)
	now           func() time.Time
}

func (s *Service) CheckReachability(ctx context.Context, id string) (ReachabilityResult, error) {
	ch, err := s.Store.Get(ctx, id)
	if err != nil {
		return ReachabilityResult{}, err
	}
	cred, err := domain.DecryptCredential(s.Key, ch.CredentialCiphertext)
	if err != nil {
		return ReachabilityResult{}, err
	}
	probe := s.CheckTelegram
	if probe == nil {
		probe = checkTelegramReachability
	}
	return probe(ctx, cred.BotToken)
}
```

`internal/httpapi/channels/handler.go` 挂路由并新增 handler：

```go
func (h *Handler) Mount(r chi.Router) {
	// 在既有路由后追加
	r.Post("/{id}/check", h.CheckReachability)
}

func (h *Handler) CheckReachability(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Svc == nil {
		writeErr(w, http.StatusInternalServerError, "channel service not configured")
		return
	}
	id := chi.URLParam(r, "id")
	res, err := h.Svc.CheckReachability(r.Context(), id)
	if errors.Is(err, domain.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "channel not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}
```

- [ ] **Step 7: 运行确认通过**

Run: `go test ./internal/channel/application/... ./internal/httpapi/channels/...`
Expected: PASS。

- [ ] **Step 8: Commit**

```bash
git add internal/channel/application/reachability.go internal/channel/application/reachability_test.go internal/channel/application/service.go internal/httpapi/channels/handler.go internal/httpapi/channels/handler_test.go
git commit -m "feat: channel Telegram reachability check API"
```

---

### Task 2: 前端消息平台详情页连通状态

**Files:**
- Modify: `web/admin/src/lib/api/channels.ts`
- Modify: `web/admin/src/features/channels/channel-detail-panel.tsx`
- Modify: `web/admin/src/lib/i18n/locales/zh.json`
- Modify: `web/admin/src/lib/i18n/locales/en.json`
- Modify: `web/admin/src/features/channels/channel-layout.contract.test.ts`

**Interfaces:**
- Consumes: `POST /api/v1/channels/{id}/check` → `{ ok: boolean; kind: 'ok'|'network'|'auth'|'other'; message: string }`
- Produces: `checkChannelReachability(id: string): Promise<ChannelReachability>`；详情页单元素状态区（loading / success Alert / warn Alert + 去设置代理 / destructive Alert / ErrorBanner）。

- [ ] **Step 1: 写前端契约测试（失败）**

`web/admin/src/features/channels/channel-layout.contract.test.ts` 追加：

```ts
it('detail panel runs a one-time reachability check and guides to proxy settings', () => {
  const detail = readFileSync(join(here, 'channel-detail-panel.tsx'), 'utf8')
  const api = readFileSync(join(here, '../../lib/api/channels.ts'), 'utf8')
  expect(api).toMatch(/checkChannelReachability/)
  expect(api).toMatch(/\/check'/)
  expect(detail).toMatch(/checkChannelReachability\(id\)/)
  expect(detail).toMatch(/channels\.reachabilityNetwork/)
  expect(detail).toMatch(/to='\/settings'/)
  expect(detail).toMatch(/search=\{\{ tab: 'network' \}\}/)
})
```

Run: `cd web/admin && npx vitest run src/features/channels/channel-layout.contract.test.ts`
Expected: FAIL，断言找不到 `checkChannelReachability`。

- [ ] **Step 2: 实现 API 函数**

`web/admin/src/lib/api/channels.ts` 追加：

```ts
export type ChannelReachability = {
  ok: boolean
  kind: 'ok' | 'network' | 'auth' | 'other'
  message: string
}

export function checkChannelReachability(id: string) {
  return apiFetch<ChannelReachability>(
    `/api/v1/channels/${encodeURIComponent(id)}/check`,
    { method: 'POST' }
  )
}
```

- [ ] **Step 3: 实现详情页连通状态元素**

`web/admin/src/features/channels/channel-detail-panel.tsx`：

1. import 追加 `CheckCircle2, KeyRound, WifiOff` 到 lucide-react；`Alert, AlertDescription, AlertTitle` 到 `@/components/ui/alert`；`checkChannelReachability, type ChannelReachability` 到 `@/lib/api/channels`。
2. `channelQuery` 之后加：

```tsx
const reachabilityQuery = useQuery({
  queryKey: ['channels', id, 'reachability'],
  queryFn: () => checkChannelReachability(id),
  enabled: Boolean(ch),
})
```

3. 在 header/meta 区块（`</div>` 后、`updateMutation.isError` 之前）插入：

```tsx
{reachabilityQuery.isPending ? (
  <p className='text-xs text-muted-foreground'>
    {t('channels.checkingReachability')}
  </p>
) : reachabilityQuery.isError ? (
  <ErrorBanner
    message={errorMessage(reachabilityQuery.error)}
    onRetry={() => void reachabilityQuery.refetch()}
  />
) : reachabilityQuery.data ? (
  <ChannelReachabilityAlert result={reachabilityQuery.data} />
) : null}
```

4. 文件末尾新增组件：

```tsx
function ChannelReachabilityAlert({ result }: { result: ChannelReachability }) {
  const { t } = useTranslation()
  if (result.kind === 'ok') {
    return (
      <Alert variant='success'>
        <CheckCircle2 aria-hidden='true' />
        <AlertTitle>{t('channels.reachabilityOK')}</AlertTitle>
      </Alert>
    )
  }
  if (result.kind === 'network') {
    return (
      <Alert variant='warn'>
        <WifiOff aria-hidden='true' />
        <AlertTitle>{t('channels.reachabilityNetwork')}</AlertTitle>
        <AlertDescription>
          <Button asChild size='sm'>
            <Link to='/settings' search={{ tab: 'network' }}>
              {t('channels.reachabilityNetworkAction')}
            </Link>
          </Button>
        </AlertDescription>
      </Alert>
    )
  }
  if (result.kind === 'auth') {
    return (
      <Alert variant='destructive'>
        <KeyRound aria-hidden='true' />
        <AlertTitle>{t('channels.reachabilityAuth')}</AlertTitle>
      </Alert>
    )
  }
  return (
    <ErrorBanner
      message={result.message || t('common.errorGeneric')}
    />
  )
}
```

- [ ] **Step 4: 补 i18n 文案**

`web/admin/src/lib/i18n/locales/zh.json` 的 `channels` 段追加：

```json
"checkingReachability": "检测中…",
"reachabilityOK": "Telegram 连接正常",
"reachabilityNetwork": "无法连接 Telegram。若网络受限，在设置中配置代理",
"reachabilityNetworkAction": "去设置代理",
"reachabilityAuth": "Bot Token 无效，检查 token"
```

`web/admin/src/lib/i18n/locales/en.json` 的 `channels` 段追加：

```json
"checkingReachability": "Checking…",
"reachabilityOK": "Telegram connected",
"reachabilityNetwork": "Cannot reach Telegram. If the network is restricted, configure a proxy in settings",
"reachabilityNetworkAction": "Configure proxy",
"reachabilityAuth": "Bot token invalid, check the token"
```

- [ ] **Step 5: 运行确认通过**

Run: `cd web/admin && npx vitest run src/features/channels/channel-layout.contract.test.ts src/lib/i18n/copy-quality.test.ts && npx tsc --noEmit`
Expected: PASS，且 copy-quality 无「请」违规。

- [ ] **Step 6: Commit**

```bash
git add web/admin/src/lib/api/channels.ts web/admin/src/features/channels/channel-detail-panel.tsx web/admin/src/lib/i18n/locales/zh.json web/admin/src/lib/i18n/locales/en.json web/admin/src/features/channels/channel-layout.contract.test.ts
git commit -m "feat: show channel reachability status with proxy guidance"
```

---

### Task 3: 设置页深链 + 全量验证

**Files:**
- Modify: `web/admin/src/routes/_app/settings/index.tsx`
- Modify: `web/admin/src/features/settings/settings-page.tsx`
- Modify: `web/admin/src/features/settings/settings-page.contract.test.ts`

**Interfaces:**
- Consumes: 详情页按钮 `Link to='/settings' search={{ tab: 'network' }}`
- Produces: `/_app/settings` 接受 `?tab=account|storage|network`；`SettingsPage({ initialTab? })`。

- [ ] **Step 1: 写设置页契约测试（失败）**

`web/admin/src/features/settings/settings-page.contract.test.ts` 追加（用该文件已有的 read helper）：

```ts
it('settings deep-links to a tab via ?tab=', () => {
  const route = read('../../routes/_app/settings/index.tsx')
  const page = read('settings-page.tsx')
  expect(route).toMatch(/validateSearch/)
  expect(route).toMatch(/search\.tab/)
  expect(page).toMatch(/initialTab/)
})
```

Run: `cd web/admin && npx vitest run src/features/settings/settings-page.contract.test.ts`
Expected: FAIL。

- [ ] **Step 2: 实现路由 search 与初始 tab**

`web/admin/src/routes/_app/settings/index.tsx` 整体替换为：

```tsx
import { createFileRoute, useSearch } from '@tanstack/react-router'
import { SettingsPage } from '@/features/settings/settings-page'

export const Route = createFileRoute('/_app/settings/')({
  component: SettingsRoute,
  validateSearch: (search: Record<string, unknown>) => ({
    tab: typeof search.tab === 'string' ? search.tab : undefined,
  }),
})

function SettingsRoute() {
  const { tab } = useSearch({ from: '/_app/settings/' })
  return <SettingsPage initialTab={tab} />
}
```

`web/admin/src/features/settings/settings-page.tsx`：

1. `SettingsPage` 改为接收 `initialTab` 并传给编辑器：

```tsx
export function SettingsPage({ initialTab }: { initialTab?: string }) {
  // ...
  const tab =
    initialTab === 'storage' || initialTab === 'network' ? initialTab : 'account'
  // ...
  {q.data?.configured && q.data.settings ? (
    <SettingsEditor
      key={q.dataUpdatedAt}
      initial={q.data.settings}
      initialTab={tab}
    />
  ) : null}
}

function SettingsEditor({
  initial,
  initialTab,
}: {
  initial: SetupDraft
  initialTab?: string
}) {
  // ...
  <Tabs defaultValue={initialTab ?? 'account'} className='max-w-2xl gap-4'>
```

- [ ] **Step 3: 运行确认通过**

Run: `cd web/admin && npx vitest run src/features/channels/channel-layout.contract.test.ts src/features/settings/settings-page.contract.test.ts && npx tsc --noEmit`
Expected: PASS。

- [ ] **Step 4: 全量验证**

Run: `go test ./... && cd web/admin && npx vitest run && npx tsc --noEmit`
Expected: 全绿。

- [ ] **Step 5: Commit**

```bash
git add web/admin/src/routes/_app/settings/index.tsx web/admin/src/features/settings/settings-page.tsx web/admin/src/features/settings/settings-page.contract.test.ts
git commit -m "feat: settings network tab deep link"
```
