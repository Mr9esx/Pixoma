---
change: mcp-caller-skill
design-doc: docs/superpowers/specs/2026-09-12-mcp-caller-skill-design.md
base-ref: 99891849405c1e26d618ad03cbdb6d73b9a45c33
---

# MCP 调用方 Skill Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 调用方 Agent 失败时能从 HTTP 401/403 与 tool `isError` 正文拿到下一步；并提供可下载的 `pixoma-mcp` skill（错误表与接口句子同义）。

**Architecture:** 方案 A：只改 MCP 边界文案，不引入错误码、不改 `RunCase` 主链路。`RequireBearer` 外包一层改写 401/403 正文（SDK 对无 Bearer 会写 `no bearer token`，对会话错配会写 `session user mismatch`）；`Resolve` 区分「凭据无效」与「渠道/用户不可用」以便 401 多一句停用说明。`toolErr` 在写出 `IsError` 前用 `errors.Is` / 关键词换成固定指导句，内部 sentinel 保留。Skill 放在 `docs/skills/pixoma-mcp/`，不进 `.agents/skills`。

**Tech Stack:** Go 1.25、`github.com/modelcontextprotocol/go-sdk` v1.7.0、现有 `internal/mcp` 测试夹具、Markdown skill 包。

## Global Constraints

- 产物语言 zh-CN；方案 A：不引入 `code` 字段或错误码枚举。
- 保持现有 HTTP 状态码（无/错 Bearer、渠道停用 = 401；会话身份不匹配 = 403）。
- 文案面向 Agent：必须含下一步；MUST NOT 要求把 Bearer/token 发给模型或贴进对话。
- 句子固定，测试用子串：`连接器`、`不要`，且不得出现「把 token 发给」「贴 token」「Bearer」当作索要凭据。
- 不改工具集合、异步语义、`RunCase` 内部错误类型；不写架构文档。
- Skill `description` 只写 Use when，不写调用流程摘要。
- 提交作者必须是 `李卓洲 <1138099359@qq.com>`（`GIT_AUTHOR_*` / `GIT_COMMITTER_*`），禁止 Cursor 身份。

## File map

- Modify: `internal/mcp/auth.go` — 对外 401/403 固定句、`Resolve` 区分不可用、改写 SDK 默认正文
- Modify: `internal/mcp/sse_guard.go` — SSE 403 改用同一句（与改写层双保险）
- Modify: `internal/mcp/auth_test.go` — 401/403 状态不变，断言指导句
- Modify: `internal/mcp/tools.go` — `toolErr` 映射
- Modify: `internal/mcp/tools_test.go` — 付费、停用、缺字段、他人任务、空 `task_id` 的正文
- Create: `docs/skills/pixoma-mcp/SKILL.md` — 产品 skill
- Create: `internal/mcp/skill_doc_test.go` — skill + README 提示词合同
- Modify: `README.md` — 一句安装提示词 + 仓库相对路径

---

### Task 1: HTTP 401/403 指导正文

**Files:**
- Modify: `internal/mcp/auth.go`
- Modify: `internal/mcp/sse_guard.go:28`
- Test: `internal/mcp/auth_test.go`

**Interfaces:**
- Consumes: 现有 `Resolver.Resolve`、`RequireBearer`、`bindSSESessionUser`；SDK `auth.RequireBearerToken`（无 Bearer 时正文为 `no bearer token`；`/mcp` 会话错配正文为 `session user mismatch`）
- Produces: 包级固定句（测试与 Task 3 错误表必须同义，不要改用词）：

```go
const (
	GuideUnauthorized = "检查聊天的 Pixoma MCP 连接器是否指向本实例且凭据仍有效。不要向用户索要 token。配好后重试。"
	GuideUnavailable  = "检查聊天的 Pixoma MCP 连接器是否指向本实例且凭据仍有效。对应的 MCP 渠道或用户可能已停用。不要向用户索要 token。配好后重试。"
	GuideForbidden    = "连接器身份与会话不一致。让用户重连该连接器后重试。不要向用户索要 token。"
)
```

