---
comet_change: shared-directory-storage
role: technical-design
canonical_spec: openspec
archived-with: 2026-08-25-shared-directory-storage
status: final
---

# shared-directory-storage 深度设计

## 背景与范围

目标（详见 `docs/openspec/changes/shared-directory-storage/proposal.md`）：文件存储支持「共享目录（SMB / NFS 挂载）」驱动 `sharedfs`，覆盖同机房多设备共享文件；向导提供挂载指引与可写校验，S3 选项补充局域网（MinIO）提示。

现状约束：

- `blob.Store` 已有 localfs/s3/tos；`factory.Check`/`EnsureBucket`、`blob-test` 端点与向导「连通性测试 + 继续」模式已落地。
- `settings.Validate` 按 placement 校验：远程禁止 localfs；向导按驱动推断 placement（localfs→本机，s3/tos→远程）。
- localfs 实现即「目录路径=存储」，SMB/NFS 挂载点对应用层就是本地路径——复用实现即可。

## 目标 / 非目标

**目标**

- 新增 `blob.driver=sharedfs`（复用 localfs），控制面与 Edge 挂载同一目录后按 key 互通读写。
- 校验：`sharedfs` 要求非空路径；远程允许 `sharedfs`、仍拒绝 `localfs`。
- 向导「文件存储配置」新增「共享目录（SMB / NFS）」选项：挂载目录输入、info 挂载指引、连通性测试、placement 推断远程；S3 加局域网提示。

**非目标**

- 不做 WebDAV/SFTP 驱动；应用不负责挂载（OS 层运维，只校验可写并给指引）。
- 不改 s3/tos 实现；不做自动挂载脚本。

## 技术方案

### 1. 后端：`sharedfs` 驱动与校验

`internal/platform/botconfig/config.go`：

```go
const BlobDriverSharedFS = "sharedfs"
```

`internal/platform/settings/settings.go` `Validate()`：

```go
switch d {
case DriverSQLite, DriverMySQL, DriverPostgres:
// ...
}
switch b {
case botconfig.BlobDriverLocalFS, botconfig.BlobDriverSharedFS,
     botconfig.BlobDriverS3, botconfig.BlobDriverTOS:
// ...
}
// 路径必填：localfs 与 sharedfs
if (b == botconfig.BlobDriverLocalFS || b == botconfig.BlobDriverSharedFS) &&
    strings.TrimSpace(s.BlobRoot) == "" {
    return fmt.Errorf("settings: %s requires blob root", b)
}
// placement 组合：remote 允许 s3/tos/sharedfs，拒绝 localfs
case PlacementRemote:
    if b == botconfig.BlobDriverLocalFS {
        return fmt.Errorf("settings: remote deployment cannot use blob.driver=localfs")
    }
    if b != botconfig.BlobDriverS3 && b != botconfig.BlobDriverTOS && b != botconfig.BlobDriverSharedFS {
        return fmt.Errorf("settings: remote requires blob.driver=s3, tos or sharedfs")
    }
```

`internal/platform/blob/factory`：`NewFromConfig` 与 `Check`/`EnsureBucket` 的 `storeForCheck` 增加 `sharedfs → localfs.New` 分发（`EnsureBucket` 对 sharedfs 与 localfs 一样只做目录检查）。

`internal/httpapi/setup/handler.go` `blobTest`：placement 推断增加 `sharedfs → remote`；校验与 `factory.Check` 走目录可写路径。

`apps/edge-agent`：`BLOB_DRIVER=sharedfs` 经 `factory.New` 自动分发（无需额外改动，但补一个分发单测）。

### 2. 前端：向导「共享目录（SMB / NFS）」

`web/admin/src/features/setup/setup-wizard.tsx`：

- 驱动下拉新增 `<SelectItem value='sharedfs'>共享目录（SMB / NFS）</SelectItem>`。
- `sharedfs` 选中时显示「挂载目录」输入（复用 localfs 表单逻辑，`blobRoot` 字段）+ info Alert：

