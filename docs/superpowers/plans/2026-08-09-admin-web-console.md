---
change: admin-web-console
design-doc: docs/superpowers/specs/2026-08-09-admin-web-console-design.md
base-ref: 250fb2d0a4e8145bfed243cc507322be7a73d23f
---

# admin-web-console Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `web/admin` 落地可运行管理控制台：裁剪 shadcn-admin、配置化侧栏、中英 i18n、中等 Dashboard，以及实例 / Case / Task / User / Session 的 Master–Detail（方案 B）页面，仅通过 `apps/admin-api` 完成基础管理。

**Architecture:** 浏览器 SPA（Vite + React + TanStack Router + Query）经单一 `lib/api` 客户端访问 `VITE_ADMIN_API_BASE`；壳子用集中 `config/menu.ts` 驱动侧栏；五资源页共用左右分栏 Master–Detail 布局；Dashboard 用既有 list API 前端聚合；无鉴权、无前端 mock。

**Tech Stack:** pnpm、Vite、React、TypeScript、TanStack Router、TanStack Query、shadcn/ui（自 satnaing/shadcn-admin 裁剪）、i18next + react-i18next、Vitest（纯函数单测）、对接 `apps/admin-api` `/api/v1/*`

**依据：**
- Design Doc：`docs/superpowers/specs/2026-08-09-admin-web-console-design.md`
- 布局方案库（本期 = B）：`docs/superpowers/specs/2026-08-09-admin-list-detail-layout-options.md`
- OpenSpec：`docs/openspec/changes/admin-web-console/{design,tasks}.md` 与 `specs/admin-web-shell|admin-resource-pages/spec.md`

## Global Constraints

- 产物语言：zh-CN（本计划与用户可见文案默认中文）
- 包管理器：**pnpm only**（禁止 npm/yarn 作为项目主锁）
- **禁止** Clerk / 强制登录挡板；打开根路径必须直接进 Dashboard
- **禁止** 独立前端 mock（MSW、假数据开关、`VITE_USE_MOCK` 等）；验收一律打真 admin-api
- **禁止** 前端直连 DB 或 bot 管理残留路径；正式数据路径只经 `VITE_ADMIN_API_BASE`
- **禁止** TG / Menu 落库管理入口与路由
- 菜单顺序固定：Dashboard → 实例 → Case → Task → User → Session
- 五资源列表/详情统一 **Master–Detail 方案 B**（左列表右详情；窄屏可降级）；Dashboard 整页不分栏
- Case：结构化多段表单为主路径；`bindings.workflow` / `input_schema` 允许受控 JSON 兜底并标明「高级/原始」
- User / Session：**只读**；Task：详情可取消；实例：CRUD + system/queue/tasks 观测
- API 错误体字段为 `{"error":"..."}`（与 admin-api 对齐）；展示优先用后端 `error` 文案
- 默认语言 `zh`；完整 `en`；偏好 key：`admin-locale:v1`
- Dashboard 计数口径：基于当前拉取样本/`limit`，UI 与 README 须注明
- 改动范围以 `web/admin/**` 为主；不改 admin-api 契约；触及架构表述时同步 `docs/architecture/`
- Comfy Mock 与 bot 执行链路本 change 不改

---

## 文件结构（先锁定职责）

基座裁剪后，以如下树为准（若上游模板目录名略有差异，**迁移到此结构**，勿保留 `_authenticated` / Clerk 路由树）：

| 路径 | 职责 |
|---|---|
| `web/admin/package.json` | pnpm 脚本：`dev` / `build` / `test` / `lint` |
| `web/admin/.env.example` | `VITE_ADMIN_API_BASE=http://127.0.0.1:8081` |
| `web/admin/.env.development` | 本地默认基址（可 gitignore；至少有 example） |
| `web/admin/README.md` | 安装、环境变量、先起 admin-api 再 `pnpm dev`、无鉴权警示 |
| `web/admin/src/main.tsx` | 挂载 Router + QueryClient + i18n；**无** AuthProvider |
| `web/admin/src/routes/__root.tsx` | 根：主题 + Toaster + Outlet |
| `web/admin/src/routes/_app.tsx` | 管理布局：侧栏 + 顶栏（主题/语言）+ `<Outlet />` |
| `web/admin/src/routes/_app/index.tsx` | Dashboard `/` |
| `web/admin/src/routes/_app/instances/route.tsx` | 实例 Master–Detail 壳 |
| `web/admin/src/routes/_app/instances/index.tsx` | `/instances` 未选中 |
| `web/admin/src/routes/_app/instances/$instanceId.tsx` | `/instances/$instanceId` |
| `web/admin/src/routes/_app/cases/**` | Case 同构路由 |
| `web/admin/src/routes/_app/tasks/**` | Task 同构路由 |
| `web/admin/src/routes/_app/users/**` | User 同构路由 |
| `web/admin/src/routes/_app/sessions/**` | Session 同构路由 |
| `web/admin/src/config/menu.ts` | 菜单单一真相源（顺序/path/titleKey/icon） |
| `web/admin/src/components/layout/app-sidebar.tsx` | 读 `menu.ts` 渲染侧栏 |
| `web/admin/src/components/layout/language-switcher.tsx` | 中英切换 |
| `web/admin/src/components/master-detail/master-detail-shell.tsx` | 方案 B：左列表右详情 + 窄屏降级 |
| `web/admin/src/components/feedback/{empty-state,error-banner,loading-skeleton}.tsx` | 通用空态/错误/骨架 |
| `web/admin/src/lib/api/client.ts` | `apiFetch`：baseURL、JSON、错误解析 |
| `web/admin/src/lib/api/types.ts` | 与 admin-api DTO 对齐的 TS 类型 |
| `web/admin/src/lib/api/{instances,cases,tasks,users,sessions}.ts` | 各资源 API 函数 |
| `web/admin/src/lib/i18n/{index.ts,locales/zh.json,locales/en.json}` | i18n 初始化与文案 |
| `web/admin/src/lib/dashboard/aggregate.ts` | Dashboard 纯聚合函数 |
| `web/admin/src/features/dashboard/dashboard-page.tsx` | 中等 Dashboard UI |
| `web/admin/src/features/instances/{list-panel,detail-panel,form}.tsx` | 实例左右栏 |
| `web/admin/src/features/cases/{list-panel,case-form}.tsx` | Case 列表 + 多段表单 |
| `web/admin/src/features/tasks/{list-panel,detail-panel}.tsx` | Task + 取消 |
| `web/admin/src/features/users/{list-panel,detail-panel}.tsx` | User 只读 |
| `web/admin/src/features/sessions/{list-panel,detail-panel}.tsx` | Session 只读 |
| `web/admin/src/lib/**/*.test.ts` | Vitest：菜单顺序、locale、聚合、api 错误解析 |
| `web/admin/vitest.config.ts` | Vitest + path alias |

