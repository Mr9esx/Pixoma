---
change: blob-tos-adapter
design-doc: docs/superpowers/specs/2026-08-12-blob-tos-adapter-design.md
base-ref: 80569755843197175d8863fa19dd423153740bfa
archived-with: 2026-08-12-blob-tos-adapter
---

# Blob TOS 适配器 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 新增 `blob.driver=tos`，用火山引擎官方 SDK 实现与现有 S3 行为对齐的 `blob.Store`，使 `split` 下可用 `redis + tos` 跨机读写，且真桶 Put→Get 硬门禁必须通过。

**Architecture:** 业务只依赖 `blob.Store` 端口；新建独立包 `internal/platform/blob/tos` 封装 `ve-tos-golang-sdk/v2`，不向领域层泄漏火山类型。`botconfig` 放宽校验矩阵（`split` = `queue=redis` 且 `blob∈{s3,tos}`）。edge-agent 按 `BLOB_DRIVER` 选型；控制面尚无完整 bot main 时，提供小型工厂供后续入口复用，不为此期强造 bot 进程重构。

**Tech Stack:** Go、`blob.Store` / `sharedkernel.BlobRef`、现有 `blob/s3` 模式、`github.com/volcengine/ve-tos-golang-sdk/v2`、`botconfig`、`apps/edge-agent`。

## Global Constraints

- 产物语言：zh-CN（文档与本计划）；代码标识符保持仓库英文风格
- `comfy_mock` / `COMFY_MOCK` 必须继续端到端可通（本 change 不改 Comfy 路径，但回归不得破坏）
- 禁止 STS / SecurityToken；禁止流式大文件新能力；禁止删 localfs/s3；禁止在 MQ 传文件字节
- 本期 `split` 合法组合仅为 `queue=redis` + `blob∈{s3,tos}`（不以 nats 为合法 queue）
- Key 规则必须与 `blob/s3` 一致：拒绝对对路径、`..`、空 key；`BlobRef.Bucket` 缺省回落 Options.Bucket
- Put 体对齐 s3：`io.ReadAll` 后再上传
- **禁止**把 AK/SK 写入仓库、本计划、OpenSpec、Design Doc、测试源码常量或提交信息；凭证只来自环境变量 / 本地 `.env.tos.local`（已 gitignore）
- 真网门禁：缺任一必需 `TOS_*` → 测试 **必须失败**（不算 skip / 不算过）
- 架构驱动列表若扩展为含 TOS，必须同步 `docs/architecture/`（overview / runtime / bounded-contexts 等触及处）

## 文件结构

| 文件 | 职责 |
|---|---|
| `internal/platform/botconfig/config.go` | `BlobDriverTOS`、可选 TOS 连接字段、`ValidateRuntimeDrivers` 放宽 |
| `internal/platform/botconfig/config_test.go` | split+tos+redis 通过；split+localfs / 非法组合失败 |
| `internal/platform/blob/tos/tos.go` | `Options` / `New` / `Put` / `Get` / `cleanKey` |
| `internal/platform/blob/tos/tos_test.go` | 空 bucket、非法 key（不连网） |
| `internal/platform/blob/tos/tos_live_test.go` | 真桶 Put→Get（`//go:build live_tos`；读环境变量，缺配置即 FAIL） |
| `internal/platform/blob/factory/factory.go`（新建） | 按 driver 装配 localfs / s3 / tos，供 edge 与后续 bot 复用 |
| `apps/edge-agent/cmd/edge-agent/main.go` | `BLOB_DRIVER` 分支（默认仍 s3） |
| `configs/bot.yaml` | 注释样例：`split` 可用 s3 或 tos + `TOS_*` |
| `docs/architecture/overview.md` 等 | 驱动列表含 TOS |
| `go.mod` / `go.sum` | 锁定 `ve-tos-golang-sdk/v2` |

### 真 TOS 凭证加载（实现与 Verify 共用）

`.env.tos.local` 已在 `.gitignore`，**不要**提交。在 shell 中注入环境后再跑测试：

