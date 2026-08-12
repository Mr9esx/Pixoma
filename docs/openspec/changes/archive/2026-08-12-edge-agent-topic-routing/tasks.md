## 1. 模式、Blob、Queue

- [x] 1.1 引入 `runtime_mode=allinone|split` 与 queue/blob 驱动配置；启动组合校验
- [x] 1.2 保留 localfs；新增 S3 兼容 `blob.Store`；前缀 `inputs/` `jobs/` `outputs/`
- [x] 1.3 保留 Memory；新增 Redis Streams `queue.Bus`；topic ↔ stream 映射与 Ack
- [x] 1.4 适配器单测/集成测（Memory、localfs、Redis、S3 或测试替身）

## 2. 方案 A 任务包与 Dispatch

- [x] 2.1 定义 job JSON 与 `job_ref`；扩展 `DispatchCommand`
- [x] 2.2 实现 prep（非图片注入 + images 清单）；禁止 prep 调家里 UploadImage
- [x] 2.3 执行面改为读 job → 本机 Upload → Submit/Wait → outputs → status
- [x] 2.4 allinone 同进程接线：prep + 执行面 + Memory + localfs
- [x] 2.5 缺输入/坏 job_ref 失败路径测试

## 3. Edge（split）

- [x] 3.1 新增 `edge-agent` 入口（Redis、S3、本机 Comfy、instance_id、subscribe topic、mock）
- [x] 3.2 云侧 split 默认不订阅生产 dispatch；Edge 心跳/在线信号最小实现
- [x] 3.3 无在线 Edge 时保持 pending；相关测试
- [x] 3.4 split 冒烟：Redis + S3 + Edge + mock/真 Comfy（可文档化手动步骤）

## 4. Mock、文档与验收

- [x] 4.1 allinone + `comfy_mock` 端到端：确认生成 → 收到产物
- [x] 4.2 更新 `docs/architecture/`（runtime/overview）：双模式、job_ref、Edge
- [x] 4.3 回归 orchestrator/actuator/confirm_run 等相关测试
- [x] 4.4 （明确移出本期范围）Topic Admin / 投放表达式 / 会员×分类分流 — 不实施，见下方「延后」

## Review notes（build `review_mode=standard`）

审查见会话 code review：Critical 已修（split 用 `ListEnabled`+`Online` 而非云侧 Comfy 探活；Redis PEL 重试；`Worker.fail` 传播 status 发布错误）。

接受延后（非 CRITICAL）：
- Redis AUTH/TLS、S3 启动 HeadBucket 探测（部署约定 / 后续加固）
- `OutputPrefix` 字段尚未驱动产物路径（行为与既有 `outputs/<task_id>` 一致）
- 方案 A 在 allinone 仍允许无 `job_ref` 回落 `WorkflowForTask`（兼容）；Edge 无 Workflows，空 `job_ref` 会失败

## 延后（另开 change，不在本 change 实施）

- Topic Admin、投放表达式、会员×分类分流