**删除/不保留（裁剪清单）：**
- `routes/(auth)/**`、`routes/clerk/**`、任何 `ClerkProvider` / `SignedIn` / 登录强制 `beforeLoad`
- demo 业务页（chats、users demo、settings demo、tasks demo 等与本期无关者）
- 任何 MSW / mock service worker 入口

---

### Task 1: 脚手架 — 裁剪 shadcn-admin 进 `web/admin`（pnpm）

**Files:**
- Replace/Create: `web/admin/**`（保留并改写现有占位 `web/admin/README.md`）
- Create: `web/admin/.env.example`
- Create: `web/admin/vitest.config.ts`（可先最小配置，后续 Task 填测试）

**Interfaces:**
- Produces: 可 `pnpm install && pnpm dev` 启动的 Vite 应用；脚本名固定为 `dev` / `build` / `test`

- [x] **Step 1: 拉取基座到临时目录并拷入 `web/admin`**

```bash
# 在仓库根执行；锁定克隆深度，避免带入无关历史
rm -rf /tmp/shadcn-admin-src
git clone --depth 1 https://github.com/satnaing/shadcn-admin.git /tmp/shadcn-admin-src

# 备份现有 README
cp web/admin/README.md /tmp/admin-readme.bak

# 清空占位后拷贝（保留 web/admin 目录本身）
find web/admin -mindepth 1 -delete
rsync -a --exclude .git /tmp/shadcn-admin-src/ web/admin/

# 恢复并稍后改写 README
cp /tmp/admin-readme.bak web/admin/README.md
```

- [x] **Step 2: 确认包管理器为 pnpm 并安装**

```bash
cd web/admin
# 若 package.json 无 packageManager 字段，追加：
# "packageManager": "pnpm@9.15.0"
corepack enable
pnpm install
```

Expected: 生成/更新 `pnpm-lock.yaml`；无 `package-lock.json` / `yarn.lock`（若有则删除）。

- [x] **Step 3: 写最小 README 安装段（先能跑起来）**

将 `web/admin/README.md` 改为至少包含：

```markdown
# Admin Web Console

Pixoma 管理控制台。只通过 `apps/admin-api` 取数；**本期无鉴权**，勿对公网暴露。

## 前置

1. 先启动 admin-api（默认 `127.0.0.1:8081`），见 `apps/admin-api/README.md`
2. Node 20+，启用 pnpm（`corepack enable`）

## 安装与启动

```bash
cd web/admin
cp .env.example .env.development
pnpm install
pnpm dev
```

环境变量：`VITE_ADMIN_API_BASE`（例：`http://127.0.0.1:8081`）
```

- [x] **Step 4: 添加 `.env.example`**

```bash
# web/admin/.env.example
VITE_ADMIN_API_BASE=http://127.0.0.1:8081
```

- [x] **Step 5: 验证开发服务器可启动**

```bash
cd web/admin && pnpm dev
```

Expected: 终端打印本地 URL（通常 `http://localhost:5173`）；浏览器能打开（此时可能仍有登录挡板，Task 2 去掉）。

- [x] **Step 6: Commit**

```bash
git add web/admin
git commit -m "$(cat <<'EOF'
chore(admin-web): scaffold shadcn-admin into web/admin with pnpm

EOF
)"
```

---

### Task 2: 去掉鉴权挡板，默认进入管理布局

**Files:**
- Modify: `web/admin/src/main.tsx`（或等价入口）
- Modify/Delete: 含 Clerk / auth guard 的 route 与 provider
- Create/Rename: `web/admin/src/routes/_app.tsx` 为无鉴权布局壳
- Delete: `web/admin/src/routes/(auth)/**`、`clerk/**` 等（按实际模板路径）

**Interfaces:**
- Produces: 访问 `/` 无需登录即可看到侧栏布局（内容可仍是占位）

- [x] **Step 1: 定位鉴权入口**

```bash
cd web/admin
rg -n "Clerk|SignedIn|auth|beforeLoad|redirect.*sign-in|login" src/routes src/main.tsx src/components --glob '!**/node_modules/**'
```

记录所有强制跳转登录的 `beforeLoad` / Provider。

- [x] **Step 2: 移除 Auth Provider 与登录路由**

在 `main.tsx`（示意）：

```tsx
// 删除类似：
// import { ClerkProvider } from '@clerk/clerk-react'
// <ClerkProvider>...</ClerkProvider>

import { RouterProvider, createRouter } from '@tanstack/react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { routeTree } from './routeTree.gen'

const queryClient = new QueryClient()
const router = createRouter({ routeTree, context: { queryClient } })

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
}
```

删除 `(auth)`、`clerk` 路由文件；从 `package.json` 移除 `@clerk/*` 依赖后 `pnpm install`。

- [x] **Step 3: 建立无鉴权 `_app` 布局路由**

`web/admin/src/routes/_app.tsx`：

```tsx
import { createFileRoute, Outlet } from '@tanstack/react-router'
import { AppSidebar } from '@/components/layout/app-sidebar' // Task 4 可先占位简单 nav
import { SidebarProvider, SidebarInset } from '@/components/ui/sidebar'

export const Route = createFileRoute('/_app')({
  component: AppLayout,
})

function AppLayout() {
  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <header className="flex h-14 items-center gap-2 border-b px-4">
          {/* Task 5: LanguageSwitcher + theme toggle */}
        </header>
        <div className="flex-1 p-4">
          <Outlet />
        </div>
      </SidebarInset>
    </SidebarProvider>
  )
}
```

将原 `_authenticated` 下页面迁到 `_app`；根 `/` 指向 Dashboard 占位：

`web/admin/src/routes/_app/index.tsx`：

```tsx
import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_app/')({
  component: () => <div>Dashboard placeholder</div>,
})
```

未知路径：在 `__root.tsx` 或专用 `$.tsx` 重定向到 `/`。

- [x] **Step 4: 重新生成路由树并手测**

```bash
cd web/admin && pnpm exec tsr generate && pnpm dev
```

手测：打开 `/` → 无登录页；侧栏/顶栏布局可见。

- [x] **Step 5: Commit**

```bash
git add web/admin
git commit -m "$(cat <<'EOF'
feat(admin-web): remove auth gates so console opens without login

EOF
)"
```

---