- `Resolve`：空/错 token 仍返回现有 `errUnauthorized`；用户不存在、渠道停用或 `Platform != mcp` 返回可 `errors.Is` 的 `errUnavailable`（新 sentinel，仅 MCP 包内）。
- Verifier 失败时：`fmt.Errorf("%s: %w", GuideUnauthorized或GuideUnavailable, auth.ErrInvalidToken)`，保证 SDK 仍给 401。
- `RequireBearer` 返回的 handler 最外层包 `rewriteMCPAuthBody`：若状态 401 且正文不含 `连接器`，改写成 `GuideUnauthorized`；若状态 403 且正文含 `session user mismatch`，改写成 `GuideForbidden`。不要改写其它 4xx。
- `sse_guard.go` 的 `http.Error` 第三参保持 `http.StatusForbidden`，正文改为 `GuideForbidden`。

- [ ] **Step 1: 写失败测试**

在 `internal/mcp/auth_test.go` 增加辅助函数与三个测试（复用 `TestMCP_RequiresBearer` 的夹具写法，不要改旧测试的状态断言；旧测试缺正文检查，本任务用新测试锁文案）。

```go
func mcpAuthBody(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	return rec.Body.String()
}

func assertAgentAuthGuide(t *testing.T, body string) {
	t.Helper()
	if !strings.Contains(body, "连接器") {
		t.Fatalf("body=%q missing 连接器", body)
	}
	if !strings.Contains(body, "不要") {
		t.Fatalf("body=%q missing 不要", body)
	}
	if strings.Contains(body, "把 token 发给") || strings.Contains(body, "贴 token") || strings.Contains(body, "Bearer") {
		t.Fatalf("body solicits credential: %q", body)
	}
}

func TestMCP_UnauthorizedBodyGuidesConnector(t *testing.T) {
	h := newAuthedMCPHandler(t) // 抽现有 TestMCP_RequiresBearer 的 handler 搭建；或内联同样 15 行

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader("{}")))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rec.Code)
	}
	assertAgentAuthGuide(t, mcpAuthBody(t, rec))

	bad := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader("{}"))
	bad.Header.Set("Authorization", "Bearer wrong")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, bad)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong bearer status=%d", rec.Code)
	}
	assertAgentAuthGuide(t, mcpAuthBody(t, rec))
}

func TestMCP_DisabledChannelBodyMentionsUnavailable(t *testing.T) {
	// 与 TestMCP_RequiresBearer 相同：有效 token 后把渠道 Enabled=false 再 POST /mcp
	// 断言 status 401，body 含 assertAgentAuthGuide，且含「停用」
}

func TestMCP_SessionMismatchBodyGuidesReconnect(t *testing.T) {
	// 复制 TestMCP_StreamableSessionRejectsOtherBearer 到拿到 strangerResp 之后：
	body, err := io.ReadAll(strangerResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if strangerResp.StatusCode != http.StatusForbidden {
		t.Fatalf("status=%d", strangerResp.StatusCode)
	}
	assertAgentAuthGuide(t, string(body))
	if !strings.Contains(string(body), "重连") && !strings.Contains(string(body), "不一致") {
		t.Fatalf("403 body=%q", body)
	}

	// 同样对 TestMCP_LegacySSESessionRejectsOtherBearer 的 strangerResp 读 body 做相同断言
	// 可写在本测试后半，或再开 TestMCP_SSESessionMismatchBodyGuidesReconnect
}
```

把 `TestMCP_RequiresBearer` 里「tokens / users / channels / handler」抽成 `newAuthedMCPHandler(t)`（返回 `http.Handler` 与 `plain` token、`channels`）避免复制三份；抽函数时不要改旧测试期望。

- [ ] **Step 2: 跑测试确认失败**

```bash
go test ./internal/mcp/ -count=1 -run 'TestMCP_UnauthorizedBodyGuidesConnector|TestMCP_DisabledChannelBodyMentionsUnavailable|TestMCP_SessionMismatchBodyGuidesReconnect'
```

Expected: FAIL——401 正文仍是 `no bearer token` / `invalid token` / `unauthorized`，403 正文仍是 `session user mismatch`，缺 `连接器`。

