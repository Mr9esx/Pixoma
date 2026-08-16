---
design-doc: docs/superpowers/specs/2026-08-15-local-dev-loop-design.md
---

# 本地调试一条命令 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 仓库根目录 `make dev` 同时拉起 pixoma 和管理页面；浏览器打开日志里的页面地址即可调后台。

**Architecture:** 开发时 Vite 把 `/api` 转到 `127.0.0.1:8080`。`pnpm dev` 下 `apiFetch` 始终走相对路径 `/api/v1/...`，忽略本地 `.env.development` 里的旧端口。Makefile 调用 `scripts/dev.sh` 起两个进程，Ctrl-C 一起停。

**Tech Stack:** Make、bash、Vite `server.proxy`、Vitest 合同测试、Go `go run`、pnpm

## Global Constraints

- 产物与提交说明语言：zh-CN
- `make dev` 默认 `COMFY_MOCK=0`（真 Comfy），与现有 `make run` 一致
- 不新增 `make dev-mock`
- 开发时不把前端 embed 进 8080
- 不把独立 `admin-api`（:8081）当作默认调试路径
- `make run` / `make run-mock` 原样保留
- 不改向导、鉴权、生产拓扑、Comfy mock 开关语义
- 不改 `docs/architecture/`
- 未经用户明确要求不要 `git commit`

## 文件地图

| 文件 | 职责 |
|---|---|
| `web/admin/src/lib/api/client.ts` | `DEV` 时 API 根路径为空，请求相对 `/api/...` |
| `web/admin/src/lib/api/client.test.ts` | 锁住 DEV 忽略 `VITE_ADMIN_API_BASE` |
| `web/admin/vite.config.ts` | `server.proxy['/api']` → `http://127.0.0.1:8080` |
| `web/admin/src/vite-proxy.contract.test.ts` | 锁住代理目标 |
| `web/admin/vitest.config.ts` | include 新合同测试 |
| `scripts/dev.sh` | 起 pixoma + Vite，打印两个地址，转发信号 |
| `Makefile` | `dev` 目标 |
| `web/admin/src/dev-loop.contract.test.ts` | 锁住 Makefile / 脚本的入口行为 |
| `README.md` | 「本地调试」怎么跑 |
| `web/admin/README.md` | 去掉过时的 8081 / `make run-admin` |
| `apps/admin-api/README.md` | 标明过渡期，新调试用 `make dev` |
| `web/admin/.env.example` | 标明开发可不设 |

---

### Task 1: 开发时走相对 /api 路径

**Files:**
- Modify: `web/admin/src/lib/api/client.ts`
- Modify: `web/admin/src/lib/api/client.test.ts`

**Interfaces:**
- Consumes: `apiFetch(path: string, init?: RequestInit)`；`import.meta.env.DEV`；`import.meta.env.VITE_ADMIN_API_BASE`
- Produces: `DEV === true` 时 `fetch` 的 URL 为传入的相对路径（如 `/api/v1/setup/status`），即使 `VITE_ADMIN_API_BASE` 有值

- [ ] **Step 1.1** 在 `client.test.ts` 的 `afterEach` 增加 `vi.unstubAllEnvs()`，并追加用例：

```ts
it('uses a relative /api path in DEV even if VITE_ADMIN_API_BASE is set', async () => {
  vi.stubEnv('DEV', 'true')
  vi.stubEnv('VITE_ADMIN_API_BASE', 'http://127.0.0.1:8081')
  const fetchMock = vi.fn().mockResolvedValue(
    new Response(JSON.stringify({ initialized: false }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    }),
  )
  vi.stubGlobal('fetch', fetchMock)

  await apiFetch('/api/v1/setup/status')

  expect(fetchMock).toHaveBeenCalledWith(
    '/api/v1/setup/status',
    expect.objectContaining({
      credentials: 'include',
    }),
  )
})
```

保留文件里现有两个用例不动。

- [ ] **Step 1.2** 跑失败测试

Run: `pnpm --dir web/admin exec vitest run src/lib/api/client.test.ts`
Expected: FAIL，新用例里 `fetch` 实际打到 `http://127.0.0.1:8081/api/v1/setup/status`

- [ ] **Step 1.3** 改 `baseURL()`：

```ts
function baseURL(): string {
  if (import.meta.env.DEV) {
    return ''
  }
  const raw = import.meta.env.VITE_ADMIN_API_BASE as string | undefined
  if (!raw) {
    return ''
  }
  return raw.replace(/\/$/, '')
}
```

`apiFetch` 其余逻辑不动。Vitest 默认 `MODE=test`、`DEV=false`，现有 `instances.test.ts` 等仍走 `VITE_ADMIN_API_BASE=http://127.0.0.1:8081`。

