package mcp

import (
	"context"
	"encoding/json"
	"io"
	"strconv"
	"strings"

	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func (b *backend) readResource(ctx context.Context, req *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
	uri := ""
	if req != nil && req.Params != nil {
		uri = req.Params.URI
	}
	if id, ok := parseCaseURI(uri); ok {
		return b.readCase(ctx, uri, id)
	}
	if taskID, n, ok := parseTaskOutputURI(uri); ok {
		return b.readTaskOutput(ctx, uri, taskID, n)
	}
	return nil, mcpsdk.ResourceNotFoundError(uri)
}

func (b *backend) readCase(ctx context.Context, uri string, id sharedkernel.CaseID) (*mcpsdk.ReadResourceResult, error) {
	if b.facade == nil {
		return nil, mcpsdk.ResourceNotFoundError(uri)
	}
	c, err := b.facade.GetCase(ctx, id)
	if err != nil {
		return nil, mcpsdk.ResourceNotFoundError(uri)
	}
	body, err := json.Marshal(map[string]any{
		"id":          c.Document.ID,
		"name":        c.Document.Name,
		"description": c.Document.Description,
		"enabled":     c.Enabled,
		"inputs":      c.Document.Inputs,
		"outputs":     c.Document.Outputs,
	})
	if err != nil {
		return nil, err
	}
	return &mcpsdk.ReadResourceResult{
		Contents: []*mcpsdk.ResourceContents{{
			URI:      uri,
			MIMEType: "application/json",
			Text:     string(body),
		}},
	}, nil
}

func (b *backend) readTaskOutput(ctx context.Context, uri string, taskID sharedkernel.TaskID, n int) (*mcpsdk.ReadResourceResult, error) {
	task, err := b.ownedTask(ctx, taskID)
	if err != nil {
		return nil, mcpsdk.ResourceNotFoundError(uri)
	}
	if n < 0 || n >= len(task.Outputs) {
		return nil, mcpsdk.ResourceNotFoundError(uri)
	}
	ref := task.Outputs[n].Blob
	if b.facade == nil || b.facade.Blob == nil || ref.Key == "" {
		return nil, mcpsdk.ResourceNotFoundError(uri)
	}
	rc, err := b.facade.Blob.Get(ctx, ref)
	if err != nil {
		return nil, mcpsdk.ResourceNotFoundError(uri)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	mime := ref.MIME
	if mime == "" {
		mime = "application/octet-stream"
	}
	return &mcpsdk.ReadResourceResult{
		Contents: []*mcpsdk.ResourceContents{{
			URI:      uri,
			MIMEType: mime,
			Blob:     data,
		}},
	}, nil
}

func parseCaseURI(uri string) (sharedkernel.CaseID, bool) {
	rest, ok := strings.CutPrefix(uri, "pixoma://case/")
	if !ok || rest == "" || strings.Contains(rest, "/") {
		return 0, false
	}
	id, err := sharedkernel.ParseCaseID(rest)
	if err != nil {
		return 0, false
	}
	return id, true
}

func parseTaskOutputURI(uri string) (sharedkernel.TaskID, int, bool) {
	rest, ok := strings.CutPrefix(uri, "pixoma://task/")
	if !ok {
		return "", 0, false
	}
	id, suffix, ok := strings.Cut(rest, "/output/")
	if !ok || id == "" {
		return "", 0, false
	}
	n, err := strconv.Atoi(suffix)
	if err != nil || n < 0 {
		return "", 0, false
	}
	return sharedkernel.TaskID(id), n, true
}