### Task 3: API 客户端 — `VITE_ADMIN_API_BASE` + 类型

**Files:**
- Create: `web/admin/src/lib/api/client.ts`
- Create: `web/admin/src/lib/api/types.ts`
- Create: `web/admin/src/lib/api/client.test.ts`
- Modify: `web/admin/vitest.config.ts`、`package.json` scripts

**Interfaces:**
- Produces:
  - `apiFetch<T>(path: string, init?: RequestInit): Promise<T>`
  - `ApiError`：`{ status: number; message: string }`
  - 类型：`ComfyInstance`、`CaseRecord`、`TaskRecord`、`UserRecord`、`SessionRecord`（字段名与 admin-api JSON 一致，snake_case）

- [x] **Step 1: 写失败测试（错误体解析）**

`web/admin/src/lib/api/client.test.ts`：

```ts
import { describe, it, expect, vi, afterEach } from 'vitest'
import { apiFetch, ApiError } from './client'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('apiFetch', () => {
  it('throws ApiError with backend error message', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: 'case not found' }), {
          status: 404,
          headers: { 'Content-Type': 'application/json' },
        }),
      ),
    )
    await expect(apiFetch('/api/v1/cases/missing')).rejects.toMatchObject({
      status: 404,
      message: 'case not found',
    } satisfies Partial<ApiError>)
  })

  it('returns parsed JSON on 2xx', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify([{ id: 'gpu-1' }]), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
      ),
    )
    const data = await apiFetch<{ id: string }[]>('/api/v1/comfy-instances')
    expect(data[0].id).toBe('gpu-1')
  })
})
```

- [x] **Step 2: 跑测试确认失败**

```bash
cd web/admin && pnpm test -- src/lib/api/client.test.ts
```

Expected: FAIL（模块/导出不存在）

- [x] **Step 3: 实现 `types.ts`（对齐 admin-api DTO）**

```ts
// web/admin/src/lib/api/types.ts
export type ComfyInstance = {
  id: string
  base_url: string
  enabled: boolean
  capabilities: string[]
  created_at: string
  updated_at: string
}

export type CaseInputField = {
  key: string
  type: string
  required: boolean
  skip_allowed?: boolean
  description?: string
  preview?: string
}

export type CaseOutputField = {
  key: string
  type: string
  description?: string
  media_type?: string
}

export type InputBinding = { key: string; node_id: string; field_path: string }
export type OutputBinding = { key: string; node_id: string; index?: number }

export type CaseRecord = {
  id: string
  name: string
  description?: string
  preview?: string
  price: number
  tags?: string[]
  menu_key?: string
  categories?: string[]
  inputs: CaseInputField[]
  outputs: CaseOutputField[]
  bindings: {
    workflow: Record<string, unknown>
    inputs: InputBinding[]
    outputs: OutputBinding[]
  }
  input_schema: Record<string, unknown>
  enabled: boolean
}

export type TaskRecord = {
  id: string
  session_id: string
  chat_id?: number
  case_id: string
  status: string
  instance_id?: string
  prompt_id?: string
  error_code?: string
  error_message?: string
  created_at: string
  updated_at: string
}

export type UserRecord = {
  id: string
  tg_user_id: number
  username: string
  first_name: string
  last_name: string
  language_code: string
  last_seen_at: string
  created_at: string
  updated_at: string
}

export type SessionDraft = {
  key: string
  text?: string
  number?: number
  bool?: boolean
  blob?: unknown
  skipped?: boolean
}

export type SessionRecord = {
  id: string
  user_id: string
  chat_id: number
  case_id: string
  status: string
  current_input_index: number
  input_keys: string[]
  draft: Record<string, SessionDraft>
  created_at: string
  updated_at: string
}

export type ListParams = {
  q?: string
  limit?: number
  offset?: number
  created_from?: string
  created_to?: string
  [key: string]: string | number | boolean | undefined
}
```

- [x] **Step 4: 实现 `client.ts`**

```ts
// web/admin/src/lib/api/client.ts
export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

function baseURL(): string {
  const raw = import.meta.env.VITE_ADMIN_API_BASE as string | undefined
  if (!raw) {
    throw new Error('VITE_ADMIN_API_BASE is not set')
  }
  return raw.replace(/\/$/, '')
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const url = `${baseURL()}${path.startsWith('/') ? path : `/${path}`}`
  let res: Response
  try {
    res = await fetch(url, {
      ...init,
      headers: {
        Accept: 'application/json',
        ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
        ...init?.headers,
      },
    })
  } catch {
    throw new ApiError(
      0,
      'Network error: check VITE_ADMIN_API_BASE and admin-api CORS',
    )
  }

  const text = await res.text()
  let body: unknown = undefined
  if (text) {
    try {
      body = JSON.parse(text)
    } catch {
      body = text
    }
  }

  if (!res.ok) {
    const msg =
      typeof body === 'object' &&
      body !== null &&
      'error' in body &&
      typeof (body as { error: unknown }).error === 'string'
        ? (body as { error: string }).error
        : `Request failed (${res.status})`
    throw new ApiError(res.status, msg)
  }

  return body as T
}

export function toQuery(params?: Record<string, string | number | boolean | undefined | null>): string {
  if (!params) return ''
  const sp = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined || v === null || v === '') continue
    sp.set(k, String(v))
  }
  const s = sp.toString()
  return s ? `?${s}` : ''
}
```

- [x] **Step 5: 跑测试确认通过**

```bash
cd web/admin && pnpm test -- src/lib/api/client.test.ts
```

Expected: PASS

- [x] **Step 6: Commit**

```bash
git add web/admin/src/lib/api web/admin/vitest.config.ts web/admin/package.json
git commit -m "$(cat <<'EOF'
feat(admin-web): add admin-api client bound to VITE_ADMIN_API_BASE

EOF
)"
```

---

### Task 4: 菜单配置 + 侧栏 + 路由绑定

**Files:**
- Create: `web/admin/src/config/menu.ts`
- Create: `web/admin/src/config/menu.test.ts`
- Create: `web/admin/src/components/layout/app-sidebar.tsx`
- Create: 五资源路由占位（`instances|cases|tasks|users|sessions`）
- Modify: `_app.tsx` 使用侧栏

**Interfaces:**
- Produces:
  - `MENU_ITEMS: readonly MenuItem[]`，顺序不可变
  - `MenuItem = { id: string; titleKey: string; path: string; icon: LucideIcon }`
  - 路径：`/`、`/instances`、`/cases`、`/tasks`、`/users`、`/sessions`

- [x] **Step 1: 写失败测试（菜单顺序）**

