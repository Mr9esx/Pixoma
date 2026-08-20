package condition

import (
	"fmt"
)

// ValidateRule checks the rule against the registry: fields must be registered,
// operators must be built-in, and values must match the attribute schema.
func ValidateRule(r Rule, reg *Registry) error {
	if reg == nil {
		return fmt.Errorf("condition: nil registry")
	}
	if len(r.And) > 0 {
		for i, sub := range r.And {
			if err := ValidateRule(sub, reg); err != nil {
				return fmt.Errorf("condition: and[%d]: %w", i, err)
			}
		}
		return nil
	}
	if len(r.Or) > 0 {
		for i, sub := range r.Or {
			if err := ValidateRule(sub, reg); err != nil {
				return fmt.Errorf("condition: or[%d]: %w", i, err)
			}
		}
		return nil
	}
	if !BuiltinOps[r.Op] {
		return fmt.Errorf("condition: unknown op %q", r.Op)
	}
	provider, err := reg.ProviderFor(r.Field)
	if err != nil {
		return err
	}
	var desc *AttributeDescriptor
	for _, d := range provider.ListAttributes() {
		if d.Key == r.Field {
			dd := d
			desc = &dd
			break
		}
	}
	if desc == nil {
		return fmt.Errorf("condition: field %q not provided by %q", r.Field, provider.Namespace())
	}
	if r.Op == "exists" {
		return nil
	}
	if err := validateValueType(r.Value, desc.Schema, r.Op); err != nil {
		return fmt.Errorf("condition: field %s: %w", r.Field, err)
	}
	return nil
}

func validateValueType(v any, schema map[string]any, op string) error {
	typ, _ := schema["type"].(string)
	if typ == "" {
		return nil
	}
	if op == "in" {
		arr, ok := v.([]any)
		if !ok {
			return fmt.Errorf("value must be array for in")
		}
		itemSchema := schema
		if items, ok := schema["items"].(map[string]any); ok {
			itemSchema = items
		}
		for _, item := range arr {
			if err := validateValueType(item, itemSchema, "eq"); err != nil {
				return err
			}
		}
		return nil
	}
	switch typ {
	case "boolean":
		if _, ok := v.(bool); !ok {
			return fmt.Errorf("value must be boolean")
		}
	case "number":
		switch v.(type) {
		case float64, int, int64:
		default:
			return fmt.Errorf("value must be number")
		}
	case "string":
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("value must be string")
		}
		if enum, ok := schema["enum"].([]any); ok && len(enum) > 0 {
			for _, e := range enum {
				if e == s {
					return nil
				}
			}
			return fmt.Errorf("value %q not in enum %v", s, enum)
		}
	case "array":
		arr, ok := v.([]any)
		if !ok {
			return fmt.Errorf("value must be array")
		}
		if itemSchema, ok := schema["items"].(map[string]any); ok {
			for _, item := range arr {
				if err := validateValueType(item, itemSchema, "eq"); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