```bash
set -a
source .env.tos.local
set +a
# 必需变量（名称固定；值由本地文件提供，切勿写入本计划或源码）：
# TOS_ENDPOINT  TOS_REGION  TOS_BUCKET  TOS_ACCESS_KEY  TOS_SECRET_KEY
go test ./internal/platform/blob/tos/ -tags=live_tos -run 'TestRealTOS|TestLive' -count=1 -v
```

---

### Task 1: botconfig — BlobDriverTOS 与 ValidateRuntimeDrivers

**Files:**
- Modify: `internal/platform/botconfig/config.go`
- Modify: `internal/platform/botconfig/config_test.go`
- Modify: `configs/bot.yaml`（注释样例，无密钥）

**Interfaces:**
- Consumes: 现有 `RuntimeModeSplit`、`QueueDriverRedis`、`BlobDriverS3`、`BlobDriverLocalFS`
- Produces:
  - `const BlobDriverTOS = "tos"`
  - `ValidateRuntimeDrivers()`：`split` 允许 `blob ∈ {s3, tos}` 且要求 `queue=redis`
  - 可选：`BlobConfig` 增加非密钥连接字段（endpoint/region/bucket），AK/SK **仅**环境变量

- [x] **Step 1: 写失败测试 — split + tos + redis 尚不被允许**

在 `config_test.go` 追加：

```go
func TestValidateRuntimeDrivers_SplitTOSOK(t *testing.T) {
	cfg := botconfig.Default()
	cfg.RuntimeMode = botconfig.RuntimeModeSplit
	cfg.Queue.Driver = botconfig.QueueDriverRedis
	cfg.Blob.Driver = botconfig.BlobDriverTOS
	if err := cfg.ValidateRuntimeDrivers(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRuntimeDrivers_SplitRejectsLocalFS(t *testing.T) {
	cfg := botconfig.Default()
	cfg.RuntimeMode = botconfig.RuntimeModeSplit
	cfg.Queue.Driver = botconfig.QueueDriverRedis
	cfg.Blob.Driver = botconfig.BlobDriverLocalFS
	err := cfg.ValidateRuntimeDrivers()
	if err == nil {
		t.Fatal("expected error for split+localfs")
	}
	if !strings.Contains(err.Error(), "blob.driver") {
		t.Fatalf("error should mention blob.driver, got %v", err)
	}
}

func TestValidateRuntimeDrivers_SplitS3StillOK(t *testing.T) {
	cfg := botconfig.Default()
	cfg.RuntimeMode = botconfig.RuntimeModeSplit
	cfg.Queue.Driver = botconfig.QueueDriverRedis
	cfg.Blob.Driver = botconfig.BlobDriverS3
	if err := cfg.ValidateRuntimeDrivers(); err != nil {
		t.Fatal(err)
	}
}
```

（`SplitTOSOK` 在实现前会因 `BlobDriverTOS` 未定义而编译失败，或常量已加但校验仍拒 tos。）

- [x] **Step 2: 跑测试确认当前失败**

Run: `go test ./internal/platform/botconfig/ -run ValidateRuntimeDrivers -count=1`

Expected: 编译失败（无 `BlobDriverTOS`）或 `SplitTOSOK` FAIL / 相关断言失败。

- [x] **Step 3: 最小实现 — 常量、注释字段、校验矩阵**

在 `config.go`：

```go
const (
	// ...existing...
	BlobDriverLocalFS = "localfs"
	BlobDriverS3      = "s3"
	BlobDriverTOS     = "tos"
)

// BlobConfig selects the blob adapter (BlobRoot still used for localfs).
type BlobConfig struct {
	Driver string `yaml:"driver"` // localfs | s3 | tos
	// TOS 非密钥连接信息（可选；密钥只用 TOS_ACCESS_KEY / TOS_SECRET_KEY）。
	TOS BlobTOSConfig `yaml:"tos"`
}

type BlobTOSConfig struct {
	Endpoint string `yaml:"endpoint"`
	Region   string `yaml:"region"`
	Bucket   string `yaml:"bucket"`
}
```

改 `ValidateRuntimeDrivers` 的 split 分支：