```ts
// web/admin/src/config/menu.test.ts
import { describe, it, expect } from 'vitest'
import { MENU_ITEMS } from './menu'

describe('MENU_ITEMS', () => {
  it('keeps required order', () => {
    expect(MENU_ITEMS.map((m) => m.id)).toEqual([
      'dashboard',
      'instances',
      'cases',
      'tasks',
      'users',
      'sessions',
    ])
  })

  it('maps paths', () => {
    expect(MENU_ITEMS.map((m) => m.path)).toEqual([
      '/',
      '/instances',
      '/cases',
      '/tasks',
      '/users',
      '/sessions',
    ])
  })
})
```

- [x] **Step 2: 跑测试确认失败**

```bash
cd web/admin && pnpm test -- src/config/menu.test.ts
```

Expected: FAIL

- [x] **Step 3: 实现 `menu.ts` 与侧栏**

```ts
// web/admin/src/config/menu.ts
import {
  LayoutDashboard,
  Server,
  Boxes,
  ListTodo,
  Users,
  MessagesSquare,
  type LucideIcon,
} from 'lucide-react'

export type MenuItem = {
  id: string
  titleKey: string
  path: string
  icon: LucideIcon
}

export const MENU_ITEMS: readonly MenuItem[] = [
  { id: 'dashboard', titleKey: 'menu.dashboard', path: '/', icon: LayoutDashboard },
  { id: 'instances', titleKey: 'menu.instances', path: '/instances', icon: Server },
  { id: 'cases', titleKey: 'menu.cases', path: '/cases', icon: Boxes },
  { id: 'tasks', titleKey: 'menu.tasks', path: '/tasks', icon: ListTodo },
  { id: 'users', titleKey: 'menu.users', path: '/users', icon: Users },
  { id: 'sessions', titleKey: 'menu.sessions', path: '/sessions', icon: MessagesSquare },
] as const
```

侧栏用 `Link` / `useRouterState` 高亮当前项；`titleKey` 暂时可先显示 `id`，Task 5 接 i18n。

为五资源建占位页（例）：

```tsx
// web/admin/src/routes/_app/instances/index.tsx
import { createFileRoute } from '@tanstack/react-router'
export const Route = createFileRoute('/_app/instances/')({
  component: () => <div data-testid="instances-page">Instances</div>,
})
```

同理 `cases` / `tasks` / `users` / `sessions`。清理所有无关 demo 菜单项。

- [x] **Step 4: 跑测试 + 手测导航**

```bash
cd web/admin && pnpm test -- src/config/menu.test.ts && pnpm exec tsr generate
```

手测：侧栏六项顺序正确；点击进入对应路由；刷新布局保持。

- [x] **Step 5: Commit**

```bash
git add web/admin/src/config web/admin/src/components/layout web/admin/src/routes
git commit -m "$(cat <<'EOF'
feat(admin-web): add configurable sidebar menu and resource routes

EOF
)"
```

---

### Task 5: 中英 i18n（默认 zh + 顶栏切换）

**Files:**
- Create: `web/admin/src/lib/i18n/index.ts`
- Create: `web/admin/src/lib/i18n/locales/zh.json`
- Create: `web/admin/src/lib/i18n/locales/en.json`
- Create: `web/admin/src/lib/i18n/locale.test.ts`
- Create: `web/admin/src/components/layout/language-switcher.tsx`
- Modify: `main.tsx` 初始化 i18n；侧栏改用 `t(titleKey)`

**Interfaces:**
- Produces:
  - `LOCALE_STORAGE_KEY = 'admin-locale:v1'`
  - `getStoredLocale(): 'zh' | 'en'`
  - `setStoredLocale(locale: 'zh' | 'en'): void`
  - 默认 `'zh'`

- [x] **Step 1: 写失败测试**

```ts
// web/admin/src/lib/i18n/locale.test.ts
import { describe, it, expect, beforeEach } from 'vitest'
import { getStoredLocale, setStoredLocale, LOCALE_STORAGE_KEY } from './index'

beforeEach(() => {
  localStorage.clear()
})

describe('locale persistence', () => {
  it('defaults to zh', () => {
    expect(getStoredLocale()).toBe('zh')
  })

  it('persists en', () => {
    setStoredLocale('en')
    expect(localStorage.getItem(LOCALE_STORAGE_KEY)).toBe('en')
    expect(getStoredLocale()).toBe('en')
  })
})
```

- [x] **Step 2: 跑测试确认失败**

```bash
cd web/admin && pnpm test -- src/lib/i18n/locale.test.ts
```

Expected: FAIL

- [x] **Step 3: 安装 i18n 并实现**

```bash
cd web/admin && pnpm add i18next react-i18next
```

```ts
// web/admin/src/lib/i18n/index.ts
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import zh from './locales/zh.json'
import en from './locales/en.json'

export const LOCALE_STORAGE_KEY = 'admin-locale:v1'
export type AppLocale = 'zh' | 'en'

export function getStoredLocale(): AppLocale {
  const v = localStorage.getItem(LOCALE_STORAGE_KEY)
  return v === 'en' ? 'en' : 'zh'
}

export function setStoredLocale(locale: AppLocale): void {
  localStorage.setItem(LOCALE_STORAGE_KEY, locale)
}

export async function initI18n() {
  const lng = typeof window !== 'undefined' ? getStoredLocale() : 'zh'
  await i18n.use(initReactI18next).init({
    resources: {
      zh: { translation: zh },
      en: { translation: en },
    },
    lng,
    fallbackLng: 'zh',
    interpolation: { escapeValue: false },
  })
  return i18n
}

export { i18n }
```

`zh.json` / `en.json` 至少包含壳子键（后续任务补齐页面键）：

```json
{
  "menu": {
    "dashboard": "Dashboard",
    "instances": "实例",
    "cases": "Case",
    "tasks": "Task",
    "users": "User",
    "sessions": "Session"
  },
  "common": {
    "loading": "加载中…",
    "empty": "暂无数据",
    "selectItem": "请选择一项",
    "save": "保存",
    "create": "新建",
    "cancel": "取消",
    "retry": "重试",
    "errorGeneric": "请求失败",
    "successSaved": "已保存",
    "sampleNote": "基于当前拉取样本（受 limit 限制）"
  },
  "lang": { "zh": "中文", "en": "English" }
}
```

英文文件对应翻译（`instances`→`Instances` 等）。`LanguageSwitcher`：按钮切换 `i18n.changeLanguage` + `setStoredLocale`。

- [x] **Step 4: 跑测试 + 手测刷新保留语言**

