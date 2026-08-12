---
comet_change: blob-tos-adapter
role: technical-design
canonical_spec: openspec
---

# Blob TOS 适配器 — 技术设计

## 1. 背景与目标

`split` 部署跨机文件目前只有 S3 兼容驱动。本期新增 `blob.driver=tos`，用火山引擎官方 Go SDK 实现同一 `blob.Store` 端口，使控制面与 Edge 可通过 TOS 读写 `inputs/`、`jobs/`、`outputs/`，且不改 `BlobRef` 契约。

**目标**

- 独立包实现 Put/Get，行为对齐现有 `blob/s3`
- `split` 允许 `blob∈{s3,tos}`，与 `queue=redis` 搭配
- edge-agent（及后续 bot 装配点）可按驱动切换
- Verify **必须**对真桶 `pixoma-test` 完成 Put→Get

**非目标**

- STS / SecurityToken
- 流式大文件优化（超出当前 s3 的 ReadAll 模型）
- 删除 localfs/s3；改 topic / 在 MQ 传文件字节
- NATS / 去 Redis（`queue-nats-adapter` 已取消）

## 2. 架构

```
botconfig (driver=tos + TOS_*)
        │
        ▼
 ┌──────────────────┐
 │  blob/tos.Store  │ ── ve-tos-golang-sdk/v2 ──▶ TOS (pixoma-test)
 └──────────────────┘
        ▲
        │ 实现 blob.Store
 edge-agent / 控制面装配
```

业务与调度只依赖 `blob.Store`；火山类型不泄漏到领域层。

## 3. 包与 API

**路径**：`internal/platform/blob/tos`

```go
type Options struct {
    Endpoint        string
    Region          string
    Bucket          string
    AccessKeyID     string
    SecretAccessKey string
}

func New(opts Options) (*Store, error)
func (s *Store) Put(ctx context.Context, key string, r io.Reader, opts blob.PutOptions) (sharedkernel.BlobRef, error)
func (s *Store) Get(ctx context.Context, ref sharedkernel.BlobRef) (io.ReadCloser, error)
```

**SDK 用法**

- 模块：`github.com/volcengine/ve-tos-golang-sdk/v2`
- `tos.NewClientV2(endpoint, tos.WithRegion(region), tos.WithCredentials(tos.NewStaticCredentials(ak, sk)))`
- Put：`PutObjectV2`（Content-Type 来自 `PutOptions.MIME`）
- Get：`GetObjectV2`，返回 `Content` 作为 `io.ReadCloser`

**Key 规则**：与 `blob/s3` 一致（拒绝对对路径、`..`、空 key）；`BlobRef.Bucket` 缺省回落到 Options.Bucket。

**Put 体**：对齐 s3，可读全量再上传（本期不对齐新流式能力）。

## 4. 配置与校验

**常量**：`BlobDriverTOS = "tos"`

**环境变量**（edge / 联调）

| 变量 | 含义 |
|------|------|
| `BLOB_DRIVER` | `localfs` / `s3` / `tos` |
| `TOS_ENDPOINT` | 例 `https://tos-cn-beijing.volces.com` |
| `TOS_REGION` | 例 `cn-beijing` |
| `TOS_BUCKET` | 例 `pixoma-test` |
| `TOS_ACCESS_KEY` / `TOS_SECRET_KEY` | 静态密钥 |

本地临时文件：`.env.tos.local`（已 gitignore）。**禁止**写入仓库、OpenSpec 产物或 Design Doc 正文。

**`ValidateRuntimeDrivers`**

- `allinone`：`queue=memory` + `blob=localfs`
- `split`：`queue=redis` + `blob∈{s3,tos}`
- 非法组合：启动失败，错误信息可诊断

YAML：可在 `blob` 下扩展嵌套 tos 字段；若现有 s3 仅靠 env，tos 可同样以 env 为主，与 edge-agent 现状一致。

## 5. 装配

**edge-agent**（现硬编码 `s3.New`）

- 读 `BLOB_DRIVER`（默认保持当前 s3 行为以免破坏既有 split）
- `tos` → `blob/tos.New` + `TOS_*`
- `s3` → 现有 `blob/s3` + `S3_*`

**控制面 / bot**

- 仓库若尚无独立 bot main，则：
  - 在 `botconfig` 完成驱动常量与校验
  - 提供小型工厂（例如 `internal/platform/blob/factory` 或 packaging 内 helper）供后续入口复用
  - 不为此期强造完整 bot 进程重构

## 6. 测试与验收

| 层级 | 内容 |
|------|------|
| 单元 | 空 bucket、非法 key；可不连网 |
| 真网门禁 | `go test -tags=live_tos` 集成用例：加载 env / `.env.tos.local`，对 `pixoma-test` Put 随机 key → Get 比对字节；**缺任一 `TOS_*` → 失败（不算过）**；默认 `go test ./...` 不编译该用例 |

Verify 阶段证据必须包含真桶往返成功日志/测试输出。用完临时 AK/SK 后作废。

## 7. Spec Patch

已确认写入 OpenSpec delta：`模式与驱动一致性校验` 去掉 nats 前瞻，明确本期 `split` = redis + (s3\|tos)。

## 8. 实现顺序

1. botconfig 常量 + 校验 + 单测  
2. 引入 SDK，实现 `blob/tos` + 单测  
3. 真网 Put→Get 测试（读 `.env.tos.local`）  
4. edge-agent 按 `BLOB_DRIVER` 分支  
5. 配置样例 / 架构文档若触及驱动列表则同步  
6. Verify：真 TOS 门禁

## 9. 风险

- SDK 版本锁定在 go.mod，封装在 tos 包内  
- 与未来其它驱动 change 合并时，保持「按维放宽」校验表，避免又写死单一组合  
- 聊天曾出现明文密钥：交付后轮换
