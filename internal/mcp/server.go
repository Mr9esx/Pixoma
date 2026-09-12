package mcp

import (
	"context"
	"strconv"
	"strings"

	catalogdomain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	"github.com/Mr9esx/Pixoma/internal/packaging/botapp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type backend struct {
	facade *botapp.Facade
	ident  Identity
}

func newServer(deps Deps, ident Identity, cache *mcpsdk.SchemaCache) *mcpsdk.Server {
	b := &backend{facade: deps.Facade, ident: ident}
	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "pixoma", Version: "v1"}, &mcpsdk.ServerOptions{
		Instructions: "List and run enabled Pixoma workflows. Poll get_task for results.",
		Capabilities: &mcpsdk.ServerCapabilities{
			Logging:      &mcpsdk.LoggingCapabilities{},
			Experimental: map[string]any{"elicitation": map[string]any{}},
		},
		CompletionHandler: b.complete,
		SchemaCache:       cache,
	})
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_workflows",
		Description: "List enabled Pixoma workflows",
	}, b.listWorkflows)
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_workflow",
		Description: "Get a workflow's input schema",
	}, b.getWorkflow)
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "run_workflow",
		Description: "Start a workflow and return task_id immediately",
	}, b.runWorkflow)
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_tasks",
		Description: "List tasks for the current MCP user",
	}, b.listTasks)
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_task",
		Description: "Get one of the current MCP user's tasks",
	}, b.getTask)
	server.AddPrompt(&mcpsdk.Prompt{
		Name:        "run_workflow",
		Description: "Run an enabled Pixoma workflow",
		Arguments: []*mcpsdk.PromptArgument{
			{Name: "case_id", Required: true, Description: "Workflow id"},
		},
	}, b.getPrompt)
	server.AddResourceTemplate(&mcpsdk.ResourceTemplate{
		Name:        "case",
		URITemplate: "pixoma://case/{id}",
		MIMEType:    "application/json",
	}, b.readResource)
	server.AddResourceTemplate(&mcpsdk.ResourceTemplate{
		Name:        "task_output",
		URITemplate: "pixoma://task/{id}/output/{n}",
	}, b.readResource)
	return server
}

func (b *backend) getPrompt(_ context.Context, req *mcpsdk.GetPromptRequest) (*mcpsdk.GetPromptResult, error) {
	caseID := ""
	if req != nil && req.Params != nil && req.Params.Arguments != nil {
		caseID = req.Params.Arguments["case_id"]
	}
	text := "Call run_workflow with the workflow id and required inputs, then poll get_task."
	if caseID != "" {
		text = "Call run_workflow with case_id=" + caseID + " and required inputs, then poll get_task."
	}
	return &mcpsdk.GetPromptResult{
		Messages: []*mcpsdk.PromptMessage{{
			Role:    "user",
			Content: &mcpsdk.TextContent{Text: text},
		}},
	}, nil
}

func (b *backend) complete(ctx context.Context, req *mcpsdk.CompleteRequest) (*mcpsdk.CompleteResult, error) {
	out := &mcpsdk.CompleteResult{Completion: mcpsdk.CompletionResultDetails{Values: []string{}}}
	if b.facade == nil || req == nil || req.Params == nil || req.Params.Ref == nil {
		return out, nil
	}
	if req.Params.Ref.Type != "ref/prompt" || req.Params.Ref.Name != "run_workflow" {
		return out, nil
	}
	if req.Params.Argument.Name != "case_id" {
		return out, nil
	}
	enabled := true
	cases, err := b.facade.ListCases(ctx, catalogdomain.ListQuery{Enabled: &enabled})
	if err != nil {
		return out, nil
	}
	prefix := req.Params.Argument.Value
	for _, c := range cases {
		id := strconv.FormatUint(uint64(c.Document.ID), 10)
		if prefix == "" || strings.HasPrefix(id, prefix) || strings.HasPrefix(c.Document.Name, prefix) {
			out.Completion.Values = append(out.Completion.Values, id)
		}
	}
	return out, nil
}
