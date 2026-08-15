package server

import (
	"net/http"

	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/adminhost"
)

// Options configures the admin-api HTTP handler.
type Options = adminhost.Options

// NewHandler returns the admin-api chi router (health, CORS, resource APIs).
func NewHandler(opts Options) http.Handler {
	return adminhost.NewHandler(opts)
}
