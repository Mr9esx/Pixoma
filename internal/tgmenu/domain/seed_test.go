package domain_test

import (
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/tgmenu/domain"
)

func TestDefaultSeedTree_ImageIsFolder(t *testing.T) {
	tree := domain.DefaultSeedTree()
	if tree.ID != domain.DocumentIDDefault || tree.BotID != domain.BotIDDefault {
		t.Fatalf("ids: %+v", tree)
	}
	roots := tree.Items
	if len(roots) != 6 {
		t.Fatalf("roots=%d", len(roots))
	}
	img := roots[0]
	if img.ID != "btn-image" || img.Kind != domain.KindFolder {
		t.Fatalf("want folder btn-image, got %+v", img)
	}
	if img.ParentID != "" {
		t.Fatal("root must have empty parent")
	}
}