```go
case RuntimeModeSplit:
	if q != QueueDriverRedis {
		return fmt.Errorf("botconfig: split requires queue.driver=redis, got %q", q)
	}
	if b != BlobDriverS3 && b != BlobDriverTOS {
		return fmt.Errorf("botconfig: split requires blob.driver=s3 or tos, got %q", b)
	}
```

在 `applyEnv` 中（仅非密钥）：

```go
if v := os.Getenv("TOS_ENDPOINT"); v != "" {
	cfg.Blob.TOS.Endpoint = strings.TrimSpace(v)
}
if v := os.Getenv("TOS_REGION"); v != "" {
	cfg.Blob.TOS.Region = strings.TrimSpace(v)
}
if v := os.Getenv("TOS_BUCKET"); v != "" {
	cfg.Blob.TOS.Bucket = strings.TrimSpace(v)
}
// 不要把 TOS_ACCESS_KEY / TOS_SECRET_KEY 读进 Config 结构体并持久化到 YAML 路径。
```

更新 `configs/bot.yaml` 注释（无密钥）：

```yaml
# runtime_mode: split      # queue.driver=redis + blob.driver=s3|tos；Edge 需 REDIS_* / S3_* 或 TOS_*
# blob:
#   driver: tos            # 或 s3
#   tos:
#     endpoint: "https://tos-cn-beijing.volces.com"
#     region: "cn-beijing"
#     bucket: "pixoma-test"
# 密钥：环境变量 TOS_ACCESS_KEY / TOS_SECRET_KEY（可用 set -a; source .env.tos.local）
```

- [x] **Step 4: 跑测试确认通过**

Run: `go test ./internal/platform/botconfig/ -count=1`

Expected: PASS（含 SplitTOSOK、SplitRejectsLocalFS、既有 SplitOK/SplitRejectsMemory）。

- [x] **Step 5: Commit**

```bash
git add internal/platform/botconfig/config.go internal/platform/botconfig/config_test.go configs/bot.yaml
git commit -m "$(cat <<'EOF'
feat(botconfig): allow split blob driver tos with redis

EOF
)"
```

---

### Task 2: 引入 SDK 并实现 `internal/platform/blob/tos`（单元路径）

**Files:**
- Create: `internal/platform/blob/tos/tos.go`
- Create: `internal/platform/blob/tos/tos_test.go`
- Modify: `go.mod` / `go.sum`

**Interfaces:**
- Consumes: `blob.Store`、`blob.PutOptions`、`sharedkernel.BlobRef`
- Produces:

```go
package tos

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

SDK：`github.com/volcengine/ve-tos-golang-sdk/v2/tos` — `NewClientV2`、`PutObjectV2`、`GetObjectV2`。

- [x] **Step 1: 写失败测试 — 空 bucket 与非法 key**

`tos_test.go`：

```go
package tos_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	blobtos "github.com/mr9esx/comfyui_tgbot/internal/platform/blob/tos"
)

func TestNew_RequiresBucket(t *testing.T) {
	_, err := blobtos.New(blobtos.Options{
		Endpoint: "https://tos-cn-beijing.volces.com",
		Region:   "cn-beijing",
	})
	if err == nil {
		t.Fatal("expected error for empty bucket")
	}
	if !strings.Contains(err.Error(), "bucket") {
		t.Fatalf("got %v", err)
	}
}