- [ ] **Step 1.4** 再跑测试

Run: `pnpm --dir web/admin exec vitest run src/lib/api/client.test.ts`
Expected: PASS（3 个用例）

---

### Task 2: Vite 把 /api 转到 pixoma

**Files:**
- Create: `web/admin/src/vite-proxy.contract.test.ts`
- Modify: `web/admin/vite.config.ts`
- Modify: `web/admin/vitest.config.ts`（`include` 加上新文件）

**Interfaces:**
- Consumes: Task 1 的相对 `/api` 请求
- Produces: `defineConfig` 的 `server.proxy['/api'].target === 'http://127.0.0.1:8080'` 且 `changeOrigin: true`

- [ ] **Step 2.1** 新建合同测试（先红）：

```ts
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const adminRoot = join(dirname(fileURLToPath(import.meta.url)), '..')

describe('vite admin API proxy', () => {
  it('proxies /api to pixoma 8080', () => {
    const src = readFileSync(join(adminRoot, 'vite.config.ts'), 'utf8')
    expect(src).toMatch(/proxy\s*:/)
    expect(src).toMatch(/['"]\/api['"]\s*:/)
    expect(src).toMatch(/target\s*:\s*['"]http:\/\/127\.0\.0\.1:8080['"]/)
    expect(src).toMatch(/changeOrigin\s*:\s*true/)
  })
})
```

- [ ] **Step 2.2** 把该文件加入 `vitest.config.ts` 的 `include` 数组（与 `src/lib/api/client.test.ts` 同级列出）。

- [ ] **Step 2.3** 跑失败测试

Run: `pnpm --dir web/admin exec vitest run src/vite-proxy.contract.test.ts`
Expected: FAIL，`vite.config.ts` 还没有 `proxy`

- [ ] **Step 2.4** 在 `vite.config.ts` 的 `defineConfig({...})` 里、`resolve` 旁增加：

```ts
  server: {
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
```

不要改 plugins / test 块。

- [ ] **Step 2.5** 再跑测试

Run: `pnpm --dir web/admin exec vitest run src/vite-proxy.contract.test.ts src/lib/api/client.test.ts`
Expected: PASS

---

### Task 3: `make dev` 一起起两个进程

**Files:**
- Create: `scripts/dev.sh`
- Modify: `Makefile`
- Create: `web/admin/src/dev-loop.contract.test.ts`
- Modify: `web/admin/vitest.config.ts`（`include` 加上该文件）

**Interfaces:**
- Consumes: Task 2 的 Vite 代理；`pnpm --dir web/admin dev`；`go run ./apps/pixoma/cmd/pixoma`
- Produces: `make dev` → `bash scripts/dev.sh`；脚本打印 `管理页面: http://127.0.0.1:5173` 与 `后台接口: http://127.0.0.1:8080`；`COMFY_MOCK=0`；INT/TERM 时杀掉两个子进程

- [ ] **Step 3.1** 写失败合同测试 `dev-loop.contract.test.ts`：

```ts
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const adminRoot = join(dirname(fileURLToPath(import.meta.url)), '..')
const repoRoot = join(adminRoot, '..')

describe('make dev loop', () => {
  it('Makefile dev target runs scripts/dev.sh', () => {
    const mk = readFileSync(join(repoRoot, 'Makefile'), 'utf8')
    expect(mk).toMatch(/^\.PHONY:.*\bdev\b/m)
    expect(mk).toMatch(/^dev:\n\tbash scripts\/dev\.sh/m)
  })

  it('dev.sh starts pixoma and vite, prints both URLs, and traps signals', () => {
    const sh = readFileSync(join(repoRoot, 'scripts/dev.sh'), 'utf8')
    expect(sh).toContain('管理页面: http://127.0.0.1:5173')
    expect(sh).toContain('后台接口: http://127.0.0.1:8080')
    expect(sh).toContain('COMFY_MOCK="${COMFY_MOCK:-0}"')
    expect(sh).toContain('go run ./apps/pixoma/cmd/pixoma')
    expect(sh).toContain('pnpm --dir web/admin dev')
    expect(sh).toMatch(/trap .* INT TERM/)
  })
})
```

- [ ] **Step 3.2** 把该文件加入 `vitest.config.ts` 的 `include`。

- [ ] **Step 3.3** 跑失败测试

Run: `pnpm --dir web/admin exec vitest run src/dev-loop.contract.test.ts`
Expected: FAIL（没有 `scripts/dev.sh` 和 Makefile `dev`）

- [ ] **Step 3.4** 新建 `scripts/dev.sh`，全文：

