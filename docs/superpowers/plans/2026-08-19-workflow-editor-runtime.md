# 工作流编辑器后端 + 运行时 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 后端在保存时强校验 workflow 图结构与绑定引用；运行时把输出绑定真正接上：ComfyUI 结果按节点解析，Worker 按绑定提取产物，通知按顺序发送全部输出。

**Architecture:** 三个独立可测的层次：校验（`catalog/infrastructure/validation`）、ComfyUI 结果解析（`runtime/infrastructure/comfyui`，新增纯函数 `parseHistory`）、执行（`runtime/infrastructure/actuator`：JobPackage 携带绑定 → Worker 按绑定提取 → Blob 落盘）。TG 通知层最后收口。

**Tech Stack:** Go（module `github.com/mr9esx/comfyui_tgbot`，GOTOOLCHAIN=go1.25.0）、标准库测试 + `go test ./...`、gin/chi（httpapi）、blob 本地存储。

## Global Constraints

- 测试命令：仓库根目录 `go test ./...`（或 `GOTOOLCHAIN=go1.25.0 go test ./...`）；构建 `go build ./...`。
- 无绑定（旧数据）时 Worker 回退“收集全部图片”，不得破坏现有任务流。
- 输出绑定缺失节点 / index 越界 → 任务失败，错误信息包含输出字段 key。
- `JobPackage` 必须自包含（执行面不读业务库）；输出绑定随 job 包下发。
- UI 文案（bot 通知里的 “Case”）保持与前端一致改为「工作流」。
- 不改 `CaseDocument` / `OutputBinding` 现有 JSON 字段结构（`outputs` 数组的 `index` 语义不变）。

---

### Task 2.1: ValidateDocument 图结构与绑定引用校验

**Files:**
- Modify: `internal/catalog/infrastructure/validation/validate.go`
- Modify: `internal/catalog/infrastructure/validation/validate_test.go`

**Interfaces:**
- Consumes: `domain.CaseDocument`（`Bindings.WorkflowJSON`、`Bindings.Inputs`、`Bindings.Outputs`）
- Produces: `ValidateDocument` 新增检查（不改签名）：workflow 非空且每个节点 `class_type` 为非空字符串、`inputs` 若存在必须为对象；输入绑定 `node_id` 必须存在于图中且 `field_path` 必须是该节点 `inputs` 的 key；输出绑定 `node_id` 必须存在且 `index >= 0`。错误通过 `domain.FieldError` 报告（key 如 `bindings.inputs[0].node_id`）。

- [ ] **Step 1: 更新现有 fixtures 并写失败测试**

现有 `text2imgDoc` / `mixedEditDoc` 的 `WorkflowJSON` 是 `map[string]any{"1": map[string]any{}}`，新校验会拒绝（缺 `class_type`）。先修正 fixtures，再追加新测试：

在 `validate_test.go` 的 `text2imgDoc` 中把：

```go
WorkflowJSON: map[string]any{"1": map[string]any{}},
```

替换为：

```go
WorkflowJSON: map[string]any{
	"1": map[string]any{"class_type": "CLIPTextEncode", "inputs": map[string]any{"text": "x"}},
	"2": map[string]any{"class_type": "SaveImage", "inputs": map[string]any{"filename_prefix": "o"}},
},
```

`mixedEditDoc` 同理：

```go
WorkflowJSON: map[string]any{
	"10": map[string]any{"class_type": "LoadImage", "inputs": map[string]any{"image": "ref.png"}},
	"20": map[string]any{"class_type": "CLIPTextEncode", "inputs": map[string]any{"text": "x"}},
	"60": map[string]any{"class_type": "SaveImage", "inputs": map[string]any{"filename_prefix": "o"}},
},
```

追加测试：

```go
func TestValidateDocumentRejectsGraphWithoutClassType(t *testing.T) {
	v := validation.New()
	doc := text2imgDoc()
	doc.Bindings.WorkflowJSON = map[string]any{"1": map[string]any{"inputs": map[string]any{"text": "x"}}}
	if err := v.ValidateDocument(doc); err == nil {
		t.Fatal("expected graph structure error")
	}
}

func TestValidateDocumentRejectsMissingBindingNode(t *testing.T) {
	v := validation.New()
	doc := text2imgDoc()
	doc.Bindings.Inputs[0].NodeID = "99"
	if err := v.ValidateDocument(doc); err == nil {
		t.Fatal("expected missing node error")
	}
}

func TestValidateDocumentRejectsMissingFieldPath(t *testing.T) {
	v := validation.New()
	doc := text2imgDoc()
	doc.Bindings.Inputs[0].FieldPath = "not_a_field"
	if err := v.ValidateDocument(doc); err == nil {
		t.Fatal("expected missing field path error")
	}
}

func TestValidateDocumentRejectsNegativeOutputIndex(t *testing.T) {
	v := validation.New()
	doc := text2imgDoc()
	doc.Bindings.Outputs[0].Index = -1
	if err := v.ValidateDocument(doc); err == nil {
		t.Fatal("expected negative index error")
	}
}

func TestValidateDocumentAcceptsValidGraphAndBindings(t *testing.T) {
	v := validation.New()
	if err := v.ValidateDocument(text2imgDoc()); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./internal/catalog/...`
Expected: FAIL（新测试因缺少校验逻辑而通过不了；`TestValidateDocumentAcceptsValidGraphAndBindings` 已过但结构测试失败）。

