package application

import (
	"context"
	"fmt"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type UpdateFlowNodeInput struct {
	ID       string `json:"id"`
	Position struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	} `json:"position"`
	SortOrder int `json:"sort_order"`
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
