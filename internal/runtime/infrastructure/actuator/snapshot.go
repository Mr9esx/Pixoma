package actuator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// CaseSnapshot builds a submit-ready workflow by loading the Case template
// and injecting staged inputs from blob storage.
type CaseSnapshot struct {
	Tasks domain.TaskRepository
	Cases catalogdomain.Repository
	Blob  blob.Store
}

func (s *CaseSnapshot) WorkflowForTask(ctx context.Context, taskID sharedkernel.TaskID) (comfyui.Graph, error) {
	if s == nil || s.Tasks == nil || s.Cases == nil || s.Blob == nil {
		return nil, fmt.Errorf("actuator: CaseSnapshot not configured")
	}

	task, err := s.Tasks.Get(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("actuator: load task: %w", err)
	}

	c, err := s.Cases.Get(ctx, task.CaseID)
	if err != nil {
		return nil, fmt.Errorf("actuator: load case: %w", err)
	}

	if len(c.Document.Bindings.WorkflowJSON) == 0 {
		return nil, fmt.Errorf("actuator: empty workflow for case %s", c.Document.ID)
	}

	graph, err := deepCopyGraph(c.Document.Bindings.WorkflowJSON)
	if err != nil {
		return nil, fmt.Errorf("actuator: copy workflow: %w", err)
	}

	bindings := indexBindings(c.Document.Bindings.Inputs)
	for _, field := range c.Document.Inputs {
		val, ok, err := s.loadStaged(ctx, task.InputPrefix, field)
		if err != nil {
			return nil, err
		}
		if !ok {
			if field.Required {
				return nil, fmt.Errorf("actuator: missing required input %q", field.Key)
			}
			continue
		}

		b, ok := bindings[field.Key]
		if !ok {
			return nil, fmt.Errorf("actuator: missing binding for input %q", field.Key)
		}

		switch field.Type {
		case "image":
			// UploadImage + node write is task 2; binding presence is still enforced.
			continue
		case "string":
			if err := writeNodeInput(graph, b.NodeID, b.FieldPath, val.text); err != nil {
				return nil, err
			}
		case "number":
			if err := writeNodeInput(graph, b.NodeID, b.FieldPath, val.number); err != nil {
				return nil, err
			}
		case "boolean":
			if err := writeNodeInput(graph, b.NodeID, b.FieldPath, val.boolean); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("actuator: unsupported input type %q for %q", field.Type, field.Key)
		}
	}

	return graph, nil
}

type stagedValue struct {
	text    string
	number  float64
	boolean bool
}

func (s *CaseSnapshot) loadStaged(ctx context.Context, prefix string, field catalogdomain.InputField) (stagedValue, bool, error) {
	base := prefix + "/" + field.Key
	switch field.Type {
	case "string":
		raw, ok, err := s.readBlob(ctx, base+".txt")
		if err != nil || !ok {
			return stagedValue{}, ok, err
		}
		return stagedValue{text: string(raw)}, true, nil
	case "number":
		raw, ok, err := s.readBlob(ctx, base+".num")
		if err != nil || !ok {
			return stagedValue{}, ok, err
		}
		n, err := strconv.ParseFloat(string(raw), 64)
		if err != nil {
			return stagedValue{}, false, fmt.Errorf("actuator: parse number %q: %w", field.Key, err)
		}
		return stagedValue{number: n}, true, nil
	case "boolean":
		raw, ok, err := s.readBlob(ctx, base+".bool")
		if err != nil || !ok {
			return stagedValue{}, ok, err
		}
		b, err := strconv.ParseBool(string(raw))
		if err != nil {
			return stagedValue{}, false, fmt.Errorf("actuator: parse bool %q: %w", field.Key, err)
		}
		return stagedValue{boolean: b}, true, nil
	case "image":
		_, ok, err := s.readBlob(ctx, base+".blob.json")
		return stagedValue{}, ok, err
	default:
		return stagedValue{}, false, fmt.Errorf("actuator: unsupported input type %q for %q", field.Type, field.Key)
	}
}

func (s *CaseSnapshot) readBlob(ctx context.Context, key string) ([]byte, bool, error) {
	rc, err := s.Blob.Get(ctx, sharedkernel.BlobRef{Key: key})
	if err != nil {
		return nil, false, nil
	}
	defer rc.Close()
	raw, err := io.ReadAll(rc)
	if err != nil {
		return nil, false, fmt.Errorf("actuator: read %s: %w", key, err)
	}
	return raw, true, nil
}

func indexBindings(in []catalogdomain.InputBinding) map[string]catalogdomain.InputBinding {
	out := make(map[string]catalogdomain.InputBinding, len(in))
	for _, b := range in {
		out[b.Key] = b
	}
	return out
}

func deepCopyGraph(src map[string]any) (comfyui.Graph, error) {
	raw, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}
	var dst comfyui.Graph
	if err := json.Unmarshal(raw, &dst); err != nil {
		return nil, err
	}
	return dst, nil
}

func writeNodeInput(graph comfyui.Graph, nodeID, fieldPath string, value any) error {
	raw, ok := graph[nodeID]
	if !ok {
		return fmt.Errorf("actuator: node %q not found", nodeID)
	}
	node, ok := raw.(map[string]any)
	if !ok {
		return fmt.Errorf("actuator: node %q has invalid shape", nodeID)
	}
	inputsRaw, ok := node["inputs"]
	if !ok {
		inputs := map[string]any{}
		node["inputs"] = inputs
		inputs[fieldPath] = value
		return nil
	}
	inputs, ok := inputsRaw.(map[string]any)
	if !ok {
		return fmt.Errorf("actuator: node %q inputs has invalid shape", nodeID)
	}
	inputs[fieldPath] = value
	return nil
}

var _ CaseSnapshotProvider = (*CaseSnapshot)(nil)
