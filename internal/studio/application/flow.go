package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type UpdateFlowNodeInput struct {
	ID        string            `json:"id"`
	Position  FlowPositionInput `json:"position"`
	SortOrder int               `json:"sort_order"`
}

type FlowPositionInput struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type CreateFlowNodeInput struct {
	Type     domain.FlowNodeType `json:"type"`
	Title    string              `json:"title"`
	Body     string              `json:"body"`
	Position FlowPositionInput   `json:"position"`
}

type CreateFlowEdgeInput struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label"`
}

// CreateFlowNode adds a user-authored SOP node. Asset nodes remain owned by
// asset creation, so a manual node cannot impersonate a stored output.
func (s *Service) CreateFlowNode(ctx context.Context, accountID, sessionID string, input CreateFlowNodeInput) (*domain.FlowNode, error) {
	if s == nil || s.Repo == nil {
		return nil, fmt.Errorf("studio: repository is required")
	}
	if _, err := s.Repo.GetSession(ctx, accountID, sessionID); err != nil {
		return nil, err
	}
	if input.Type == domain.FlowNodeAsset {
		return nil, fmt.Errorf("%w: asset nodes are created from assets", domain.ErrInvalid)
	}
	nodes, _, err := s.Repo.GetFlow(ctx, accountID, sessionID)
	if err != nil {
		return nil, err
	}
	sortOrder := 0
	for _, existing := range nodes {
		if existing.SortOrder >= sortOrder {
			sortOrder = existing.SortOrder + 10
		}
	}
	node, err := domain.NewFlowNode(s.nextID(), sessionID, accountID, input.Type, strings.TrimSpace(input.Title), sortOrder, s.now())
	if err != nil {
		return nil, err
	}
	node.Body = strings.TrimSpace(input.Body)
	node.PositionX, node.PositionY = input.Position.X, input.Position.Y
	if err := s.Repo.SaveFlowNode(ctx, node); err != nil {
		return nil, err
	}
	return node, nil
}

func (s *Service) DeleteFlowNode(ctx context.Context, accountID, sessionID, nodeID string) error {
	if s == nil || s.Repo == nil {
		return fmt.Errorf("studio: repository is required")
	}
	if _, err := s.Repo.GetSession(ctx, accountID, sessionID); err != nil {
		return err
	}
	return s.Repo.DeleteFlowNode(ctx, accountID, sessionID, nodeID)
}

func (s *Service) CreateFlowEdge(ctx context.Context, accountID, sessionID string, input CreateFlowEdgeInput) (*domain.FlowEdge, error) {
	if s == nil || s.Repo == nil {
		return nil, fmt.Errorf("studio: repository is required")
	}
	if _, err := s.Repo.GetSession(ctx, accountID, sessionID); err != nil {
		return nil, err
	}
	nodes, edges, err := s.Repo.GetFlow(ctx, accountID, sessionID)
	if err != nil {
		return nil, err
	}
	known := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		known[node.ID] = struct{}{}
	}
	if _, ok := known[input.Source]; !ok {
		return nil, domain.ErrNotFound
	}
	if _, ok := known[input.Target]; !ok {
		return nil, domain.ErrNotFound
	}
	for _, existing := range edges {
		if existing.SourceNodeID == input.Source && existing.TargetNodeID == input.Target {
			return nil, fmt.Errorf("%w: flow edge already exists", domain.ErrInvalid)
		}
	}
	edge, err := domain.NewFlowEdge(s.nextID(), sessionID, accountID, input.Source, input.Target, s.now())
	if err != nil {
		return nil, err
	}
	edge.Label = strings.TrimSpace(input.Label)
	if err := s.Repo.SaveFlowEdge(ctx, edge); err != nil {
		return nil, err
	}
	return edge, nil
}

func (s *Service) DeleteFlowEdge(ctx context.Context, accountID, sessionID, edgeID string) error {
	if s == nil || s.Repo == nil {
		return fmt.Errorf("studio: repository is required")
	}
	if _, err := s.Repo.GetSession(ctx, accountID, sessionID); err != nil {
		return err
	}
	return s.Repo.DeleteFlowEdge(ctx, accountID, sessionID, edgeID)
}

// UpdateFlowNodes persists the user's ordering and layout edits. Agent-created
// metadata remains immutable here, so the asset road can be reorganized without
// accidentally rewriting the recorded production provenance.
func (s *Service) UpdateFlowNodes(ctx context.Context, accountID, sessionID string, updates []UpdateFlowNodeInput) error {
	if s == nil || s.Repo == nil {
		return fmt.Errorf("studio: repository is required")
	}
	if _, err := s.Repo.GetSession(ctx, accountID, sessionID); err != nil {
		return err
	}
	if len(updates) == 0 {
		return nil
	}
	nodes, _, err := s.Repo.GetFlow(ctx, accountID, sessionID)
	if err != nil {
		return err
	}
	byID := make(map[string]*domain.FlowNode, len(nodes))
	for _, node := range nodes {
		byID[node.ID] = node
	}
	now := s.now()
	for _, update := range updates {
		node := byID[update.ID]
		if node == nil || update.SortOrder < 0 {
			return domain.ErrNotFound
		}
		node.PositionX, node.PositionY, node.SortOrder, node.UpdatedAt = update.Position.X, update.Position.Y, update.SortOrder, now
		if err := s.Repo.SaveFlowNode(ctx, node); err != nil {
			return err
		}
	}
	return nil
}
