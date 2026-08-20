package actuator_test

import (
	"context"
	"testing"
	"time"

	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/actuator"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestQueryAdapterGetRunMapsTask(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(10, 0).UTC()
	task := runtimedomain.NewPending("t1", "s1", sharedkernel.CaseID(1), "inputs/t1", now)
	_ = task.MarkQueued("local", now)
	_ = task.MarkRunning("prompt-1", now)
	_ = task.MarkSucceeded([]runtimedomain.OutputRef{
		{Key: "out-0", Blob: sharedkernel.BlobRef{Key: "outputs/t1/0.png"}},
	}, now)
	if err := tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}

	q := &actuator.QueryAdapter{Tasks: tasks}
	view, err := q.GetRun(ctx, "t1")
	if err != nil {
		t.Fatal(err)
	}
	if view.Phase != "succeeded" {
		t.Fatalf("phase=%s", view.Phase)
	}
	if view.PromptID != "prompt-1" {
		t.Fatalf("prompt=%s", view.PromptID)
	}
	if len(view.Outputs) != 1 || view.Outputs[0].Key != "outputs/t1/0.png" {
		t.Fatalf("outputs=%+v", view.Outputs)
	}
}

func TestQueryAdapterGetRunUnknownWhenMissing(t *testing.T) {
	ctx := context.Background()
	q := &actuator.QueryAdapter{Tasks: runtimedomain.NewMemoryTaskRepository()}
	view, err := q.GetRun(ctx, "missing")
	if err != nil {
		t.Fatal(err)
	}
	if view.Phase != "unknown" {
		t.Fatalf("phase=%s", view.Phase)
	}
}
