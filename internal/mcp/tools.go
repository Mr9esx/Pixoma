package mcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	catalogdomain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	"github.com/Mr9esx/Pixoma/internal/packaging/botapp"
	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	runtimedomain "github.com/Mr9esx/Pixoma/internal/tasks/domain"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const maxInputBytes = 8 << 20

type workflowItem struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled"`
}

type listWorkflowsOut struct {
	Workflows []workflowItem `json:"workflows"`
}

type getWorkflowIn struct {
	CaseID uint64 `json:"case_id" jsonschema:"Workflow id"`
}

type getWorkflowOut struct {
	ID          uint64                      `json:"id"`
	Name        string                      `json:"name"`
	Description string                      `json:"description,omitempty"`
	Enabled     bool                        `json:"enabled"`
	Inputs      []catalogdomain.InputField  `json:"inputs"`
	Outputs     []catalogdomain.OutputField `json:"outputs"`
}

type runWorkflowIn struct {
	CaseID uint64         `json:"case_id" jsonschema:"Workflow id"`
	Inputs map[string]any `json:"inputs"`
}

type runWorkflowOut struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}

type listTasksOut struct {
	Tasks []taskItem `json:"tasks"`
}

type getTaskIn struct {
	TaskID string `json:"task_id" jsonschema:"Task id"`
}

type taskItem struct {
	TaskID  string       `json:"task_id"`
	Status  string       `json:"status"`
	CaseID  uint64       `json:"case_id"`
	Error   string       `json:"error,omitempty"`
	Outputs []outputItem `json:"outputs,omitempty"`
}

type outputItem struct {
	Key  string `json:"key"`
	URI  string `json:"uri"`
	MIME string `json:"mime,omitempty"`
}

func (b *backend) listWorkflows(ctx context.Context, _ *mcpsdk.CallToolRequest, _ struct{}) (*mcpsdk.CallToolResult, listWorkflowsOut, error) {
	out := listWorkflowsOut{Workflows: []workflowItem{}}
	if b.facade == nil {
		return nil, out, nil
	}
	enabled := true
	cases, err := b.facade.ListCases(ctx, catalogdomain.ListQuery{Enabled: &enabled})
	if err != nil {
		return toolErr[listWorkflowsOut](err)
	}
	for _, c := range cases {
		out.Workflows = append(out.Workflows, workflowItem{
			ID:          uint64(c.Document.ID),
			Name:        c.Document.Name,
			Description: c.Document.Description,
			Enabled:     c.Enabled,
		})
	}
	return nil, out, nil
}

func (b *backend) getWorkflow(ctx context.Context, _ *mcpsdk.CallToolRequest, in getWorkflowIn) (*mcpsdk.CallToolResult, getWorkflowOut, error) {
	if b.facade == nil {
		return toolErr[getWorkflowOut](errors.New("workflow not found"))
	}
	c, err := b.facade.GetCase(ctx, sharedkernel.CaseID(in.CaseID))
	if err != nil {
		return toolErr[getWorkflowOut](err)
	}
	return nil, getWorkflowOut{
		ID:          uint64(c.Document.ID),
		Name:        c.Document.Name,
		Description: c.Document.Description,
		Enabled:     c.Enabled,
		Inputs:      c.Document.Inputs,
		Outputs:     c.Document.Outputs,
	}, nil
}

func (b *backend) runWorkflow(ctx context.Context, req *mcpsdk.CallToolRequest, in runWorkflowIn) (*mcpsdk.CallToolResult, runWorkflowOut, error) {
	if b.facade == nil {
		return toolErr[runWorkflowOut](errors.New("mcp not wired"))
	}
	if !b.ident.ok() {
		return toolErr[runWorkflowOut](errors.New("mcp user required"))
	}
	if in.CaseID == 0 {
		return toolErr[runWorkflowOut](errors.New("case_id required"))
	}
	c, err := b.facade.GetCase(ctx, sharedkernel.CaseID(in.CaseID))
	if err != nil {
		return toolErr[runWorkflowOut](err)
	}
	if !c.Enabled {
		return toolErr[runWorkflowOut](catalogdomain.ErrDisabled)
	}
	values, err := toInputValues(ctx, b.facade.Blob, c.Document, in.Inputs)
	if err != nil {
		return toolErr[runWorkflowOut](err)
	}
	if missing := missingRequired(c.Document, values); len(missing) > 0 {
		filled, elicitErr := b.elicitMissing(ctx, req, missing)
		if elicitErr != nil {
			return toolErr[runWorkflowOut](elicitErr)
		}
		if filled == nil {
			return toolErr[runWorkflowOut](fmt.Errorf("missing required: %s", missingKeys(missing)))
		}
		for k, v := range filled {
			if in.Inputs == nil {
				in.Inputs = map[string]any{}
			}
			in.Inputs[k] = v
		}
		values, err = toInputValues(ctx, b.facade.Blob, c.Document, in.Inputs)
		if err != nil {
			return toolErr[runWorkflowOut](err)
		}
	}
	res, err := b.facade.RunCase(ctx, botapp.RunCaseCmd{
		CaseID: c.Document.ID,
		Inputs: values,
		Actor:  b.ident.ChatID(),
		UserID: b.ident.UserID,
	})
	if err != nil {
		return toolErr[runWorkflowOut](err)
	}
	return nil, runWorkflowOut{TaskID: string(res.TaskID), Status: string(res.Status)}, nil
}

