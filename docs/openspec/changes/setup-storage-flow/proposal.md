## Why

Setup 向导目前先问「出图机器在哪」（本机/远程），再进对象存储配置。对多数用户这是多余的一步：部署位置本质上由对象存储选型决定（本机目录=本机，S3/TOS=远程），且对象存储配置目前没有任何连通性校验，填错 endpoint/密钥/bucket 要到保存后才暴露。向导应直接进入对象存储配置，并像数据库一样提供连通性测试。

## What Changes

- **去掉「出图机器在哪」步骤**：Setup 步骤变为 密码 → 数据库 → 对象存储；`placement` 由对象存储驱动推断（`localfs`→本机，`s3`/`tos`→远程），后端 `settings.Validate` 的「远程禁止 localfs」约束继续生效。
- **对象存储连通性测试**：对象存储步骤新增「连通性测试 + 继续」按钮（复用数据库步骤模式）；`localfs` 校验目录可写，`s3`/`tos` 用 HeadBucket 校验 endpoint/region/bucket/密钥可达；通过显示 success Alert，失败显示 destructive Alert（中文友好文案 + 实际详情）。
- **后端**：新增对象存储连通性检查能力（`blob` 包各驱动 `Check`、工厂级带显式凭据的检查）；Setup 新增 `POST /api/v1/setup/blob-test` 端点。
- **规格与文档**：更新 `setup-wizard`（移除本机/远程选择、对象存储直接配置与连通性测试）、`shared-blob-store`（连通性测试需求）；README 同步说明。

## Non-Goals

- 不新增 OSS / Azure Blob 等新对象存储驱动（本期只对现有 `localfs`/`s3`/`tos` 做连通性测试；OSS 等作为后续驱动扩展）。
- 不做自动创建 bucket（连通性测试只校验可达，bucket 不存在时报错，不自动创建）。
- 不改数据库步骤行为与设置页对象存储配置。
- 不在向导内重做 Edge 部署指引（部署说明保留在 README/文档）。

## Capabilities

### New Capabilities

（无新能力，均为既有能力的行为变更。）

### Modified Capabilities
- `setup-wizard`: 移除「出图机器在哪」步骤，直接进入对象存储配置；对象存储步骤支持连通性测试（同数据库模式）。
- `shared-blob-store`: 对象存储驱动支持连通性检查（localfs 目录可写、S3/TOS HeadBucket 校验）。

## Impact

- 前端：`web/admin` Setup 向导（步骤序列、placement 推断、对象存储表单 + 连通性测试/继续按钮）、合同测试与步骤测试。
- 后端：`internal/platform/blob`（localfs/s3/tos 新增 `Check`）、`internal/platform/blob/factory`（带显式凭据的检查入口）、`internal/httpapi/setup`（`blob-test` 端点）、设置校验兼容。
- 文档：`setup-wizard`、`shared-blob-store` 规格、README。
