## Context

现状（动机见 `proposal.md`）：

- `blob.Store` 已有 localfs/s3/tos 三驱动；`factory.Check`/`EnsureBucket`、向导连通性测试与 `blob-test` 端点刚落地。
- `settings.Validate` 按 placement 校验：远程禁止 localfs；向导按 blob 驱动推断 placement（localfs→本机，s3/tos→远程）。
- localfs 本质是「目录路径即存储」，SMB/NFS 挂载点对应用层就是本地路径——复用 localfs 实现即可获得跨机共享，缺的是独立驱动标识、校验语义与向导引导。

## Goals / Non-Goals

**Goals:**

- 新增 `blob.driver=sharedfs`（复用 localfs 实现），覆盖同机房多设备通过 SMB/NFS 挂载共享目录。
- 校验与 placement：远程允许 sharedfs、仍拒绝 localfs；sharedfs 要求非空路径。
- 向导「文件存储配置」新增「共享目录（SMB / NFS）」选项：挂载路径输入、info 挂载指引、连通性测试（目录可写）、placement 推断远程；S3 选项加局域网（MinIO）提示。

**Non-Goals:**

- 不做 WebDAV/SFTP 驱动；应用不负责挂载（OS 层运维）。
- 不改 s3/tos 实现；不做自动挂载脚本。

## Decisions

### D1：新增驱动值 `sharedfs`，复用 localfs

- `botconfig` 新增 `BlobDriverSharedFS = "sharedfs"`；`blob/factory` 将 `sharedfs` 分发到 `localfs.New`（同一实现，语义不同）。
- `settings.Validate`：`sharedfs` 与 `localfs` 一样要求非空 `BlobRoot`；placement 允许 local/remote（远程+sharedfs 通过，远程+localfs 仍拒绝）。
- 向导/设置页按驱动值区分选项与文案；`blob-test` 的 placement 推断 `sharedfs`→remote。

### D2：向导「共享目录（SMB / NFS）」选项

- 驱动下拉新增 `sharedfs`（文案「共享目录（SMB / NFS）」）。
- 选中时：目录路径输入（复用 localfs 表单）+ info Alert 挂载指引（先在所有机器挂载同一目录，示例 `mount -t nfs` / `mount -t cifs`）+ 连通性测试/继续（复用现有模式）。
- localfs 保留「仅同机」warn；S3 选中时 Endpoint 下加局域网提示（MinIO 等，示例 `http://192.168.x.x:9000`）。

### D3：错误映射与文档

- `blob-error.ts` 增加 sharedfs 相关文案（目录不可写等走现有 localfs 映射路径）。
- README 增加 SMB/NFS 挂载指引与 `sharedfs` 说明；规格同步。

## Risks / Trade-offs

- [挂载是 OS 层职责] → 应用只校验可写并给指引；README 明确「先挂载再配向导」。
- [Windows 与 Linux 挂载命令差异] → 指引同时给出 NFS（Linux）与 SMB/CIFS（Windows/Linux）示例。
- [sharedfs 与 localfs 同实现易混] → 用独立驱动值区分，校验与文案不共用「仅本机」语义。

## Migration Plan

- 无破坏性 schema 变更；新增驱动值与向导选项，既有部署不受影响。

## Open Questions

- 是否需要在设置页也展示 sharedfs（本期仅向导，设置页保持只读展示驱动名即可）。