- [ ] **Step 3: 实现校验**

在 `validate.go` 的 `ValidateDocument` 中，`InputSchema` 检查之后追加：

```go
if err := validateWorkflowGraph(doc.Bindings.WorkflowJSON); err != nil {
	fields = append(fields, *err)
}
if err := validateInputBindings(doc.Bindings.WorkflowJSON, doc.Bindings.Inputs); err != nil {
	fields = append(fields, err...)
}
if err := validateOutputBindings(doc.Bindings.WorkflowJSON, doc.Bindings.Outputs); err != nil {
	fields = append(fields, err...)
}
```

新增函数（放在 `ValidateDocument` 之后）：

```go
func workflowNodes(g map[string]any) (map[string]map[string]any, *domain.FieldError) {
	nodes := make(map[string]map[string]any, len(g))
	for id, raw := range g {
		node, ok := raw.(map[string]any)
		if !ok {
			return nil, &domain.FieldError{Key: "bindings.workflow", Message: "node " + id + " must be an object"}
		}
		classType, _ := node["class_type"].(string)
		if classType == "" {
			return nil, &domain.FieldError{Key: "bindings.workflow", Message: "node " + id + " missing class_type"}
		}
		if inputs, exists := node["inputs"]; exists {
			if _, ok := inputs.(map[string]any); !ok {
				return nil, &domain.FieldError{Key: "bindings.workflow", Message: "node " + id + " inputs must be an object"}
			}
		}
		nodes[id] = node
	}
	return nodes, nil
}

func validateWorkflowGraph(g map[string]any) *domain.FieldError {
	if len(g) == 0 {
		return &domain.FieldError{Key: "bindings.workflow", Message: "required"}
	}
	_, err := workflowNodes(g)
	return err
}

func validateInputBindings(g map[string]any, bindings []domain.InputBinding) []domain.FieldError {
	nodes, graphErr := workflowNodes(g)
	if graphErr != nil {
		return nil
	}
	var out []domain.FieldError
	for i, b := range bindings {
		node, ok := nodes[b.NodeID]
		if !ok {
			out = append(out, domain.FieldError{Key: fmt.Sprintf("bindings.inputs[%d].node_id", i), Message: "node not found"})
			continue
		}
		inputs, _ := node["inputs"].(map[string]any)
		if _, ok := inputs[b.FieldPath]; !ok {
			out = append(out, domain.FieldError{Key: fmt.Sprintf("bindings.inputs[%d].field_path", i), Message: "field not found"})
		}
	}
	return out
}

func validateOutputBindings(g map[string]any, bindings []domain.OutputBinding) []domain.FieldError {
	nodes, graphErr := workflowNodes(g)
	if graphErr != nil {
		return nil
	}
	var out []domain.FieldError
	for i, b := range bindings {
		if _, ok := nodes[b.NodeID]; !ok {
			out = append(out, domain.FieldError{Key: fmt.Sprintf("bindings.outputs[%d].node_id", i), Message: "node not found"})
			continue
		}
		if b.Index < 0 {
			out = append(out, domain.FieldError{Key: fmt.Sprintf("bindings.outputs[%d].index", i), Message: "must be >= 0"})
		}
	}
	return out
}
```

`validate.go` 现有 import 已含 `fmt`（第 5 行），无需新增。

- [ ] **Step 4: 运行确认通过**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./internal/catalog/...`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/catalog/infrastructure/validation/validate.go internal/catalog/infrastructure/validation/validate_test.go
git commit -m "feat(catalog): validate workflow graph and binding references"
```

---

### Task 2.2: ComfyUI history 解析纯函数（按节点，含 images / text）

**Files:**
- Create: `internal/runtime/infrastructure/comfyui/history.go`
- Create: `internal/runtime/infrastructure/comfyui/history_test.go`

**Interfaces:**
- Consumes: 无
- Produces:
  - 类型：`type NodeImage struct { OutputFile; Subfolder string; Type string }`、`type NodeOutput struct { Images []NodeImage; Texts []string }`
  - `type HistoryResult = map[string]NodeOutput`（key 为 ComfyUI 节点 ID 的字符串）
  - `func ParseHistory(raw []byte) (HistoryResult, error)`：解析 `GET /history/{prompt_id}` 响应体；节点缺失 / 无产出时跳过该节点；空结果返回 `map[string]NodeOutput{}`（不报错）

- [ ] **Step 1: 写失败测试**

创建 `history_test.go`：

```go
package comfyui_test

import (
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
)

func TestParseHistoryGroupsByNode(t *testing.T) {
	raw := []byte(`{
		"prompt-id-1": {
			"outputs": {
				"4": { "images": [{ "filename": "a.png", "subfolder": "", "type": "output" }] },
				"9": { "text": ["hello", "world"] }
			},
			"status": { "status_str": "success", "completed": true }
		}
	}`)
	got, err := comfyui.ParseHistory(raw)
	if err != nil {
		t.Fatal(err)
	}
	node4, ok := got["4"]
	if !ok || len(node4.Images) != 1 || node4.Images[0].Filename != "a.png" {
		t.Fatalf("node 4 images: %+v", got["4"])
	}
	node9, ok := got["9"]
	if !ok || len(node9.Texts) != 2 || node9.Texts[0] != "hello" {
		t.Fatalf("node 9 texts: %+v", got["9"])
	}
}

func TestParseHistorySkipsEmptyNodes(t *testing.T) {
	raw := []byte(`{
		"prompt-id-1": {
			"outputs": { "7": { "images": [] } },
			"status": { "status_str": "success", "completed": true }
		}
	}`)
	got, err := comfyui.ParseHistory(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("want empty result, got %+v", got)
	}
}

func TestParseHistoryRejectsBadJSON(t *testing.T) {
	if _, err := comfyui.ParseHistory([]byte("not json")); err == nil {
		t.Fatal("expected error")
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./internal/runtime/infrastructure/comfyui/`
Expected: FAIL（`ParseHistory` undefined）。

