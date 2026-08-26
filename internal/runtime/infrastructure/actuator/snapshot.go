package actuator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"

	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// ImageUploader uploads local image bytes to ComfyUI and returns the remote filename.
type ImageUploader interface {
	UploadImage(ctx context.Context, filename, mime string, data []byte) (string, error)
}

// CaseSnapshot builds a submit-ready workflow by loading the Case template
// and injecting staged inputs from blob storage.
type CaseSnapshot struct {
	Tasks    domain.TaskRepository
	Cases    catalogdomain.Repository
	Blob     blob.Store
	Uploader ImageUploader
}

func (s *CaseSnapshot) WorkflowForTask(ctx context.Context, taskID sharedkernel.TaskID, uploader ImageUploader) (comfyui.Graph, error) {
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
		return nil, fmt.Errorf("actuator: empty workflow for case %d", c.Document.ID)
	}

	graph, err := deepCopyGraph(c.Document.Bindings.WorkflowJSON)
	if err != nil {
		return nil, fmt.Errorf("actuator: copy workflow: %w", err)
	}
	graph = stripAnnotationNodes(graph)

	// Prefer per-dispatch uploader; fall back to struct field. Never mutate s.Uploader.
	upload := uploader
	if upload == nil {
		upload = s.Uploader
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
			remote, err := s.uploadStagedImage(ctx, upload, val.blob)
			if err != nil {
				return nil, err
			}
			if err := writeNodeInput(graph, b.NodeID, b.FieldPath, remote); err != nil {
				return nil, err
			}
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

// BuildJobPackage assembles a scheme-A job: non-image fields injected into workflow;
// images listed as BlobRefs for the executor to upload locally. The package is
// edge-agnostic: the claiming node is chosen after prep.
func (s *CaseSnapshot) BuildJobPackage(ctx context.Context, taskID sharedkernel.TaskID) (JobPackage, error) {
	if s == nil || s.Tasks == nil || s.Cases == nil || s.Blob == nil {
		return JobPackage{}, fmt.Errorf("actuator: CaseSnapshot not configured")
	}

	task, err := s.Tasks.Get(ctx, taskID)
	if err != nil {
		return JobPackage{}, fmt.Errorf("actuator: load task: %w", err)
	}

	c, err := s.Cases.Get(ctx, task.CaseID)
	if err != nil {
		return JobPackage{}, fmt.Errorf("actuator: load case: %w", err)
	}

	if len(c.Document.Bindings.WorkflowJSON) == 0 {
		return JobPackage{}, fmt.Errorf("actuator: empty workflow for case %d", c.Document.ID)
	}

	graph, err := deepCopyGraph(c.Document.Bindings.WorkflowJSON)
	if err != nil {
		return JobPackage{}, fmt.Errorf("actuator: copy workflow: %w", err)
	}
	graph = stripAnnotationNodes(graph)

	var images []JobImage
	bindings := indexBindings(c.Document.Bindings.Inputs)
	for _, field := range c.Document.Inputs {
		val, ok, err := s.loadStaged(ctx, task.InputPrefix, field)
		if err != nil {
			return JobPackage{}, err
		}
		if !ok {
			if field.Required {
				return JobPackage{}, fmt.Errorf("actuator: missing required input %q", field.Key)
			}
			continue
		}

		b, ok := bindings[field.Key]
		if !ok {
			return JobPackage{}, fmt.Errorf("actuator: missing binding for input %q", field.Key)
		}

		switch field.Type {
		case "image":
			images = append(images, JobImage{NodeID: b.NodeID, FieldPath: b.FieldPath, Blob: val.blob})
		case "string":
			if err := writeNodeInput(graph, b.NodeID, b.FieldPath, val.text); err != nil {
				return JobPackage{}, err
			}
		case "number":
			if err := writeNodeInput(graph, b.NodeID, b.FieldPath, val.number); err != nil {
				return JobPackage{}, err
			}
		case "boolean":
			if err := writeNodeInput(graph, b.NodeID, b.FieldPath, val.boolean); err != nil {
				return JobPackage{}, err
			}
		default:
			return JobPackage{}, fmt.Errorf("actuator: unsupported input type %q for %q", field.Type, field.Key)
		}
	}

	return JobPackage{
		TaskID:       taskID,
		Workflow:     graph,
		Images:       images,
		OutputPrefix: fmt.Sprintf("outputs/%s", taskID),
		Outputs:      append([]catalogdomain.OutputBinding(nil), c.Document.Bindings.Outputs...),
	}, nil
}

// PrepareJob builds a job package and writes it to Blob at jobs/<task_id>/job.json.
func (s *CaseSnapshot) PrepareJob(ctx context.Context, taskID sharedkernel.TaskID) (sharedkernel.BlobRef, error) {
	job, err := s.BuildJobPackage(ctx, taskID)
	if err != nil {
		return sharedkernel.BlobRef{}, err
	}
	raw, err := json.Marshal(job)
	if err != nil {
		return sharedkernel.BlobRef{}, fmt.Errorf("actuator: marshal job: %w", err)
	}
	key := fmt.Sprintf("jobs/%s/job.json", taskID)
	ref, err := s.Blob.Put(ctx, key, bytes.NewReader(raw), blob.PutOptions{MIME: "application/json"})
	if err != nil {
		return sharedkernel.BlobRef{}, fmt.Errorf("actuator: put job: %w", err)
	}
	return ref, nil
}

type stagedValue struct {
	text    string
	number  float64
	boolean bool
	blob    sharedkernel.BlobRef
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
		raw, ok, err := s.readBlob(ctx, base+".blob.json")
		if err != nil || !ok {
			return stagedValue{}, ok, err
		}
		var ref sharedkernel.BlobRef
		if err := json.Unmarshal(raw, &ref); err != nil {
			return stagedValue{}, false, fmt.Errorf("actuator: parse blob meta %q: %w", field.Key, err)
		}
		if ref.Key == "" {
			return stagedValue{}, false, fmt.Errorf("actuator: empty blob key for %q", field.Key)
		}
		return stagedValue{blob: ref}, true, nil
	default:
		return stagedValue{}, false, fmt.Errorf("actuator: unsupported input type %q for %q", field.Type, field.Key)
	}
}