- [ ] **Step 3: 写最小实现**

`auth.go`：

```go
var errUnavailable = errors.New("mcp channel or user unavailable")

func (r *Resolver) Resolve(ctx context.Context, bearer string) (Identity, error) {
	// 空 token、hash/decrypt 失败 → errUnauthorized
	// user 取不到 → errUnavailable
	// 渠道 Get 失败 / !Enabled / Platform != mcp → errUnavailable
}

func (r *Resolver) Resolve(...) {
	// verifier:
	id, err := res.Resolve(ctx, token)
	if err != nil {
		msg := GuideUnauthorized
		if errors.Is(err, errUnavailable) {
			msg = GuideUnavailable
		}
		return nil, fmt.Errorf("%s: %w", msg, auth.ErrInvalidToken)
	}
}

func rewriteMCPAuthBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&authGuideWriter{ResponseWriter: w}, r)
	})
}

type authGuideWriter struct {
	http.ResponseWriter
	code int
}

func (w *authGuideWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *authGuideWriter) Write(p []byte) (int, error) {
	code := w.code
	if code == 0 {
		code = http.StatusOK
	}
	if code == http.StatusUnauthorized && !bytes.Contains(p, []byte("连接器")) {
		p = []byte(GuideUnauthorized + "\n")
	}
	if code == http.StatusForbidden && bytes.Contains(p, []byte("session user mismatch")) {
		p = []byte(GuideForbidden + "\n")
	}
	return w.ResponseWriter.Write(p)
}
```

`RequireBearer`：`return rewriteMCPAuthBody(mw(http.HandlerFunc(...)))`；内层 `http.Error` 的 `"unauthorized"` 改成 `GuideUnauthorized`。

`sse_guard.go`：`http.Error(w, GuideForbidden, http.StatusForbidden)`。

- [ ] **Step 4: 跑测试确认通过**

```bash
go test ./internal/mcp/ -count=1 -run 'TestMCP_'
```

Expected: PASS（含旧鉴权与会话测试）。

- [ ] **Step 5: Commit**

```bash
GIT_AUTHOR_NAME="李卓洲" GIT_AUTHOR_EMAIL="1138099359@qq.com" \
GIT_COMMITTER_NAME="李卓洲" GIT_COMMITTER_EMAIL="1138099359@qq.com" \
git add internal/mcp/auth.go internal/mcp/sse_guard.go internal/mcp/auth_test.go
git commit -m "$(cat <<'EOF'
fix(mcp): 401/403 正文改成连接器指导句

EOF
)"
```

---

### Task 2: toolErr 固定指导句

**Files:**
- Modify: `internal/mcp/tools.go`（`toolErr` 与新 `guideToolError`）
- Test: `internal/mcp/tools_test.go`

**Interfaces:**
- Consumes: `botapp.ErrAccessDenied`（`user access denied`）、`catalogdomain.ErrNotFound` / `ErrDisabled`、`runtimedomain.ErrTaskNotFound`、现有 `missing required: prompt`、`task_id required`
- Produces: `func guideToolError(err error) string`；`toolErr` 的 `TextContent.Text` 只用该函数结果。不增加 JSON `code`。

固定句（与 Task 3 表逐字对齐）：

```go
const (
	GuideToolAccessDenied = "该 MCP 用户当前不能跑工作流。请管理员在 MCP 渠道用户里改为允许后再试。"
	GuideToolWorkflowGone = "工作流不存在或已停用。先 list_workflows 换已启用的 id。"
	GuideToolMissingInput = "缺少必填：%s。先 get_workflow 再带齐 inputs 重试。" // %s 为缺的 key 列表
	GuideToolTaskHidden   = "只能查当前连接器用户的任务。核对 task_id 或先 list_tasks。"
)
```

映射：

| 条件 | 对外 |
|---|---|
| `errors.Is(err, botapp.ErrAccessDenied)` | `GuideToolAccessDenied` |
| `errors.Is(..., catalogdomain.ErrNotFound)` 或 `ErrDisabled`，或原文含 `workflow not found` / `case not found` | `GuideToolWorkflowGone` |
| 原文含 `missing required:` | `fmt.Sprintf(GuideToolMissingInput, 冒号后 key 列表)` |
| `errors.Is(..., runtimedomain.ErrTaskNotFound)` 或原文为 `task_id required` | `GuideToolTaskHidden` |
| 其它 | `err.Error()` 原样透传，不加前缀 |

