## Why

文件存储目前只有三类：`localfs`（本机目录，明确警告只适合同机）、`s3`/`tos`（跨网络对象存储）。同一机房多台设备共享文件的场景夹在中间：不需要 S3/TOS 的开销，但本机目录语义又不成立。SMB / NFS 这类操作系统级共享目录正好覆盖这个空档——两边挂载同一个目录即可互通，应用层只需要把它作为一等选项并提供挂载指引与可写校验。

## What Changes

- **新增 `sharedfs` 共享目录驱动**：复用 `localfs` 实现（挂载点即路径），作为独立驱动值 `blob.driver=sharedfs`；配置字段仍是目录路径。
- **校验与 placement 语义**：`sharedfs` 允许本机或远程部署（跨机场景），API 层「远程禁止 localfs」继续只针对本机目录；`sharedfs` 必须有非空路径。
- **向导「文件存储配置」**：驱动下拉新增「共享目录（SMB / NFS）」选项；选中时展示挂载路径输入、info 提示（需先在所有机器上挂载同一目录，如 `mount -t nfs` / `mount -t cifs`）、连通性测试（目录可写）与继续按钮；本机目录仍保留「仅同机」warn。
- **局域网 S3 提示**：选择 S3 时在 Endpoint 下加提示「局域网可用 MinIO 等 S3 兼容服务，如 `http://192.168.x.x:9000`」。
- **文档**：README 与规格补充 SMB/NFS 挂载指引、`sharedfs` 说明。

## Non-Goals

- 不做 WebDAV / SFTP 驱动（应用内直连协议，另行评估）。
- 应用不负责挂载 SMB/NFS（操作系统层运维职责，仅校验挂载点可写并给指引）。
- 不改 `s3`/`tos` 驱动实现；不做自动挂载脚本。

## Capabilities

### New Capabilities
- `shared-directory-storage`: 文件存储支持共享目录（SMB/NFS 挂载）驱动 `sharedfs`，覆盖同机房多设备共享文件场景。

### Modified Capabilities
- `shared-blob-store`: 新增 `sharedfs` 驱动（复用 localfs 实现），连通性检查与合法组合校验相应扩展。
- `setup-wizard`: 「文件存储配置」新增「共享目录（SMB / NFS）」选项与挂载指引、局域网 S3 提示。

## Impact

- 后端：`internal/platform/botconfig`（驱动常量）、`internal/platform/settings`（校验）、`internal/platform/blob/factory`（`sharedfs` 分发）、`internal/httpapi/setup`（blob-test 与 draft 兼容）。
- 前端：`web/admin` Setup 向导（驱动选项、共享目录表单与提示、S3 局域网提示）、合同测试。
- 文档：`shared-blob-store`、`setup-wizard` 规格、README。
