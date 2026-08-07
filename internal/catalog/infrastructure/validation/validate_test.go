package validation_test

import (
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/validation"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func text2imgDoc() domain.CaseDocument {
	return domain.CaseDocument{
		ID:   "text2img-demo",
		Name: "Demo",
		Inputs: []domain.InputField{
			{Key: "prompt", Type: "string", Required: true},
			{Key: "seed", Type: "number", Required: false, SkipAllowed: true},
		},
		Outputs: []domain.OutputField{{Key: "image", Type: "image"}},
		Bindings: domain.ComfyBindings{
			WorkflowJSON: map[string]any{"1": map[string]any{}},
			Inputs:       []domain.InputBinding{{Key: "prompt", NodeID: "1", FieldPath: "text"}},
			Outputs:      []domain.OutputBinding{{Key: "image", NodeID: "2"}},
		},
		InputSchema: map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []any{"prompt"},
			"properties": map[string]any{
				"prompt": map[string]any{"type": "string", "minLength": 1, "maxLength": 200},
				"seed":   map[string]any{"type": "integer", "minimum": 0},
			},
		},
	}
}

func TestValidateInputsAcceptsPrompt(t *testing.T) {
	v := validation.New()
	prompt := "a cat"
	err := v.ValidateInputs(text2imgDoc(), []domain.InputValue{{Key: "prompt", Text: &prompt}})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestValidateInputsRejectsMissingPrompt(t *testing.T) {
	v := validation.New()
	err := v.ValidateInputs(text2imgDoc(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	var ve *domain.ValidationError
	if !asValidation(err, &ve) {
		t.Fatalf("want ValidationError, got %T %v", err, err)
	}
}

func TestValidateInputsRejectsEmptyPrompt(t *testing.T) {
	v := validation.New()
	empty := ""
	err := v.ValidateInputs(text2imgDoc(), []domain.InputValue{{Key: "prompt", Text: &empty}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateDocumentRequiresID(t *testing.T) {
	v := validation.New()
	doc := text2imgDoc()
	doc.ID = ""
	if err := v.ValidateDocument(doc); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateInputsAllowsSkippedOptional(t *testing.T) {
	v := validation.New()
	prompt := "hi"
	err := v.ValidateInputs(text2imgDoc(), []domain.InputValue{{Key: "prompt", Text: &prompt}})
	if err != nil {
		t.Fatalf("skip seed should be ok: %v", err)
	}
}

func TestValidateInputsRejectsBadEnum(t *testing.T) {
	v := validation.New()
	doc := text2imgDoc()
	doc.Inputs = append(doc.Inputs, domain.InputField{Key: "style", Type: "enum", Required: true})
	doc.InputSchema = map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"prompt", "style"},
		"properties": map[string]any{
			"prompt": map[string]any{"type": "string", "minLength": 1},
			"style":  map[string]any{"type": "string", "enum": []any{"anime", "photo"}},
		},
	}
	prompt := "x"
	style := "oil"
	err := v.ValidateInputs(doc, []domain.InputValue{
		{Key: "prompt", Text: &prompt},
		{Key: "style", Text: &style},
	})
	if err == nil {
		t.Fatal("expected enum error")
	}
}

func TestValidateInputsRejectsTextForImage(t *testing.T) {
	v := validation.New()
	doc := text2imgDoc()
	doc.Inputs = []domain.InputField{{Key: "source", Type: "image", Required: true}}
	doc.InputSchema = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"source": map[string]any{"type": "object"},
		},
		"required": []any{"source"},
	}
	txt := "not-an-image"
	err := v.ValidateInputs(doc, []domain.InputValue{{Key: "source", Text: &txt}})
	if err == nil {
		t.Fatal("expected media type error")
	}
}

func TestValidateInputsAcceptsImageBlob(t *testing.T) {
	v := validation.New()
	doc := text2imgDoc()
	doc.Inputs = []domain.InputField{{Key: "source", Type: "image", Required: true}}
	doc.InputSchema = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"source": map[string]any{"type": "object"},
		},
		"required": []any{"source"},
	}
	blob := &sharedkernel.BlobRef{Key: "inputs/a.png", MIME: "image/png"}
	err := v.ValidateInputs(doc, []domain.InputValue{{Key: "source", Blob: blob}})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func asValidation(err error, target **domain.ValidationError) bool {
	if ve, ok := err.(*domain.ValidationError); ok {
		*target = ve
		return true
	}
	return false
}