```tsx
{blobDriver === 'sharedfs' ? (
  <Alert variant='info'>
    <CircleInfo aria-hidden='true' />
    <AlertTitle>注意！</AlertTitle>
    <AlertDescription>
      需要先在所有机器上挂载同一共享目录（SMB / NFS）。
    </AlertDescription>
    <pre className='...'>
      {`# Linux NFS
mount -t nfs 192.168.1.10:/srv/pixoma /mnt/pixoma-shared

# Linux / macOS CIFS (SMB)
mount -t cifs //192.168.1.10/pixoma /mnt/pixoma-shared -o username=admin`}
    </pre>
  </Alert>
) : null}
```

- `draft()` 的 placement 推断改为：`localfs`→local，`s3`/`tos`/`sharedfs`→remote。
- `blobKey` 计算已含 `blobDriver`/`blobRoot`，切换 sharedfs 自动重置测试状态。
- S3 选中时 Endpoint 下加 hint：`局域网可用 MinIO 等 S3 兼容服务，如 http://192.168.x.x:9000`。

`blob-test` 请求体对 sharedfs 只需 `blob_driver` + `blob_root`（后端 Validate 要求路径）。

### 3. 文档

README「对象存储」段落补充：`sharedfs`（SMB/NFS 共享目录）适用于同机房多设备；先挂载再配置；NFS/CIFS 示例命令。

## 数据流：共享目录装配

```text
运维：在所有机器挂载同一共享目录（NFS / CIFS）
  → 向导「文件存储配置」选「共享目录（SMB / NFS）」填挂载路径
  → 连通性测试（factory.Check → localfs.Check 目录可写）→ 连接正常
  → draft 保存 blob.driver=sharedfs + blob_root；placement=remote
  → 重启装配：控制面与 Edge 各用 localfs 实现读写同一目录，按 BlobRef key 互通
```

## 边界条件

- `sharedfs` 空路径 → Validate 拒绝。
- `remote + localfs` 仍拒绝；`remote + sharedfs` 通过。
- 挂载点未挂载/不可写 → `localfs.Check` 报可诊断错误。
- 驱动值大小写/空白 → 统一 trim+lower（现有行为）。

## 测试策略

- Go 单测：`botconfig` 驱动常量；`settings.Validate` 组合矩阵（sharedfs 空路径、remote+localfs 拒绝、remote+sharedfs 通过、local+sharedfs 通过）；`factory` 分发（NewFromConfig/Check/EnsureBucket 的 sharedfs→localfs）；`blob-test` handler 对 sharedfs 走目录校验。
- 前端：`setup-pages.contract.test.ts` 断言 sharedfs 选项、挂载指引（NFS/CIFS 示例）、S3 局域网提示、placement 推断；`setup-steps` 无改动。
- 回归：`go build ./... && go test ./...`；`cd web/admin && pnpm tsc -b && pnpm vitest run`。

## 风险与缓解

- [挂载是 OS 层职责] → 应用只校验可写并给指引；README 明确「先挂载再配置」。
- [Windows/Linux 挂载命令差异] → 指引同时给 NFS（Linux）与 CIFS（Windows/Linux）示例。
- [sharedfs 与 localfs 同实现易混] → 独立驱动值区分，校验与文案不共用「仅本机」语义。

## 交付清单

- `internal/platform/botconfig/config.go`：`BlobDriverSharedFS`。
- `internal/platform/settings/settings.go`：校验组合 + 单测。
- `internal/platform/blob/factory/factory.go`、`check.go`：sharedfs 分发 + 单测。
- `internal/httpapi/setup/handler.go`：`blobTest` placement 推断 + 单测。
- `web/admin/src/features/setup/setup-wizard.tsx`：sharedfs 选项/表单/挂载指引、S3 局域网提示、placement 推断 + 合同测试。
- `README.md`：共享目录挂载指引。