- [ ] **Step 3: 实现 history.go**

```go
package comfyui

import (
	"encoding/json"
	"fmt"
)

// NodeOutput groups everything ComfyUI recorded for one node in a prompt history entry.
type NodeOutput struct {
	Images []NodeImage
	Texts  []string
}

type NodeImage struct {
	OutputFile
	Subfolder string
	Type      string
}

// HistoryResult maps node IDs to their outputs for one prompt.
type HistoryResult map[string]NodeOutput

type historyEntry struct {
	Outputs map[string]struct {
		Images []struct {
			Filename  string `json:"filename"`
			Subfolder string `json:"subfolder"`
			Type      string `json:"type"`
		} `json:"images"`
		Text []string `json:"text"`
	} `json:"outputs"`
	Status struct {
		StatusStr string `json:"status_str"`
		Completed bool   `json:"completed"`
	} `json:"status"`
}

// ParseHistory parses a GET /history/{prompt_id} response body.
func ParseHistory(raw []byte) (HistoryResult, error) {
	var hist map[string]historyEntry
	if err := json.Unmarshal(raw, &hist); err != nil {
		return nil, fmt.Errorf("comfyui history decode: %w", err)
	}
	out := HistoryResult{}
	for _, entry := range hist {
		for nodeID, node := range entry.Outputs {
			var images []NodeImage
			for _, img := range node.Images {
				images = append(images, NodeImage{
					OutputFile: OutputFile{Filename: img.Filename},
					Subfolder:  img.Subfolder,
					Type:       img.Type,
				})
			}
			if len(images) == 0 && len(node.Text) == 0 {
				continue
			}
			out[nodeID] = NodeOutput{Images: images, Texts: append([]string(nil), node.Text...)}
		}
	}
	return out, nil
}
```

- [ ] **Step 4: 运行确认通过**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./internal/runtime/infrastructure/comfyui/`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/runtime/infrastructure/comfyui/history.go internal/runtime/infrastructure/comfyui/history_test.go
git commit -m "feat(comfyui): parse history outputs per node"
```

---

### Task 2.3: Result 结构升级 + Mock 适配

**Files:**
- Modify: `internal/runtime/infrastructure/comfyui/client.go`
- Modify: `internal/runtime/infrastructure/comfyui/http.go`（Wait 使用 `ParseHistory`）
- Modify: `internal/runtime/infrastructure/comfyui/client_select_test.go`

**Interfaces:**
- Consumes: `history.go`（`ParseHistory` / `HistoryResult` / `NodeOutput`）
- Produces:
  - `type Result struct { PromptID string; Outputs HistoryResult }`
  - `func (h *HTTP) Wait(ctx, promptID) (*Result, error)`：调用 `fetchHistory`（现有实现）后 `ParseHistory`，随后按 `OutputFile` 逐张 `fetchView` 填充 MIME/Data；无图片且未 completed 时继续轮询；completed 但完全无输出时报错（保持现状语义）。
  - `Mock.Wait` 返回 `Outputs: HistoryResult{"1": {Images: [...]}}`。

- [ ] **Step 1: 写失败测试（client_select_test 更新）**

把 `client_select_test.go` 的 `TestNewClient_MockDefaultPath` 断言改为：

```go
res, err := c.Wait(context.Background(), id)
if err != nil {
	t.Fatal(err)
}
var found []byte
for _, node := range res.Outputs {
	for _, img := range node.Images {
		found = append(found, img.Data...)
	}
}
if len(found) == 0 {
	t.Fatal("mock must return image bytes")
}
```