func TestPut_RejectsInvalidKeys(t *testing.T) {
	store, err := blobtos.New(blobtos.Options{
		Endpoint:        "https://tos-cn-beijing.volces.com",
		Region:          "cn-beijing",
		Bucket:          "pixoma-test",
		AccessKeyID:     "unused-in-key-check",
		SecretAccessKey: "unused-in-key-check",
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, key := range []string{"", "/abs", "../x", ".."} {
		_, err := store.Put(ctx, key, bytes.NewReader([]byte("x")), blob.PutOptions{})
		if err == nil {
			t.Fatalf("expected error for key %q", key)
		}
	}
}
```

说明：非法 key 必须在发网前由 `cleanKey` 拒绝（对齐 `blob/s3`）。占位凭证仅用于构造客户端，**不要**写成真实 AK/SK。

- [x] **Step 2: 跑测试确认失败**

Run: `go test ./internal/platform/blob/tos/ -count=1`

Expected: 包不存在 / 编译失败。

- [x] **Step 3: 引入依赖**

```bash
go get github.com/volcengine/ve-tos-golang-sdk/v2@latest
```

（锁定进 `go.mod`/`go.sum`；具体版本以命令解析结果为准。）

- [x] **Step 4: 最小实现 `tos.go`**

对齐 `internal/platform/blob/s3/s3.go` 结构：

```go
package tos

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type Options struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
}

type Store struct {
	client *tos.ClientV2
	bucket string
}

func New(opts Options) (*Store, error) {
	if strings.TrimSpace(opts.Bucket) == "" {
		return nil, errors.New("blob/tos: empty bucket")
	}
	endpoint := strings.TrimSpace(opts.Endpoint)
	if endpoint == "" {
		return nil, errors.New("blob/tos: empty endpoint")
	}
	region := strings.TrimSpace(opts.Region)
	if region == "" {
		return nil, errors.New("blob/tos: empty region")
	}
	client, err := tos.NewClientV2(endpoint,
		tos.WithRegion(region),
		tos.WithCredentials(tos.NewStaticCredentials(opts.AccessKeyID, opts.SecretAccessKey)),
	)
	if err != nil {
		return nil, fmt.Errorf("blob/tos: new client: %w", err)
	}
	return &Store{client: client, bucket: opts.Bucket}, nil
}

func (s *Store) Put(ctx context.Context, key string, r io.Reader, opts blob.PutOptions) (sharedkernel.BlobRef, error) {
	if err := ctx.Err(); err != nil {
		return sharedkernel.BlobRef{}, err
	}
	rel, err := cleanKey(key)
	if err != nil {
		return sharedkernel.BlobRef{}, err
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return sharedkernel.BlobRef{}, fmt.Errorf("blob/tos: read body: %w", err)
	}
	in := &tos.PutObjectV2Input{
		PutObjectBasicInput: tos.PutObjectBasicInput{
			Bucket: s.bucket,
			Key:    rel,
		},
		Content: bytes.NewReader(data),
	}
	if opts.MIME != "" {
		in.ContentType = opts.MIME
	}
	if _, err := s.client.PutObjectV2(ctx, in); err != nil {
		return sharedkernel.BlobRef{}, fmt.Errorf("blob/tos: put %s: %w", rel, err)
	}
	return sharedkernel.BlobRef{
		Bucket: s.bucket,
		Key:    rel,
		MIME:   opts.MIME,
		Size:   int64(len(data)),
	}, nil
}

func (s *Store) Get(ctx context.Context, ref sharedkernel.BlobRef) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rel, err := cleanKey(ref.Key)
	if err != nil {
		return nil, err
	}
	bucket := ref.Bucket
	if bucket == "" {
		bucket = s.bucket
	}
	out, err := s.client.GetObjectV2(ctx, &tos.GetObjectV2Input{
		Bucket: bucket,
		Key:    rel,
	})
	if err != nil {
		return nil, fmt.Errorf("blob/tos: get %s: %w", rel, err)
	}
	return out.Content, nil
}

func cleanKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", errors.New("blob/tos: empty key")
	}
	if path.IsAbs(key) || strings.HasPrefix(key, "/") {
		return "", errors.New("blob/tos: absolute key not allowed")
	}
	clean := path.Clean(key)
	if clean == "." || strings.HasPrefix(clean, "../") || clean == ".." {
		return "", errors.New("blob/tos: invalid key")
	}
	return clean, nil
}

var _ blob.Store = (*Store)(nil)
```

- [x] **Step 5: 跑单元测试确认通过**

Run: `go test ./internal/platform/blob/tos/ -run 'TestNew_RequiresBucket|TestPut_RejectsInvalidKeys' -count=1`

Expected: PASS（不依赖真网）。

- [x] **Step 6: Commit**

```bash
git add go.mod go.sum internal/platform/blob/tos/
git commit -m "$(cat <<'EOF'
feat(blob): add Volcengine TOS Store adapter

