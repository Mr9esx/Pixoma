package application

import (
	"reflect"
	"testing"

	studiopersist "github.com/Mr9esx/Pixoma/internal/studio/infrastructure/persistence"
)

func TestApplicationModelsIncludeStudioSchema(t *testing.T) {
	models := applicationModels()
	want := []any{
		&studiopersist.SessionRow{},
		&studiopersist.MessageRow{},
		&studiopersist.RunRow{},
		&studiopersist.EventRow{},
		&studiopersist.ApprovalRow{},
		&studiopersist.AssetRow{},
		&studiopersist.AssetVersionRow{},
		&studiopersist.LibraryFolderRow{},
		&studiopersist.LibraryAssetRow{},
		&studiopersist.FlowNodeRow{},
		&studiopersist.FlowEdgeRow{},
	}
	for _, expected := range want {
		if !containsModelType(models, expected) {
			t.Errorf("applicationModels() missing %T", expected)
		}
	}
}

func containsModelType(models []any, expected any) bool {
	want := reflect.TypeOf(expected)
	for _, model := range models {
		if reflect.TypeOf(model) == want {
			return true
		}
	}
	return false
}