```bash
cd web/admin && pnpm test -- src/lib/i18n/locale.test.ts
```

Expected: PASS；手测切 en 刷新仍为英文。

- [x] **Step 5: Commit**

```bash
git add web/admin/src/lib/i18n web/admin/src/components/layout/language-switcher.tsx web/admin/package.json pnpm-lock.yaml
git commit -m "$(cat <<'EOF'
feat(admin-web): add zh/en i18n with persisted language switcher

EOF
)"
```

---

### Task 6: Master–Detail 壳（方案 B）+ 通用反馈组件

**Files:**
- Create: `web/admin/src/components/master-detail/master-detail-shell.tsx`
- Create: `web/admin/src/components/feedback/empty-state.tsx`
- Create: `web/admin/src/components/feedback/error-banner.tsx`
- Create: `web/admin/src/components/feedback/loading-skeleton.tsx`

**Interfaces:**
- Produces:
  - `MasterDetailShell({ list, detail, hasSelection }: { list: ReactNode; detail: ReactNode; hasSelection: boolean })`
  - 桌面：`grid` 左约 `320–380px` + 右自适应；窄屏：`hasSelection` 时只显示 detail，提供返回列表回调可选
  - `EmptyState` / `ErrorBanner` / `LoadingSkeleton` 文案走 i18n

- [ ] **Step 1: 实现 MasterDetailShell**

```tsx
// web/admin/src/components/master-detail/master-detail-shell.tsx
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'

type Props = {
  list: ReactNode
  detail: ReactNode
  hasSelection: boolean
  onBackToList?: () => void
  className?: string
}

export function MasterDetailShell({
  list,
  detail,
  hasSelection,
  onBackToList,
  className,
}: Props) {
  const { t } = useTranslation()
  return (
    <div
      className={cn(
        'grid h-[calc(100vh-5rem)] gap-0 overflow-hidden rounded-md border',
        'md:grid-cols-[minmax(280px,360px)_1fr]',
        className,
      )}
    >
      <aside
        className={cn(
          'min-h-0 overflow-auto border-r',
          hasSelection ? 'hidden md:block' : 'block',
        )}
      >
        {list}
      </aside>
      <section
        className={cn(
          'min-h-0 overflow-auto p-4',
          !hasSelection ? 'hidden md:block' : 'block',
        )}
      >
        {hasSelection ? (
          <>
            {onBackToList ? (
              <button
                type="button"
                className="mb-3 text-sm underline md:hidden"
                onClick={onBackToList}
              >
                {t('common.backToList', { defaultValue: '返回列表' })}
              </button>
            ) : null}
            {detail}
          </>
        ) : (
          <p className="text-muted-foreground">{t('common.selectItem')}</p>
        )}
      </section>
    </div>
  )
}
```

同步补 `zh`/`en` 的 `common.backToList`。

- [ ] **Step 2: 实现反馈三件套**

```tsx
// empty-state.tsx — 显示 t('common.empty') + 可选 action
// error-banner.tsx — 显示 message + Retry 按钮
// loading-skeleton.tsx — 几条 pulse 条即可
```

- [ ] **Step 3: 用实例路由占位接入壳（验证布局）**

在 `/instances` 临时用假列表 2 项验证选中高亮与窄屏降级（下一 Task 换真 API；**不要**引入持久 mock 开关，此占位提交前删掉或直接进入 Task 7）。

手测：桌面左右同屏；缩窄窗口先列表再详情。

- [ ] **Step 4: Commit**

```bash
git add web/admin/src/components/master-detail web/admin/src/components/feedback web/admin/src/lib/i18n/locales
git commit -m "$(cat <<'EOF'
feat(admin-web): add Master-Detail shell (layout B) and feedback primitives

EOF
)"
```

---

### Task 7: 资源 API 模块 + Query hooks 约定

**Files:**
- Create: `web/admin/src/lib/api/instances.ts`
- Create: `web/admin/src/lib/api/cases.ts`
- Create: `web/admin/src/lib/api/tasks.ts`
- Create: `web/admin/src/lib/api/users.ts`
- Create: `web/admin/src/lib/api/sessions.ts`
- Create: `web/admin/src/lib/api/query-keys.ts`

**Interfaces:**
- Produces（函数签名固定，后续页面只消费这些）：

```ts
// instances
listInstances(): Promise<ComfyInstance[]>
getInstance(id: string): Promise<ComfyInstance>
createInstance(body: { id: string; base_url: string; enabled?: boolean; capabilities?: string[] }): Promise<ComfyInstance>
patchInstance(id: string, body: { base_url?: string; enabled?: boolean; capabilities?: string[] }): Promise<ComfyInstance>
deleteInstance(id: string): Promise<void>
getInstanceSystem(id: string): Promise<unknown>
getInstanceQueue(id: string): Promise<unknown>
listInstanceTasks(id: string, params?: { limit?: number }): Promise<TaskRecord[]>

// cases
listCases(params?: { q?: string; enabled?: boolean; menu_key?: string; limit?: number; offset?: number }): Promise<CaseRecord[]>
getCase(id: string): Promise<CaseRecord>
createCase(body: Omit<CaseRecord, never>): Promise<CaseRecord> // body 含 document 字段 + enabled
patchCase(id: string, body: Partial<CaseRecord>): Promise<CaseRecord>
enableCase(id: string): Promise<CaseRecord>
disableCase(id: string): Promise<CaseRecord>

// tasks
listTasks(params?: { status?: string; q?: string; limit?: number; offset?: number }): Promise<TaskRecord[]>
getTask(id: string): Promise<TaskRecord>
cancelTask(id: string): Promise<TaskRecord>

// users / sessions — GET only
listUsers(params?: { q?: string; tg_user_id?: number; limit?: number; offset?: number }): Promise<UserRecord[]>
getUser(id: string): Promise<UserRecord>
listSessions(params?: { user_id?: string; status?: string; q?: string; limit?: number; offset?: number }): Promise<SessionRecord[]>
getSession(id: string): Promise<SessionRecord>
```

- Query key 工厂：`queryKeys.instances.all`、`queryKeys.instances.detail(id)` 等。

- [ ] **Step 1: 实现 `instances.ts`（示例）**

```ts
import { apiFetch, toQuery } from './client'
import type { ComfyInstance, TaskRecord } from './types'

export function listInstances() {
  return apiFetch<ComfyInstance[]>('/api/v1/comfy-instances')
}

export function getInstance(id: string) {
  return apiFetch<ComfyInstance>(`/api/v1/comfy-instances/${encodeURIComponent(id)}`)
}

export function createInstance(body: {
  id: string
  base_url: string
  enabled?: boolean
  capabilities?: string[]
}) {
  return apiFetch<ComfyInstance>('/api/v1/comfy-instances', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function patchInstance(
  id: string,
  body: { base_url?: string; enabled?: boolean; capabilities?: string[] },
) {
  return apiFetch<ComfyInstance>(`/api/v1/comfy-instances/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
  })
}

