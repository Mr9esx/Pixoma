# Split 模式冒烟（手动）

前置：Redis、S3 兼容（MinIO）、可选本机 Comfy；或 `COMFY_MOCK=true`。

1. 启动 MinIO / Redis；创建 bucket（如 `pixoma`）。
2. Bot（云/本机）配置：
   - `runtime_mode: split`
   - `queue.driver: redis` + `REDIS_ADDR`
   - `blob.driver: s3` + `S3_*` 凭证
   - 启动后 Bot **不**订阅生产 dispatch。
3. Edge：
   ```bash
   INSTANCE_ID=local \
   REDIS_ADDR=127.0.0.1:6379 \
   S3_ENDPOINT=http://127.0.0.1:9000 \
   S3_BUCKET=pixoma S3_ACCESS_KEY=... S3_SECRET_KEY=... S3_PATH_STYLE=1 \
   COMFY_MOCK=true \
   go run ./apps/edge-agent/cmd/edge-agent
   ```
4. TG 确认生成：应收到产物；Edge 日志有 dispatch；S3 出现 `jobs/` 与 `outputs/`。

Edge 心跳 key：`edge:online:<INSTANCE_ID>`（TTL 约 30s）。Bot 在 `runtime_mode=split` 时用 Redis EXISTS 接 `orch.Online`；无在线 Edge 则任务保持 pending。
