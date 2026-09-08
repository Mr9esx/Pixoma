package sharedkernel_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

func TestDispatchCommand_JobRefRoundTrip(t *testing.T) {
	cmd := sharedkernel.DispatchCommand{
		TaskID: "t1",
		EdgeID: "local",
		JobRef: sharedkernel.BlobRef{Key: "jobs/t1/job.json", MIME: "application/json"},
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
	if got.TaskID != "t1" || got.EdgeID != "local" {
		t.Fatalf("got=%+v", got)
	}
}

func TestTaskStatusEvent_JSONUsesEdgeID(t *testing.T) {
	ev := sharedkernel.TaskStatusEvent{TaskID: "t1", EdgeID: "gpu-1"}
	raw, err := json.Marshal(ev)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`"edge_id":"gpu-1"`)) {
		t.Fatalf("json=%s", raw)
	}
	if bytes.Contains(raw, []byte("instance_id")) {
		t.Fatalf("old json key still present: %s", raw)
	}
}
