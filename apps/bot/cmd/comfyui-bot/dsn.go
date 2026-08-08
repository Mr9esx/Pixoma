package main

import (
	"os"
	"path/filepath"
)

// resolveDSN picks the SQLite DSN shared with admin-api.
// Non-empty configured (yaml/env) wins; otherwise DATA_DIR/app.db (default data/).
func resolveDSN(configured string) string {
	if configured != "" {
		return configured
	}
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	return filepath.Join(dataDir, "app.db")
}
