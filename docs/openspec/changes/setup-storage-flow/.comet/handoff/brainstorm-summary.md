# Brainstorm Summary

- Change: setup-storage-flow
- Date: 2026-08-25

## 确认的技术方案

- **移除 placement 步骤**：`SETUP_STEPS` 变 密码 → 数据库 → 对象存储；`draft()` 的 `placement` 由 `blob_driver` 推断（localfs→local，s3/tos→remote）；后端 `settings.Validate` 的远程禁止 localfs 约束保留（API 层）。
- **对象存储连通性测试（后端）**：`blob/localfs`、`blob/s3`、`blob/tos` 新增 `Check(ctx)`（目录可写探测 / S3 HeadBucket / TOS `ClientV2.HeadBucket`）；`blob/factory` 新增 `Check(ctx, CheckOptions)`（显式凭据，不依赖环境变量）；Setup 新增 `POST /api/v1/setup/blob-test`，先按推断 placement 走 `settings.Validate` 再 `factory.Check`。
- **前端对象存储步骤**：复用数据库步骤的「连通性测试 + 继续」与 success/destructive Alert；S3/TOS 展示 endpoint/region/bucket/access key/secret key 字段；新增 `blob-error.ts` 映射常见对象存储错误为中文友好标题 + 实际详情。
- **不自动建 bucket**：连通性测试只校验，bucket 不存在报错，README 说明需预先创建。

- **自动建 bucket（用户确认后）**：`s3`/`tos` 各新增 `EnsureBucket`（HeadBucket 探测 → CreateBucket/CreateBucketV2 → 复检）；blob 包新增 `ErrBucketNotFound`；`blob-test` 返回 `bucket_not_found` 让前端提示「是否帮你创建」，确认后带 `auto_create_bucket:true` 重试创建。OSS 驱动不在本期（另行开 change）。

## 关键取舍与风险

- 不自动建 bucket → 用户需先建好；HeadBucket 404 报「bucket 不存在」。
- OSS/Azure 等新驱动非目标；`blob-test` 按 driver 分发，后续加驱动只需补 `Check`。
- placement 推断不影响既有已初始化部署（putSettings 仍从引导态读 DB 连接）。

## 测试策略

- Go 单测：localfs/s3/tos `Check` 的构造与错误路径、`factory.Check` 分发、`blob-test` 端点（非法配置、未知驱动、成功路径用可注入 fake）。
- 前端：`setup-steps.test.ts` 3 步序列；合同测试断言无「出图机器在哪」、对象存储测试按钮与 Alert、`blob-error.ts` 映射单测。
- 回归：`go test ./...`、`pnpm vitest run` 全绿。

## Spec Patch

无需额外回写；open 阶段 delta spec 已覆盖（setup-wizard 移除部署位置 + 对象存储连通性测试；shared-blob-store 连通性检查）。