- [ ] **Step 2: 运行确认失败**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./internal/runtime/infrastructure/comfyui/`
Expected: FAIL（`res.Outputs[0]` 不再编译）。

- [ ] **Step 3: 升级结构并接线**

`client.go` 中：

```go
type Result struct {
	PromptID string
	Outputs  HistoryResult
}
```

`Mock.Wait` 返回改为：

```go
return &Result{
	PromptID: promptID,
	Outputs: HistoryResult{
		"1": {
			Images: []NodeImage{{
				OutputFile: OutputFile{
					Filename: "out.png",
					Mime:     "image/png",
					Data:     GenerateMockPNG(320, 240, c),
				},
			}},
		},
	},
}, nil
```

`http.go` 的 `Wait` 轮询逻辑中，把现有 history 解码 + `fetchView` 循环替换为：

```go
hist, err := ParseHistory(raw)
if err != nil {
	return nil, false, err
}
var nodeOutputs HistoryResult
for nodeID, node := range hist {
	out := NodeOutput{}
	for _, img := range node.Images {
		if img.Filename == "" {
			continue
		}
		data, mime, err := h.fetchView(ctx, img.Filename, img.Subfolder, img.Type)
		if err != nil {
			return nil, false, err
		}
		out.Images = append(out.Images, NodeImage{
			OutputFile: OutputFile{Filename: img.Filename, Mime: mime, Data: data},
			Subfolder:  img.Subfolder,
			Type:       img.Type,
		})
	}
	out.Texts = append(out.Texts, node.Texts...)
	if len(out.Images) > 0 || len(out.Texts) > 0 {
		nodeOutputs[nodeID] = out
	}
}
if len(nodeOutputs) == 0 && !entry.Status.Completed {
	return nil, false, nil
}
if len(nodeOutputs) == 0 {
	return nil, false, fmt.Errorf("comfyui wait: completed without outputs")
}
return &Result{PromptID: promptID, Outputs: nodeOutputs}, true, nil
```

`NodeImage` / `NodeOutput` 已在 Task 2.2 的 `history.go` 定义（`NodeImage` 含 `Subfolder` / `Type`），此处直接使用。

同时最小适配 `worker.go`（保持仓库可编译、行为回退为「收集全部图片」）：

```go
var outs []sharedkernel.BlobRef
var nodeIDs []string
for nodeID := range res.Outputs {
	nodeIDs = append(nodeIDs, nodeID)
}
sort.Strings(nodeIDs)
i := 0
for _, nodeID := range nodeIDs {
	for _, img := range res.Outputs[nodeID].Images {
		key := fmt.Sprintf("outputs/%s/%d_%s", ev.TaskID, i, img.Filename)
		ref, err := w.Blob.Put(ctx, key, bytes.NewReader(img.Data), blob.PutOptions{MIME: img.Mime})
		if err != nil {
			return w.fail(ctx, ev, "blob_put", err.Error(), w.now())
		}
		outs = append(outs, ref)
		i++
	}
}
```

`worker.go` 顶部 import 增加 `sort`。

- [ ] **Step 4: 运行确认通过**

Run: `cd /Users/mr9esx/Documents/Pixoma && go build ./... && go test ./internal/runtime/...`
Expected: 全部 PASS（Task 2.3 提交后仓库保持可编译、可测试）。

- [ ] **Step 5: 提交**

```bash
git add internal/runtime/infrastructure/comfyui internal/runtime/infrastructure/actuator/worker.go
git commit -m "refactor(comfyui): per-node history result"
```

---

### Task 2.4: JobPackage 携带输出绑定

**Files:**
- Modify: `internal/runtime/infrastructure/actuator/job.go`
- Modify: `internal/runtime/infrastructure/actuator/snapshot.go`
- Modify: `internal/runtime/infrastructure/actuator/job_test.go`
- Modify: `internal/runtime/infrastructure/actuator/snapshot_test.go`

**Interfaces:**
- Consumes: `catalogdomain.OutputBinding`（`internal/catalog/domain/document.go`）
- Produces:
  - `JobPackage` 新增字段 `Outputs []catalogdomain.OutputBinding \`json:"outputs,omitempty"\``
  - `BuildJobPackage` 设置 `Outputs: c.Document.Bindings.Outputs`

- [ ] **Step 1: 更新测试（失败先行）**

`job_test.go` 的 `TestJobPackage_RoundTrip` 追加：

```go
Outputs: []catalogdomain.OutputBinding{
	{Key: "image", NodeID: "9", Index: 0},
},
```

并断言：

```go
if len(got.Outputs) != 1 || got.Outputs[0].Key != "image" || got.Outputs[0].NodeID != "9" {
	t.Fatalf("outputs=%+v", got.Outputs)
}
```

同时 `job_test.go` 顶部加 import `catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"`。

在 `snapshot_test.go` 中，`textWorkflowCase` 的 `Bindings` 追加输出绑定：

```go
Bindings: catalogdomain.ComfyBindings{
	WorkflowJSON: map[string]any{
		"20": map[string]any{
			"class_type": "CLIPTextEncode",
			"inputs": map[string]any{
				"text": "placeholder",
				"clip": []any{"19", 0},
			},
		},
	},
	Inputs: []catalogdomain.InputBinding{
		{Key: "prompt", NodeID: "20", FieldPath: "text"},
	},
	Outputs: []catalogdomain.OutputBinding{{Key: "image", NodeID: "9"}},
},
```

（原有 `textWorkflowCase` 已有前四行；只把 `Outputs` 追加进 `Bindings`。）

追加完整测试（复用现有 `memCases` / `recordingUploader` / localfs 模式）：

```go
func TestBuildJobPackageCarriesOutputBindings(t *testing.T) {
	ctx := context.Background()
	store, err := localfs.New(filepath.Join(t.TempDir(), "blob"))
	if err != nil {
		t.Fatal(err)
	}
	prefix := "inputs/task-out"
	if _, err := store.Put(ctx, prefix+"/prompt.txt", bytes.NewReader([]byte("out prompt")), blob.PutOptions{MIME: "text/plain"}); err != nil {
		t.Fatal(err)
	}
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(1, 0).UTC()
	if err := tasks.Create(ctx, runtimedomain.NewPending("task-out", "s1", "text-inject", prefix, now)); err != nil {
		t.Fatal(err)
	}
	cases := &memCases{}
	if err := cases.Create(ctx, &catalogdomain.Case{Document: textWorkflowCase(), Enabled: true}); err != nil {
		t.Fatal(err)
	}
	snap := &actuator.CaseSnapshot{Tasks: tasks, Cases: cases, Blob: store}
	job, err := snap.BuildJobPackage(ctx, "task-out", "local")
	if err != nil {
		t.Fatal(err)
	}
	if len(job.Outputs) != 1 || job.Outputs[0].Key != "image" || job.Outputs[0].NodeID != "9" {
		t.Fatalf("outputs=%+v", job.Outputs)
	}
}
```

