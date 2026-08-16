---
role: technical-design
status: proposed
---

# 本地调试一条命令（make dev）技术设计

## 1. 目标与边界

日常调试只要一条命令：同时拉起控制面 `pixoma` 和管理页面 Vite。浏览器打开命令打印的页面地址；接口地址也打出来，方便对一下。Ctrl-C 两个一起停。

默认走**真 Comfy**（和现在 `make run` 一样）。这轮不加 Mock 专用新命令。

**非目标：**

- 不新增 `make dev-mock`
- 开发时不把前端打进 8080（发布仍用 `make embed-admin`）
- 不把独立 `admin-api`（旧 8081）当作默认调试路径
- 不改向导、鉴权、生产拓扑、Comfy mock 开关语义

## 2. 已确认决定

| 项 | 选择 |
|---|---|
| 入口 | `make dev`（仓库根目录） |
| Comfy | 真机（`COMFY_MOCK=0`） |
| 浏览器 | 打开 Vite 页面地址（一般 `http://127.0.0.1:5173`） |
| 启动日志 | 同时打印页面地址和后台接口地址（一般 `http://127.0.0.1:8080`） |
| 页面怎么找到接口 | 开发时 Vite 把 `/api` 转到 pixoma；不依赖 `VITE_ADMIN_API_BASE` |
| 旧命令 | `make run` / `make run-mock` 原样保留 |

## 3. 日常差别

| 你做什么 | 会发生什么 |
|---|---|
| `make dev` | 起 pixoma + 管理页面；日志里有两个地址 |
| 改 `web/admin` | 保存后页面自己刷新 |
| 改 Go | 停掉再跑一次 `make dev` |
| 发布 | 仍先 `make embed-admin`，再构建二进制；用户只开 8080 |

`make run` 仍然只起后端。没 embed 时打开 8080 仍会看到「页面没打进二进制」的提示，这是发布路径，不是日常调试入口。

## 4. 两个进程怎么连

```text
浏览器 ──► Vite :5173（管理页面）
              │
              └── /api/*  代理 ──► pixoma :8080
```

开发时 `apiFetch` 用空的 API 根路径，请求写成 `/api/v1/...`。浏览器打到 5173，Vite 转到 8080。这样：

- 不用配、也不用翻本地 `.env.development`
- 开发请求走同源，不靠 CORS 才能打开后台
- 发布 embed 后页面和接口本来就在 8080，空根路径仍然正确

若以后有人显式设置 `VITE_ADMIN_API_BASE`，仍可覆盖（测试夹具可以继续写死一个地址）。默认开发路径不需要它。

`make dev` 若发现 `web/admin` 还没装依赖，先 `pnpm install` 再起 Vite。不必先手动 `cp .env.example`。

实现上 Makefile 目标可以调用小脚本，保证 Ctrl-C 能停掉两个子进程；不要只 `go run` 完再起 Vite（那样停一个另一个还挂着）。

## 5. 文档

根 `README.md` 增加「本地调试」：

1. `make dev`
2. 打开日志里的页面地址
3. 接口地址是哪一个
4. 改页面 vs 改 Go
5. 发布才 `make embed-admin`

`web/admin/README.md` 去掉过时的 `make run-admin` / `make run-all` / 默认 8081 联调，改成指向根目录 `make dev`。`apps/admin-api/README.md` 若仍把 `make run-admin` 写成默认联调，改为「过渡期独立进程，新调试用 `make dev`」。

不改 `docs/architecture/`：生产仍是「开发独立 Vite、发布 embed」，拓扑没变。

## 6. 验收

- 仓库根目录 `make dev`，两个进程都起来；Ctrl-C 后都退出
- 启动输出里能看到页面 URL 和接口 URL
- 未设置 `VITE_ADMIN_API_BASE` 时，浏览器打开页面能打到 `/api/v1/setup/status`（经 Vite 代理）
- 根 README 按上面步骤能跟做；`web/admin` README 不再把 8081 / `make run-admin` 写成默认调试
