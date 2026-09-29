package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/assetmedia"
)

type PaletteWorker struct {
	Repo domain.Repository
	Blob blob.Store
	Now  func() time.Time
}

func (w *PaletteWorker) ProcessOnce(ctx context.Context, limit int) error {
	if w == nil || w.Repo == nil || w.Blob == nil {
		return fmt.Errorf("studio: palette worker is not configured")
	}
	now := time.Now().UTC()
	if w.Now != nil {
		now = w.Now().UTC()
	}
	jobs, err := w.Repo.ClaimPendingPaletteJobs(ctx, limit, now)
	if err != nil {
		return err
	}
	var failures []error
	for _, job := range jobs {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := w.analyze(ctx, job, now); err != nil {
			failures = append(failures, fmt.Errorf("version %s: %w", job.AssetVersionID, err))
			delay := time.Minute * time.Duration(1<<min(job.Attempts, 6))
			if updateErr := w.Repo.FailAssetVersionPalette(context.WithoutCancel(ctx), job.AccountID, job.AssetVersionID, "analysis_failed", now.Add(delay)); updateErr != nil {
				failures = append(failures, fmt.Errorf("version %s: %w", job.AssetVersionID, updateErr))
			}
		}
	}
	return errors.Join(failures...)
}

func (w *PaletteWorker) analyze(ctx context.Context, job *domain.AssetVersionPalette, now time.Time) error {
	version, err := w.Repo.GetAssetVersion(ctx, job.AccountID, job.AssetVersionID)
	if err != nil {
		return err
	}
	reader, err := w.Blob.Get(ctx, sharedkernel.BlobRef{Key: version.BlobKey, MIME: version.MIMEType, Size: version.SizeBytes})
	if err != nil {
		return err
	}
	defer reader.Close()
	var colors []assetmedia.Color
	var points []float64
	var width, height int
	switch {
	case version.MIMEType == "image/png", version.MIMEType == "image/jpeg", version.MIMEType == "image/gif", version.MIMEType == "image/webp", version.MIMEType == "image/svg+xml":
		result, err := assetmedia.AnalyzeImage(ctx, reader)
		if err != nil {
			return err
		}
		colors = result.Colors
		width, height = result.Width, result.Height
	case version.MIMEType == "video/mp4", version.MIMEType == "video/webm":
		probe, err := w.Blob.Get(ctx, sharedkernel.BlobRef{Key: version.BlobKey, MIME: version.MIMEType, Size: version.SizeBytes})
		if err != nil {
			return err
		}
		defer probe.Close()
		result, err := assetmedia.AnalyzeVideoStream(ctx, probe, reader)
		if err != nil {
			return err
		}
		colors = result.Colors
		points = result.SamplePoints
		width, height = result.Width, result.Height
	default:
		return fmt.Errorf("studio: unsupported palette format %s", version.MIMEType)
	}
	paletteColors := make([]domain.PaletteColor, 0, len(colors))
	for _, color := range colors {
		paletteColors = append(paletteColors, domain.PaletteColor{Hex: color.Hex, Ratio: color.Ratio})
	}
	return w.Repo.CompleteAssetVersionPalette(ctx, job.AccountID, job.AssetVersionID, paletteColors, points, width, height, now)
}
