package linkhealth

import (
	"net/url"
	"strconv"
	"strings"

	mcdomain "github.com/mr9esx/comfyui_tgbot/internal/menucard/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/presence"
)

type Kind string

const (
	KindPlatform Kind = "platform"
	KindCase     Kind = "case"
	KindTopic    Kind = "topic"
	KindEdge     Kind = "edge"
)

type Health string

const (
	HealthOK      Health = "ok"
	HealthWarn    Health = "warn"
	HealthPending Health = "pending"
)

func NodeID(kind Kind, id string) string {
	return string(kind) + ":" + id
}

type Breakpoint struct {
	Stage  string            `json:"stage"`
	Fix    string            `json:"fix"`
	Key    string            `json:"key"`
	Params map[string]string `json:"params,omitempty"`
	Action Action            `json:"action"`
	Guide  string            `json:"guide"`
}

type Action struct {
	To  string `json:"to"`
	Key string `json:"key"`
}

type Ref struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State Health `json:"state"`
	To    string `json:"to"`
}

type Node struct {
	ID          string       `json:"id"`
	Kind        Kind         `json:"kind"`
	RefID       string       `json:"ref_id"`
	Name        string       `json:"name"`
	Health      Health       `json:"health"`
	Breakpoints []Breakpoint `json:"breakpoints"`
	Upstream    []Ref        `json:"upstream"`
	Downstream  []Ref        `json:"downstream"`
}

