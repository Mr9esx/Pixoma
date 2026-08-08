package actuator_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob/localfs"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/actuator"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type statusCap struct {
	msgs []queue.Message
}

func (s *statusCap) Publish(_ context.Context, msg queue.Message) error {
	s.msgs = append(s.msgs, msg)
	return nil
}

func TestHandleDispatchPublishesRunningAndSucceeded(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := localfs.New(filepath.Join(dir, "blob"))
	if err != nil {
		t.Fatal(err)
	}
	cap := &statusCap{}
	w := &actuator.Worker{
		InstanceID: "local",
		Comfy:      &comfyui.Mock{},
		Blob:       store,
		Status:     cap,
		Workflows:  actuator.StaticWorkflows{},
		Now:        func() time.Time { return time.Unix(1, 0).UTC() },
	}
	if err := w.HandleDispatch(ctx, sharedkernel.DispatchCommand{TaskID: "t1", InstanceID: "local"}); err != nil {
		t.Fatal(err)
	}
	if len(cap.msgs) != 2 {
		t.Fatalf("status msgs=%d", len(cap.msgs))
	}
	var running, done sharedkernel.TaskStatusEvent
	_ = json.Unmarshal(cap.msgs[0].Payload, &running)
	_ = json.Unmarshal(cap.msgs[1].Payload, &done)
	if running.Status != sharedkernel.TaskRunning || done.Status != sharedkernel.TaskSucceeded {
		t.Fatalf("running=%s done=%s", running.Status, done.Status)
	}
	if len(done.Outputs) == 0 {
		t.Fatal("expected outputs")
	}
}
