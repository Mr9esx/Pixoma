package application

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

func MigrateLegacyWorkflowOutputs(ctx context.Context, repo domain.Repository, accountID, sessionID string) error {
	nodes, edges, err := repo.GetFlow(ctx, accountID, sessionID)
	if err != nil {
		return err
	}
	byID := make(map[string]*domain.FlowNode, len(nodes))
	for _, node := range nodes {
		byID[node.ID] = node
	}
	executions, err := repo.ListSessionWorkflowExecutions(ctx, accountID, sessionID)
	if err != nil {
		return err
	}
	for _, execution := range executions {
		prefix := "workflow-output-node-" + execution.TaskID + "-"
		operation := byID[execution.OperationNodeID]
		for _, legacy := range nodes {
			if !strings.HasPrefix(legacy.ID, prefix) {
				continue
			}
			if operation == nil {
				return fmt.Errorf("studio: legacy workflow operation %s is missing", execution.OperationNodeID)
			}
			index, err := strconv.Atoi(strings.TrimPrefix(legacy.ID, prefix))
			if err != nil || index < 0 {
				return fmt.Errorf("%w: invalid legacy workflow output node %s", domain.ErrInvalid, legacy.ID)
			}
			asset, err := repo.GetAsset(ctx, accountID, legacy.AssetID)
			if err != nil {
				return fmt.Errorf("studio: read legacy workflow output asset %s: %w", legacy.AssetID, err)
			}
			portKey, outputType := fmt.Sprintf("out-%d", index), string(asset.Kind)
			if index < len(execution.OutputFields) {
				portKey, outputType = execution.OutputFields[index].Key, execution.OutputFields[index].Type
			}
			found := false
			for _, current := range operation.Outputs {
				if current.Key == portKey && current.AssetID == legacy.AssetID {
					found = true
					break
				}
			}
			if !found {
				operation.Outputs = append(operation.Outputs, domain.FlowOutput{Key: portKey, Type: outputType, Name: legacy.Title, AssetID: legacy.AssetID, AssetVersionID: legacy.AssetVersionID})
				if err := repo.SaveFlowNode(ctx, operation); err != nil {
					return fmt.Errorf("studio: save migrated workflow output: %w", err)
				}
			}
			for _, edge := range edges {
				if edge.SourceNodeID != legacy.ID {
					continue
				}
				edge.SourceNodeID, edge.SourceOutputKey = operation.ID, portKey
				if edge.TargetInputKey == "" {
					edge.Label = "使用产物"
				}
				if err := repo.SaveFlowEdge(ctx, edge); err != nil {
					return fmt.Errorf("studio: save migrated workflow relationship: %w", err)
				}
			}
			if err := repo.DeleteFlowNode(ctx, accountID, sessionID, legacy.ID); err != nil {
				return fmt.Errorf("studio: remove legacy workflow output node: %w", err)
			}
		}
	}
	return nil
}
