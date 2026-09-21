package application

import (
	"context"
	"fmt"
	"strings"
)

// DispatchEngine keeps the deterministic Mock workflow isolated from the
// configured Eino single Agent. A run selects the online path only when the
// user explicitly chose an Agent-enabled model for its session.
type DispatchEngine struct {
	Mock   AgentEngine
	Online AgentEngine
}

func (e DispatchEngine) Execute(ctx context.Context, request AgentRequest, sink AgentSink) error {
	if strings.TrimSpace(request.Run.ModelConfigID) == "" {
		if e.Mock == nil {
			return fmt.Errorf("studio: mock engine is not configured")
		}
		return e.Mock.Execute(ctx, request, sink)
	}
	if e.Online == nil {
		return fmt.Errorf("studio: online agent engine is not configured")
	}
	return e.Online.Execute(ctx, request, sink)
}

var _ AgentEngine = (*DispatchEngine)(nil)