func (b *backend) listTasks(ctx context.Context, _ *mcpsdk.CallToolRequest, _ struct{}) (*mcpsdk.CallToolResult, listTasksOut, error) {
	out := listTasksOut{Tasks: []taskItem{}}
	if b.facade == nil || !b.ident.ok() {
		return nil, out, nil
	}
	tasks, err := b.facade.ListMyTasks(ctx, b.ident.ChatID(), 50)
	if err != nil {
		return toolErr[listTasksOut](err)
	}
	for _, task := range tasks {
		if !b.owns(ctx, task) {
			continue
		}
		out.Tasks = append(out.Tasks, toTaskItem(task))
	}
	return nil, out, nil
}

func (b *backend) getTask(ctx context.Context, _ *mcpsdk.CallToolRequest, in getTaskIn) (*mcpsdk.CallToolResult, taskItem, error) {
	if strings.TrimSpace(in.TaskID) == "" {
		return toolErr[taskItem](errors.New("task_id required"))
	}
	task, err := b.ownedTask(ctx, sharedkernel.TaskID(in.TaskID))
	if err != nil {
		return toolErr[taskItem](err)
	}
	return nil, toTaskItem(task), nil
}

func (b *backend) ownedTask(ctx context.Context, id sharedkernel.TaskID) (*runtimedomain.Task, error) {
	if b.facade == nil || !b.ident.ok() {
		return nil, runtimedomain.ErrTaskNotFound
	}
	task, err := b.facade.Tasks.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !b.owns(ctx, task) {
		return nil, runtimedomain.ErrTaskNotFound
	}
	return task, nil
}

func (b *backend) owns(ctx context.Context, task *runtimedomain.Task) bool {
	if task == nil || !b.ident.ok() {
		return false
	}
	if task.ChatID != "" && task.ChatID == b.ident.ChatID() {
		return true
	}
	if b.facade.SessionStore != nil && task.SessionID != "" {
		sess, err := b.facade.SessionStore.GetByID(ctx, task.SessionID)
		if err == nil && sess != nil && sess.UserID == b.ident.UserID {
			return true
		}
	}
	return false
}