`textWorkflowCase` 完整函数如下（在现有定义上追加 `Outputs` 一行）：

```go
func textWorkflowCase() catalogdomain.CaseDocument {
	return catalogdomain.CaseDocument{
		ID:   "text-inject",
		Name: "Text inject",
		Inputs: []catalogdomain.InputField{
			{Key: "prompt", Type: "string", Required: true},
		},
		Outputs: []catalogdomain.OutputField{{Key: "image", Type: "image"}},
		Bindings: catalogdomain.ComfyBindings{
			WorkflowJSON: map[string]any{
				"20": map[string]any{
					"class_type": "CLIPTextEncode",
					"inputs": map[string]any{
						"text": "placeholder",
						"clip": []any{"19", 0},
					},
				},
			},
			Inputs: []catalogdomain.InputBinding{
				{Key: "prompt", NodeID: "20", FieldPath: "text"},
			},
			Outputs: []catalogdomain.OutputBinding{{Key: "image", NodeID: "9"}},
		},
		InputSchema: map[string]any{"type": "object"},
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./internal/runtime/infrastructure/actuator/`
Expected: 编译失败（`JobPackage` 无 `Outputs` 字段）。

- [ ] **Step 3: 实现**

`job.go`：

```go
import (
	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type JobPackage struct {
	TaskID       sharedkernel.TaskID `json:"task_id"`
	EdgeID       sharedkernel.EdgeID `json:"edge_id"`
	Workflow     comfyui.Graph       `json:"workflow"`
	Images       []JobImage          `json:"images,omitempty"`
	OutputPrefix string              `json:"output_prefix"`
	Outputs      []catalogdomain.OutputBinding `json:"outputs,omitempty"`
}
```

`snapshot.go` 的 `BuildJobPackage` 返回值中：

```go
return JobPackage{
	TaskID:       taskID,
	EdgeID:       edgeID,
	Workflow:     graph,
	Images:       images,
	OutputPrefix: fmt.Sprintf("outputs/%s", taskID),
	Outputs:      append([]catalogdomain.OutputBinding(nil), c.Document.Bindings.Outputs...),
}, nil
```

- [ ] **Step 4: 运行确认通过**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./internal/runtime/infrastructure/actuator/`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/runtime/infrastructure/actuator/job.go internal/runtime/infrastructure/actuator/snapshot.go internal/runtime/infrastructure/actuator/job_test.go internal/runtime/infrastructure/actuator/snapshot_test.go
git commit -m "feat(actuator): carry output bindings in job package"
```

---

### Task 2.5: Worker 按输出绑定提取产物（含 fallback）

**Files:**
- Modify: `internal/runtime/infrastructure/actuator/worker.go`
- Modify: `internal/runtime/infrastructure/actuator/worker_test.go`

**Interfaces:**
- Consumes: `comfyui.Result.Outputs HistoryResult`、`JobPackage.Outputs`、`catalogdomain.OutputBinding`
- Produces:
  - `func (w *Worker) resolveJob(ctx context.Context, ev sharedkernel.DispatchCommand, cli comfyui.Client) (comfyui.Graph, []catalogdomain.OutputBinding, error)`（替换 `resolveGraph`；job 路径返回 `JobPackage.Workflow` + `JobPackage.Outputs`，workflow-provider 路径返回 `(graph, nil, err)`）
  - `func (w *Worker) storeOutputs(ctx context.Context, taskID sharedkernel.TaskID, res *comfyui.Result, bindings []catalogdomain.OutputBinding) ([]sharedkernel.BlobRef, error)`
  - fallback：无绑定 → 按节点 ID 排序拍平全部图片；有绑定 → 按绑定顺序取 `res.Outputs[nodeID].Images[index]`，Blob key `outputs/<taskID>/<key>_<i>.<ext>`

- [ ] **Step 1: 写失败测试**

`worker_test.go` 追加（复用现有 `statusCap` / localfs store；`strings` 需加到 import）：

