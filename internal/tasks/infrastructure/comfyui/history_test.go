package comfyui_test

import (
	"testing"

	"github.com/Mr9esx/Pixoma/internal/tasks/infrastructure/comfyui"
)

func TestParseHistoryGroupsByNode(t *testing.T) {
	raw := []byte(`{
		"prompt-id-1": {
			"outputs": {
				"4": { "images": [{ "filename": "a.png", "subfolder": "", "type": "output" }] },
				"5": { "audio": [{ "filename": "a.mp3", "subfolder": "audio", "type": "output" }] },
				"6": { "gifs": [{ "filename": "a.mp4", "subfolder": "video", "type": "output" }] },
				"9": { "text": ["hello", "world"] }
			},
			"status": { "status_str": "success", "completed": true }
		}
	}`)
	got, err := comfyui.ParseHistory(raw)
	if err != nil {
		t.Fatal(err)
	}
	node4, ok := got["4"]
	if !ok || len(node4.Images) != 1 || node4.Images[0].Filename != "a.png" {
		t.Fatalf("node 4 images: %+v", got["4"])
	}
	if node4.Images[0].Subfolder != "" || node4.Images[0].Type != "output" {
		t.Fatalf("node 4 image meta: %+v", node4.Images[0])
	}
	node5, ok := got["5"]
	if !ok || len(node5.Images) != 1 || node5.Images[0].Filename != "a.mp3" {
		t.Fatalf("node 5 audio: %+v", got["5"])
	}
	node6, ok := got["6"]
	if !ok || len(node6.Images) != 1 || node6.Images[0].Filename != "a.mp4" {
		t.Fatalf("node 6 gifs: %+v", got["6"])
	}
	node9, ok := got["9"]
	if !ok || len(node9.Texts) != 2 || node9.Texts[0] != "hello" {
		t.Fatalf("node 9 texts: %+v", got["9"])
	}
}

func TestParseHistorySkipsEmptyNodes(t *testing.T) {
	raw := []byte(`{
		"prompt-id-1": {
			"outputs": { "7": { "images": [] } },
			"status": { "status_str": "success", "completed": true }
		}
	}`)
	got, err := comfyui.ParseHistory(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("want empty result, got %+v", got)
	}
}

func TestParseHistoryRejectsBadJSON(t *testing.T) {
	if _, err := comfyui.ParseHistory([]byte("not json")); err == nil {
		t.Fatal("expected error")
	}
}
