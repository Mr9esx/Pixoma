package application

import (
	"context"
	"testing"

	"github.com/Mr9esx/Pixoma/internal/platform/db"
)

func TestRequireStudioAssetSchemaRejectsPreviousTables(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:studio_schema_guard?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := requireStudioAssetSchema(context.Background(), gdb); err != nil {
		t.Fatal(err)
	}
	if err := gdb.Exec("CREATE TABLE studio_library_assets (id TEXT PRIMARY KEY)").Error; err != nil {
		t.Fatal(err)
	}
	if err := requireStudioAssetSchema(context.Background(), gdb); err == nil {
		t.Fatal("previous asset schema was accepted")
	}
}