```go
func writeJob(t *testing.T, ctx context.Context, store blob.Store, job actuator.JobPackage) sharedkernel.BlobRef {
	t.Helper()
	raw, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := store.Put(ctx, "jobs/"+string(job.TaskID)+"/job.json", bytes.NewReader(raw), blob.PutOptions{MIME: "application/json"})
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func TestWorkerExtractsOutputsByBinding(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := localfs.New(filepath.Join(dir, "blob"))
	if err != nil {
		t.Fatal(err)
	}
	cap := &statusCap{}
	mock := &comfyui.Mock{
		WaitFn: func(_ context.Context, _ string) (*comfyui.Result, error) {
			return &comfyui.Result{
				PromptID: "p1",
				Outputs: comfyui.HistoryResult{
					"9": {
						Images: []comfyui.NodeImage{
							{OutputFile: comfyui.OutputFile{Filename: "out.png", Mime: "image/png", Data: []byte("png")}},
						},
					},
				},
			}, nil
		},
	}
	w := &actuator.Worker{
		EdgeID:    "local",
		Comfy:     mock,
		Blob:      store,
		Status:    cap,
		Workflows: actuator.StaticWorkflows{},
		Now:       func() time.Time { return time.Unix(1, 0).UTC() },
	}
	jobRef := writeJob(t, ctx, store, actuator.JobPackage{
		TaskID: "t1",
		EdgeID: "local",
		Workflow: comfyui.Graph{"1": map[string]any{"class_type": "SaveImage", "inputs": map[string]any{}}},
		Outputs: []catalogdomain.OutputBinding{{Key: "image", NodeID: "9", Index: 0}},
	})
	err = w.HandleDispatch(ctx, sharedkernel.DispatchCommand{
		TaskID: "t1", EdgeID: "local", JobRef: jobRef,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(cap.msgs) != 2 {
		t.Fatalf("msgs=%d", len(cap.msgs))
	}
	var done sharedkernel.TaskStatusEvent
	_ = json.Unmarshal(cap.msgs[1].Payload, &done)
	if done.Status != sharedkernel.TaskSucceeded || len(done.Outputs) != 1 {
		t.Fatalf("done=%+v", done)
	}
	if !strings.HasPrefix(done.Outputs[0].Key, "outputs/t1/image_") {
		t.Fatalf("want keyed output, got %q", done.Outputs[0].Key)
	}
}

func TestWorkerFailsWhenBoundNodeMissing(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := localfs.New(filepath.Join(dir, "blob"))
	if err != nil {
		t.Fatal(err)
	}
	cap := &statusCap{}
	mock := &comfyui.Mock{
		WaitFn: func(_ context.Context, _ string) (*comfyui.Result, error) {
			return &comfyui.Result{PromptID: "p1", Outputs: comfyui.HistoryResult{}}, nil
		},
	}
	w := &actuator.Worker{
		EdgeID:    "local",
		Comfy:     mock,
		Blob:      store,
		Status:    cap,
		Workflows: actuator.StaticWorkflows{},
		Now:       func() time.Time { return time.Unix(1, 0).UTC() },
	}
	jobRef := writeJob(t, ctx, store, actuator.JobPackage{
		TaskID:   "t2",
		EdgeID:   "local",
		Workflow: comfyui.Graph{"1": map[string]any{"class_type": "SaveImage", "inputs": map[string]any{}}},
		Outputs:  []catalogdomain.OutputBinding{{Key: "image", NodeID: "9", Index: 0}},
	})
	if err := w.HandleDispatch(ctx, sharedkernel.DispatchCommand{
		TaskID: "t2", EdgeID: "local", JobRef: jobRef,
	}); err != nil {
		t.Fatal(err)
	}
	// HandleDispatch 对任务失败返回 nil，失败信息在 status 事件里
	if len(cap.msgs) != 2 {
		t.Fatalf("msgs=%d", len(cap.msgs))
	}
	var failed sharedkernel.TaskStatusEvent
	_ = json.Unmarshal(cap.msgs[1].Payload, &failed)
	if failed.Status != sharedkernel.TaskFailed || failed.ErrorCode != "output_extract" {
		t.Fatalf("failed=%+v", failed)
	}
}

func TestWorkerFallsBackToAllImagesWithoutBindings(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := localfs.New(filepath.Join(dir, "blob"))
	if err != nil {
		t.Fatal(err)
	}
	cap := &statusCap{}
	mock := &comfyui.Mock{
		WaitFn: func(_ context.Context, _ string) (*comfyui.Result, error) {
			return &comfyui.Result{
				PromptID: "p1",
				Outputs: comfyui.HistoryResult{
					"1": {Images: []comfyui.NodeImage{
						{OutputFile: comfyui.OutputFile{Filename: "a.png", Mime: "image/png", Data: []byte("a")}},
					}},
					"2": {Images: []comfyui.NodeImage{
						{OutputFile: comfyui.OutputFile{Filename: "b.png", Mime: "image/png", Data: []byte("b")}},
					}},
				},
			}, nil
		},
	}
	w := &actuator.Worker{
		EdgeID:    "local",
		Comfy:     mock,
		Blob:      store,
		Status:    cap,
		Workflows: actuator.StaticWorkflows{},
		Now:       func() time.Time { return time.Unix(1, 0).UTC() },
	}
	jobRef := writeJob(t, ctx, store, actuator.JobPackage{
		TaskID:   "t3",
		EdgeID:   "local",
		Workflow: comfyui.Graph{"1": map[string]any{"class_type": "SaveImage", "inputs": map[string]any{}}},
	})
	if err := w.HandleDispatch(ctx, sharedkernel.DispatchCommand{
		TaskID: "t3", EdgeID: "local", JobRef: jobRef,
	}); err != nil {
		t.Fatal(err)
	}
	var done sharedkernel.TaskStatusEvent
	_ = json.Unmarshal(cap.msgs[1].Payload, &done)
	if done.Status != sharedkernel.TaskSucceeded || len(done.Outputs) != 2 {
		t.Fatalf("done=%+v", done)
	}
	if done.Outputs[0].Key != "outputs/t3/0_a.png" || done.Outputs[1].Key != "outputs/t3/1_b.png" {
		t.Fatalf("keys=%q %q", done.Outputs[0].Key, done.Outputs[1].Key)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./internal/runtime/infrastructure/actuator/ -run TestWorker`
Expected: 编译失败 / FAIL。

- [ ] **Step 3: 实现**

`worker.go` 中：

