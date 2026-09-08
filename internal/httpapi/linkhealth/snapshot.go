package linkhealth

import (
	"context"
	"errors"
	"fmt"

	catalogdomain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	channelapp "github.com/Mr9esx/Pixoma/internal/channels/application"
	channeldomain "github.com/Mr9esx/Pixoma/internal/channels/domain"
	mcdomain "github.com/Mr9esx/Pixoma/internal/menus/domain"
	"github.com/Mr9esx/Pixoma/internal/packaging/linkhealth"
	edge "github.com/Mr9esx/Pixoma/internal/edge/domain"
	"github.com/Mr9esx/Pixoma/internal/edge/infrastructure/presence"
	topicdomain "github.com/Mr9esx/Pixoma/internal/topics/domain"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	"gorm.io/gorm"
)

type Source struct {
	Channels channelapp.Repository
	Adapter  func(ctx context.Context, id string) (state string, lastErr string, found bool)
	Menus    MenuTreeReader
	Cases    catalogdomain.Repository
	Topics   topicdomain.Repository
	Edges    edge.Repository
	Presence *presence.Store
}

type MenuTreeReader interface {
	GetTree(ctx context.Context, channelID string) (mcdomain.MenuTree, error)
}

func Collect(ctx context.Context, src Source) (linkhealth.Snapshot, error) {
	if src.Channels == nil || src.Menus == nil || src.Cases == nil || src.Topics == nil || src.Edges == nil {
		return linkhealth.Snapshot{}, fmt.Errorf("linkhealth: incomplete source")
	}
	channels, err := src.Channels.List(ctx)
	if err != nil {
		return linkhealth.Snapshot{}, fmt.Errorf("linkhealth: list channels: %w", err)
	}
	cases, err := src.Cases.List(ctx, catalogdomain.ListQuery{})
	if err != nil {
		return linkhealth.Snapshot{}, fmt.Errorf("linkhealth: list cases: %w", err)
	}
	topics, err := src.Topics.List(ctx, nil)
	if err != nil {
		return linkhealth.Snapshot{}, fmt.Errorf("linkhealth: list topics: %w", err)
	}
	edges, err := src.Edges.List(ctx)
	if err != nil {
		return linkhealth.Snapshot{}, fmt.Errorf("linkhealth: list edges: %w", err)
	}

	snap := linkhealth.Snapshot{
		AdapterKnown:  src.Adapter != nil,
		PresenceKnown: src.Presence != nil,
		Menus:         map[string]mcdomain.MenuTree{},
		Presence:      map[string]presence.Snapshot{},
	}
	for _, ch := range channels {
		row := channelSnap(ch)
		if src.Adapter != nil {
			state, _, found := src.Adapter(ctx, ch.ID)
			row.AdapterFound = found
			row.AdapterState = state
		}
		snap.Channels = append(snap.Channels, row)
		tree, err := src.Menus.GetTree(ctx, ch.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return linkhealth.Snapshot{}, fmt.Errorf("linkhealth: menu %s: %w", ch.ID, err)
		}
		snap.Menus[ch.ID] = tree
	}
	for _, c := range cases {
		if c == nil {
			continue
		}
		snap.Cases = append(snap.Cases, linkhealth.CaseSnap{
			ID:      uint64(c.Document.ID),
			Name:    c.Document.Name,
			Topics:  routingTopics(c),
			Enabled: c.Enabled,
		})
	}
	for _, tp := range topics {
		name := tp.Name
		if name == "" {
			name = tp.Key
		}
		snap.Topics = append(snap.Topics, linkhealth.TopicSnap{Key: tp.Key, Name: name, Enabled: tp.Enabled})
	}
	for _, e := range edges {
		if e == nil {
			continue
		}
		id := string(e.ID)
		snap.Edges = append(snap.Edges, linkhealth.EdgeSnap{
			ID:      id,
			Name:    e.Name,
			Enabled: e.Enabled,
			Topics:  e.EffectiveTopics(),
		})
		if src.Presence != nil {
			snap.Presence[id] = src.Presence.Snapshot(sharedkernel.EdgeID(id))
		}
	}
	return snap, nil
}

func channelSnap(ch channeldomain.Channel) linkhealth.ChannelSnap {
	return linkhealth.ChannelSnap{
		ID:               ch.ID,
		Name:             ch.Name,
		Enabled:          ch.Enabled,
		LastCheckKind:    ch.LastCheckKind,
		LastCheckMessage: ch.LastCheckMessage,
	}
}

func routingTopics(c *catalogdomain.Case) []string {
	if c.Document.Routing == nil {
		return nil
	}
	raw := make([]string, 0, len(c.Document.Routing.Rules))
	for _, rule := range c.Document.Routing.Rules {
		raw = append(raw, rule.Topic)
	}
	return topicdomain.NormalizeTopics(raw)
}
