// Package conversation contains the platform-independent portion of an IM
// conversation. Platform adapters decode inbound events and render effects;
// this package constructs capability calls and owns short-lived action tokens.
package conversation

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	texttpl "github.com/Mr9esx/Pixoma/internal/channels/domain/templates"
	"github.com/Mr9esx/Pixoma/internal/channels/protocol"
	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	identitydomain "github.com/Mr9esx/Pixoma/internal/users/domain"
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

// CallbackCodec encodes platform-neutral action tokens and navigation targets
// into a platform callback payload.
type CallbackCodec interface {
	Invoke(token string) string
	MainMenu() string
	Back(target string) string
}

// Controller owns capability invocation and one-shot action tokens.
type Controller struct {
	channelID string
	invoker   Invoker
	renderer  Renderer
	// Outbound and Callbacks are platform rendering ports. The controller owns
	// result decisions while adapters own payload encoding and delivery.
	Outbound      protocol.Outbound
	Callbacks     CallbackCodec
	Blob          blob.Store
	Texts         texttpl.Renderer
	Notifications *NotificationStore
	// Users resolves an external platform user to Pixoma's internal identity.
	// It is optional so pure protocol tests need no persistence dependency.
	Users protocol.IdentityResolver

	actions *ActionStore
}

// New creates a platform-independent controller for one channel.
func New(channelID string, invoker Invoker, renderer Renderer) *Controller {
	return NewWithActionStore(channelID, invoker, renderer, nil)
}

