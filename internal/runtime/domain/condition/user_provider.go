package condition

import (
	"context"
	"fmt"
)

// UserProvider resolves user.* attributes via an injected lookup.
type UserProvider struct {
	Lookup func(ctx context.Context, userID string) (*bool, error)
}

// Namespace returns "user".
func (p *UserProvider) Namespace() string { return "user" }

// ListAttributes returns the user attributes this provider can resolve.
func (p *UserProvider) ListAttributes() []AttributeDescriptor {
	return []AttributeDescriptor{
		{
			Key:     "user.is_premium",
			Context: "user",
			Label:   "用户是否付费（Premium）",
			Schema:  map[string]any{"type": "boolean", "description": "用户 Telegram Premium 状态"},
		},
	}
}

// Value resolves a single user attribute.
func (p *UserProvider) Value(ctx context.Context, field string) (any, error) {
	if field != "user.is_premium" {
		return nil, fmt.Errorf("%w: %s", ErrUnknownField, field)
	}
	userID, ok := UserIDFrom(ctx)
	if !ok {
		return nil, ErrAttributeMissing
	}
	if p.Lookup == nil {
		return nil, fmt.Errorf("condition: user provider lookup not configured")
	}
	v, err := p.Lookup(ctx, userID)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, ErrAttributeMissing
	}
	return *v, nil
}
