// Package conversation contains the platform-independent portion of an IM
// conversation. Platform adapters decode inbound events and render effects;
// this package constructs capability calls and owns short-lived action tokens.
package conversation

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/Mr9esx/Pixoma/internal/channels/protocol"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

// Inbound is the platform-neutral identity of one user interaction.
type Inbound struct {
	Addr           sharedkernel.ChannelAddr
	ExternalUserID string
}

// Invoker is implemented by the capability registry.
type Invoker interface {
	Invoke(ctx context.Context, inv protocol.CapabilityInvoke) (protocol.Result, error)
}

// Renderer emits a text effect. Rich platform renderers are layered on this
// narrow common port as the adapters are migrated.
type Renderer interface {
	SendText(ctx context.Context, addr sharedkernel.ChannelAddr, text string) error
}

// Controller owns capability invocation and one-shot action tokens.
type Controller struct {
	channelID string
	invoker   Invoker
	renderer  Renderer

	mu      sync.Mutex
	seq     uint64
	actions map[string]protocol.CapabilityInvoke
}

// New creates a platform-independent controller for one channel.
func New(channelID string, invoker Invoker, renderer Renderer) *Controller {
	return &Controller{
		channelID: channelID,
		invoker:   invoker,
		renderer:  renderer,
		actions:   make(map[string]protocol.CapabilityInvoke),
	}
}

// HandleMedia submits an already-materialized image to the active workflow.
func (c *Controller) HandleMedia(ctx context.Context, in Inbound, ref sharedkernel.BlobRef) error {
	return c.invoke(ctx, in, protocol.CapabilityInvoke{
		CapabilityID: "open_case",
		Params: map[string]any{
			"step": "media",
			"blob": map[string]any{"key": ref.Key, "mime": ref.MIME},
		},
	})
}

// RememberAction stores a capability invocation and returns its opaque token.
// Platform adapters are responsible for encoding the token into callback data.
func (c *Controller) RememberAction(inv protocol.CapabilityInvoke) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seq++
	token := strconv.FormatUint(c.seq, 36)
	c.actions[token] = inv
	return token
}

// HandleAction consumes and invokes a previously issued action token.
func (c *Controller) HandleAction(ctx context.Context, in Inbound, token string) error {
	c.mu.Lock()
	inv, ok := c.actions[token]
	if ok {
		delete(c.actions, token)
	}
	c.mu.Unlock()
	if !ok {
		if c.renderer == nil {
			return nil
		}
		return c.renderer.SendText(ctx, in.Addr, "操作已过期，重新选择。")
	}
	return c.invoke(ctx, in, inv)
}

func (c *Controller) invoke(ctx context.Context, in Inbound, inv protocol.CapabilityInvoke) error {
	if c == nil || c.invoker == nil {
		return fmt.Errorf("conversation: capability invoker not configured")
	}
	inv.Account.ChannelID = c.channelID
	inv.Account.ExternalUserID = in.ExternalUserID
	inv.ChatID = sharedkernel.FormatChatID(in.Addr)
	if inv.Nav.Back == "" {
		inv.Nav.Back = "root"
	}
	_, err := c.invoker.Invoke(ctx, inv)
	return err
}
