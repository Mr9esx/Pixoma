package smoke_test

import (
	"context"

	"github.com/Mr9esx/Pixoma/internal/tasks/domain/condition"
)

type smokeStubProvider struct {
	val func(ctx context.Context, field string) (any, error)
}

func (s *smokeStubProvider) Namespace() string { return "user" }
func (s *smokeStubProvider) ListAttributes() []condition.AttributeDescriptor {
	return []condition.AttributeDescriptor{{
		Key:     "user.level",
		Context: "user",
		Label:   "Level",
		Schema:  map[string]any{"type": "string", "enum": []any{"image", "video"}},
	}}
}
func (s *smokeStubProvider) Value(ctx context.Context, field string) (any, error) {
	if s.val == nil {
		return nil, condition.ErrAttributeMissing
	}
	return s.val(ctx, field)
}
