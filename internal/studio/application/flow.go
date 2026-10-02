package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

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
	Source          string `json:"source"`
	Target          string `json:"target"`
	Label           string `json:"label"`
	SourceOutputKey string `json:"source_output_key"`
	TargetInputKey  string `json:"target_input_key"`
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
	executions, err := s.Repo.ListSessionWorkflowExecutions(ctx, accountID, sessionID)
	if err != nil {
		return err
	}
	for _, execution := range executions {
		if execution.OperationNodeID == nodeID {
			return fmt.Errorf("%w: workflow execution nodes cannot be deleted", domain.ErrInvalid)
		}
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
	known := make(map[string]*domain.FlowNode, len(nodes))
	for _, node := range nodes {
		known[node.ID] = node
	}
	source := known[input.Source]
	if source == nil {
		return nil, domain.ErrNotFound
	}
	if known[input.Target] == nil {
		return nil, domain.ErrNotFound
	}
	var output *domain.FlowOutput
	for i := range source.Outputs {
		if source.Outputs[i].Key == input.SourceOutputKey {
			output = &source.Outputs[i]
			break
		}
	}
	if (len(source.Outputs) > 0 && output == nil) || (len(source.Outputs) == 0 && input.SourceOutputKey != "") {
		return nil, fmt.Errorf("%w: unknown workflow output", domain.ErrInvalid)
	}
	executions, err := s.Repo.ListSessionWorkflowExecutions(ctx, accountID, sessionID)
	if err != nil {
		return nil, err
	}
	var field *domain.FlowPort
	var targetExecution *domain.WorkflowExecution
	for _, execution := range executions {
		if execution.OperationNodeID == input.Target {
			targetExecution = execution
			break
		}
	}
	if targetExecution != nil {
		for i := range targetExecution.InputFields {
			if targetExecution.InputFields[i].Key == input.TargetInputKey {
				field = &targetExecution.InputFields[i]
				break
			}
		}
	}
	if (targetExecution != nil && len(targetExecution.InputFields) > 0 && field == nil) || (field == nil && input.TargetInputKey != "") {
		return nil, fmt.Errorf("%w: unknown workflow input", domain.ErrInvalid)
	}
	if (output == nil) != (field == nil) {
		if field == nil || source.Type != domain.FlowNodeAsset || source.AssetID == "" || source.AssetVersionID == "" {
			return nil, fmt.Errorf("%w: workflow relationship requires matching ports", domain.ErrInvalid)
		}
	}
	if output != nil && field != nil && output.Type != field.Type {
		return nil, fmt.Errorf("%w: incompatible workflow ports", domain.ErrInvalid)
	}
	if field != nil {
		assetID, versionID := source.AssetID, source.AssetVersionID
		if output != nil {
			assetID, versionID = output.AssetID, output.AssetVersionID
		} else {
			asset, err := s.Repo.GetAsset(ctx, accountID, assetID)
			if err != nil {
				return nil, err
			}
			if string(asset.Kind) != field.Type {
				return nil, fmt.Errorf("%w: incompatible workflow asset", domain.ErrInvalid)
			}
		}
		matched := false
		for _, item := range targetExecution.Inputs {
			if item.Key == field.Key && item.AssetID == assetID && item.AssetVersionID == versionID {
				matched = true
				break
			}
		}
		if !matched {
			return nil, fmt.Errorf("%w: workflow input did not use this output", domain.ErrInvalid)
		}
	}
	for _, existing := range edges {
		if input.TargetInputKey != "" && existing.TargetNodeID == input.Target && existing.TargetInputKey == input.TargetInputKey {
			return nil, fmt.Errorf("%w: workflow input already connected", domain.ErrInvalid)
		}
		if existing.SourceNodeID == input.Source && existing.TargetNodeID == input.Target && existing.SourceOutputKey == input.SourceOutputKey && existing.TargetInputKey == input.TargetInputKey {
			return nil, fmt.Errorf("%w: flow edge already exists", domain.ErrInvalid)
		}
	}
	edge, err := domain.NewFlowEdge(s.nextID(), sessionID, accountID, input.Source, input.Target, s.now())
	if err != nil {
		return nil, err
	}
	edge.Label = strings.TrimSpace(input.Label)
	edge.SourceOutputKey, edge.TargetInputKey = input.SourceOutputKey, input.TargetInputKey
	if err := s.Repo.SaveFlowEdge(ctx, edge); err != nil {
		return nil, err
	}
	return edge, nil
}

type flowRelationshipRepository interface {
	GetFlow(context.Context, string, string) ([]*domain.FlowNode, []*domain.FlowEdge, error)
	SaveFlowEdge(context.Context, *domain.FlowEdge) error
}

func RecordWorkflowInputEdges(ctx context.Context, repo flowRelationshipRepository, execution *domain.WorkflowExecution, now time.Time) error {
	nodes, edges, err := repo.GetFlow(ctx, execution.AccountID, execution.SessionID)
	if err != nil {
		return err
	}
	for _, input := range execution.Inputs {
		if input.AssetVersionID == "" {
			continue
		}
		for _, source := range nodes {
			if source.ID == execution.OperationNodeID {
				continue
			}
			for _, output := range source.Outputs {
				if output.AssetID != input.AssetID || output.AssetVersionID != input.AssetVersionID {
					continue
				}
				alreadyConnected := false
				for _, edge := range edges {
					if edge.TargetNodeID == execution.OperationNodeID && edge.TargetInputKey == input.Key {
						if edge.SourceNodeID != source.ID || edge.SourceOutputKey != output.Key {
							return fmt.Errorf("%w: workflow input has a different source", domain.ErrInvalid)
						}
						alreadyConnected = true
						break
					}
				}
				if alreadyConnected {
					break
				}
				edgeID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(execution.ID+"\x00flow-input\x00"+input.Key)).String()
				edge, err := domain.NewFlowEdge(edgeID, execution.SessionID, execution.AccountID, source.ID, execution.OperationNodeID, now)
				if err != nil {
					return err
				}
				edge.SourceOutputKey, edge.TargetInputKey = output.Key, input.Key
				if err := repo.SaveFlowEdge(ctx, edge); err != nil {
					return err
				}
				edges = append(edges, edge)
				break
			}
		}
	}
	return nil
}

func (s *Service) DeleteFlowEdge(ctx context.Context, accountID, sessionID, edgeID string) error {
	if s == nil || s.Repo == nil {
		return fmt.Errorf("studio: repository is required")
	}
	if _, err := s.Repo.GetSession(ctx, accountID, sessionID); err != nil {
		return err
	}
	_, edges, err := s.Repo.GetFlow(ctx, accountID, sessionID)
	if err != nil {
		return err
	}
	for _, edge := range edges {
		if edge.ID == edgeID && edge.TargetInputKey != "" {
			return fmt.Errorf("%w: workflow input edges cannot be deleted", domain.ErrInvalid)
		}
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
