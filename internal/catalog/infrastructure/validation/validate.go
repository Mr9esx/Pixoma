package validation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
)

type Validator struct {
	compiler *jsonschema.Compiler
}

func New() *Validator {
	return &Validator{compiler: jsonschema.NewCompiler()}
}

func (v *Validator) ValidateDocument(doc domain.CaseDocument) error {
	var fields []domain.FieldError
	if strings.TrimSpace(doc.Name) == "" {
		fields = append(fields, domain.FieldError{Key: "name", Message: "required"})
	}
	if len(doc.Inputs) == 0 {
		fields = append(fields, domain.FieldError{Key: "inputs", Message: "required"})
	}
	if len(doc.Bindings.WorkflowJSON) == 0 {
		fields = append(fields, domain.FieldError{Key: "bindings.workflow", Message: "required"})
	}
	if doc.InputSchema == nil {
		fields = append(fields, domain.FieldError{Key: "input_schema", Message: "required"})
	}
	if err := validateWorkflowGraph(doc.Bindings.WorkflowJSON); err != nil {
		fields = append(fields, *err)
	}
	fields = append(fields, validateInputBindings(doc.Inputs, doc.Bindings.WorkflowJSON, doc.Bindings.Inputs)...)
	fields = append(fields, validateOutputBindings(doc.Bindings.WorkflowJSON, doc.Bindings.Outputs)...)
	if len(fields) > 0 {
		return &domain.ValidationError{Fields: fields}
	}
	return nil
}

func workflowNodes(g map[string]any) (map[string]map[string]any, *domain.FieldError) {
	nodes := make(map[string]map[string]any, len(g))
	for id, raw := range g {
		node, ok := raw.(map[string]any)
		if !ok {
			return nil, &domain.FieldError{Key: "bindings.workflow", Message: "node " + id + " must be an object"}
		}
		classType, _ := node["class_type"].(string)
		if classType == "" {
			return nil, &domain.FieldError{Key: "bindings.workflow", Message: "node " + id + " missing class_type"}
		}
		if inputs, exists := node["inputs"]; exists {
			if _, ok := inputs.(map[string]any); !ok {
				return nil, &domain.FieldError{Key: "bindings.workflow", Message: "node " + id + " inputs must be an object"}
			}
		}
		nodes[id] = node
	}
	return nodes, nil
}

func validateWorkflowGraph(g map[string]any) *domain.FieldError {
	if len(g) == 0 {
		return &domain.FieldError{Key: "bindings.workflow", Message: "required"}
	}
	_, err := workflowNodes(g)
	return err
}

func validateInputBindings(inputs []domain.InputField, g map[string]any, bindings []domain.InputBinding) []domain.FieldError {
	nodes, graphErr := workflowNodes(g)
	if graphErr != nil {
		return nil
	}
	byKey := make(map[string]domain.InputBinding, len(bindings))
	for _, b := range bindings {
		byKey[b.Key] = b
	}
	var out []domain.FieldError
	for i, in := range inputs {
		b, ok := byKey[in.Key]
		if !ok || b.NodeID == "" || b.FieldPath == "" {
			out = append(out, domain.FieldError{
				Key:     fmt.Sprintf("bindings.inputs[%d].node_id", i),
				Message: "binding required for input " + in.Key,
			})
			continue
		}
		node, ok := nodes[b.NodeID]
		if !ok {
			out = append(out, domain.FieldError{Key: fmt.Sprintf("bindings.inputs[%d].node_id", i), Message: "node not found"})
			continue
		}
		inputs, _ := node["inputs"].(map[string]any)
		if _, ok := inputs[b.FieldPath]; !ok {
			out = append(out, domain.FieldError{Key: fmt.Sprintf("bindings.inputs[%d].field_path", i), Message: "field not found"})
		}
	}
	return out
}

func validateOutputBindings(g map[string]any, bindings []domain.OutputBinding) []domain.FieldError {
	nodes, graphErr := workflowNodes(g)
	if graphErr != nil {
		return nil
	}
	var out []domain.FieldError
	for i, b := range bindings {
		if _, ok := nodes[b.NodeID]; !ok {
			out = append(out, domain.FieldError{Key: fmt.Sprintf("bindings.outputs[%d].node_id", i), Message: "node not found"})
			continue
		}
		if b.Index < 0 {
			out = append(out, domain.FieldError{Key: fmt.Sprintf("bindings.outputs[%d].index", i), Message: "must be >= 0"})
		}
	}
	return out
}

