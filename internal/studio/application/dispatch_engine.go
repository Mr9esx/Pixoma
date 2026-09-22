package application

import (
	"context"
	"fmt"
	"strings"
)

// DispatchEngine runs the configured Eino single Agent. A Studio run is never
// allowed to silently fall back to a deterministic local agent: a model choice
// is required before a conversation can execute.
type DispatchEngine struct {
	Online AgentEngine
}

func (e DispatchEngine) Execute(ctx context.Context, request AgentRequest, sink AgentSink) error {
	if strings.TrimSpace(request.Run.ModelConfigID) == "" {
		return fmt.Errorf("studio: model is required before starting an Agent run")
	}
	if e.Online == nil {
		return fmt.Errorf("studio: online agent engine is not configured")
	}
	return e.Online.Execute(ctx, request, sink)
}

var _ AgentEngine = (*DispatchEngine)(nil)
