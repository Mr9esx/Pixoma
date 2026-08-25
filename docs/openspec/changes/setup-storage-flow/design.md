## Context

现状（动机见 `proposal.md`）：

- Setup 向导步骤为 密码 → 数据库 → 出图机器在哪（placement）→ 对象存储；placement 决定 blob 驱动合法组合（远程禁止 localfs）。
- 对象存储步骤没有任何连通性校验，填错 endpoint/密钥/bucket 要到 draft 保存后才暴露。
- `settings.Settings.Validate` 已按 placement 校验 blob 合法组合；`blob` 各驱动（localfs/s3/tos）只有 Put/Get，无连通性检查。
- 数据库步骤已建立「连通性测试 + 继续」双按钮与 success/destructive Alert 模式，可直接复用到对象存储步骤。

## Goals / Non-Goals

**Goals:**

- Setup 去掉「出图机器在哪」步骤，数据库配置后直接进入对象存储配置；`placement` 由 blob 驱动推断，远程禁止 localfs 约束在 API 层继续生效。
- 对象存储步骤支持连通性测试：localfs 校验目录可写；S3/TOS 用 HeadBucket 校验 endpoint/region/bucket/密钥；成功 success Alert、失败 destructive Alert。

**Non-Goals:**

- 不新增 OSS/Azure 等对象存储驱动（本期只对现有 localfs/s3/tos 做连通性测试）。
- 不自动创建 bucket（连通性测试只校验，不创建）。
- 不改数据库步骤行为、不改设置页对象存储配置、不在向导内重做 Edge 部署指引。

## Decisions

### D1：移除 placement 步骤，placement 由 blob 驱动推断

前端 `SETUP_STEPS` 变为 密码 → 数据库 → 对象存储；`draft()` 不再携带用户选择的 placement，而是按 `blob_driver` 推断：`localfs`→`local`，`s3`/`tos`→`remote`。后端 `settings.Validate` 不变，继续拒绝 `remote + localfs`（API 层保护）；前端不提供该组合输入。

### D2：对象存储连通性检查（后端）

- `blob/localfs`、`blob/s3`、`blob/tos` 各新增 `Check(ctx) error`：
  - localfs：`os.MkdirAll` + 探测文件写入/删除。
  - s3：`HeadBucket`。
  - tos：TOS SDK `ClientV2.HeadBucket`。
- `blob/factory` 新增 `Check(ctx, CheckOptions)`，`CheckOptions` 带显式凭据（endpoint/region/bucket/accessKey/secretKey/localRoot），不依赖环境变量回退，供向导测试用。
- Setup 新增 `POST /api/v1/setup/blob-test`：请求体为 blob 配置（driver/root/endpoint/region/bucket/keys），先做合法性校验，再调用 `factory.Check`；成功返回 ok，失败返回可诊断错误。
- 自动建 bucket：`s3`/`tos` 各新增 `EnsureBucket(ctx)`（`HeadBucket` 探测 → 不存在时 `CreateBucket`/`CreateBucketV2` → 再 `HeadBucket` 确认）；`blob` 包新增哨兵错误 `ErrBucketNotFound`。`blob-test` 检查到 bucket 不存在且未确认创建时返回 `{ok:false, code:"bucket_not_found", bucket}`，前端提示「是否帮你创建」，用户确认后带 `auto_create_bucket:true` 重试，后端创建后返回 ok。

### D3：前端对象存储步骤复用数据库步骤交互

- 对象存储步骤增加「连通性测试」按钮（调 `blob-test`）+「继续」按钮；通过显示 success Alert（`CircleCheck` + 连接正常），失败显示 destructive Alert。
- 新增 `blob-error.ts`：常见对象存储错误（InvalidAccessKeyId、SignatureDoesNotMatch、NoSuchBucket、AccessDenied、connection refused、timeout、空配置等）映射中文友好标题 + 实际详情，复用 `setupErrorCopy` 的 Alert 形态。
- localfs 选择仍展示目录输入；S3/TOS 展示 endpoint/region/bucket/access key/secret key。
- bucket 不存在时：测试返回 `bucket_not_found` → 展示提示「bucket `xxx` 不存在，帮你创建？」（创建/取消），确认后带 `auto_create_bucket` 重试，创建成功显示连接正常。
- 存储步骤文案：Title「文件存储配置」、Desc「决定了生成的图和视频存放的位置。」；选择 `localfs`（本机目录）时展示 warn Alert（`CircleAlert` 图标 + Title「注意！」+ Description 连续文案「这个配置只适合 ComfyUI 和后台在同一台机器上使用，无法使用远程节点。」（自然换行，不强制断行））；对象存储驱动标签 S3/TOS/localfs 用「S3 / 火山 TOS / 本机目录」。

`blob-error.ts` 错误映射只认 `NoSuchBucket`/`NoSuchKey` 为 bucket 不存在；裸 `404`（如旧后端未更新返回的 `Request failed (404)`）走通用「对象存储配置失败，请重试」，避免误报。

### D4：步骤序列与测试

- `setup-steps.ts`：删除 `placement` 步骤与 copy；`initialSetupStep`/`previousSetupStep`/`setupStepIndex` 相应更新。
- 更新 `setup-steps.test.ts` 与合同测试：3 步流程、无「出图机器在哪」、对象存储测试按钮与 Alert 断言。

## Risks / Trade-offs

- [创建 bucket 需要账号权限] → 用户确认创建后若权限不足，报可诊断错误；README 说明需 `CreateBucket`/`PutBucket` 权限。
- [OSS 等新驱动缺失] → 本期非目标；`blob-test` 端点按 driver 分发，后续加驱动只需补 `Check`。
- [placement 推断改变既有保存语义] → 前端 draft 发送推断值，后端 Validate 行为不变；既有已初始化部署不受影响（putSettings 仍从引导态读 DB 连接）。

## Migration Plan

- 无破坏性 schema 变更；仅向导流程与新增端点。
- 新部署按新步骤完成向导；既有部署不受影响。

## Open Questions

- 是否需要支持 MinIO 等 S3 兼容自建（当前 s3 驱动已支持 endpoint + path-style）——连通性测试直接复用 HeadBucket，无需额外决策。