- [ ] **Step 1: 写失败测试**

在 `tools_test.go` 增加：

```go
func toolErrorText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
			continue
		}
		b.WriteString(fmt.Sprint(c))
	}
	return b.String()
}

func TestMCP_RunWorkflow_PaidUserGuidesAdmin(t *testing.T) {
	// 与 TestMCP_RunWorkflow_PaidUserDoesNotCreateTask 相同前置
	// 断言 IsError，且 toolErrorText 含「不能跑工作流」与「MCP 渠道用户」
	// 断言不创建任务（ListByChat 长度为 0）
}

func TestMCP_RunWorkflow_DisabledGuidesList(t *testing.T) {
	facade, tasks, ident := mcpHarness(t)
	ctx := context.Background()
	c, err := facade.Cases.Get(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	c.Enabled = false
	if err := facade.Cases.Save(ctx, c); err != nil {
		t.Fatal(err)
	}
	cs, cleanup := connectMCP(t, pixmcp.Deps{Facade: facade, Identity: ident})
	defer cleanup()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "run_workflow",
		Arguments: map[string]any{
			"case_id": 1,
			"inputs":  map[string]any{"prompt": "a cat"},
		},
	})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error")
	}
	got := toolErrorText(t, res)
	if !strings.Contains(got, "list_workflows") || !strings.Contains(got, "停用") {
		t.Fatalf("guide=%q", got)
	}
	list, err := tasks.ListByChat(ctx, ident.ChatID(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("created tasks=%d", len(list))
	}
}

func TestMCP_GetWorkflow_MissingGuidesList(t *testing.T) {
	facade, _, ident := mcpHarness(t)
	cs, cleanup := connectMCP(t, pixmcp.Deps{Facade: facade, Identity: ident})
	defer cleanup()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_workflow",
		Arguments: map[string]any{"case_id": 999},
	})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error")
	}
	got := toolErrorText(t, res)
	if !strings.Contains(got, "list_workflows") {
		t.Fatalf("guide=%q", got)
	}
}

func TestMCP_RunWorkflow_MissingRequiredGuidesGetWorkflow(t *testing.T) {
	// 同 TestMCP_RunWorkflow_MissingRequired 调用
	// 断言正文含「prompt」、`get_workflow`、`inputs`
}

func TestMCP_GetTask_ForeignOrEmptyGuidesListTasks(t *testing.T) {
	// 1) 复用 TestMCP_ListTasks_OnlyOwnUser：stranger get_task 正文含「连接器」或「list_tasks」
	// 2) 本用户 get_task 且 task_id="" 或省略：正文同样含 list_tasks / task_id
}
```

可把付费用户旧测试扩成断言正文，但必须先让新断言失败再改 `toolErr`。不要删「不创建任务」断言。

- [ ] **Step 2: 跑测试确认失败**

```bash
go test ./internal/mcp/ -count=1 -run 'TestMCP_RunWorkflow_PaidUserGuidesAdmin|TestMCP_RunWorkflow_DisabledGuidesList|TestMCP_GetWorkflow_MissingGuidesList|TestMCP_RunWorkflow_MissingRequiredGuidesGetWorkflow|TestMCP_GetTask_ForeignOrEmptyGuidesListTasks'
```

Expected: FAIL——正文仍是 `user access denied` / `case disabled` / `case not found` / `missing required: prompt` / `task not found`。

- [ ] **Step 3: 写最小实现**

`tools.go`：

