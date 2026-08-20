package main

import (
	"github.com/mr9esx/comfyui_tgbot/internal/channel/capability"
	"github.com/mr9esx/comfyui_tgbot/internal/packaging/botapp"
)

func botCapabilities(facade *botapp.Facade) *capability.Registry {
	r := capability.NewRegistry()
	_ = r.Register(capability.OpenCase{App: facade})
	return r
}