// NewWithActionStore creates a controller that shares the supplied action
// store with its platform adapter. A nil store creates a private store.
func NewWithActionStore(channelID string, invoker Invoker, renderer Renderer, actions *ActionStore) *Controller {
	if actions == nil {
		actions = NewActionStore()
	}
	return &Controller{
		channelID:     channelID,
		invoker:       invoker,
		renderer:      renderer,
		actions:       actions,
		Notifications: NewNotificationStore(),
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

// HandleText submits a text input to the active workflow.
func (c *Controller) HandleText(ctx context.Context, in Inbound, text string) error {
	return c.invoke(ctx, in, protocol.CapabilityInvoke{
		CapabilityID: "open_case",
		Params:       map[string]any{"step": "text", "text": text},
	})
}

// RememberAction stores a capability invocation and returns its opaque token.
// Platform adapters are responsible for encoding the token into callback data.
func (c *Controller) RememberAction(inv protocol.CapabilityInvoke) string {
	return c.actions.Put(inv)
}

// HandleAction consumes and invokes a previously issued action token.
func (c *Controller) HandleAction(ctx context.Context, in Inbound, token string) error {
	inv, ok := c.actions.Get(token)
	if !ok {
		if c.renderer == nil {
			return nil
		}
		return c.renderer.SendText(ctx, in.Addr, "操作已过期，重新选择。")
	}
	if err := c.invoke(ctx, in, inv); err != nil {
		return err
	}
	c.actions.Consume(token)
	return nil
}

// RenderResult turns a capability result into platform-neutral outbound calls.
func (c *Controller) RenderResult(ctx context.Context, in Inbound, base protocol.CapabilityInvoke, res protocol.Result) error {
	if c == nil || c.Outbound == nil || c.Callbacks == nil {
		return fmt.Errorf("conversation: result renderer not configured")
	}
	rows := make([][]protocol.Button, 0, len(res.Options)+1)
	for _, opt := range res.Options {
		next := base
		next.Params = opt.Value
		rows = append(rows, []protocol.Button{{Text: opt.Label, Data: c.Callbacks.Invoke(c.actions.Put(next))}})
	}
	if len(rows) > 0 {
		rows = append(rows, c.backButton(base.Nav))
	}

	hasMedia := len(res.Media) > 0 || len(res.MediaURLs) > 0
	first := true
	for _, media := range res.Media {
		caption, buttons := mediaExtras(res.Text, rows, first)
		first = false
		if err := c.Outbound.SendMedia(ctx, in.Addr, sharedkernel.BlobRef{Key: media.Key, MIME: media.MIME}, caption, buttons); err != nil {
			return err
		}
	}
	for _, mediaURL := range res.MediaURLs {
		caption, buttons := mediaExtras(res.Text, rows, first)
		first = false
		if err := c.Outbound.SendMediaURL(ctx, in.Addr, mediaURL, "", caption, buttons); err != nil {
			return err
		}
	}
	if hasMedia {
		return nil
	}
	if len(rows) > 0 {
		return c.Outbound.SendList(ctx, in.Addr, res.Text, rows)
	}
	if res.Text != "" {
		return c.Outbound.SendText(ctx, in.Addr, res.Text)
	}
	return nil
}

// HandleNotify renders an asynchronous task notification at its persisted
// conversation address. It is shared by all IM platform adapters.
func (c *Controller) HandleNotify(ctx context.Context, n sharedkernel.UserNotify) error {
	if c == nil || c.Outbound == nil {
		return fmt.Errorf("conversation: notification renderer not configured")
	}
	if c.Notifications == nil {
		c.Notifications = NewNotificationStore()
	}
	key := string(n.TaskID) + ":" + n.Kind
	if !c.Notifications.Mark(key) {
		return nil
	}
	addr, err := sharedkernel.ParseChatID(string(n.ChatID))
	if err != nil {
		return err
	}
	if n.Kind == "task_succeeded" {
		if len(n.Outputs) > 0 {
			for i, ref := range n.Outputs {
				caption := ""
				if i == 0 {
					caption = c.renderText(ctx, texttpl.KeyWorkflowDone, map[string]string{"task_id": string(n.TaskID)})
				}
				if strings.HasPrefix(ref.MIME, "text/") {
					if c.Blob == nil {
						continue
					}
					rc, err := c.Blob.Get(ctx, ref)
					if err != nil {
						return err
					}
					raw, readErr := io.ReadAll(rc)
					rc.Close()
					if readErr != nil {
						return readErr
					}
					if err := c.Outbound.SendText(ctx, addr, string(raw)); err != nil {
						return err
					}
					continue
				}
				if err := c.Outbound.SendMedia(ctx, addr, ref, caption, nil); err != nil {
					return err
				}
			}
		} else if err := c.Outbound.SendText(ctx, addr, c.renderText(ctx, texttpl.KeyWorkflowDone, map[string]string{"task_id": string(n.TaskID)})); err != nil {
			return err
		}
		return c.Outbound.SendMenu(ctx, addr, c.renderText(ctx, texttpl.KeyWorkflowDoneFollowp, nil), nil)
	}
	if n.Kind == "session_terminated" {
		return c.Outbound.SendText(ctx, addr, c.renderText(ctx, texttpl.KeySessionTerminated, map[string]string{"error_msg": n.ErrorMsg}))
	}
	if n.Kind == "task_failed" || n.Kind == "task_cancelled" {
		template := texttpl.KeyTaskFailed
		if n.Kind == "task_cancelled" {
			template = texttpl.KeyTaskCancelled
		}
		return c.Outbound.SendText(ctx, addr, c.renderText(ctx, template, map[string]string{
			"task_id": string(n.TaskID), "status": strings.TrimPrefix(n.Kind, "task_"), "error_msg": n.ErrorMsg,
		}))
	}
	msg := fmt.Sprintf("任务 %s: %s", n.TaskID, n.Kind)
	if n.ErrorMsg != "" {
		msg += " — " + n.ErrorMsg
	}
	return c.Outbound.SendText(ctx, addr, msg)
}

func (c *Controller) renderText(ctx context.Context, key string, vars map[string]string) string {
	if c.Texts == nil {
		return texttpl.Render(texttpl.Default(key), vars)
	}
	return c.Texts.Render(ctx, c.channelID, key, vars)
}

func (c *Controller) backButton(nav protocol.Nav) []protocol.Button {
	if nav.Back == "" || nav.Back == "root" {
		return []protocol.Button{{Text: "返回主菜单", Data: c.Callbacks.MainMenu()}}
	}
	return []protocol.Button{{Text: "返回", Data: c.Callbacks.Back(nav.Back)}}
}

func mediaExtras(text string, rows [][]protocol.Button, first bool) (string, [][]protocol.Button) {
	if !first {
		return "", nil
	}
	return text, rows
}

func (c *Controller) invoke(ctx context.Context, in Inbound, inv protocol.CapabilityInvoke) error {
	if c == nil || c.invoker == nil {
		return fmt.Errorf("conversation: capability invoker not configured")
	}
	inv.Account.ChannelID = c.channelID
	inv.Account.ExternalUserID = in.ExternalUserID
	if c.Users != nil {
		id, err := c.Users.Resolve(ctx, in.Addr, identitydomain.UpsertFrom{
			ChannelID: c.channelID, ExternalUserID: in.ExternalUserID,
		})
		if err != nil {
			slog.Error("conversation resolve account", "err", err, "channel_id", c.channelID)
		} else {
			inv.Account.InternalUserID = id
		}
	}
	inv.ChatID = sharedkernel.FormatChatID(in.Addr)
	if inv.Nav.Back == "" {
		inv.Nav.Back = "root"
	}
	_, err := c.invoker.Invoke(ctx, inv)
	return err
}
