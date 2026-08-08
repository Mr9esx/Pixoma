# Admin API

独立管理 HTTP 进程：健康检查、CORS、后续实例管理 API。

**本期无鉴权。** 只在本机或可信内网使用，不要对公网暴露。

## 运行

```bash
# 可选：复制示例配置（DSN 必须与 bot 同一库）
cp configs/admin-api.example.yaml configs/admin-api.yaml

make run-admin-api
# 或
go run ./apps/admin-api/cmd/admin-api
```

默认监听 `:8081`。配置路径：`ADMIN_CONFIG` 或 `configs/admin-api.yaml`。

## 健康检查

```bash
curl -s localhost:8081/healthz
# ok
```

## 与 bot 的边界

| 进程 | 端口（默认） | 职责 |
|---|---|---|
| bot | `:8080` | 对话 / 编排 / TG |
| admin-api | `:8081` | 管理 HTTP（本 change 起） |

两边共用同一 `database_dsn`。admin-api **不依赖** `channel/tg`。

实例 CRUD/观测路由在后续任务挂载到 `/api/v1/comfy-instances`。