EOF
)"
```

---

### Task 3: 真 TOS Put→Get 硬门禁

**Files:**
- Create: `internal/platform/blob/tos/tos_live_test.go`

**Interfaces:**
- Consumes: `blobtos.New` / `Put` / `Get`；环境变量 `TOS_*`
- Produces: 可重复跑的真桶往返测试；缺配置 → `t.Fatal`（非 Skip）

- [x] **Step 1: 写真网测试（缺 env 即失败）**

```go
package tos_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	blobtos "github.com/mr9esx/comfyui_tgbot/internal/platform/blob/tos"
)

func requireTOSEnv(t *testing.T) blobtos.Options {
	t.Helper()
	opts := blobtos.Options{
		Endpoint:        os.Getenv("TOS_ENDPOINT"),
		Region:          os.Getenv("TOS_REGION"),
		Bucket:          os.Getenv("TOS_BUCKET"),
		AccessKeyID:     os.Getenv("TOS_ACCESS_KEY"),
		SecretAccessKey: os.Getenv("TOS_SECRET_KEY"),
	}
	var missing []string
	if opts.Endpoint == "" {
		missing = append(missing, "TOS_ENDPOINT")
	}
	if opts.Region == "" {
		missing = append(missing, "TOS_REGION")
	}
	if opts.Bucket == "" {
		missing = append(missing, "TOS_BUCKET")
	}
	if opts.AccessKeyID == "" {
		missing = append(missing, "TOS_ACCESS_KEY")
	}
	if opts.SecretAccessKey == "" {
		missing = append(missing, "TOS_SECRET_KEY")
	}
	if len(missing) > 0 {
		t.Fatalf("real TOS gate requires env %v; load with: set -a; source .env.tos.local; set +a", missing)
	}
	return opts
}

func TestRealTOS_PutGetRoundTrip(t *testing.T) {
	opts := requireTOSEnv(t)
	store, err := blobtos.New(opts)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	key := fmt.Sprintf("jobs/tos-gate/%d/ping.txt", time.Now().UnixNano())
	payload := []byte("tos-live-roundtrip")

	ref, err := store.Put(ctx, key, bytes.NewReader(payload), blob.PutOptions{MIME: "text/plain"})
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if ref.Key != key {
		t.Fatalf("key=%q", ref.Key)
	}
	if ref.Bucket != opts.Bucket {
		t.Fatalf("bucket=%q", ref.Bucket)
	}

	rc, err := store.Get(ctx, ref)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer rc.Close()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("got %q want %q", got, payload)
	}
}
```

注意：测试只读 `os.Getenv`；**不要**在仓库内提交 `.env.tos.local` 或硬编码密钥。桶名期望为联调桶（如 `pixoma-test`），由 `TOS_BUCKET` 决定。

- [x] **Step 2: 未加载 env 时跑一次，确认硬失败**

Run: `env -u TOS_ENDPOINT -u TOS_REGION -u TOS_BUCKET -u TOS_ACCESS_KEY -u TOS_SECRET_KEY go test ./internal/platform/blob/tos/ -tags=live_tos -run TestRealTOS_PutGetRoundTrip -count=1 -v`

Expected: FAIL，消息含 `real TOS gate requires env`（不是 `SKIP`）。

- [x] **Step 3: 加载本地 env 后跑通**

```bash
set -a
source .env.tos.local
set +a
go test ./internal/platform/blob/tos/ -tags=live_tos -run TestRealTOS_PutGetRoundTrip -count=1 -v
```

Expected: PASS；日志可见 Put/Get 成功。Verify 阶段保留此输出作证据。用完临时密钥后在火山控制台作废。

- [x] **Step 4: Commit（仅测试文件，无密钥）**

```bash
git add internal/platform/blob/tos/tos_live_test.go
git commit -m "$(cat <<'EOF'
test(blob/tos): add real TOS Put-Get gate from env

