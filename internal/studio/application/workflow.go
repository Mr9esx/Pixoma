package application

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	catalogdomain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/platform/queue"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	studiodomain "github.com/Mr9esx/Pixoma/internal/studio/domain"
	runtimedomain "github.com/Mr9esx/Pixoma/internal/tasks/domain"
)

// ResolvedWorkflow is the safe, short-lived Case view exposed to the Agent
// runtime. It intentionally carries only the Case's existing display metadata
// and input schema; bindings and workflow graph JSON never leave the catalog.
type ResolvedWorkflow struct {
	ID          string          `json:"id"`
	ToolName    string          `json:"tool_name"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// ResolveWorkflows returns Cases that are both globally enabled and explicitly
// enabled for Agent use by this account. Tool names derive only from immutable
// Case IDs so an Agent conversation can safely refer to them across requests.
func (s *CapabilityConfigService) ResolveWorkflows(ctx context.Context, accountID string) ([]ResolvedWorkflow, error) {
	if s == nil || s.Repo == nil || s.WorkflowCatalog == nil {
		return nil, fmt.Errorf("studio: workflow catalog is not configured")
	}
	cases, err := s.WorkflowCatalog.List(ctx, catalogdomain.ListQuery{})
	if err != nil {
		return nil, fmt.Errorf("studio: list workflows: %w", err)
	}
	settings, err := s.Repo.ListAgentWorkflowSettings(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("studio: list agent workflow settings: %w", err)
	}
	enabled := make(map[string]bool, len(settings))
	for _, setting := range settings {
		if setting != nil {
			enabled[setting.WorkflowID] = setting.AgentEnabled
		}
	}
	workflows := make([]ResolvedWorkflow, 0, len(cases))
	for _, workflow := range cases {
		if workflow == nil || !workflow.Enabled {
			continue
		}
		id := strconv.FormatUint(uint64(workflow.Document.ID), 10)
		if !enabled[id] {
			continue
		}
		schema, err := json.Marshal(workflow.Document.InputSchema)
		if err != nil {
			return nil, fmt.Errorf("studio: marshal workflow %s input schema: %w", id, err)
		}
		if !json.Valid(schema) {
			return nil, fmt.Errorf("studio: workflow %s has invalid input schema", id)
		}
		workflows = append(workflows, ResolvedWorkflow{
			ID:          id,
			ToolName:    workflowToolName(id),
			Name:        workflow.Document.Name,
			Description: workflow.Document.Description,
			InputSchema: append(json.RawMessage(nil), schema...),
		})
	}
	sort.Slice(workflows, func(i, j int) bool { return workflows[i].ID < workflows[j].ID })
	return workflows, nil
}

func workflowToolName(workflowID string) string {
	return "studio_workflow_" + workflowID
}

type workflowCaseLookup interface {
	Get(context.Context, sharedkernel.CaseID) (*catalogdomain.Case, error)
}

type workflowStudioRepository interface {
	GetWorkflowExecutionByRunTool(context.Context, string, string, string) (*studiodomain.WorkflowExecution, error)
	CreateWorkflowExecution(context.Context, *studiodomain.WorkflowExecution) error
	GetAsset(context.Context, string, string) (*studiodomain.Asset, error)
}

// WorkflowStartInput identifies an Agent tool call and its normalized JSON
// arguments. OperationNodeID is supplied by the runtime after it adds the
// user-visible SOP operation node to the Session Road.
type WorkflowStartInput struct {
	AccountID       string
	SessionID       string
	RunID           string
	ToolCallID      string
	WorkflowID      string
	OperationNodeID string
	Inputs          map[string]any
}

type WorkflowStartResult struct {
	TaskID     string
	WorkflowID string
	Reused     bool
}

// WorkflowStarter is the application boundary between Studio Agent tools and
// the existing asynchronous task runtime. It does not use Bot users or Bot
// sessions: a Studio task is deliberately marked by a Studio-prefixed session
// ID and an empty ChatID.
type WorkflowStarter struct {
	StudioRepo workflowStudioRepository
	Catalog    workflowCaseLookup
	Validator  catalogdomain.Validator
	Tasks      runtimedomain.TaskRepository
	Blob       blob.Store
	Publisher  queue.Publisher
	NewID      func() string
	NewTaskID  func() sharedkernel.TaskID
	Now        func() time.Time
}

func (s *WorkflowStarter) Start(ctx context.Context, input WorkflowStartInput) (*WorkflowStartResult, error) {
	if s == nil || s.StudioRepo == nil || s.Catalog == nil || s.Validator == nil || s.Tasks == nil || s.Blob == nil || s.Publisher == nil || s.NewTaskID == nil {
		return nil, fmt.Errorf("studio: workflow starter is not configured")
	}
	if anyWorkflowBlank(input.AccountID, input.SessionID, input.RunID, input.ToolCallID, input.WorkflowID, input.OperationNodeID) {
		return nil, fmt.Errorf("%w: workflow start identity is required", studiodomain.ErrInvalid)
	}
	if existing, err := s.StudioRepo.GetWorkflowExecutionByRunTool(ctx, input.AccountID, input.RunID, input.ToolCallID); err == nil {
		return &WorkflowStartResult{TaskID: existing.TaskID, WorkflowID: existing.WorkflowID, Reused: true}, nil
	} else if err != studiodomain.ErrNotFound {
		return nil, fmt.Errorf("studio: find workflow execution: %w", err)
	}

	caseID, err := parseWorkflowID(input.WorkflowID)
	if err != nil {
		return nil, err
	}
	workflow, err := s.Catalog.Get(ctx, caseID)
	if err != nil {
		return nil, fmt.Errorf("studio: get workflow %s: %w", input.WorkflowID, err)
	}
	if workflow == nil || !workflow.Enabled {
		return nil, catalogdomain.ErrDisabled
	}
	values, err := s.inputValues(ctx, input.AccountID, workflow.Document, input.Inputs)
	if err != nil {
		return nil, err
	}
	if err := s.Validator.ValidateInputs(workflow.Document, values); err != nil {
		return nil, fmt.Errorf("studio: validate workflow inputs: %w", err)
	}

	now := s.now()
	taskID := s.NewTaskID()
	if strings.TrimSpace(string(taskID)) == "" {
		return nil, fmt.Errorf("studio: workflow task id is required")
	}
	inputPrefix := "studio-workflow-inputs/" + string(taskID)
	if err := stageWorkflowInputs(ctx, s.Blob, inputPrefix, values); err != nil {
		return nil, fmt.Errorf("studio: stage workflow inputs: %w", err)
	}
	task := runtimedomain.NewPending(taskID, sharedkernel.SessionID("studio-session-"+input.SessionID), caseID, inputPrefix, now)
	if err := s.Tasks.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("studio: create workflow task: %w", err)
	}
	execution, err := studiodomain.NewWorkflowExecution(s.newID(), input.AccountID, input.SessionID, input.RunID, input.ToolCallID, string(taskID), input.WorkflowID, input.OperationNodeID, now)
	if err != nil {
		return nil, err
	}
	if err := s.StudioRepo.CreateWorkflowExecution(ctx, execution); err != nil {
		if err == studiodomain.ErrAlreadyExists {
			existing, findErr := s.StudioRepo.GetWorkflowExecutionByRunTool(ctx, input.AccountID, input.RunID, input.ToolCallID)
			if findErr == nil {
				return &WorkflowStartResult{TaskID: existing.TaskID, WorkflowID: existing.WorkflowID, Reused: true}, nil
			}
		}
		return nil, fmt.Errorf("studio: record workflow execution: %w", err)
	}
	payload, err := json.Marshal(sharedkernel.TaskCreated{TaskID: taskID, CaseID: caseID, CreatedAt: now})
	if err != nil {
		return nil, fmt.Errorf("studio: encode workflow task event: %w", err)
	}
	if err := s.Publisher.Publish(ctx, queue.Message{Topic: sharedkernel.TopicTaskCreated, Key: string(taskID), Payload: payload}); err != nil {
		return nil, fmt.Errorf("studio: publish workflow task: %w", err)
	}
	return &WorkflowStartResult{TaskID: string(taskID), WorkflowID: input.WorkflowID}, nil
}

func (s *WorkflowStarter) inputValues(ctx context.Context, accountID string, document catalogdomain.CaseDocument, inputs map[string]any) ([]catalogdomain.InputValue, error) {
	fields := make(map[string]catalogdomain.InputField, len(document.Inputs))
	for _, field := range document.Inputs {
		fields[field.Key] = field
	}
	values := make([]catalogdomain.InputValue, 0, len(inputs))
	for key, raw := range inputs {
		field, ok := fields[key]
		if !ok {
			return nil, fmt.Errorf("%w: unknown workflow input %q", studiodomain.ErrInvalid, key)
		}
		value, err := s.inputValue(ctx, accountID, field, raw)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Key < values[j].Key })
	return values, nil
}

func (s *WorkflowStarter) inputValue(ctx context.Context, accountID string, field catalogdomain.InputField, raw any) (catalogdomain.InputValue, error) {
	value := catalogdomain.InputValue{Key: field.Key}
	switch field.Type {
	case "image", "video":
		assetID, ok := raw.(string)
		if !ok || strings.TrimSpace(assetID) == "" {
			return value, fmt.Errorf("%w: workflow input %q requires an asset ID", studiodomain.ErrInvalid, field.Key)
		}
		asset, err := s.StudioRepo.GetAsset(ctx, accountID, assetID)
		if err != nil {
			return value, fmt.Errorf("studio: get workflow asset %q: %w", field.Key, err)
		}
		if asset == nil || (field.Type == "image" && asset.Kind != studiodomain.AssetImage) || (field.Type == "video" && asset.Kind != studiodomain.AssetVideo) || len(asset.Versions) == 0 {
			return value, fmt.Errorf("%w: workflow input %q has incompatible asset", studiodomain.ErrInvalid, field.Key)
		}
		version := asset.Versions[len(asset.Versions)-1]
		value.Blob = &sharedkernel.BlobRef{Key: version.BlobKey, MIME: version.MIMEType, Size: version.SizeBytes}
	case "number":
		number, ok := asFloat64(raw)
		if !ok {
			return value, fmt.Errorf("%w: workflow input %q requires a number", studiodomain.ErrInvalid, field.Key)
		}
		value.Number = &number
	case "boolean":
		boolean, ok := raw.(bool)
		if !ok {
			return value, fmt.Errorf("%w: workflow input %q requires a boolean", studiodomain.ErrInvalid, field.Key)
		}
		value.Bool = &boolean
	default:
		text, ok := raw.(string)
		if !ok {
			return value, fmt.Errorf("%w: workflow input %q requires text", studiodomain.ErrInvalid, field.Key)
		}
		value.Text = &text
	}
	return value, nil
}

func asFloat64(value any) (float64, bool) {
	switch number := value.(type) {
	case float64:
		return number, true
	case float32:
		return float64(number), true
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	case json.Number:
		parsed, err := number.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

func stageWorkflowInputs(ctx context.Context, store blob.Store, prefix string, values []catalogdomain.InputValue) error {
	for _, value := range values {
		key := prefix + "/" + value.Key
		switch {
		case value.Blob != nil:
			metadata, err := json.Marshal(value.Blob)
			if err != nil {
				return err
			}
			if _, err := store.Put(ctx, key+".blob.json", bytes.NewReader(metadata), blob.PutOptions{MIME: "application/json"}); err != nil {
				return err
			}
		case value.Text != nil:
			if _, err := store.Put(ctx, key+".txt", bytes.NewReader([]byte(*value.Text)), blob.PutOptions{MIME: "text/plain"}); err != nil {
				return err
			}
		case value.Number != nil:
			encoded := strconv.FormatFloat(*value.Number, 'f', -1, 64)
			if _, err := store.Put(ctx, key+".num", bytes.NewReader([]byte(encoded)), blob.PutOptions{MIME: "text/plain"}); err != nil {
				return err
			}
		case value.Bool != nil:
			encoded := strconv.FormatBool(*value.Bool)
			if _, err := store.Put(ctx, key+".bool", bytes.NewReader([]byte(encoded)), blob.PutOptions{MIME: "text/plain"}); err != nil {
				return err
			}
		}
	}
	return nil
}

func parseWorkflowID(workflowID string) (sharedkernel.CaseID, error) {
	id, err := strconv.ParseUint(strings.TrimSpace(workflowID), 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("%w: invalid workflow ID", studiodomain.ErrInvalid)
	}
	return sharedkernel.CaseID(id), nil
}

func anyWorkflowBlank(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return true
		}
	}
	return false
}

func (s *WorkflowStarter) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s *WorkflowStarter) newID() string {
	if s.NewID != nil {
		return s.NewID()
	}
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}
