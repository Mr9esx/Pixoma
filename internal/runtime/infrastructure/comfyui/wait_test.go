package comfyui_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
)

// TestHTTPWaitCollectsOutputs 回归：pollHistory 对已完成任务收输出时
// 不能因为 HistoryResult 是 nil map 而 panic（assignment to entry in nil map）。
func TestHTTPWaitCollectsOutputs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/view"):
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("png-bytes"))
		case r.URL.Path == "/history/p1":
			_, _ = w.Write([]byte(`{
				"p1": {
					"status": {"status_str": "success", "completed": true},
					"outputs": {
						"60": {"images": [{"filename": "out.png", "subfolder": "", "type": "output"}]}
					}
				}
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	h := comfyui.NewHTTP(srv.URL)
	res, err := h.Wait(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	out, ok := res.Outputs["60"]
	if !ok || len(out.Images) != 1 || string(out.Images[0].Data) != "png-bytes" {
		t.Fatalf("outputs=%+v", res.Outputs)
	}
}

func TestHTTPWaitCollectsAudioAndVideoOutputs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/view"):
			filename := r.URL.Query().Get("filename")
			if strings.HasSuffix(filename, ".mp4") {
				w.Header().Set("Content-Type", "video/mp4")
				_, _ = w.Write([]byte("mp4-bytes"))
				return
			}
			w.Header().Set("Content-Type", "audio/mpeg")
			_, _ = w.Write([]byte("mp3-bytes"))
		case r.URL.Path == "/history/p1":
			_, _ = w.Write([]byte(`{
				"p1": {
					"status": {"status_str": "success", "completed": true},
					"outputs": {
						"61": {"audio": [{"filename": "out.mp3", "subfolder": "", "type": "output"}]},
						"62": {"gifs": [{"filename": "out.mp4", "subfolder": "", "type": "output"}]}
					}
				}
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	h := comfyui.NewHTTP(srv.URL)
	res, err := h.Wait(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	audio, ok := res.Outputs["61"]
	if !ok || len(audio.Images) != 1 || string(audio.Images[0].Data) != "mp3-bytes" {
		t.Fatalf("audio outputs=%+v", res.Outputs)
	}
	video, ok := res.Outputs["62"]
	if !ok || len(video.Images) != 1 || string(video.Images[0].Data) != "mp4-bytes" {
		t.Fatalf("video outputs=%+v", res.Outputs)
	}
}
