package app

import (
	"context"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/capability"
)

// MenuCapabilityChecker adapts the capability registry to the menu validator.
type MenuCapabilityChecker struct {
	Reg *capability.Registry
}

func (c MenuCapabilityChecker) CapabilityExists(_ context.Context, id string) (bool, error) {
	if c.Reg == nil {
		return false, nil
	}
	_, ok := c.Reg.Get(id)
	return ok, nil
}

func (c MenuCapabilityChecker) ValidateParams(ctx context.Context, id string, params map[string]any) error {
	if c.Reg == nil {
		return nil
	}
	return c.Reg.ValidateParams(ctx, id, params)
}