EOF
)"
```

---

### Task 4: 工厂 + edge-agent `BLOB_DRIVER` 装配

**Files:**
- Create: `internal/platform/blob/factory/factory.go`
- Create: `internal/platform/blob/factory/factory_test.go`（可选但推荐：非法 driver / 校验信息）
- Modify: `apps/edge-agent/cmd/edge-agent/main.go`

**Interfaces:**
- Consumes: `botconfig.BlobDriver*`、`localfs.New`、`s3.New`、`tos.New`、环境变量 `S3_*` / `TOS_*` / `BLOB_DRIVER`
- Produces:

```go
package factory

// NewFromEnv builds a blob.Store from BLOB_DRIVER (default "s3" for edge split compat).
func NewFromEnv(blobRoot string) (blob.Store, error)
```

或更显式：

```go
func New(driver string, localRoot string) (blob.Store, error)
```

仓库当前无完整 `apps/bot` main；本任务用工厂覆盖 tasks「控制面装配」意图，edge-agent 立刻接线。不为此期新建完整 bot 进程。

- [x] **Step 1: 写工厂失败测试 — 未知 driver**

```go
package factory_test

import (
	"strings"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob/factory"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/botconfig"
)

func TestNew_UnknownDriver(t *testing.T) {
	_, err := factory.New("nope", t.TempDir())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "driver") {
		t.Fatalf("got %v", err)
	}
}

