package actuator_test

import (
	"encoding/json"
	"testing"

	catalogdomain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	"github.com/Mr9esx/Pixoma/internal/tasks/infrastructure/actuator"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

func TestJobPackage_RoundTrip(t *testing.T) {
	job := actuator.JobPackage{
		TaskID: "t1",
		EdgeID: "local",
		Workflow: map[string]any{
			"1": map[string]any{"inputs": map[string]any{"text": "hi"}},
		},
		Images: []actuator.JobImage{
			{
				NodeID:    "2",
				FieldPath: "image",
				Blob:      sharedkernel.BlobRef{Key: "inputs/t1/photo.png", MIME: "image/png"},
			},
		},
		OutputPrefix: "outputs/t1",
		Outputs: []catalogdomain.OutputBinding{
			{Key: "image", NodeID: "9", Index: 0},
		},
	}
	raw, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	var got actuator.JobPackage
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.TaskID != "t1" || got.OutputPrefix != "outputs/t1" {
		t.Fatalf("got=%+v", got)
	}
	if len(got.Outputs) != 1 || got.Outputs[0].Key != "image" || got.Outputs[0].NodeID != "9" {
		t.Fatalf("outputs=%+v", got.Outputs)
	}
	if len(got.Images) != 1 || got.Images[0].Blob.Key != "inputs/t1/photo.png" {
		t.Fatalf("images=%+v", got.Images)
	}
	if got.Workflow["1"] == nil {
		t.Fatal("missing workflow node")
	}
}
