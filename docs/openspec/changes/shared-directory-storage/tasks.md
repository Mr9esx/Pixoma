## 1. 后端 sharedfs 驱动与校验

- [x] 1.1 `botconfig` 新增 `BlobDriverSharedFS = "sharedfs"`；`blob/factory` 将 `sharedfs` 分发到 `localfs.New`，含单测
- [x] 1.2 `settings.Validate`：`sharedfs` 要求非空 `BlobRoot`，允许 local/remote（远程+sharedfs 通过、远程+localfs 仍拒绝），含单测
- [x] 1.3 `blob-test` 的 placement 推断与校验支持 `sharedfs`（含单测）

## 2. 前端向导「共享目录」选项

- [x] 2.1 向导驱动下拉新增「共享目录（SMB / NFS）」`sharedfs` 选项：目录路径输入 + info 挂载指引（NFS/CIFS 示例）+ 连通性测试/继续；local fs 保留仅同机 warn
- [x] 2.2 选择 S3 时 Endpoint 下加局域网提示（MinIO 等 S3 兼容服务示例）
- [x] 2.3 合同测试：sharedfs 选项、挂载指引、S3 局域网提示、placement 推断

## 3. 文档与验证

- [x] 3.1 README 增加 SMB/NFS 挂载指引与 `sharedfs` 说明
- [x] 3.2 `go build ./...` + `go test ./...`；`pnpm tsc -b` + `pnpm vitest run`（web/admin）
