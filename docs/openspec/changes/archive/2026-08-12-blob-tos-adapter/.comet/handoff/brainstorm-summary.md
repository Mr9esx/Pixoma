# Brainstorm Summary

- Change: blob-tos-adapter
- Date: 2026-08-12

## Confirmed Technical Approach

采用独立包 `internal/platform/blob/tos`，基于火山官方 SDK `github.com/volcengine/ve-tos-golang-sdk/v2` 实现 `blob.Store`（Put/Get）。配置为静态 AK/SK + endpoint/region/bucket；`split` 下 `blob∈{s3,tos}` 且 `queue=redis`。edge-agent 按 `BLOB_DRIVER` 装配。验收必须真连 `pixoma-test` 做 Put→Get。

## Key Trade-offs and Risks

- 密钥仅存 `.env.tos.local`（gitignore）；用完作废临时密钥
- Put 内存模型对齐现有 s3（可读全量），本期不做流式优化
- nats change 已取消，校验矩阵不包含 nats

## Testing Strategy

- 单元：非法 key、空 bucket 等离线路径
- Verify 硬门禁：加载 `TOS_*` 对真桶 Put→Get；缺配置失败

## Spec Patches

- 收窄 delta：去掉「与 nats 任意搭配」前瞻表述，改为 `split` 下与 redis 搭配、`blob∈{s3,tos}`
