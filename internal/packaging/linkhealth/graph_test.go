package linkhealth

import (
	"testing"

	mcdomain "github.com/mr9esx/comfyui_tgbot/internal/menucard/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/presence"
)

func openWorkflowMenu(workflowID string) mcdomain.MenuTree {
	return mcdomain.MenuTree{
		ID:      "menu",
		Columns: 1,
		Items: []mcdomain.TreeButton{{
			ID:    "btn",
			Label: "run",
			Action: mcdomain.TreeAction{
				Type:       "open_workflow",
				WorkflowID: workflowID,
			},
		}},
	}
}

func liveChain(platforms []ChannelSnap, edgeReady bool) Snapshot {
	snap := Snapshot{
		AdapterKnown:  true,
		PresenceKnown: true,
		Channels:      platforms,
		Menus:         map[string]mcdomain.MenuTree{},
		Cases: []CaseSnap{{
			ID:      10,
			Name:    "海报",
			Topics:  []string{"jobs"},
			Enabled: true,
		}},
		Topics: []TopicSnap{{Key: "jobs", Name: "出图", Enabled: true}},
		Edges: []EdgeSnap{{
			ID:      "e1",
			Name:    "机房-1",
			Enabled: true,
			Topics:  []string{"jobs"},
		}},
		Presence: map[string]presence.Snapshot{
			"e1": {ID: "e1", EdgeOnline: edgeReady, ComfyRunning: edgeReady},
		},
	}
	for _, p := range platforms {
		snap.Menus[p.ID] = openWorkflowMenu("10")
	}
	return snap
}

func usablePlatform(id, name string) ChannelSnap {
	return ChannelSnap{
		ID:            id,
		Name:          name,
		Enabled:       true,
		AdapterFound:  true,
		AdapterState:  "running",
		LastCheckKind: "ok",
	}
}

func unusablePlatform(id, name string) ChannelSnap {
	p := usablePlatform(id, name)
	p.LastCheckKind = "network"
	p.LastCheckMessage = "i/o timeout"
	return p
}

func TestAssembleLivePathInvariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		snap Snapshot
		want map[string]Health
	}{
		{
			name: "唯一入口不可用则工作流不得 ok",
			snap: liveChain([]ChannelSnap{unusablePlatform("tg", "电报")}, true),
			want: map[string]Health{
				"platform:tg": HealthWarn,
				"case:10":     HealthWarn,
				"topic:jobs":  HealthWarn,
				"edge:e1":     HealthWarn,
			},
		},
		{
			name: "双入口有活路则工作流 ok、坏入口只在引用里 warn",
			snap: liveChain([]ChannelSnap{
				unusablePlatform("tg", "电报"),
				usablePlatform("fs", "飞书"),
			}, true),
			want: map[string]Health{
				"platform:tg": HealthWarn,
				"platform:fs": HealthOK,
				"case:10":     HealthOK,
				"topic:jobs":  HealthOK,
				"edge:e1":     HealthOK,
			},
		},
		{
			name: "唯一节点不可用则工作流与队列不得 ok",
			snap: liveChain([]ChannelSnap{usablePlatform("tg", "电报")}, false),
			want: map[string]Health{
				"platform:tg": HealthWarn,
				"case:10":     HealthWarn,
				"topic:jobs":  HealthWarn,
				"edge:e1":     HealthWarn,
			},
		},
		{
			name: "从未探测但适配器在跑则活路仍可 ok",
			snap: liveChain([]ChannelSnap{{
				ID:           "tg",
				Name:         "电报",
				Enabled:      true,
				AdapterFound: true,
				AdapterState: "running",
			}}, true),
			want: map[string]Health{
				"platform:tg": HealthOK,
				"case:10":     HealthOK,
				"topic:jobs":  HealthOK,
				"edge:e1":     HealthOK,
			},
		},
		{
			name: "适配器读不到则 pending 不得 ok",
			snap: func() Snapshot {
				s := liveChain([]ChannelSnap{{
					ID:           "tg",
					Name:         "电报",
					Enabled:      true,
					AdapterFound: true,
					AdapterState: "running",
				}}, true)
				s.AdapterKnown = false
				return s
			}(),
			want: map[string]Health{
				"platform:tg": HealthPending,
				"case:10":     HealthPending,
				"topic:jobs":  HealthPending,
				"edge:e1":     HealthPending,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := Assemble(tt.snap)
			got := map[string]Health{}
			for _, n := range g.Nodes {
				got[n.ID] = n.Health
			}
			for id, want := range tt.want {
				if got[id] != want {
					t.Errorf("%s: got %s, want %s", id, got[id], want)
				}
			}
			if got["case:10"] == HealthOK && tt.want["platform:tg"] == HealthWarn {
				caseNode := node(t, g, "case:10")
				if refState(caseNode.Upstream, "platform:tg") != HealthWarn {
					t.Errorf("case upstream platform:tg want warn, got %+v", caseNode.Upstream)
				}
				if refState(caseNode.Upstream, "platform:fs") != HealthOK {
					t.Errorf("case upstream platform:fs want ok, got %+v", caseNode.Upstream)
				}
			}
		})
	}
}

func TestAssemblePendingExposesBreakpoints(t *testing.T) {
	t.Parallel()
	snap := liveChain([]ChannelSnap{{
		ID:           "tg",
		Name:         "电报",
		Enabled:      true,
		AdapterFound: true,
		AdapterState: "running",
	}}, true)
	snap.AdapterKnown = false
	g := Assemble(snap)
	for _, id := range []string{"platform:tg", "case:10", "topic:jobs", "edge:e1"} {
		n := node(t, g, id)
		if n.Health != HealthPending {
			t.Fatalf("%s health=%s, want pending", id, n.Health)
		}
		if len(n.Breakpoints) == 0 {
			t.Fatalf("%s pending with no breakpoints", id)
		}
	}
	plat := node(t, g, "platform:tg")
	if plat.Breakpoints[0].Key != "linkHealth.pathNotReady" {
		t.Fatalf("platform breakpoint=%q", plat.Breakpoints[0].Key)
	}
}

func TestAssembleDisabledUniqueTopicBlocksLivePath(t *testing.T) {
	t.Parallel()
	snap := liveChain([]ChannelSnap{usablePlatform("tg", "电报")}, true)
	snap.Topics[0].Enabled = false
	g := Assemble(snap)
	got := map[string]Health{}
	for _, n := range g.Nodes {
		got[n.ID] = n.Health
	}
	if got["topic:jobs"] == HealthOK || got["case:10"] == HealthOK || got["edge:e1"] == HealthOK {
		t.Fatalf("disabled unique topic must not keep a live path: %+v", got)
	}
}

func node(t *testing.T, g Graph, id string) Node {
	t.Helper()
	for _, n := range g.Nodes {
		if n.ID == id {
			return n
		}
	}
	t.Fatalf("missing node %s", id)
	return Node{}
}

func refState(refs []Ref, id string) Health {
	for _, r := range refs {
		if r.ID == id {
			return r.State
		}
	}
	return ""
}
