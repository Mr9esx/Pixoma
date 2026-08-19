package main

import (
	"context"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/capability"
	"github.com/mr9esx/comfyui_tgbot/internal/packaging/botapp"
)

func botCapabilities(facade *botapp.Facade) *capability.Registry {
	r := capability.NewRegistry()
	_ = r.Register(capability.OpenCase{App: facade})
	return r
}

type botMenuCaps struct {
	reg *capability.Registry
}

func (c botMenuCaps) CapabilityExists(_ context.Context, id string) (bool, error) {
	if c.reg == nil {
		return false, nil
	}
	_, ok := c.reg.Get(id)
	return ok, nil
}

func (c botMenuCaps) ValidateParams(ctx context.Context, id string, params map[string]any) error {
	if c.reg == nil {
		return nil
	}
	return c.reg.ValidateParams(ctx, id, params)
}
