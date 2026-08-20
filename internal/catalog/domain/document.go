package domain

import (
	"encoding/json"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// CaseDocument is the protocol document stored for a Case.
type CaseDocument struct {
	ID               sharedkernel.CaseID `json:"id"`
	Name             string              `json:"name"`
	Description      string              `json:"description,omitempty"`
	Preview          string              `json:"preview,omitempty"`
	Price            float64             `json:"price"`
	Tags             []string            `json:"tags,omitempty"`
	Categories       []string            `json:"categories,omitempty"`
	Routing          *RoutingConfig      `json:"routing,omitempty"`
	Inputs           []InputField        `json:"inputs"`
	Outputs          []OutputField       `json:"outputs"`
	Bindings         ComfyBindings       `json:"bindings"`
	InputSchema      map[string]any      `json:"input_schema"` // JSON Schema object for values map
	WorkflowFilename string              `json:"workflow_filename,omitempty"`
}

// RoutingConfig declares how tasks of this Case are delivered to topics.
// Rules are evaluated in order; the first match wins. No match falls back to
// the system default topic.
type RoutingConfig struct {
	Rules []RoutingRule `json:"rules"`
}

// RoutingRule binds a condition to a target topic key.
type RoutingRule struct {
	When  json.RawMessage `json:"when"`
	Topic string          `json:"topic"`
}

type InputField struct {
	Key         string `json:"key"`
	Type        string `json:"type"` // string|image|video|number|boolean|enum
	Required    bool   `json:"required"`
	SkipAllowed bool   `json:"skip_allowed,omitempty"`
	Description string `json:"description,omitempty"`
	Preview     string `json:"preview,omitempty"`
}

type OutputField struct {
	Key         string `json:"key"`
	Type        string `json:"type"` // image|text|file
	Description string `json:"description,omitempty"`
	MediaType   string `json:"media_type,omitempty"`
}

type ComfyBindings struct {
	WorkflowJSON map[string]any  `json:"workflow"` // prompt graph template
	Inputs       []InputBinding  `json:"inputs"`
	Outputs      []OutputBinding `json:"outputs"`
}

type InputBinding struct {
	Key       string `json:"key"`
	NodeID    string `json:"node_id"`
	FieldPath string `json:"field_path"`
}

type OutputBinding struct {
	Key    string `json:"key"`
	NodeID string `json:"node_id"`
	Index  int    `json:"index,omitempty"`
}

// InputValue is a logical input after collection / staging.
type InputValue struct {
	Key    string
	Text   *string
	Number *float64
	Bool   *bool
	Blob   *sharedkernel.BlobRef
}

type Validator interface {
	ValidateInputs(doc CaseDocument, values []InputValue) error
}

type FieldError struct {
	Key     string
	Message string
}

type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	if len(e.Fields) == 0 {
		return "validation failed"
	}
	return "validation failed: " + e.Fields[0].Key + ": " + e.Fields[0].Message
}