func (s *CaseSnapshot) uploadStagedImage(ctx context.Context, uploader ImageUploader, ref sharedkernel.BlobRef) (string, error) {
	if uploader == nil {
		return "", fmt.Errorf("actuator: image uploader not configured")
	}
	rc, err := s.Blob.Get(ctx, ref)
	if err != nil {
		return "", fmt.Errorf("actuator: get image blob %s: %w", ref.Key, err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("actuator: read image blob %s: %w", ref.Key, err)
	}
	filename := path.Base(ref.Key)
	if filename == "" || filename == "." || filename == "/" {
		filename = "image.png"
	}
	mime := ref.MIME
	if mime == "" {
		mime = "application/octet-stream"
	}
	remote, err := uploader.UploadImage(ctx, filename, mime, data)
	if err != nil {
		return "", fmt.Errorf("actuator: upload image %s: %w", ref.Key, err)
	}
	if remote == "" {
		return "", fmt.Errorf("actuator: upload image %s returned empty filename", ref.Key)
	}
	return remote, nil
}

func (s *CaseSnapshot) readBlob(ctx context.Context, key string) ([]byte, bool, error) {
	rc, err := s.Blob.Get(ctx, sharedkernel.BlobRef{Key: key})
	if err != nil {
		if isMissingBlob(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("actuator: get %s: %w", key, err)
	}
	defer rc.Close()
	raw, err := io.ReadAll(rc)
	if err != nil {
		return nil, false, fmt.Errorf("actuator: read %s: %w", key, err)
	}
	return raw, true, nil
}

// isMissingBlob reports whether err means the blob key was absent.
// localfs.Get surfaces this via os.Open → errors.Is(..., os.ErrNotExist).
func isMissingBlob(err error) bool {
	return errors.Is(err, os.ErrNotExist)
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
var _ ImageUploader = (comfyui.Client)(nil)
