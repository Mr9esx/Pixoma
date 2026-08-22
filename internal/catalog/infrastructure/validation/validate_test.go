package validation_test

import (
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/validation"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func text2imgDoc() domain.CaseDocument {
	return domain.CaseDocument{
		ID:   1,
		Name: "Demo",
		Inputs: []domain.InputField{
			{Key: "prompt", Type: "string", Required: true},
			{Key: "seed", Type: "number", Required: false, SkipAllowed: true},
		},
		Outputs: []domain.OutputField{{Key: "image", Type: "image"}},
		Bindings: domain.ComfyBindings{
			WorkflowJSON: map[string]any{
				"1": map[string]any{"class_type": "CLIPTextEncode", "inputs": map[string]any{"text": "x", "seed": 42}},
				"2": map[string]any{"class_type": "SaveImage", "inputs": map[string]any{"filename_prefix": "o"}},
			},
			Inputs: []domain.InputBinding{
				{Key: "prompt", NodeID: "1", FieldPath: "text"},
				{Key: "seed", NodeID: "1", FieldPath: "seed"},
			},
			Outputs: []domain.OutputBinding{{Key: "image", NodeID: "2"}},
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

func TestValidateDocumentRejectsUnboundInput(t *testing.T) {
	v := validation.New()
	doc := text2imgDoc()
	doc.Bindings.Inputs = []domain.InputBinding{{Key: "prompt", NodeID: "1", FieldPath: "text"}}
	err := v.ValidateDocument(doc)
	if err == nil {
		t.Fatal("expected error for unbound seed input")
	}
	var ve *domain.ValidationError
	if !asValidation(err, &ve) {
		t.Fatalf("want ValidationError, got %T %v", err, err)
	}
	found := false
	for _, f := range ve.Fields {
		if f.Message == "binding required for input seed" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected unbound seed error, got %+v", ve.Fields)
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

func TestValidateDocumentAllowsAutoAssignedID(t *testing.T) {
	v := validation.New()
	doc := text2imgDoc()
	doc.ID = 0
	if err := v.ValidateDocument(doc); err != nil {
		t.Fatalf("id=0 应允许自动分配，got %v", err)
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

func mixedEditDoc() domain.CaseDocument {
	return domain.CaseDocument{
		ID:   2,
		Name: "图片编辑",
		Inputs: []domain.InputField{
			{Key: "reference", Type: "image", Required: true},
			{Key: "prompt", Type: "string", Required: true},
		},
		Outputs: []domain.OutputField{{Key: "image", Type: "image"}},
		Bindings: domain.ComfyBindings{
			WorkflowJSON: map[string]any{
				"10": map[string]any{"class_type": "LoadImage", "inputs": map[string]any{"image": "ref.png"}},
				"20": map[string]any{"class_type": "CLIPTextEncode", "inputs": map[string]any{"text": "x"}},
				"60": map[string]any{"class_type": "SaveImage", "inputs": map[string]any{"filename_prefix": "o"}},
			},
			Inputs: []domain.InputBinding{
				{Key: "reference", NodeID: "10", FieldPath: "image"},
				{Key: "prompt", NodeID: "20", FieldPath: "text"},
			},
			Outputs: []domain.OutputBinding{{Key: "image", NodeID: "60"}},
		},
		InputSchema: map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []any{"reference", "prompt"},
			"properties": map[string]any{
				"reference": map[string]any{
					"type":     "object",
					"required": []any{"key"},
					"properties": map[string]any{
						"key":  map[string]any{"type": "string", "minLength": 1},
						"mime": map[string]any{"type": "string"},
						"size": map[string]any{"type": "number"},
					},
				},
				"prompt": map[string]any{"type": "string", "minLength": 1, "maxLength": 2000},
			},
		},
	}
}

func TestValidateInputsRejectsTextForImage(t *testing.T) {
	v := validation.New()
	doc := mixedEditDoc()
	txt := "not-an-image"
	prompt := "edit the photo"
	err := v.ValidateInputs(doc, []domain.InputValue{
		{Key: "reference", Text: &txt},
		{Key: "prompt", Text: &prompt},
	})
	if err == nil {
		t.Fatal("expected media type error")
	}
	var ve *domain.ValidationError
	if !asValidation(err, &ve) {
		t.Fatalf("want ValidationError, got %T %v", err, err)
	}
	if len(ve.Fields) == 0 || ve.Fields[0].Key != "reference" {
		t.Fatalf("want reference field error, got %#v", ve.Fields)
	}
	if ve.Fields[0].Message != "expected media blob, got text" {
		t.Fatalf("want media blob message, got %q", ve.Fields[0].Message)
	}
}

func TestValidateInputsAcceptsMixedTextAndImage(t *testing.T) {
	v := validation.New()
	doc := mixedEditDoc()
	prompt := "edit the photo"
	blob := &sharedkernel.BlobRef{Key: "inputs/a.png", MIME: "image/png"}
	err := v.ValidateInputs(doc, []domain.InputValue{
		{Key: "reference", Blob: blob},
		{Key: "prompt", Text: &prompt},
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestValidateInputsAcceptsImageBlob(t *testing.T) {
	v := validation.New()
	doc := text2imgDoc()
	doc.Inputs = []domain.InputField{{Key: "source", Type: "image", Required: true}}
	doc.Bindings.Inputs = []domain.InputBinding{{Key: "source", NodeID: "1", FieldPath: "text"}}
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

func TestValidateDocumentRejectsGraphWithoutClassType(t *testing.T) {
	v := validation.New()
	doc := text2imgDoc()
	doc.Bindings.WorkflowJSON = map[string]any{
		"1": map[string]any{"inputs": map[string]any{"text": "x"}},
	}
	if err := v.ValidateDocument(doc); err == nil {
		t.Fatal("expected graph structure error")
	}
}

func TestValidateDocumentRejectsMissingBindingNode(t *testing.T) {
	v := validation.New()
	doc := text2imgDoc()
	doc.Bindings.Inputs[0].NodeID = "99"
	if err := v.ValidateDocument(doc); err == nil {
		t.Fatal("expected missing node error")
	}
}

func TestValidateDocumentRejectsMissingFieldPath(t *testing.T) {
	v := validation.New()
	doc := text2imgDoc()
	doc.Bindings.Inputs[0].FieldPath = "not_a_field"
	if err := v.ValidateDocument(doc); err == nil {
		t.Fatal("expected missing field path error")
	}
}

func TestValidateDocumentRejectsNegativeOutputIndex(t *testing.T) {
	v := validation.New()
	doc := text2imgDoc()
	doc.Bindings.Outputs[0].Index = -1
	if err := v.ValidateDocument(doc); err == nil {
		t.Fatal("expected negative index error")
	}
}

func TestValidateDocumentAcceptsValidGraphAndBindings(t *testing.T) {
	v := validation.New()
	if err := v.ValidateDocument(text2imgDoc()); err != nil {
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
