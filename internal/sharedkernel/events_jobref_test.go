package sharedkernel_test

import (
	"encoding/json"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestDispatchCommand_JobRefRoundTrip(t *testing.T) {
	cmd := sharedkernel.DispatchCommand{
		TaskID:     "t1",
		InstanceID: "local",
		JobRef:     sharedkernel.BlobRef{Key: "jobs/t1/job.json", MIME: "application/json"},
	}
	raw, err := json.Marshal(cmd)
	if err != nil {
		t.Fatal(err)
	}
	var got sharedkernel.DispatchCommand
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.JobRef.Key != "jobs/t1/job.json" {
		t.Fatalf("job_ref.key=%q", got.JobRef.Key)
	}
	if got.TaskID != "t1" || got.InstanceID != "local" {
		t.Fatalf("got=%+v", got)
	}
}