```bash
#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

PAGE_URL="http://127.0.0.1:5173"
API_URL="http://127.0.0.1:8080"

if [[ ! -d web/admin/node_modules ]]; then
  pnpm --dir web/admin install
fi

echo "管理页面: ${PAGE_URL}"
echo "后台接口: ${API_URL}"

export COMFY_MOCK="${COMFY_MOCK:-0}"
go run ./apps/pixoma/cmd/pixoma &
pixoma_pid=$!

pnpm --dir web/admin dev &
vite_pid=$!

cleanup() {
  trap - INT TERM EXIT
  kill "$pixoma_pid" "$vite_pid" 2>/dev/null || true
  wait "$pixoma_pid" "$vite_pid" 2>/dev/null || true
}
trap cleanup INT TERM EXIT

wait
```

默认 `COMFY_MOCK=0`。调用前若已 `export COMFY_MOCK=1` 则尊重，但不新增 `make dev-mock` 目标。

- [ ] **Step 3.5** 改 Makefile：`.PHONY` 加上 `dev`；增加目标：

```makefile
dev:
	bash scripts/dev.sh
```

不要改 `run` / `run-mock` / `embed-admin`。

- [ ] **Step 3.6** 再跑合同测试

Run: `pnpm --dir web/admin exec vitest run src/dev-loop.contract.test.ts src/vite-proxy.contract.test.ts src/lib/api/client.test.ts`
Expected: PASS

- [ ] **Step 3.7** 手测（本机 8080/5173 没被占用时）：仓库根目录 `make dev`，日志里同时出现「管理页面」和「后台接口」两行；Ctrl-C 后两个进程都退出（`lsof -iTCP:8080 -sTCP:LISTEN` 与 `5173` 应为空）。不要在计划执行时强行杀掉用户已有的 `make run`。

---

### Task 4: 文档写成能跟着做的调试步骤

**Files:**
- Modify: `README.md`（「实例管理」一节换成「本地调试」+ 发布补充）
- Modify: `web/admin/README.md`
- Modify: `apps/admin-api/README.md` 的「运行」段
- Modify: `web/admin/.env.example`

**Interfaces:**
- Consumes: Task 3 的 `make dev`、两个 URL
- Produces: 根 README 能从零跟做；`web/admin` / `admin-api` README 不再把 8081 / `make run-admin` 写成默认调试

- [ ] **Step 4.1** 把根 `README.md` 里「## 实例管理（pixoma 后台）」整节换成：

```markdown
## 本地调试

一条命令同时起控制面和管理页面（真 Comfy，和 `make run` 一样）：

```bash
make dev
```

启动日志里有两个地址：

- **管理页面**（浏览器打开这个）：`http://127.0.0.1:5173`
- **后台接口**：`http://127.0.0.1:8080`

Ctrl-C 两个一起停。改 `web/admin` 保存后页面会自己刷新；改 Go 需要再跑一次 `make dev`。

只起后端、不看页面时继续用 `make run` / `make run-mock`。没把前端打进二进制时，打开 8080 会看到提示页，这是发布路径，不是日常调试入口。

发布：`make embed-admin` 之后再构建 `pixoma`，用户只开 8080 就是完整后台。
```

保留该节后面的 `make build` / `make test` / `make clean` 代码块。过渡期 `make run-admin-api` 那句改成：独立 `admin-api` 仅过渡期保留，调试请用 `make dev`。

- [ ] **Step 4.2** 重写 `web/admin/README.md` 的「安装与启动」「联调」为指向根目录 `make dev`。删掉 `make run-admin`、`make run-all`、默认 `8081` 示例。可以保留「只起前端」的拆开命令，但必须写清接口是 pixoma `:8080`，且 `pnpm dev` 会把 `/api` 代理过去，不必设 `VITE_ADMIN_API_BASE`。

- [ ] **Step 4.3** `apps/admin-api/README.md`「运行」段开头加一句：新部署与日常调试用仓库根目录 `make dev`（pixoma + 管理页面）。下面的 `make run-admin` 若 Makefile 里不存在，删掉该行，只保留 `make run-admin-api` 并标明过渡期。

- [ ] **Step 4.4** `web/admin/.env.example` 改成：

```
# 开发（pnpm dev / make dev）会走 Vite 代理，不必设置。
# 非 DEV 且要打到别的地址时再取消注释：
# VITE_ADMIN_API_BASE=http://127.0.0.1:8080
```

- [ ] **Step 4.5** 自检：`rg 'make run-admin[^-]|make run-all' README.md web/admin/README.md apps/admin-api/README.md Makefile` 应无默认调试入口命中（`run-admin-api` 允许）。`rg '8081' README.md web/admin/README.md` 不应再把 8081 写成打开管理页面的地址。
