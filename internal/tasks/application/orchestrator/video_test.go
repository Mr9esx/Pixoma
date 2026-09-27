package orchestrator_test

import (
	"context"
	"io"
	"testing"
	"time"

	catalogdomain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	casepersist "github.com/Mr9esx/Pixoma/internal/cases/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/platform/blob/localfs"
	"github.com/Mr9esx/Pixoma/internal/platform/db"
	sessionpersist "github.com/Mr9esx/Pixoma/internal/sessions/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	"github.com/Mr9esx/Pixoma/internal/tasks/application/orchestrator"
	runtimedomain "github.com/Mr9esx/Pixoma/internal/tasks/domain"
	taskpersist "github.com/Mr9esx/Pixoma/internal/tasks/infrastructure/persistence"
)

type videoCaseReader struct{ repo *casepersist.GormRepository }

func (r videoCaseReader) GetCase(ctx context.Context, id sharedkernel.CaseID) (*catalogdomain.CaseDocument, error) {
	c, err := r.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return &c.Document, nil
}

func TestStudioVideoRunsForThirtySecondsAndStoresMP4(t *testing.T) {
	ctx := context.Background()
	gdb, err := db.Open(db.Options{DSN: "file:video_execution?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &casepersist.CaseRow{}, &sessionpersist.SessionRow{}, &taskpersist.TaskRow{}); err != nil {
		t.Fatal(err)
	}
	tasks := taskpersist.NewTaskRepository(gdb)
	cases := casepersist.NewGormRepository(gdb)
	for _, c := range []*catalogdomain.Case{
		{Enabled: true, Document: catalogdomain.CaseDocument{ID: 1, Name: "文生视频", Outputs: []catalogdomain.OutputField{{Key: "video", Type: "video", MediaType: "video/mp4"}}}},
		{Enabled: true, Document: catalogdomain.CaseDocument{ID: 2, Name: "文生图片", Outputs: []catalogdomain.OutputField{{Key: "image", Type: "image"}}}},
		{Enabled: true, Document: catalogdomain.CaseDocument{ID: 3, Name: "图生视频", Outputs: []catalogdomain.OutputField{{Key: "video", Type: "video"}}}},
	} {
		if err := cases.Create(ctx, c); err != nil {
			t.Fatal(err)
		}
	}
	store, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(100, 0).UTC()
	svc := orchestrator.New(tasks, nil, nil, nil)
	svc.Sessions = sessionpersist.NewSessionRepository(gdb)
	svc.Now = func() time.Time { return now }
	svc.VideoBlob = store
	svc.Cases = videoCaseReader{repo: cases}
	videoTask := runtimedomain.NewPending("video-task", "studio-session-one", 1, "inputs/video-task", now)
	imageTask := runtimedomain.NewPending("image-task", "studio-session-one", 2, "inputs/image-task", now)
	otherVideoTask := runtimedomain.NewPending("other-video-task", "studio-session-one", 3, "inputs/other-video-task", now)
	for _, task := range []*runtimedomain.Task{videoTask, imageTask, otherVideoTask} {
		if err := tasks.Create(ctx, task); err != nil {
			t.Fatal(err)
		}
		if err := svc.OnTaskCreated(ctx, sharedkernel.TaskCreated{TaskID: task.ID}); err != nil {
			t.Fatal(err)
		}
	}
	videoTask, err = tasks.Get(ctx, videoTask.ID)
	if err != nil || videoTask.Status != sharedkernel.TaskRunning {
		t.Fatalf("video task status = %v, err = %v", videoTask.Status, err)
	}
	imageTask, err = tasks.Get(ctx, imageTask.ID)
	if err != nil || imageTask.Status != sharedkernel.TaskFailed {
		t.Fatalf("image task status = %v, err = %v", imageTask.Status, err)
	}
	otherVideoTask, err = tasks.Get(ctx, otherVideoTask.ID)
	if err != nil || otherVideoTask.Status != sharedkernel.TaskFailed {
		t.Fatalf("other video task status = %v, err = %v", otherVideoTask.Status, err)
	}
	now = now.Add(29 * time.Second)
	if err := svc.CompleteVideo(ctx); err != nil {
		t.Fatal(err)
	}
	videoTask, err = tasks.Get(ctx, videoTask.ID)
	if err != nil || videoTask.Status != sharedkernel.TaskRunning {
		t.Fatalf("early video task status = %v, err = %v", videoTask.Status, err)
	}
	now = now.Add(time.Second)
	if err := svc.CompleteVideo(ctx); err != nil {
		t.Fatal(err)
	}
	videoTask, err = tasks.Get(ctx, videoTask.ID)
	if err != nil || videoTask.Status != sharedkernel.TaskSucceeded || len(videoTask.Outputs) != 1 {
		t.Fatalf("completed video task = %+v, err = %v", videoTask, err)
	}
	ref := videoTask.Outputs[0].Blob
	if ref.MIME != "video/mp4" || ref.Size == 0 {
		t.Fatalf("video output = %+v", ref)
	}
	reader, err := store.Get(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil || int64(len(data)) != ref.Size {
		t.Fatalf("video bytes = %d, err = %v", len(data), err)
	}
}