```go
func guideToolError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, botapp.ErrAccessDenied) {
		return GuideToolAccessDenied
	}
	if errors.Is(err, catalogdomain.ErrNotFound) || errors.Is(err, catalogdomain.ErrDisabled) {
		return GuideToolWorkflowGone
	}
	if errors.Is(err, runtimedomain.ErrTaskNotFound) {
		return GuideToolTaskHidden
	}
	raw := err.Error()
	if strings.Contains(raw, "workflow not found") || strings.Contains(raw, "case not found") {
		return GuideToolWorkflowGone
	}
	if _, keys, ok := strings.Cut(raw, "missing required:"); ok {
		return fmt.Sprintf(GuideToolMissingInput, strings.TrimSpace(keys))
	}
	if raw == "task_id required" {
		return GuideToolTaskHidden
	}
	return raw
}

func toolErr[T any](err error) (*mcpsdk.CallToolResult, T, error) {
	var zero T
	return &mcpsdk.CallToolResult{
		IsError: true,
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: guideToolError(err)}},
	}, zero, nil
}
```

常量可与 Task 1 一样放在 `auth.go`，或新建 `internal/mcp/guide.go` 只放字符串（不要新包）。优先 `guide.go`，避免 `auth.go` 再膨胀。

- [ ] **Step 4: 跑测试确认通过**

```bash
go test ./internal/mcp/ -count=1
```

Expected: PASS。

- [ ] **Step 5: Commit**

```bash
GIT_AUTHOR_NAME="李卓洲" GIT_AUTHOR_EMAIL="1138099359@qq.com" \
GIT_COMMITTER_NAME="李卓洲" GIT_COMMITTER_EMAIL="1138099359@qq.com" \
git add internal/mcp/tools.go internal/mcp/tools_test.go internal/mcp/guide.go
git commit -m "$(cat <<'EOF'
fix(mcp): tool 错误改成带下一步的固定句

EOF
)"
```

---

### Task 3: 产品 skill 包

**Files:**
- Create: `docs/skills/pixoma-mcp/SKILL.md`
- Create: `internal/mcp/skill_doc_test.go`

**Interfaces:**
- Consumes: Task 1/2 的 `Guide*` 原文（错误表步骤必须同义，可复述不可改成另一套话）
- Produces: `name: pixoma-mcp`；公开路径约定 `https://pixoma.miaoplus.com/skills/pixoma-mcp/SKILL.md`

- [ ] **Step 1: 写失败测试**

```go
package mcp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func TestPixomaMCPSkill_ProductAttachment(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "docs", "skills", "pixoma-mcp", "SKILL.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read skill: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, "name: pixoma-mcp") {
		t.Fatal("missing name")
	}
	if !strings.Contains(text, "Use when") {
		t.Fatal("description must be Use when only")
	}
	descEnd := strings.Index(text, "---\n\n")
	if descEnd < 0 {
		t.Fatal("missing frontmatter close")
	}
	desc := text[:descEnd]
	for _, banned := range []string{"list_workflows", "get_workflow", "run_workflow", "get_task"} {
		if strings.Contains(desc, banned) {
			t.Fatalf("description leaked flow %q", banned)
		}
	}
	if !strings.Contains(text, "错误") || !strings.Contains(text, "连接器") {
		t.Fatal("missing error table / connector")
	}
	for _, need := range []string{"list_workflows", "get_workflow", "run_workflow", "get_task", "pixoma://task/"} {
		if !strings.Contains(text, need) {
			t.Fatalf("missing call chain %q", need)
		}
	}
	if !strings.Contains(text, "task_id") {
		t.Fatal("run_workflow must say it returns task_id")
	}
	if strings.Contains(text, "签发") || strings.Contains(text, "粘贴 Bearer") {
		t.Fatal("must not teach bearer")
	}
	devSkill := filepath.Join(root, ".agents", "skills", "pixoma-mcp")
	if st, err := os.Stat(devSkill); err == nil && st.IsDir() {
		t.Fatal("must not install into .agents/skills")
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

```bash
go test ./internal/mcp/ -count=1 -run TestPixomaMCPSkill_ProductAttachment
```

Expected: FAIL `read skill: ... no such file`。

- [ ] **Step 3: 写 skill 全文**

创建 `docs/skills/pixoma-mcp/SKILL.md`（不要放到 `.agents/skills`）：

```markdown
---
name: pixoma-mcp
description: "Use when the user wants to run a Pixoma workflow, generate an image through Pixoma, check their own Pixoma tasks, or Pixoma MCP tools are missing from the chat."
---

