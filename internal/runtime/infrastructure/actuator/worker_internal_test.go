package actuator

import (
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
)

func TestStripAnnotationNodes(t *testing.T) {
	graph := comfyui.Graph{
		"10": map[string]any{
			"class_type": "LoadImage",
			"inputs":     map[string]any{"image": "ref.png"},
		},
		"99": map[string]any{
			"class_type": "MarkdownNote",
			"inputs":     map[string]any{"widget_0": "## note"},
		},
	}
	stripAnnotationNodes(graph)
	if _, ok := graph["99"]; ok {
		t.Fatal("standalone MarkdownNote must be stripped")
	}
	if _, ok := graph["10"]; !ok {
		t.Fatal("executable node must be kept")
	}
}

func TestStripAnnotationNodesKeepsReferenced(t *testing.T) {
	graph := comfyui.Graph{
		"1": map[string]any{
			"class_type": "MarkdownNote",
			"inputs":     map[string]any{"widget_0": "## note"},
		},
		"2": map[string]any{
			"class_type": "KSampler",
			"inputs":     map[string]any{"note": []any{"1", 0}},
		},
	}
	stripAnnotationNodes(graph)
	if _, ok := graph["1"]; !ok {
		t.Fatal("referenced note node must be kept")
	}
}
