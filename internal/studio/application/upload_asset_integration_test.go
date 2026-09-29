package application_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/platform/blob/localfs"
	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

func TestUploadAssetCreatesProjectPlacementAndSessionUsage(t *testing.T) {
	ctx := context.Background()
	repo := openRepository(t)
	now := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	project, err := domain.NewProject("project-a", "account-a", "设计项目", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	session, err := domain.NewSession("session-a", "account-a", now)
	if err != nil {
		t.Fatal(err)
	}
	session.ProjectID = project.ID
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	blobs, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var file bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 4, 3))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	if err := png.Encode(&file, img); err != nil {
		t.Fatal(err)
	}
	ids := &idSequence{}
	service := &studioapp.Service{Repo: repo, IDs: ids.Next, Now: func() time.Time { return now }}
	asset, err := service.UploadAsset(ctx, studioapp.UploadAssetInput{
		AccountID: session.AccountID, SessionID: session.ID, Name: "sample.jpg",
		MIMEType: "image/jpeg", Content: bytes.NewReader(file.Bytes()),
	}, blobs)
	if err != nil {
		t.Fatal(err)
	}
	page, err := repo.ListProjectAssets(ctx, session.AccountID, domain.ProjectAssetListQuery{ProjectID: project.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].AssetID != asset.ID || page.Items[0].Version.Format != "png" {
		t.Fatalf("project assets = %#v", page.Items)
	}
	usages, err := repo.ListSessionAssetUsages(ctx, session.AccountID, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(usages) != 1 || usages[0].AssetID != asset.ID || usages[0].AssetVersionID != page.Items[0].AssetVersionID {
		t.Fatalf("session usages = %#v", usages)
	}
	intents, err := repo.ClaimExpiredBlobWriteIntents(ctx, 10, now.Add(30*time.Minute))
	if err != nil || len(intents) != 0 {
		t.Fatalf("blob intents = (%#v, %v)", intents, err)
	}
}

func TestUploadAssetRequestIDReusesContentIdentity(t *testing.T) {
	ctx := context.Background()
	repo := openRepository(t)
	blobs, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	ids := &idSequence{}
	service := &studioapp.Service{Repo: repo, IDs: ids.Next, Now: func() time.Time { return now }}
	requestID := "8aa2388c-e27d-48e6-92de-2cf81d6f1d7f"
	upload := func(content string) (*domain.Asset, error) {
		return service.UploadAsset(ctx, studioapp.UploadAssetInput{
			AccountID: "account-a", ProjectID: "", RequestID: requestID,
			Name: "draft.txt", MIMEType: "text/plain", Content: bytes.NewBufferString(content),
		}, blobs)
	}
	first, err := upload("draft content")
	if err != nil {
		t.Fatal(err)
	}
	second, err := upload("draft content")
	if err != nil || second.ID != first.ID {
		t.Fatalf("retry = (%#v, %v)", second, err)
	}
	if _, err := upload("different content"); err != domain.ErrConflict {
		t.Fatalf("different content error = %v", err)
	}
	project, err := domain.NewProject("project-b", "account-a", "其他项目", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UploadAsset(ctx, studioapp.UploadAssetInput{
		AccountID: "account-a", ProjectID: project.ID, RequestID: requestID,
		Name: "draft.txt", MIMEType: "text/plain", Content: bytes.NewBufferString("draft content"),
	}, blobs); err != domain.ErrConflict {
		t.Fatalf("different project error = %v", err)
	}
	page, err := repo.ListProjectAssets(ctx, "account-a", domain.ProjectAssetListQuery{ProjectID: ""})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("project assets = (%#v, %v)", page, err)
	}
}
