package controlplane

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Mr9esx/Pixoma/internal/platform/queue"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

// StatusPublisher adapts Client to queue.Publisher for actuator.Worker.

type StatusPublisher struct {
	Client *Client
}

func NewStatusPublisher(c *Client) *StatusPublisher {
	return &StatusPublisher{Client: c}
}

func (p *StatusPublisher) Publish(ctx context.Context, msg queue.Message) error {
	var ev sharedkernel.TaskStatusEvent
	if err := json.Unmarshal(msg.Payload, &ev); err != nil {
		return fmt.Errorf("pull: status payload: %w", err)
	}
	return p.Client.ReportStatus(ctx, ev)
}
