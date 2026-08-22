## 1. 代码与构建清理

- [x] 1.1 迁移 backfill 至 `apps/pixoma/cmd/backfill-task-stats`（环境变量 DSN：`DB_DRIVER` / `DATABASE_DSN` / `DATA_DIR`）
- [x] 1.2 删除 `apps/bot`、`apps/admin-api` 目录
- [x] 1.3 删除 `internal/platform/adminconfig` 与 `configs/admin-api.yaml`、`configs/admin-api.example.yaml`
- [x] 1.4 Makefile：`build` 仅产出 pixoma / pixoma-edge-agent，删除 `run-admin-api`

## 2. 文档同步

- [x] 2.1 README：服务表与环境变量表移除 admin-api 过渡期描述，补充 backfill 环境变量
- [x] 2.2 `docs/architecture/overview.md` / `data-model.md` / `runtime.md` 去掉独立 admin-api / bot 入口描述

## 3. 验证

- [x] 3.1 `go build ./...` + `go test ./...` 通过且无被删包引用
- [x] 3.2 `rg apps/admin-api|apps/bot` 仅剩历史归档/报告文档
- [x] 3.3 backfill 冒烟：临时库 seed 终态任务后重算统计表正确
