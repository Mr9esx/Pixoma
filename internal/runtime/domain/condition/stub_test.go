package condition

import "context"

type stubProvider struct {
	ns    string
	attrs []AttributeDescriptor
	val   func(ctx context.Context, field string) (any, error)
}

func (s *stubProvider) Namespace() string { return s.ns }

func (s *stubProvider) ListAttributes() []AttributeDescriptor { return s.attrs }

func (s *stubProvider) Value(ctx context.Context, field string) (any, error) {
	if s.val != nil {
		return s.val(ctx, field)
	}
	return nil, ErrAttributeMissing
}
