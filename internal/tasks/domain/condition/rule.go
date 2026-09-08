package condition

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Rule is a declarative condition: a leaf {field, op, value} or an and/or
// combination. The field references a registered attribute provider key.
type Rule struct {
	Field  string
	Op     string
	Value  any
	And    []Rule
	Or     []Rule
	Always bool
}

// BuiltinOps is the fixed operator set supported by the engine. New
// operators require an engine change; new attributes do not.
var BuiltinOps = map[string]bool{
	"eq": true, "ne": true, "in": true,
	"gt": true, "gte": true, "lt": true, "lte": true,
	"exists": true,
}

type rawRule struct {
	Always *bool             `json:"always"`
	Field  *string           `json:"field"`
	Op     *string           `json:"op"`
	Value  json.RawMessage   `json:"value"`
	And    []json.RawMessage `json:"and"`
	Or     []json.RawMessage `json:"or"`
}

// ParseRule parses a JSON rule into a Rule.
func ParseRule(raw json.RawMessage) (Rule, error) {
	var r rawRule
	if err := json.Unmarshal(raw, &r); err != nil {
		return Rule{}, fmt.Errorf("condition: invalid rule: %w", err)
	}
	hasLeaf := r.Field != nil || r.Op != nil || len(r.Value) > 0 && string(r.Value) != "null"
	hasAnd := len(r.And) > 0
	hasOr := len(r.Or) > 0
	if r.Always != nil {
		if !*r.Always || hasLeaf || hasAnd || hasOr {
			return Rule{}, errors.New("condition: always must be true and cannot mix with other rules")
		}
		return Rule{Always: true}, nil
	}
	if hasLeaf && (hasAnd || hasOr) {
		return Rule{}, errors.New("condition: rule cannot mix leaf fields with and/or")
	}
	if hasAnd {
		if hasOr {
			return Rule{}, errors.New("condition: rule cannot mix and with or at the same level")
		}
		out := Rule{}
		for _, item := range r.And {
			sub, err := ParseRule(item)
			if err != nil {
				return Rule{}, fmt.Errorf("condition: and[%d]: %w", len(out.And), err)
			}
			out.And = append(out.And, sub)
		}
		return out, nil
	}
	if hasOr {
		out := Rule{}
		for _, item := range r.Or {
			sub, err := ParseRule(item)
			if err != nil {
				return Rule{}, fmt.Errorf("condition: or[%d]: %w", len(out.Or), err)
			}
			out.Or = append(out.Or, sub)
		}
		return out, nil
	}
	if r.Field == nil || r.Op == nil {
		return Rule{}, errors.New("condition: leaf rule requires field and op")
	}
	var val any
	if len(r.Value) > 0 && string(r.Value) != "null" {
		if err := json.Unmarshal(r.Value, &val); err != nil {
			return Rule{}, fmt.Errorf("condition: invalid value: %w", err)
		}
	}
	return Rule{Field: *r.Field, Op: *r.Op, Value: val}, nil
}