# Pixoma MCP

## 前置

用户的聊天产品必须能配置 MCP 连接器。本 skill 不教签发或粘贴 token。

## 调用顺序

能看到 Pixoma tools 时按这次序：

1. `list_workflows` 列出已启用工作流
2. `get_workflow` 看必填 `inputs`
3. `run_workflow`：立刻返回 `task_id`，不要等它一次吐成品
4. 轮询 `get_task`
5. 读 `pixoma://task/{id}/output/{n}`

只使用列出、查看、发起与查询。用户要求通过 MCP 创建或修改 Case 时拒绝。

## 错误对照

先读接口正文，再按本表做。步骤与接口固定句同义。

| 情况 | Agent 步骤 |
|---|---|
| 工具列表里没有 `list_workflows` 等 | 明确告诉用户：当前聊天还没有配置 Pixoma MCP 连接器，要在该聊天的 MCP / 连接器设置里加上本实例后再试。配好之前无法跑工作流。不要向用户索要 token。 |
| HTTP 401，或正文说渠道/用户可能已停用 | 检查聊天的 Pixoma MCP 连接器是否指向本实例且凭据仍有效。对应的 MCP 渠道或用户可能已停用。不要向用户索要 token。配好后重试。 |
| HTTP 403，或连接器身份与会话不一致 | 连接器身份与会话不一致。让用户重连该连接器后重试。不要向用户索要 token。不要改走管理端代跑。 |
| 不能跑工作流 / MCP 渠道用户 | 该 MCP 用户当前不能跑工作流。请管理员在 MCP 渠道用户里改为允许后再试。 |
| 缺少必填 / `inputs` | 列出缺的字段。先 `get_workflow` 再带齐 `inputs` 重试。 |
| 工作流不存在或已停用 | 先 `list_workflows` 换已启用的 id。 |
| 任务不可见 / 查失败 | 只能查当前连接器用户的任务。核对 `task_id` 或先 `list_tasks`。 |

## 禁区

- 不创建或修改 Case
- 不向用户索要 token，不把 Bearer 贴进对话
- 不改走管理端代跑工作流
```

- [ ] **Step 4: 跑测试确认通过**

```bash
go test ./internal/mcp/ -count=1 -run TestPixomaMCPSkill_ProductAttachment
```

Expected: PASS。

- [ ] **Step 5: Commit**

```bash
GIT_AUTHOR_NAME="李卓洲" GIT_AUTHOR_EMAIL="1138099359@qq.com" \
GIT_COMMITTER_NAME="李卓洲" GIT_COMMITTER_EMAIL="1138099359@qq.com" \
git add docs/skills/pixoma-mcp/SKILL.md internal/mcp/skill_doc_test.go
git commit -m "$(cat <<'EOF'
docs: 增加 pixoma-mcp 调用方 skill

EOF
)"
```

---

### Task 4: 安装提示词

**Files:**
- Modify: `README.md`（「它能做什么」里 MCP 那条之后加一小节，不要改 `docs/install.md` 的装机脚本）
- Test: `internal/mcp/skill_doc_test.go`（同一文件追加）

**Interfaces:**
- Consumes: 公网 URL `https://pixoma.miaoplus.com/skills/pixoma-mcp/SKILL.md`；仓库路径 `docs/skills/pixoma-mcp/SKILL.md`
- Produces: 文档中恰好一句可复制提示词，不含密钥

固定提示词（整句写入 README，测试按整句包含断言）：

```text
把 https://pixoma.miaoplus.com/skills/pixoma-mcp/SKILL.md 装进本机 skills 目录；用户的聊天产品需要能配置 MCP 连接器。仓库内同源文件：docs/skills/pixoma-mcp/SKILL.md。
```

- [ ] **Step 1: 写失败测试**

在 `skill_doc_test.go` 追加：