```go
func (w *Worker) HandleDispatch(ctx context.Context, ev sharedkernel.DispatchCommand) error {
	now := w.now()
	cli, err := w.clientFor(ev.EdgeID)
	if err != nil {
		return w.fail(ctx, ev, "comfy_client", err.Error(), now)
	}
	graph, outputBindings, err := w.resolveJob(ctx, ev, cli)
	if err != nil {
		return w.fail(ctx, ev, "workflow", err.Error(), now)
	}
	promptID, err := cli.Submit(ctx, graph)
	if err != nil {
		return w.fail(ctx, ev, "comfy_submit", err.Error(), now)
	}
	if err := w.publishStatus(ctx, sharedkernel.TaskStatusEvent{
		TaskID: ev.TaskID, EdgeID: ev.EdgeID, Status: sharedkernel.TaskRunning,
		PromptID: promptID, At: w.now(),
	}); err != nil {
		return err
	}
	res, err := cli.Wait(ctx, promptID)
	if err != nil {
		return w.fail(ctx, ev, "comfy_wait", err.Error(), w.now())
	}
	outs, err := w.storeOutputs(ctx, ev.TaskID, res, outputBindings)
	if err != nil {
		return w.fail(ctx, ev, "output_extract", err.Error(), w.now())
	}
	return w.publishStatus(ctx, sharedkernel.TaskStatusEvent{
		TaskID: ev.TaskID, EdgeID: ev.EdgeID, Status: sharedkernel.TaskSucceeded,
		PromptID: promptID, Outputs: outs, At: w.now(),
	})
}

func (w *Worker) resolveJob(ctx context.Context, ev sharedkernel.DispatchCommand, cli comfyui.Client) (comfyui.Graph, []catalogdomain.OutputBinding, error) {
	if ev.JobRef.Key != "" {
		job, err := w.jobFromBlob(ctx, ev.JobRef, cli)
		if err != nil {
			return nil, nil, err
		}
		return job.Workflow, job.Outputs, nil
	}
	if w.Workflows == nil {
		return nil, nil, fmt.Errorf("actuator: missing job_ref and workflows provider")
	}
	graph, err := w.Workflows.WorkflowForTask(ctx, ev.TaskID, cli)
	return graph, nil, err
}

func (w *Worker) storeOutputs(ctx context.Context, taskID sharedkernel.TaskID, res *comfyui.Result, bindings []catalogdomain.OutputBinding) ([]sharedkernel.BlobRef, error) {
	if len(bindings) == 0 {
		var keys []string
		for nodeID := range res.Outputs {
			keys = append(keys, nodeID)
		}
		sort.Strings(keys)
		var outs []sharedkernel.BlobRef
		i := 0
		for _, nodeID := range keys {
			for _, img := range res.Outputs[nodeID].Images {
				key := fmt.Sprintf("outputs/%s/%d_%s", taskID, i, img.Filename)
				ref, err := w.Blob.Put(ctx, key, bytes.NewReader(img.Data), blob.PutOptions{MIME: img.Mime})
				if err != nil {
					return nil, err
				}
				outs = append(outs, ref)
				i++
			}
		}
		return outs, nil
	}

	var outs []sharedkernel.BlobRef
	for _, binding := range bindings {
		node, ok := res.Outputs[binding.NodeID]
		if !ok || len(node.Images) == 0 {
			return nil, fmt.Errorf("output binding %q: node %s produced no image outputs", binding.Key, binding.NodeID)
		}
		index := binding.Index
		if index < 0 {
			index = 0
		}
		if index >= len(node.Images) {
			return nil, fmt.Errorf("output binding %q: node %s index %d out of range", binding.Key, binding.NodeID, index)
		}
		img := node.Images[index]
		key := fmt.Sprintf("outputs/%s/%s_%d_%s", taskID, binding.Key, index, img.Filename)
		ref, err := w.Blob.Put(ctx, key, bytes.NewReader(img.Data), blob.PutOptions{MIME: img.Mime})
		if err != nil {
			return nil, err
		}
		outs = append(outs, ref)
	}
	return outs, nil
}
```

删除 `resolveGraph`，把 `graphFromJob` 改为 `jobFromBlob`（返回 `*JobPackage`）：

```go
func (w *Worker) jobFromBlob(ctx context.Context, ref sharedkernel.BlobRef, uploader ImageUploader) (*JobPackage, error) {
	if w.Blob == nil {
		return nil, fmt.Errorf("actuator: blob store not configured")
	}
	rc, err := w.Blob.Get(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("actuator: get job: %w", err)
	}
	defer rc.Close()
	raw, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("actuator: read job: %w", err)
	}
	var job JobPackage
	if err := json.Unmarshal(raw, &job); err != nil {
		return nil, fmt.Errorf("actuator: parse job: %w", err)
	}
	if len(job.Workflow) == 0 {
		return nil, fmt.Errorf("actuator: empty workflow in job")
	}
	for _, img := range job.Images {
		remote, err := w.uploadJobImage(ctx, uploader, img.Blob)
		if err != nil {
			return nil, err
		}
		if err := writeNodeInput(job.Workflow, img.NodeID, img.FieldPath, remote); err != nil {
			return nil, err
		}
	}
	return &job, nil
}
```

`WorkflowForTask` 接口不变。

import 新增：`sort`、`catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"`。

- [ ] **Step 4: 运行确认通过**