func (b *backend) elicitMissing(ctx context.Context, req *mcpsdk.CallToolRequest, missing []catalogdomain.InputField) (map[string]any, error) {
	if req == nil || req.Session == nil || !clientSupportsElicitation(req.Session) {
		return nil, fmt.Errorf("missing required: %s", missingKeys(missing))
	}
	props := map[string]any{}
	required := make([]string, 0, len(missing))
	for _, f := range missing {
		schemaType := "string"
		switch f.Type {
		case "number":
			schemaType = "number"
		case "boolean":
			schemaType = "boolean"
		}
		prop := map[string]any{"type": schemaType}
		if f.Description != "" {
			prop["description"] = f.Description
		}
		props[f.Key] = prop
		required = append(required, f.Key)
	}
	res, err := req.Session.Elicit(ctx, &mcpsdk.ElicitParams{
		Mode:    "form",
		Message: "Missing required workflow inputs: " + missingKeys(missing),
		RequestedSchema: map[string]any{
			"type":       "object",
			"properties": props,
			"required":   required,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("missing required: %s", missingKeys(missing))
	}
	if res == nil || res.Action != "accept" || res.Content == nil {
		return nil, fmt.Errorf("missing required: %s", missingKeys(missing))
	}
	return res.Content, nil
}

func clientSupportsElicitation(ss *mcpsdk.ServerSession) bool {
	if ss == nil {
		return false
	}
	p := ss.InitializeParams()
	return p != nil && p.Capabilities != nil && p.Capabilities.Elicitation != nil
}

func toTaskItem(task *runtimedomain.Task) taskItem {
	item := taskItem{
		TaskID: string(task.ID),
		Status: string(task.Status),
		CaseID: uint64(task.CaseID),
	}
	if task.ErrorMessage != "" {
		item.Error = task.ErrorMessage
	}
	for i, out := range task.Outputs {
		item.Outputs = append(item.Outputs, outputItem{
			Key:  out.Key,
			URI:  fmt.Sprintf("pixoma://task/%s/output/%d", task.ID, i),
			MIME: out.Blob.MIME,
		})
	}
	return item
}

func missingRequired(doc catalogdomain.CaseDocument, values []catalogdomain.InputValue) []catalogdomain.InputField {
	byKey := map[string]catalogdomain.InputValue{}
	for _, v := range values {
		byKey[v.Key] = v
	}
	var missing []catalogdomain.InputField
	for _, in := range doc.Inputs {
		if !in.Required {
			continue
		}
		val, ok := byKey[in.Key]
		if !ok || emptyValue(in, val) {
			missing = append(missing, in)
		}
	}
	return missing
}

func emptyValue(in catalogdomain.InputField, val catalogdomain.InputValue) bool {
	switch in.Type {
	case "image", "video":
		return val.Blob == nil || strings.TrimSpace(val.Blob.Key) == ""
	case "number":
		return val.Number == nil
	case "boolean":
		return val.Bool == nil
	default:
		return val.Text == nil || strings.TrimSpace(*val.Text) == ""
	}
}

func missingKeys(fields []catalogdomain.InputField) string {
	keys := make([]string, 0, len(fields))
	for _, f := range fields {
		keys = append(keys, f.Key)
	}
	return strings.Join(keys, ", ")
}

func toInputValues(ctx context.Context, store blob.Store, doc catalogdomain.CaseDocument, raw map[string]any) ([]catalogdomain.InputValue, error) {
	if raw == nil {
		raw = map[string]any{}
	}
	out := make([]catalogdomain.InputValue, 0, len(doc.Inputs))
	for _, field := range doc.Inputs {
		v, ok := raw[field.Key]
		if !ok || v == nil {
			continue
		}
		iv, err := coerceInput(ctx, store, field, v)
		if err != nil {
			return nil, err
		}
		out = append(out, iv)
	}
	return out, nil
}

func coerceInput(ctx context.Context, store blob.Store, field catalogdomain.InputField, v any) (catalogdomain.InputValue, error) {
	iv := catalogdomain.InputValue{Key: field.Key}
	switch field.Type {
	case "number":
		n, err := asFloat(v)
		if err != nil {
			return iv, fmt.Errorf("%s: %w", field.Key, err)
		}
		iv.Number = &n
	case "boolean":
		b, ok := v.(bool)
		if !ok {
			return iv, fmt.Errorf("%s: expected boolean", field.Key)
		}
		iv.Bool = &b
	case "image", "video":
		ref, err := asBlob(ctx, store, field, v)
		if err != nil {
			return iv, err
		}
		iv.Blob = &ref
	default:
		s, err := asString(v)
		if err != nil {
			return iv, fmt.Errorf("%s: %w", field.Key, err)
		}
		iv.Text = &s
	}
	return iv, nil
}

func asFloat(v any) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case json.Number:
		return n.Float64()
	case string:
		return strconv.ParseFloat(n, 64)
	default:
		return 0, fmt.Errorf("expected number")
	}
}

func asString(v any) (string, error) {
	switch s := v.(type) {
	case string:
		return s, nil
	default:
		return fmt.Sprint(v), nil
	}
}

func asBlob(ctx context.Context, store blob.Store, field catalogdomain.InputField, v any) (sharedkernel.BlobRef, error) {
	raw, mime, err := blobPayload(v)
	if err != nil {
		return sharedkernel.BlobRef{}, fmt.Errorf("%s: %w", field.Key, err)
	}
	if len(raw) > maxInputBytes {
		return sharedkernel.BlobRef{}, fmt.Errorf("%s: too large", field.Key)
	}
	if store == nil {
		return sharedkernel.BlobRef{}, fmt.Errorf("%s: blob store not configured", field.Key)
	}
	if mime == "" {
		if field.Type == "video" {
			mime = "video/mp4"
		} else {
			mime = "image/png"
		}
	}
	ref, err := store.Put(ctx, "mcp-inputs/"+field.Key, bytes.NewReader(raw), blob.PutOptions{MIME: mime})
	if err != nil {
		return sharedkernel.BlobRef{}, fmt.Errorf("%s: %w", field.Key, err)
	}
	return ref, nil
}

func blobPayload(v any) ([]byte, string, error) {
	switch x := v.(type) {
	case string:
		return decodeBytes(x)
	case map[string]any:
		mime, _ := x["mime"].(string)
		data, _ := x["data"].(string)
		if data == "" {
			return nil, "", fmt.Errorf("expected data")
		}
		raw, _, err := decodeBytes(data)
		return raw, mime, err
	default:
		return nil, "", fmt.Errorf("expected media bytes")
	}
}

func decodeBytes(s string) ([]byte, string, error) {
	s = strings.TrimSpace(s)
	mime := ""
	if payload, ok := strings.CutPrefix(s, "data:"); ok {
		header, data, ok := strings.Cut(payload, ",")
		if !ok {
			return nil, "", fmt.Errorf("invalid data url")
		}
		mime, _, _ = strings.Cut(header, ";")
		s = data
	}
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, "", fmt.Errorf("invalid base64")
	}
	return raw, mime, nil
}

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