```go
func TestPixomaMCPSkill_InstallPromptInREADME(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	const prompt = "把 https://pixoma.miaoplus.com/skills/pixoma-mcp/SKILL.md 装进本机 skills 目录；用户的聊天产品需要能配置 MCP 连接器。仓库内同源文件：docs/skills/pixoma-mcp/SKILL.md。"
	if !strings.Contains(text, prompt) {
		t.Fatalf("README missing install prompt")
	}
	if !strings.Contains(text, "MCP 连接器") {
		t.Fatal("must say chat needs MCP connector")
	}
	lower := strings.ToLower(text)
	idx := strings.Index(text, "pixoma-mcp/SKILL.md")
	if idx < 0 {
		t.Fatal("missing skill url")
	}
	window := text
	if idx >= 0 && idx+400 < len(text) {
		window = text[idx : idx+400]
	}
	if strings.Contains(window, "sk-") || strings.Contains(strings.ToLower(window), "bearer ") || strings.Contains(window, "mcp-plain") {
		t.Fatalf("prompt window looks like a secret: %q", window)
	}
	_ = lower
}
```

- [ ] **Step 2: 跑测试确认失败**

```bash
go test ./internal/mcp/ -count=1 -run TestPixomaMCPSkill_InstallPromptInREADME
```

Expected: FAIL `README missing install prompt`。

- [ ] **Step 3: 写入 README**

在 `README.md`「它能做什么」列表后插入：

```markdown
### 给聊天装 Pixoma MCP skill

把 https://pixoma.miaoplus.com/skills/pixoma-mcp/SKILL.md 装进本机 skills 目录；用户的聊天产品需要能配置 MCP 连接器。仓库内同源文件：docs/skills/pixoma-mcp/SKILL.md。
```

不要写 token、Bearer、占位密钥。

- [ ] **Step 4: 跑测试确认通过**

```bash
go test ./internal/mcp/ -count=1
```

Expected: PASS。

- [ ] **Step 5: Commit**

```bash
GIT_AUTHOR_NAME="李卓洲" GIT_AUTHOR_EMAIL="1138099359@qq.com" \
GIT_COMMITTER_NAME="李卓洲" GIT_COMMITTER_EMAIL="1138099359@qq.com" \
git add README.md internal/mcp/skill_doc_test.go
git commit -m "$(cat <<'EOF'
docs: 增加 pixoma-mcp skill 安装提示词

EOF
)"
```

---

### Task 5: 对照 spec 验收

**Files:**
- 不改代码（只跑检查）

**Interfaces:**
- Consumes: 本计划 Task 1–4 产物与 `docs/openspec/changes/mcp-caller-skill/specs/mcp-caller-skill/spec.md`
- Produces: 本地验收记录（不必新文件；在 PR/change 备注勾选即可）

- [ ] **Step 1: 跑包测试**

```bash
go test ./internal/mcp/ -count=1
```

Expected: PASS。

- [ ] **Step 2: 对照 spec 勾选**

| Requirement | 证据 |
|---|---|
| 一句提示词即可安装 | README 含公网 URL；无 Bearer/token 密钥 |
| 提示词写明需 MCP 连接器 | 同一句含「聊天产品需要能配置 MCP 连接器」 |
| 调用顺序、run 立刻返回 task_id、鉴权从略 | `SKILL.md` 调用顺序；frontmatter 无 Bearer 教程 |
| 找不到工具 → 配连接器、不索要 token | skill 错误表第一行 |
| HTTP 401/403 与 tool 指导句 | Task 1/2 测试 |
| 权限 / 缺字段 / 停用或不存在 / 任务不可见 | Task 2 测试 + skill 表同义 |
| 403 不改走管理端代跑 | skill 403 行 |
| 不进开发技能树 | `TestPixomaMCPSkill_ProductAttachment` 断言无 `.agents/skills/pixoma-mcp` |

- [ ] **Step 3: 无提交**（无代码变更则跳过）

---

## Self-review

1. Spec coverage：提示词、调用链、无工具、接口文案、错误表、不进 `.agents/skills` 均有任务。方案 A 无错误码。未改架构文档（设计第 9 节）。
2. Placeholder scan：无 TBD /「类似 Task N」。固定句已写出。
3. 类型：`GuideUnauthorized` / `GuideUnavailable` / `GuideForbidden` / `guideToolError` 前后任务一致。