Run: `cd /Users/mr9esx/Documents/Pixoma && go build ./... && go test ./internal/runtime/... ./internal/channel/...`
Expected: PASS（channel 包若受影响仅编译层面，见 Task 2.6）。

- [ ] **Step 5: 提交**

```bash
git add internal/runtime/infrastructure/actuator/worker.go internal/runtime/infrastructure/actuator/worker_test.go
git commit -m "feat(actuator): extract outputs by binding"
```

---

### Task 2.6: TG 通知按顺序发送全部输出

**Files:**
- Modify: `internal/channel/tg/adapter.go`
- Modify: `internal/channel/tg/callback_test.go`（如涉及通知路径；无则不动）

**Interfaces:**
- Consumes: `sharedkernel.UserNotify.Outputs []BlobRef`（`Outputs` 保持全量、顺序 = 任务输出顺序）
- Produces: `HandleUserNotify` 不再只发 `Outputs[0]`：图片/视频逐张 `SendMedia`，`text/*` 输出通过 `a.Blob.Get` 读取后 `SendText`；第一条附 caption「✅ 工作流完成\ntask=…」，末尾 `SendMenu` 不变。

- [ ] **Step 1: 更新失败测试**

`callback_test.go` 或新建 `notify_test.go`（`package tg_test`）：

```go
type recordingOutbound struct {
	media []sharedkernel.BlobRef
	texts []string
}

func (r *recordingOutbound) SendText(_ context.Context, _ sharedkernel.ChannelAddr, text string) error {
	r.texts = append(r.texts, text)
	return nil
}
func (r *recordingOutbound) SendMenu(_ context.Context, _ sharedkernel.ChannelAddr, _ string, _ []ports.MenuEntry) error {
	return nil
}
func (r *recordingOutbound) SendList(_ context.Context, _ sharedkernel.ChannelAddr, _ string, _ [][]ports.Button) error {
	return nil
}
func (r *recordingOutbound) SendMedia(_ context.Context, _ sharedkernel.ChannelAddr, ref sharedkernel.BlobRef, caption string) error {
	r.media = append(r.media, ref)
	return nil
}
func (r *recordingOutbound) SendMediaURL(_ context.Context, _ sharedkernel.ChannelAddr, _, _ string) error {
	return nil
}

func TestHandleUserNotifySendsAllOutputs(t *testing.T) {
	out := &recordingOutbound{}
	a := tg.New(out)
	a.ChannelID = "tg"
	err := a.HandleUserNotify(context.Background(), sharedkernel.UserNotify{
		ChatID: sharedkernel.ChatID("tg:123"),
		TaskID: "t1",
		Kind:   "task_succeeded",
		Outputs: []sharedkernel.BlobRef{
			{Key: "outputs/t1/image_0_a.png", MIME: "image/png"},
			{Key: "outputs/t1/image_1_b.png", MIME: "image/png"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.media) != 2 {
		t.Fatalf("media=%d", len(out.media))
	}
}
```

测试文件 import 需含：`github.com/mr9esx/comfyui_tgbot/internal/channel/ports`、`github.com/mr9esx/comfyui_tgbot/internal/channel/tg`、`github.com/mr9esx/comfyui_tgbot/internal/sharedkernel`。`HandleUserNotify` 内部走 `addrOf`（`tg:123` → `ChannelAddr{ChannelID: "tg", ExternalChatID: "123"}`），`a.Blob` 为 nil 且输出全是图片，不走 Blob 读取分支。

- [ ] **Step 2: 运行确认失败**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./internal/channel/tg/`
Expected: FAIL（当前只发一张）。

- [ ] **Step 3: 实现**

`adapter.go` 的 `HandleUserNotify` 成功分支替换为：

```go
if n.Kind == "task_succeeded" && len(n.Outputs) > 0 {
	for i, ref := range n.Outputs {
		caption := ""
		if i == 0 {
			caption = fmt.Sprintf("✅ 工作流完成\ntask=%s", n.TaskID)
		}
		if strings.HasPrefix(ref.MIME, "text/") {
			if a.Blob == nil {
				continue
			}
			rc, err := a.Blob.Get(ctx, ref)
			if err != nil {
				return err
			}
			raw, readErr := io.ReadAll(rc)
			rc.Close()
			if readErr != nil {
				return readErr
			}
			if err := a.Out.SendText(ctx, addr, string(raw)); err != nil {
				return err
			}
			continue
		}
		if err := a.Out.SendMedia(ctx, addr, ref, caption); err != nil {
			return err
		}
	}
	return a.Out.SendMenu(ctx, addr, "还要继续？点菜单再选一个工作流。", nil)
}
```

`import` 增加 `"io"`（如已有 `bytes` 则复用其 io 语义，按实际缺失补）。

- [ ] **Step 4: 运行确认通过**

Run: `cd /Users/mr9esx/Documents/Pixoma && go build ./... && go test ./internal/channel/... ./internal/runtime/...`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/channel/tg
git commit -m "feat(tg): deliver all workflow outputs in order"
```

---

### Task 2.7: 全量验证

**Files:** 无新增。

- [ ] **Step 1: 全量检查**

```bash
cd /Users/mr9esx/Documents/Pixoma
GOTOOLCHAIN=go1.25.0 go test ./...
go build ./...
```
Expected: 全部通过。

- [ ] **Step 2: 提交遗留改动**

```bash
git add -A
git commit -m "chore: finalize workflow editor runtime"
```