func (v *Validator) ValidateInputs(doc domain.CaseDocument, values []domain.InputValue) error {
	if err := v.ValidateDocument(doc); err != nil {
		return err
	}

	byKey := indexValues(values)
	var fields []domain.FieldError

	// Media hooks: image/video fields must carry Blob when present / required.
	for _, in := range doc.Inputs {
		val, ok := byKey[in.Key]
		switch in.Type {
		case "image", "video":
			if ok && val.Text != nil {
				fields = append(fields, domain.FieldError{Key: in.Key, Message: "expected media blob, got text"})
				continue
			}
			if in.Required && (!ok || val.Blob == nil || strings.TrimSpace(val.Blob.Key) == "") {
				fields = append(fields, domain.FieldError{Key: in.Key, Message: "media blob required"})
				continue
			}
			if ok && val.Blob != nil && val.Blob.MIME != "" {
				if err := checkMIME(in.Type, val.Blob.MIME); err != nil {
					fields = append(fields, domain.FieldError{Key: in.Key, Message: err.Error()})
				}
			}
		}
	}
	if len(fields) > 0 {
		return &domain.ValidationError{Fields: fields}
	}

	obj := valuesToObject(doc, byKey)
	schemaBytes, err := json.Marshal(doc.InputSchema)
	if err != nil {
		return fmt.Errorf("marshal input_schema: %w", err)
	}
	sch, err := compileSchema(v.compiler, strconv.FormatUint(uint64(doc.ID), 10), schemaBytes)
	if err != nil {
		return fmt.Errorf("compile input_schema: %w", err)
	}
	if err := sch.Validate(obj); err != nil {
		return schemaToValidationError(err)
	}
	return nil
}

func compileSchema(c *jsonschema.Compiler, id string, raw []byte) (*jsonschema.Schema, error) {
	url := "mem://" + id + ".json"
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	if err := c.AddResource(url, doc); err != nil {
		// Recompile with fresh compiler on duplicate resource (tests reuse ids).
		c2 := jsonschema.NewCompiler()
		if err2 := c2.AddResource(url, doc); err2 != nil {
			return nil, err
		}
		return c2.Compile(url)
	}
	return c.Compile(url)
}

func indexValues(values []domain.InputValue) map[string]domain.InputValue {
	m := make(map[string]domain.InputValue, len(values))
	for _, v := range values {
		m[v.Key] = v
	}
	return m
}

func valuesToObject(doc domain.CaseDocument, byKey map[string]domain.InputValue) map[string]any {
	obj := map[string]any{}
	for _, in := range doc.Inputs {
		val, ok := byKey[in.Key]
		if !ok {
			continue
		}
		switch {
		case val.Text != nil:
			obj[in.Key] = *val.Text
		case val.Number != nil:
			obj[in.Key] = *val.Number
		case val.Bool != nil:
			obj[in.Key] = *val.Bool
		case val.Blob != nil:
			obj[in.Key] = map[string]any{
				"key":  val.Blob.Key,
				"mime": val.Blob.MIME,
				"size": val.Blob.Size,
			}
		}
	}
	return obj
}

func checkMIME(fieldType, mime string) error {
	mime = strings.ToLower(mime)
	switch fieldType {
	case "image":
		if !strings.HasPrefix(mime, "image/") {
			return fmt.Errorf("mime must be image/*, got %s", mime)
		}
	case "video":
		if !strings.HasPrefix(mime, "video/") {
			return fmt.Errorf("mime must be video/*, got %s", mime)
		}
	}
	return nil
}

func schemaToValidationError(err error) error {
	var fields []domain.FieldError
	if ve, ok := err.(*jsonschema.ValidationError); ok {
		for _, cause := range flattenCauses(ve) {
			key := instanceLocationKey(cause.InstanceLocation)
			fields = append(fields, domain.FieldError{Key: key, Message: cause.Error()})
		}
	}
	if len(fields) == 0 {
		fields = append(fields, domain.FieldError{Key: "_", Message: err.Error()})
	}
	return &domain.ValidationError{Fields: fields}
}

func flattenCauses(ve *jsonschema.ValidationError) []*jsonschema.ValidationError {
	if len(ve.Causes) == 0 {
		return []*jsonschema.ValidationError{ve}
	}
	var out []*jsonschema.ValidationError
	for _, c := range ve.Causes {
		out = append(out, flattenCauses(c)...)
	}
	return out
}

func instanceLocationKey(loc []string) string {
	if len(loc) == 0 {
		return "_"
	}
	return loc[0]
}
