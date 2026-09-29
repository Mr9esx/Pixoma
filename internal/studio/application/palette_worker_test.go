package application_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/platform/blob/localfs"
	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

func TestPaletteWorkerAnalyzesUploadedImage(t *testing.T) {
	ctx := context.Background()
	repo := openRepository(t)
	blobs, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 29, 11, 0, 0, 0, time.UTC)
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	var content bytes.Buffer
	if err := png.Encode(&content, img); err != nil {
		t.Fatal(err)
	}
	service := &studioapp.Service{Repo: repo, IDs: (&idSequence{}).Next, Now: func() time.Time { return now }}
	asset, err := service.UploadAsset(ctx, studioapp.UploadAssetInput{
		AccountID: "account-a", Name: "red.png", MIMEType: "image/png", Content: bytes.NewReader(content.Bytes()),
	}, blobs)
	if err != nil {
		t.Fatal(err)
	}
	worker := &studioapp.PaletteWorker{Repo: repo, Blob: blobs, Now: func() time.Time { return now }}
	if err := worker.ProcessOnce(ctx, 10); err != nil {
		t.Fatal(err)
	}
	palette, err := repo.GetAssetVersionPalette(ctx, "account-a", asset.Versions[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if palette.Status != "ready" || len(palette.Colors) != 1 || palette.Colors[0].Hex != "#FF0000" {
		t.Fatalf("palette = %#v", palette)
	}
}

func TestPaletteWorkerAnalyzesUploadedSVG(t *testing.T) {
	ctx := context.Background()
	repo := openRepository(t)
	blobs, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 29, 11, 0, 0, 0, time.UTC)
	content := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="20" height="10" viewBox="0 0 20 10"><rect width="20" height="10" fill="#ff0000"/></svg>`)
	service := &studioapp.Service{Repo: repo, IDs: (&idSequence{}).Next, Now: func() time.Time { return now }}
	asset, err := service.UploadAsset(ctx, studioapp.UploadAssetInput{
		AccountID: "account-a", Name: "red.svg", MIMEType: "image/png", Content: bytes.NewReader(content),
	}, blobs)
	if err != nil {
		t.Fatal(err)
	}
	worker := &studioapp.PaletteWorker{Repo: repo, Blob: blobs, Now: func() time.Time { return now }}
	if err := worker.ProcessOnce(ctx, 10); err != nil {
		t.Fatal(err)
	}
	palette, err := repo.GetAssetVersionPalette(ctx, "account-a", asset.Versions[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if palette.Status != "ready" || len(palette.Colors) != 1 || palette.Colors[0].Hex != "#FF0000" {
		t.Fatalf("SVG palette = %#v", palette)
	}
}

func TestBlobIntentWorkerRemovesExpiredUnreferencedContent(t *testing.T) {
	ctx := context.Background()
	repo := openRepository(t)
	blobs, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	ref, err := blobs.Put(ctx, "studio/account-a/orphan.png", bytes.NewReader([]byte("orphan")), blob.PutOptions{MIME: "image/png"})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.RegisterBlobWriteIntent(ctx, &domain.BlobWriteIntent{
		BlobKey: ref.Key, AccountID: "account-a", CreatedAt: now.Add(-2 * time.Minute), ExpiresAt: now.Add(-time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	worker := &studioapp.BlobIntentWorker{Repo: repo, Blob: blobs, Now: func() time.Time { return now }}
	if err := worker.ProcessOnce(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if _, err := blobs.Get(ctx, ref); err == nil {
		t.Fatal("expired content remains")
	}
	remaining, err := repo.ClaimExpiredBlobWriteIntents(ctx, 10, now.Add(2*time.Minute))
	if err != nil || len(remaining) != 0 {
		t.Fatalf("remaining intents = (%#v, %v)", remaining, err)
	}
}
