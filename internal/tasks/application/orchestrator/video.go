package orchestrator

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	runtimedomain "github.com/Mr9esx/Pixoma/internal/tasks/domain"
)

const videoExecutionDuration = 30 * time.Second

//go:embed video.mp4
var videoMP4 []byte

func (s *Service) startVideo(ctx context.Context, task *runtimedomain.Task) (bool, error) {
	caseDoc, err := s.Cases.GetCase(ctx, task.CaseID)
	if err != nil {
		return false, nil
	}
	if caseDoc == nil || caseDoc.Name != "文生视频" || len(caseDoc.Outputs) != 1 || caseDoc.Outputs[0].Type != "video" {
		return false, nil
	}
	claimed, err := s.Tasks.ClaimQueued(ctx, task.ID, "", s.Now())
	if err != nil || !claimed {
		return claimed, err
	}
	return true, s.OnStatus(ctx, sharedkernel.TaskStatusEvent{
		TaskID: task.ID, Status: sharedkernel.TaskRunning,
		PromptID: "video:" + string(task.ID), At: s.Now(),
	})
}

func (s *Service) CompleteVideo(ctx context.Context) error {
	if s.VideoBlob == nil {
		return nil
	}
	tasks, err := s.Tasks.ListByStatus(ctx, sharedkernel.TaskRunning, 0)
	if err != nil {
		return err
	}
	now := s.Now()
	for _, task := range tasks {
		if task.PromptID != "video:"+string(task.ID) || now.Sub(task.StartedAt) < videoExecutionDuration {
			continue
		}
		ref, err := s.VideoBlob.Put(ctx, fmt.Sprintf("outputs/%s/video.mp4", task.ID), bytes.NewReader(videoMP4), blob.PutOptions{MIME: "video/mp4"})
		if err != nil {
			return fmt.Errorf("store video output: %w", err)
		}
		if err := s.OnStatus(ctx, sharedkernel.TaskStatusEvent{
			TaskID: task.ID, Status: sharedkernel.TaskSucceeded,
			Outputs: []sharedkernel.BlobRef{ref}, At: now,
		}); err != nil {
			return fmt.Errorf("complete video task: %w", err)
		}
	}
	return nil
}
