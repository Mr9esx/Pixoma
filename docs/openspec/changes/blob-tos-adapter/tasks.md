## 1. 配置与校验

- [x] 1.1 在 `botconfig` 增加 `BlobDriverTOS` 及 TOS 连接配置字段
- [x] 1.2 放宽 `ValidateRuntimeDrivers`：`split` 允许 `blob ∈ {s3, tos}`，与 `queue=redis` 搭配
- [x] 1.3 补充校验单测与 YAML/env 样例

## 2. TOS 适配器

- [x] 2.1 引入火山引擎 TOS 官方 Go SDK 依赖
- [x] 2.2 实现 `internal/platform/blob/tos`：New / Put / Get，key 清洗规则对齐 s3/localfs
- [x] 2.3 单元测试覆盖 Put→Get、非法 key、空 bucket 等失败路径

## 3. 进程装配

- [x] 3.1 bot 控制面按 `blob.driver` 装配 tos（保留 localfs/s3）— 通过 `blob/factory` 供后续入口复用
- [x] 3.2 edge-agent 按 `blob.driver` 装配 tos，保证 inputs/jobs/outputs 读写主路径
- [x] 3.3 确认 `split + localfs` 等非法组合启动失败信息可诊断

## 4. 文档与验收

- [x] 4.1 若触及架构边界，更新架构/配置文档中的 blob 驱动列表
- [x] 4.2 真 TOS 硬门禁：加载 `.env.tos.local` / `TOS_*` 对 `pixoma-test` Put→Get 成功；缺配置则失败
- [x] 4.3 localfs/s3 回归通过；`split + tos + redis` 启动校验通过