export function deleteInstance(id: string) {
  return apiFetch<void>(`/api/v1/comfy-instances/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}

export function getInstanceSystem(id: string) {
  return apiFetch<unknown>(`/api/v1/comfy-instances/${encodeURIComponent(id)}/system`)
}

export function getInstanceQueue(id: string) {
  return apiFetch<unknown>(`/api/v1/comfy-instances/${encodeURIComponent(id)}/queue`)
}

export function listInstanceTasks(id: string, params?: { limit?: number }) {
  return apiFetch<TaskRecord[]>(
    `/api/v1/comfy-instances/${encodeURIComponent(id)}/tasks${toQuery(params)}`,
  )
}
```

其余资源按 README curl 路径同样实现（cases 含 `POST .../enable|disable`）。

- [ ] **Step 2: 对跑着的 admin-api 做一次冒烟（可选但推荐）**

```bash
# 另开终端：make run-admin-api
cd web/admin
node -e "
const b=process.env.VITE_ADMIN_API_BASE||'http://127.0.0.1:8081'
fetch(b+'/api/v1/comfy-instances').then(r=>r.status).then(console.log)
"
```

Expected: `200`

- [ ] **Step 3: Commit**

```bash
git add web/admin/src/lib/api
git commit -m "$(cat <<'EOF'
feat(admin-web): add typed API modules for all admin resources

EOF
)"
```

---

### Task 8: Dashboard（中等：卡片 + 简单分布）

**Files:**
- Create: `web/admin/src/lib/dashboard/aggregate.ts`
- Create: `web/admin/src/lib/dashboard/aggregate.test.ts`
- Create: `web/admin/src/features/dashboard/dashboard-page.tsx`
- Modify: `web/admin/src/routes/_app/index.tsx`
- Modify: i18n 文案

**Interfaces:**
- Produces:
  - `aggregateDashboard({ instances, cases, tasks }): DashboardStats`
  - `DashboardStats`：`instanceTotal`、`instanceEnabled`、`caseEnabled`、`taskByStatus: Record<string, number>`
  - 页面：各卡片独立 `useQuery`，失败互不影响；卡片可 `Link` 到对应资源

- [ ] **Step 1: 写失败测试**

```ts
import { describe, it, expect } from 'vitest'
import { aggregateDashboard } from './aggregate'

describe('aggregateDashboard', () => {
  it('counts instances and task statuses', () => {
    const stats = aggregateDashboard({
      instances: [
        { id: 'a', enabled: true },
        { id: 'b', enabled: false },
      ],
      cases: [
        { id: 'c1', enabled: true },
        { id: 'c2', enabled: false },
      ],
      tasks: [
        { id: 't1', status: 'pending' },
        { id: 't2', status: 'pending' },
        { id: 't3', status: 'succeeded' },
      ],
    })
    expect(stats.instanceTotal).toBe(2)
    expect(stats.instanceEnabled).toBe(1)
    expect(stats.caseEnabled).toBe(1)
    expect(stats.taskByStatus).toEqual({ pending: 2, succeeded: 1 })
  })
})
```

- [ ] **Step 2: 跑测试确认失败**

```bash
cd web/admin && pnpm test -- src/lib/dashboard/aggregate.test.ts
```

Expected: FAIL

- [ ] **Step 3: 实现聚合与页面**

```ts
// aggregate.ts
export type AggregateInput = {
  instances: { id: string; enabled: boolean }[]
  cases: { id: string; enabled: boolean }[]
  tasks: { id: string; status: string }[]
}

export type DashboardStats = {
  instanceTotal: number
  instanceEnabled: number
  caseEnabled: number
  taskByStatus: Record<string, number>
}

export function aggregateDashboard(input: AggregateInput): DashboardStats {
  const taskByStatus: Record<string, number> = {}
  for (const t of input.tasks) {
    taskByStatus[t.status] = (taskByStatus[t.status] ?? 0) + 1
  }
  return {
    instanceTotal: input.instances.length,
    instanceEnabled: input.instances.filter((i) => i.enabled).length,
    caseEnabled: input.cases.filter((c) => c.enabled).length,
    taskByStatus,
  }
}
```

页面用三个独立 query（`limit` 建议 Dashboard 用 `200`，并显示 `common.sampleNote`）：

```tsx
// 伪结构
function InstanceCard() {
  const q = useQuery({ queryKey: queryKeys.instances.all, queryFn: listInstances })
  if (q.isError) return <ErrorBanner message={q.error.message} onRetry={q.refetch} />
  if (q.isLoading) return <LoadingSkeleton />
  // 展示数字 + Link to /instances
}
```

简单分布：用 CSS 条形（`div` + width %）即可，不必上重型图表库。

- [ ] **Step 4: 跑测试确认通过 + 手测**

```bash
cd web/admin && pnpm test -- src/lib/dashboard/aggregate.test.ts
```

admin-api 可用时打开 `/`：见卡片；停掉某一后端路径时仅对应卡片错误。

- [ ] **Step 5: Commit**

```bash
git add web/admin/src/lib/dashboard web/admin/src/features/dashboard web/admin/src/routes/_app/index.tsx web/admin/src/lib/i18n/locales
git commit -m "$(cat <<'EOF'
feat(admin-web): add medium dashboard with list-API aggregation

EOF
)"
```

---

### Task 9: 实例页 — CRUD + 观测（方案 B）

**Files:**
- Create: `web/admin/src/features/instances/list-panel.tsx`
- Create: `web/admin/src/features/instances/detail-panel.tsx`
- Create: `web/admin/src/features/instances/instance-form.tsx`
- Modify: `web/admin/src/routes/_app/instances/route.tsx`
- Modify: `web/admin/src/routes/_app/instances/index.tsx`
- Modify: `web/admin/src/routes/_app/instances/$instanceId.tsx`
- Modify: i18n

**Interfaces:**
- 路由：`/instances`、`/instances/$instanceId`；新建用 `/instances/new` 或右栏 `mode=create`（推荐 path：`$instanceId` 允许字面值 `new`）
- 左栏列：`id`、`enabled`、`base_url`
- 右栏：表单字段 `id`(创建时可写)、`base_url`、`enabled`、`capabilities`（逗号/标签输入）；观测区只读 JSON/预格式化展示 system、queue、tasks
- mutation 成功后 `invalidateQueries(queryKeys.instances.*)` 并 toast

- [ ] **Step 1: 路由壳**

```tsx
// route.tsx — path: '/_app/instances'
// 提供 Outlet；列表与详情由子路由组合
// 推荐结构：instances/route 渲染 MasterDetailShell，
// list 常挂载；detail 读 useParams().instanceId
```

具体实现要点：

```tsx
const { instanceId } = useParams({ strict: false })
const navigate = useNavigate()
const listQuery = useQuery({ queryKey: queryKeys.instances.all, queryFn: listInstances })

<MasterDetailShell
  hasSelection={Boolean(instanceId)}
  onBackToList={() => navigate({ to: '/instances' })}
  list={<InstanceListPanel items={listQuery.data ?? []} selectedId={instanceId} />}
  detail={
    instanceId === 'new' ? (
      <InstanceForm mode="create" />
    ) : instanceId ? (
      <InstanceDetailPanel id={instanceId} />
    ) : null
  }
/>
```

- [ ] **Step 2: 实现表单与观测**

- Create：`POST /api/v1/comfy-instances`
- Update：`PATCH`
- Delete：确认后 `DELETE`，成功回 `/instances`
- 观测：三个 `useQuery`，key 含 instanceId；错误用 `ErrorBanner` 局部显示

- [ ] **Step 3: 手测主路径**

1. 起 admin-api  
2. 打开 `/instances` → 列表来自 API  
3. 新建 `gpu-plan-test` → 列表刷新  
4. 编辑 `base_url` / `enabled`  
5. 打开 system/queue/tasks 观测区有响应或明确错误  
6. 删除测试实例  

- [ ] **Step 4: Commit**

```bash
git add web/admin/src/features/instances web/admin/src/routes/_app/instances web/admin/src/lib/i18n/locales
git commit -m "$(cat <<'EOF'
feat(admin-web): add instances Master-Detail with CRUD and observation

EOF
)"
```

---

### Task 10: Case 页 — 结构化多段表单（方案 B）

**Files:**
- Create: `web/admin/src/features/cases/list-panel.tsx`
- Create: `web/admin/src/features/cases/case-form.tsx`
- Create: `web/admin/src/features/cases/sections/{basics,io-fields,bindings,workflow-json,input-schema}.tsx`
- Modify: `web/admin/src/routes/_app/cases/**`
- Modify: i18n

**Interfaces:**
- 路由同构：`/cases`、`/cases/$caseId`（`new` = 创建）
- 过滤：左栏顶部 `enabled` / `menu_key` / `q` → `listCases`
- 表单分段（可滚动，**禁止**整页唯一 JSON 编辑器）：

| 段组件 | 字段 |
|---|---|
| basics | id, name, description, preview, price, tags, menu_key, categories, enabled |
| io-fields | `inputs[]` / `outputs[]` 行编辑（增删行） |
| bindings | `bindings.inputs` / `bindings.outputs` 行编辑 |
| workflow-json | `bindings.workflow` 受控 `<textarea>` JSON，标签标明高级/原始 |
| input-schema | `input_schema` 受控 JSON，同上 |

- 动作：Create `POST`、Update `PATCH`、Enable/Disable `POST .../enable|disable`
- 提交前：`JSON.parse` 校验两个 JSON 区；失败在段内显示错误，不提交

- [ ] **Step 1: 实现 `CaseForm` 状态**

用单个 `CaseRecord` draft state；各 section 通过 `value` + `onChange` 更新切片。创建模式下 `id` 可编辑；编辑模式 `id` 只读。

空文档默认值：

```ts
export function emptyCase(): CaseRecord {
  return {
    id: '',
    name: '',
    price: 0,
    inputs: [],
    outputs: [],
    bindings: { workflow: {}, inputs: [], outputs: [] },
    input_schema: {},
    enabled: true,
  }
}
```

- [ ] **Step 2: 列表 + enable/disable**

左栏行显示 `id` / `name` / `enabled`；行内或详情顶栏按钮调用 `enableCase` / `disableCase`，成功 invalidate。

- [ ] **Step 3: 手测**

1. 列表过滤 `enabled=true`  
2. 新建最小 Case（必填字段按后端校验）→ 201/列表可见  
3. 编辑 inputs 行 → PATCH 成功  
4. Disable → 列表状态更新  
5. workflow JSON 写非法 JSON → 段内错误、不发请求  

- [ ] **Step 4: Commit**

```bash
git add web/admin/src/features/cases web/admin/src/routes/_app/cases web/admin/src/lib/i18n/locales
git commit -m "$(cat <<'EOF'
feat(admin-web): add Case Master-Detail with multi-section form

EOF
)"
```

---

### Task 11: Task 页 — 列表/详情 + 取消

**Files:**
- Create: `web/admin/src/features/tasks/list-panel.tsx`
- Create: `web/admin/src/features/tasks/detail-panel.tsx`
- Modify: `web/admin/src/routes/_app/tasks/**`
- Modify: i18n

**Interfaces:**
- `/tasks`、`/tasks/$taskId`
- 左栏：`id`、`status`、`case_id`；过滤 `status`、`q`
- 右栏只读字段 + **取消**按钮 → `cancelTask`
- `ApiError` status `409` → 展示后端/i18n「不可取消」；`404` → 资源不存在

- [ ] **Step 1: 实现详情取消**

```tsx
const cancelMut = useMutation({
  mutationFn: () => cancelTask(taskId),
  onSuccess: () => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.tasks.all })
    void queryClient.invalidateQueries({ queryKey: queryKeys.tasks.detail(taskId) })
  },
})
// 按钮 disabled={cancelMut.isPending}
```

- [ ] **Step 2: 手测**

有可取消任务时点取消 → 状态变化或明确错误；对已完成任务 → 409 文案可见。

- [ ] **Step 3: Commit**

```bash
git add web/admin/src/features/tasks web/admin/src/routes/_app/tasks web/admin/src/lib/i18n/locales
git commit -m "$(cat <<'EOF'
feat(admin-web): add Task Master-Detail with cancel action

EOF
)"
```

---

### Task 12: User / Session 只读页（方案 B）

**Files:**
- Create: `web/admin/src/features/users/{list-panel,detail-panel}.tsx`
- Create: `web/admin/src/features/sessions/{list-panel,detail-panel}.tsx`
- Modify: routes under `_app/users`、`_app/sessions`
- Modify: i18n

**Interfaces:**
- **禁止** 渲染 Create/Edit/Delete 按钮
- User 左栏：`id`、`tg_user_id`、`username`；过滤 `q`、`tg_user_id`
- User 右栏：类型全部字段只读展示
- Session 左栏：`id`、`status`、`user_id`、`case_id`；过滤 `user_id`、`status`、`q`
- Session 右栏：字段 + `draft` 只读 JSON/结构化展示

- [ ] **Step 1: 实现两页并接入 MasterDetailShell**

与 Task 9 路由同构；detail 仅 `useQuery(getUser|getSession)`。

- [ ] **Step 2: 手测**

从列表点选 → 右栏字段来自 admin-api；确认无写操作控件；Network 面板无 POST/PATCH/DELETE。

- [ ] **Step 3: Commit**

```bash
git add web/admin/src/features/users web/admin/src/features/sessions web/admin/src/routes/_app/users web/admin/src/routes/_app/sessions web/admin/src/lib/i18n/locales
git commit -m "$(cat <<'EOF'
feat(admin-web): add read-only User and Session Master-Detail pages

EOF
)"
```

---

### Task 13: 统一空态/加载/错误 + i18n 补齐 + 清扫

**Files:**
- Modify: 各 feature 页确保使用 feedback 组件
- Modify: `zh.json` / `en.json` — 壳子、Dashboard、五资源文案齐全
- Delete: 残留 demo 路由、未用依赖、任何 mock 入口
- Modify: `web/admin/README.md` 完整联调说明

**Interfaces:**
- 验收清单（全部勾上才算本 Task 完成）：
  1. 无 Clerk/登录页  
  2. 菜单六项顺序正确且无 TG/Menu  
  3. 默认中文；英中切换刷新保持  
  4. 所有资源请求 Host = `VITE_ADMIN_API_BASE`  
  5. 代码库无 MSW / `VITE_USE_MOCK`  
  6. Case 主路径是分段表单  

- [ ] **Step 1: 全文搜索清扫**

```bash
cd web/admin
rg -n "Clerk|MSW|msw|VITE_USE_MOCK|mockServiceWorker|sign-in|SignIn" src package.json
rg -n "menu\\.(tg|telegram)|/menus" src
```

Expected: 无命中（或仅文档说明「不做」）。

- [ ] **Step 2: 补齐双语 key**

保证 `t('...')` 在 zh/en 均有定义；跑一次页面切换抽查 Dashboard + Case + Task。

- [ ] **Step 3: 完善 README 联调**

追加：

```markdown
## 联调

1. `make run-admin-api`（或 `go run ./apps/admin-api/cmd/admin-api`）
2. `cd web/admin && cp .env.example .env.development && pnpm install && pnpm dev`
3. 浏览器打开 Vite URL；根路径应为 Dashboard
4. DevTools Network：请求前缀为 `VITE_ADMIN_API_BASE` + `/api/v1/...`

Dashboard 数字基于 list 拉取样本，受 `limit` 限制，不是全库精确统计。

## 非目标

无登录鉴权；无前端 mock；无 TG/Menu 管理。
```

- [ ] **Step 4: Commit**

```bash
git add web/admin
git commit -m "$(cat <<'EOF'
chore(admin-web): polish i18n, feedback UX, and integration README

EOF
)"
```

---

### Task 14: 联调验收（对照 OpenSpec tasks 4.x）

**Files:**
- Modify（仅当架构表述不足时）: `docs/architecture/overview.md` — 一句提及 `web/admin` SPA 经 admin-api
- 不改 API 契约

**Interfaces:**
- 无新代码接口；产出：人工验收记录（可写在 PR 描述或 verify 阶段报告）

- [ ] **Step 1: 跑前端单测与生产构建**

```bash
cd web/admin
pnpm test
pnpm build
```

Expected: 测试通过；`build` 无错误。

- [ ] **Step 2: 端到端手测清单**

| # | 步骤 | 期望 |
|---|---|---|
| 1 | 仅开前端、错误基址 | 错误提示含检查 `VITE_ADMIN_API_BASE` |
| 2 | 开 admin-api + 前端 `/` | Dashboard 卡片有数据或空态，非整页崩溃 |
| 3 | 侧栏点完全部菜单 | 六页可达，顺序正确 |
| 4 | 实例 CRUD + 观测 | 写读成功 |
| 5 | Case 分段创建/编辑/禁用 | 请求打到 `/api/v1/cases*` |
| 6 | Task 取消 | 成功或明确 409 |
| 7 | User/Session | 只读详情 |
| 8 | 切英文刷新 | 文案保持英文 |
| 9 | Network | 无 bot `:8080` 管理 CRUD；无 mock |

- [ ] **Step 3:（如需要）同步架构一句**

若 `docs/architecture/overview.md` 仍写 admin 前端「预留」，改为：管理 SPA 位于 `web/admin`，经 `VITE_ADMIN_API_BASE` 访问 admin-api。

- [ ] **Step 4: Commit（若有文档变更）**

```bash
git add docs/architecture/overview.md web/admin
git commit -m "$(cat <<'EOF'
docs: note web/admin console talks only to admin-api

EOF
)"
```

---

## Self-Review（写作时已执行）

**1. Spec coverage**
| 需求来源 | 对应 Task |
|---|---|
| pnpm + shadcn-admin 进 `web/admin` | 1 |
| 去鉴权、默认 Dashboard | 2、4、8 |
| `VITE_ADMIN_API_BASE` + 无 mock | 3、7、13、14 |
| 菜单顺序六项 | 4 |
| 中英 i18n | 5、13 |
| Master–Detail 方案 B | 6、9–12 |
| 中等 Dashboard + 失败隔离 | 8 |
| 实例 CRUD+观测 | 9 |
| Case 多段表单 | 10 |
| Task 取消 | 11 |
| User/Session 只读 | 12 |
| 空态/加载/错误 + 联调文档 | 13、14 |
| OpenSpec tasks 1–4 | 1–14 覆盖 |

**2. Placeholder scan：** 无 TBD/TODO；关键 API/组件均给出签名或示例代码。

**3. Type consistency：** `CaseRecord` / `ComfyInstance` / `apiFetch` / `MENU_ITEMS` / `MasterDetailShell` / `queryKeys` 前后任务命名一致；资源路径统一 `/api/v1/...`。

---

## 执行交接

Plan complete and saved to `docs/superpowers/plans/2026-08-09-admin-web-console.md`. Two execution options:

**1. Subagent-Driven (recommended)** — 每 Task 新开 subagent，Task 间审查，迭代快  
**2. Inline Execution** — 本会话用 executing-plans 按检查点批量执行  

Which approach?
