package condition

import (
	"context"
	"fmt"
)

// CaseProvider resolves case.* attributes via an injected lookup.
type CaseProvider struct {
	Lookup func(ctx context.Context, caseID string) (category string, tags []string, err error)
}

// Namespace returns "case".
func (p *CaseProvider) Namespace() string { return "case" }

// ListAttributes returns the case attributes this provider can resolve.
func (p *CaseProvider) ListAttributes() []AttributeDescriptor {
	return []AttributeDescriptor{
		{
			Key:     "case.category",
			Context: "case",
			Label:   "Case 分类",
			Schema:  map[string]any{"type": "string", "enum": []any{"image", "video", "audio"}, "description": "Case 类别"},
		},
		{
			Key:     "case.tags",
			Context: "case",
			Label:   "Case 标签",
			Schema:  map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Case 标签列表"},
		},
	}
}

// Value resolves a single case attribute.
func (p *CaseProvider) Value(ctx context.Context, field string) (any, error) {
	switch field {
	case "case.category", "case.tags":
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownField, field)
	}
	caseID, ok := CaseIDFrom(ctx)
	if !ok {
		return nil, ErrAttributeMissing
	}
	if p.Lookup == nil {
		return nil, fmt.Errorf("condition: case provider lookup not configured")
	}
	category, tags, err := p.Lookup(ctx, caseID)
	if err != nil {
		return nil, err
	}
	switch field {
	case "case.category":
		if category == "" {
			return nil, ErrAttributeMissing
		}
		return category, nil
	default:
		if len(tags) == 0 {
			return nil, ErrAttributeMissing
		}
		return append([]string(nil), tags...), nil
	}
}