type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type Graph struct {
	Nodes []Node      `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

type ChannelSnap struct {
	ID               string
	Name             string
	Enabled          bool
	AdapterFound     bool
	AdapterState     string
	LastCheckKind    string
	LastCheckMessage string
}

type CaseSnap struct {
	ID      uint64
	Name    string
	Topics  []string
	Enabled bool
}

type TopicSnap struct {
	Key     string
	Name    string
	Enabled bool
}

type EdgeSnap struct {
	ID      string
	Name    string
	Enabled bool
	Topics  []string
}

type Snapshot struct {
	Channels      []ChannelSnap
	Menus         map[string]mcdomain.MenuTree
	Cases         []CaseSnap
	Topics        []TopicSnap
	Edges         []EdgeSnap
	Presence      map[string]presence.Snapshot
	AdapterKnown  bool
	PresenceKnown bool
}

type localState int

const (
	localOK localState = iota
	localWarn
	localPending
)

func Assemble(snap Snapshot) Graph {
	nodes := make(map[string]*Node)
	add := func(kind Kind, refID, name string) {
		id := NodeID(kind, refID)
		if _, ok := nodes[id]; ok {
			return
		}
		nodes[id] = &Node{
			ID:    id,
			Kind:  kind,
			RefID: refID,
			Name:  name,
		}
	}

	for _, ch := range snap.Channels {
		add(KindPlatform, ch.ID, ch.Name)
	}
	for _, c := range snap.Cases {
		add(KindCase, strconv.FormatUint(c.ID, 10), c.Name)
	}
	for _, tp := range snap.Topics {
		add(KindTopic, tp.Key, tp.Name)
	}
	for _, e := range snap.Edges {
		add(KindEdge, e.ID, e.Name)
	}

	fwd := map[string][]string{}
	rev := map[string][]string{}
	var edges []GraphEdge
	seenEdge := map[string]struct{}{}
	link := func(from, to string) {
		if nodes[from] == nil || nodes[to] == nil || from == to {
			return
		}
		key := from + "\x00" + to
		if _, ok := seenEdge[key]; ok {
			return
		}
		seenEdge[key] = struct{}{}
		fwd[from] = append(fwd[from], to)
		rev[to] = append(rev[to], from)
		edges = append(edges, GraphEdge{From: from, To: to})
	}

	caseIDs := map[string]struct{}{}
	for _, c := range snap.Cases {
		caseIDs[strconv.FormatUint(c.ID, 10)] = struct{}{}
	}
	topicIDs := map[string]struct{}{}
	for _, tp := range snap.Topics {
		topicIDs[tp.Key] = struct{}{}
	}

	for _, ch := range snap.Channels {
		tree, ok := snap.Menus[ch.ID]
		if !ok {
			continue
		}
		from := NodeID(KindPlatform, ch.ID)
		for _, p := range mcdomain.WalkWorkflowPlacements(tree) {
			if _, exists := caseIDs[p.WorkflowID]; !exists {
				continue
			}
			link(from, NodeID(KindCase, p.WorkflowID))
		}
	}
	for _, c := range snap.Cases {
		from := NodeID(KindCase, strconv.FormatUint(c.ID, 10))
		for _, key := range c.Topics {
			if _, exists := topicIDs[key]; !exists {
				continue
			}
			link(from, NodeID(KindTopic, key))
		}
	}
	for _, e := range snap.Edges {
		to := NodeID(KindEdge, e.ID)
		for _, key := range e.Topics {
			if _, exists := topicIDs[key]; !exists {
				continue
			}
			link(NodeID(KindTopic, key), to)
		}
	}

	platLocal := map[string]localState{}
	var usable, pendingPlat []string
	for _, ch := range snap.Channels {
		id := NodeID(KindPlatform, ch.ID)
		st := platformLocal(ch, snap.AdapterKnown)
		platLocal[id] = st
		switch st {
		case localOK:
			usable = append(usable, id)
		case localPending:
			pendingPlat = append(pendingPlat, id)
		}
	}

	edgeLocalMap := map[string]localState{}
	var ready, pendingEdge []string
	for _, e := range snap.Edges {
		id := NodeID(KindEdge, e.ID)
		st := edgeLocal(e, snap)
		edgeLocalMap[id] = st
		switch st {
		case localOK:
			ready = append(ready, id)
		case localPending:
			pendingEdge = append(pendingEdge, id)
		}
	}

	caseEnabled := map[string]bool{}
	for _, c := range snap.Cases {
		caseEnabled[NodeID(KindCase, strconv.FormatUint(c.ID, 10))] = c.Enabled
	}
	topicEnabled := map[string]bool{}
	for _, tp := range snap.Topics {
		topicEnabled[NodeID(KindTopic, tp.Key)] = tp.Enabled
	}
	hopOpen := map[string]bool{}
	for id, n := range nodes {
		switch n.Kind {
		case KindCase:
			hopOpen[id] = caseEnabled[id]
		case KindTopic:
			hopOpen[id] = topicEnabled[id]
		default:
			hopOpen[id] = true
		}
	}

	liveFwd := filterAdj(fwd, hopOpen)
	liveRev := filterAdj(rev, hopOpen)
	live := intersect(walk(usable, liveFwd), walk(ready, liveRev))
	maybeStarts := append(append([]string{}, usable...), pendingPlat...)
	maybeEnds := append(append([]string{}, ready...), pendingEdge...)
	maybe := intersect(walk(maybeStarts, liveFwd), walk(maybeEnds, liveRev))

	healthOf := map[string]Health{}
	for id, n := range nodes {
		selfOK := true
		switch n.Kind {
		case KindCase:
			selfOK = caseEnabled[id]
		case KindTopic:
			selfOK = topicEnabled[id]
		}
		healthOf[id] = displayHealth(n.Kind, platLocal[id], edgeLocalMap[id], live[id], maybe[id], selfOK)
		n.Health = healthOf[id]
	}

	toPath := func(kind Kind, refID string) string {
		switch kind {
		case KindPlatform:
			return "/channels/" + url.PathEscape(refID)
		case KindCase:
			return "/cases/" + url.PathEscape(refID)
		case KindTopic:
			return "/topics/" + url.PathEscape(refID)
		default:
			return "/edges/" + url.PathEscape(refID)
		}
	}
	toRef := func(id string) Ref {
		n := nodes[id]
		return Ref{ID: id, Name: n.Name, State: healthOf[id], To: toPath(n.Kind, n.RefID)}
	}

	out := Graph{Edges: edges}
	appendNodes := func(kind Kind, ids []string) {
		for _, refID := range ids {
			n := nodes[NodeID(kind, refID)]
			if n == nil {
				continue
			}
			for _, up := range rev[n.ID] {
				n.Upstream = append(n.Upstream, toRef(up))
			}
			for _, down := range fwd[n.ID] {
				n.Downstream = append(n.Downstream, toRef(down))
			}
			n.Breakpoints = breakpoints(n, snap, platLocal, edgeLocalMap)
			out.Nodes = append(out.Nodes, *n)
		}
	}
	var platIDs, caseIDsOrd, topicKeys, edgeIDs []string
	for _, ch := range snap.Channels {
		platIDs = append(platIDs, ch.ID)
	}
	for _, c := range snap.Cases {
		caseIDsOrd = append(caseIDsOrd, strconv.FormatUint(c.ID, 10))
	}
	for _, tp := range snap.Topics {
		topicKeys = append(topicKeys, tp.Key)
	}
	for _, e := range snap.Edges {
		edgeIDs = append(edgeIDs, e.ID)
	}
	appendNodes(KindPlatform, platIDs)
	appendNodes(KindCase, caseIDsOrd)
	appendNodes(KindTopic, topicKeys)
	appendNodes(KindEdge, edgeIDs)
	return out
}

func platformLocal(ch ChannelSnap, adapterKnown bool) localState {
	if !adapterKnown || ch.LastCheckKind == "" {
		return localPending
	}
	if ch.Enabled && ch.AdapterFound && ch.AdapterState == "running" && ch.LastCheckKind == "ok" {
		return localOK
	}
	return localWarn
}

func edgeLocal(e EdgeSnap, snap Snapshot) localState {
	if !snap.PresenceKnown {
		return localPending
	}
	if !e.Enabled {
		return localWarn
	}
	p := snap.Presence[e.ID]
	if p.EdgeOnline && p.ComfyRunning {
		return localOK
	}
	return localWarn
}

func displayHealth(kind Kind, plat, edge localState, onLive, onMaybe, selfOK bool) Health {
	switch kind {
	case KindPlatform:
		switch plat {
		case localPending:
			return HealthPending
		case localWarn:
			return HealthWarn
		default:
			return pathHealth(onLive, onMaybe)
		}
	case KindEdge:
		switch edge {
		case localPending:
			return HealthPending
		case localWarn:
			return HealthWarn
		default:
			return pathHealth(onLive, onMaybe)
		}
	default:
		h := pathHealth(onLive, onMaybe)
		if h == HealthOK && !selfOK {
			return HealthWarn
		}
		return h
	}
}

func pathHealth(onLive, onMaybe bool) Health {
	if onLive {
		return HealthOK
	}
	if onMaybe {
		return HealthPending
	}
	return HealthWarn
}

func filterAdj(adj map[string][]string, open map[string]bool) map[string][]string {
	out := map[string][]string{}
	for from, tos := range adj {
		if !open[from] {
			continue
		}
		for _, to := range tos {
			if !open[to] {
				continue
			}
			out[from] = append(out[from], to)
		}
	}
	return out
}

func walk(starts []string, adj map[string][]string) map[string]bool {
	seen := map[string]bool{}
	q := make([]string, 0, len(starts))
	for _, s := range starts {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		q = append(q, s)
	}
	for len(q) > 0 {
		cur := q[0]
		q = q[1:]
		for _, n := range adj[cur] {
			if seen[n] {
				continue
			}
			seen[n] = true
			q = append(q, n)
		}
	}
	return seen
}

func intersect(a, b map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k := range a {
		if b[k] {
			out[k] = true
		}
	}
	return out
}

func breakpoints(n *Node, snap Snapshot, platLocal, edgeLocalMap map[string]localState) []Breakpoint {
	if n.Health == HealthOK || n.Health == HealthPending {
		return nil
	}
	switch n.Kind {
	case KindPlatform:
		return platformBreakpoints(n.RefID, snap)
	case KindCase:
		return caseBreakpoints(n, platLocal)
	case KindTopic:
		return topicBreakpoints(n, edgeLocalMap)
	case KindEdge:
		return edgeBreakpoints(n, snap, edgeLocalMap)
	default:
		return nil
	}
}

func platformBreakpoints(id string, snap Snapshot) []Breakpoint {
	ch, ok := channelByID(snap, id)
	if !ok {
		return nil
	}
	to := "/channels/" + url.PathEscape(id)
	switch ch.LastCheckKind {
	case "network":
		return []Breakpoint{bp("entry", "runtime", "linkHealth.channelUnreachable", nil, "/settings", "linkHealth.actionConfigureProxy")}
	case "auth":
		return []Breakpoint{bp("entry", "config", "linkHealth.channelTokenInvalid", nil, to, "linkHealth.actionEditChannel")}
	case "other":
		return []Breakpoint{bp("entry", "runtime", "linkHealth.channelCheckFailed", map[string]string{"message": ch.LastCheckMessage}, "/settings", "linkHealth.actionConfigureProxy")}
	}
	if !ch.Enabled || !ch.AdapterFound || ch.AdapterState != "running" {
		return []Breakpoint{bp("entry", "runtime", "linkHealth.entryChannelUnusable", map[string]string{"channel": id}, "/channels", "linkHealth.actionManageChannels")}
	}
	return nil
}

func caseBreakpoints(n *Node, platLocal map[string]localState) []Breakpoint {
	var out []Breakpoint
	if len(n.Upstream) == 0 {
		out = append(out, bp("entry", "config", "linkHealth.noMenuEntry", nil, "/channels", "linkHealth.actionAddEntry"))
	}
	for _, up := range n.Upstream {
		if platLocal[up.ID] == localWarn {
			out = append(out, bp("entry", "runtime", "linkHealth.entryChannelUnusable", map[string]string{"channel": nodesRefID(up.ID)}, "/channels", "linkHealth.actionManageChannels"))
		}
	}
	if len(n.Downstream) == 0 {
		out = append(out, bp("topic", "config", "linkHealth.noCaseRoutes", nil, n.path(), "linkHealth.actionConfigureRouting"))
	}
	for _, down := range n.Downstream {
		if down.State != HealthOK {
			out = append(out, bp("node", "runtime", "linkHealth.topicNoReadyNode", map[string]string{"topic": nodesRefID(down.ID)}, "/edges", "linkHealth.actionManageNodes"))
		}
	}
	return out
}

func topicBreakpoints(n *Node, edgeLocalMap map[string]localState) []Breakpoint {
	var out []Breakpoint
	if len(n.Upstream) == 0 {
		out = append(out, bp("workflow", "config", "linkHealth.noCaseRoutes", nil, "/cases", "linkHealth.actionConfigureRouting"))
	}
	if len(n.Downstream) == 0 {
		out = append(out, bp("node", "config", "linkHealth.noEdgeSubscribers", nil, "/edges", "linkHealth.actionBindTopic"))
		return out
	}
	ready := 0
	for _, down := range n.Downstream {
		if edgeLocalMap[down.ID] == localOK {
			ready++
		}
	}
	if ready == 0 {
		out = append(out, bp("node", "runtime", "linkHealth.subscribersOffline", nil, "/edges", "linkHealth.actionManageNodes"))
	}
	return out
}

func edgeBreakpoints(n *Node, snap Snapshot, edgeLocalMap map[string]localState) []Breakpoint {
	var out []Breakpoint
	if len(n.Upstream) == 0 {
		out = append(out, bp("topic", "config", "linkHealth.noTopicBinding", nil, n.path(), "linkHealth.actionDeployNode"))
	} else if !edgeHasCase(n.RefID, snap) {
		out = append(out, bp("workflow", "config", "linkHealth.noCaseReachable", nil, "/cases", "linkHealth.actionConfigureRouting"))
	}
	if edgeLocalMap[n.ID] != localOK {
		out = append(out, bp("node", "runtime", "linkHealth.edgeNotReady", nil, n.path(), "linkHealth.actionCheckNode"))
	}
	return out
}

func edgeHasCase(edgeID string, snap Snapshot) bool {
	var topics []string
	for _, e := range snap.Edges {
		if e.ID == edgeID {
			topics = e.Topics
			break
		}
	}
	for _, c := range snap.Cases {
		for _, key := range c.Topics {
			for _, want := range topics {
				if key == want {
					return true
				}
			}
		}
	}
	return false
}

func (n *Node) path() string {
	switch n.Kind {
	case KindPlatform:
		return "/channels/" + url.PathEscape(n.RefID)
	case KindCase:
		return "/cases/" + url.PathEscape(n.RefID)
	case KindTopic:
		return "/topics/" + url.PathEscape(n.RefID)
	default:
		return "/edges/" + url.PathEscape(n.RefID)
	}
}

func bp(stage, fix, key string, params map[string]string, to, actionKey string) Breakpoint {
	rest := strings.TrimPrefix(key, "linkHealth.")
	guide := key
	if rest != key && rest != "" {
		guide = "linkHealth.guide" + strings.ToUpper(rest[:1]) + rest[1:]
	}
	return Breakpoint{
		Stage:  stage,
		Fix:    fix,
		Key:    key,
		Params: params,
		Action: Action{To: to, Key: actionKey},
		Guide:  guide,
	}
}

func channelByID(snap Snapshot, id string) (ChannelSnap, bool) {
	for _, ch := range snap.Channels {
		if ch.ID == id {
			return ch, true
		}
	}
	return ChannelSnap{}, false
}

func nodesRefID(nodeID string) string {
	_, id, ok := splitNodeID(nodeID)
	if !ok {
		return nodeID
	}
	return id
}

func splitNodeID(id string) (Kind, string, bool) {
	kind, ref, ok := strings.Cut(id, ":")
	if !ok {
		return "", "", false
	}
	return Kind(kind), ref, true
}