func TestNew_LocalFS(t *testing.T) {
	store, err := factory.New(botconfig.BlobDriverLocalFS, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if store == nil {
		t.Fatal("nil store")
	}
}
```

- [x] **Step 2: 跑测试确认失败**

Run: `go test ./internal/platform/blob/factory/ -count=1`

Expected: 包不存在 / 编译失败。

- [x] **Step 3: 实现 factory**

```go
package factory

import (
	"fmt"
	"os"
	"strings"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob/localfs"
	blobs3 "github.com/mr9esx/comfyui_tgbot/internal/platform/blob/s3"
	blobtos "github.com/mr9esx/comfyui_tgbot/internal/platform/blob/tos"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/botconfig"
)

func New(driver, localRoot string) (blob.Store, error) {
	switch strings.TrimSpace(driver) {
	case "", botconfig.BlobDriverLocalFS:
		return localfs.New(localRoot)
	case botconfig.BlobDriverS3:
		return blobs3.New(blobs3.Options{
			Endpoint:        os.Getenv("S3_ENDPOINT"),
			Region:          envOr("S3_REGION", "us-east-1"),
			Bucket:          envOr("S3_BUCKET", "pixoma"),
			AccessKeyID:     os.Getenv("S3_ACCESS_KEY"),
			SecretAccessKey: os.Getenv("S3_SECRET_KEY"),
			UsePathStyle:    envBool("S3_PATH_STYLE", true),
		})
	case botconfig.BlobDriverTOS:
		return blobtos.New(blobtos.Options{
			Endpoint:        os.Getenv("TOS_ENDPOINT"),
			Region:          os.Getenv("TOS_REGION"),
			Bucket:          os.Getenv("TOS_BUCKET"),
			AccessKeyID:     os.Getenv("TOS_ACCESS_KEY"),
			SecretAccessKey: os.Getenv("TOS_SECRET_KEY"),
		})
	default:
		return nil, fmt.Errorf("blob/factory: unknown driver %q", driver)
	}
}

func envOr(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func envBool(k string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}
```

- [x] **Step 4: 改 edge-agent — 按 BLOB_DRIVER 选型（默认 s3）**

将 `main.go` 中硬编码 `s3.New(...)` 替换为：

```go
import (
	// remove direct blobs3 if unused
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob/factory"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/botconfig"
)

// inside run():
driver := envOr("BLOB_DRIVER", botconfig.BlobDriverS3)
blobStore, err := factory.New(driver, "") // localRoot 仅 localfs 需要；edge 默认 s3/tos
if err != nil {
	return err
}
```

若 `factory.New` 在 localfs 且 `localRoot==""` 会失败——edge 默认 s3，可接受；或对 non-localfs 忽略 root。保持默认 `BLOB_DRIVER` 未设时行为与现网一致（s3 + `S3_*`）。

日志可增加：`slog.Info("edge-agent running", ..., "blob_driver", driver)`。

- [x] **Step 5: 校验非法组合可诊断（配置层）**

复用 Task 1 测试即可；额外用一小段确认错误文案：

```bash
go test ./internal/platform/botconfig/ -run 'SplitRejectsLocalFS|SplitTOSOK' -count=1 -v
```

Expected: localfs 错误含 `blob.driver`；tos+redis 通过。

- [x] **Step 6: 跑相关包测试**

```bash
go test ./internal/platform/blob/factory/ ./internal/platform/botconfig/ ./internal/platform/blob/s3/ ./internal/platform/blob/localfs/ -count=1
```

Expected: PASS。

- [x] **Step 7: Commit**

```bash
git add internal/platform/blob/factory/ apps/edge-agent/cmd/edge-agent/main.go
git commit -m "$(cat <<'EOF'
feat(edge-agent): select blob store via BLOB_DRIVER including tos

EOF
)"
```

---

### Task 5: 架构文档与回归

**Files:**
- Modify: `docs/architecture/overview.md`（Blob：localfs | S3 | TOS；split 描述）
- Modify: `docs/architecture/runtime.md`（双模式表 Blob 列）
- Modify: `docs/architecture/bounded-contexts.md`（platform 表含 `blob/tos`）
- Modify（若文案写死仅 S3）: `docs/architecture/task-data-walkthrough.md`、`docs/architecture/diagrams/system.html`
- 视需要：根 `README.md` 架构入口无需改除非新增专章

**Interfaces:**
- Consumes: 本 change 已落地的驱动矩阵
- Produces: 文档与实现一致的「blob ∈ {localfs, s3, tos}；split = redis + s3|tos」

- [x] **Step 1: 更新 overview / runtime 驱动列表**

将「S3」扩展为「S3 或 TOS」；例如 overview：

- 部署形态：`split`（Redis Streams + S3/TOS）
- 图注 / 表格：`localfs | S3 | TOS`
- 外部系统行：Redis Streams / S3 或 TOS

`runtime.md` 双模式表：

| 模式 | Blob |
|---|---|
| allinone | localfs |
| split | S3 兼容 **或** TOS |

- [x] **Step 2: 更新 bounded-contexts platform 表**

```text
| `blob` + `blob/localfs` + `blob/s3` + `blob/tos` | 文件端口 |
```

- [x] **Step 3: 对照 task-data-walkthrough / system.html**

若仍写「仅 S3」，改为「S3 或 TOS（同 key 空间）」；diagram 文案 `localfs | S3 | TOS`。

- [x] **Step 4: 回归命令**

```bash
# 单元 + 配置
go test ./internal/platform/botconfig/ ./internal/platform/blob/... -count=1

# 真 TOS 门禁（需先 source）
set -a; source .env.tos.local; set +a
go test ./internal/platform/blob/tos/ -tags=live_tos -run TestRealTOS_PutGetRoundTrip -count=1 -v

# 既有 s3 / localfs
go test ./internal/platform/blob/s3/ ./internal/platform/blob/localfs/ -count=1
```

Expected: 全部 PASS；缺 `TOS_*` 时真网测试 FAIL（符合硬门禁）。

- [x] **Step 5: Commit**

```bash
git add docs/architecture/
git commit -m "$(cat <<'EOF'
docs(architecture): list TOS as split blob driver

EOF
)"
```

---

## Spec 覆盖自检

| 需求 / 场景 | 任务 |
|---|---|
| `blob.driver=tos` 实现 `blob.Store`，逻辑 key + BlobRef | Task 2–3 |
| split 控制面 Put / Edge Get 同 key（TOS） | Task 3–4（适配器 + 装配） |
| Edge Put outputs / 云可读 | 同端口 Put/Get；装配后主路径复用 |
| Blob 三驱动：localfs / s3 / tos | Task 2 + factory |
| split 允许 tos+redis、s3+redis；拒 localfs | Task 1 |
| 真桶 Put→Get；缺配置失败 | Task 3 |
| 架构文档驱动列表 | Task 5 |
| 无 AK/SK 入库 | Global Constraints + Task 3 加载方式 |

## 占位符扫描

无 TBD /「稍后实现」；密钥仅以环境变量名出现。
