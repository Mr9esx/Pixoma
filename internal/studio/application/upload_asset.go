package application

import (
	"context"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"

	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type UploadAssetInput struct {
	AccountID string
	SessionID string
	Name      string
	MIMEType  string
	Content   io.Reader
}

func (s *Service) UploadAsset(ctx context.Context, input UploadAssetInput, blobs blob.Store) (*domain.Asset, error) {
	if s == nil || s.Repo == nil || blobs == nil || input.Content == nil {
		return nil, fmt.Errorf("studio: upload asset service is not configured")
	}
	if strings.TrimSpace(input.AccountID) == "" || strings.TrimSpace(input.Name) == "" {
		return nil, fmt.Errorf("%w: asset name is required", domain.ErrInvalid)
	}
	if input.SessionID != "" {
		if _, err := s.Repo.GetSession(ctx, input.AccountID, input.SessionID); err != nil {
			return nil, err
		}
	}
	mimeType := strings.TrimSpace(input.MIMEType)
	if mimeType == "" {
		mimeType = mime.TypeByExtension(filepath.Ext(input.Name))
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	kind := domain.AssetFile
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		kind = domain.AssetImage
	case strings.HasPrefix(mimeType, "video/"):
		kind = domain.AssetVideo
	case strings.HasPrefix(mimeType, "audio/"):
		kind = domain.AssetAudio
	case mimeType == "text/markdown" || strings.HasPrefix(mimeType, "text/"):
		kind = domain.AssetDocument
	}
	asset, err := domain.NewAsset(s.nextID(), input.SessionID, input.AccountID, filepath.Base(input.Name), kind, domain.AssetOriginUser, s.now())
	if err != nil {
		return nil, err
	}
	key := filepath.ToSlash(filepath.Join("studio", input.AccountID, "uploads", asset.ID, "v1", asset.Name))
	ref, err := blobs.Put(ctx, key, io.LimitReader(input.Content, 50<<20), blob.PutOptions{MIME: mimeType})
	if err != nil {
		return nil, fmt.Errorf("studio: save uploaded asset: %w", err)
	}
	if _, err := asset.AppendVersion(s.nextID(), mimeType, ref.Key, ref.Size, s.now()); err != nil {
		return nil, err
	}
	if err := s.Repo.CreateAsset(ctx, asset); err != nil {
		return nil, err
	}
	if input.SessionID == "" {
		if err := s.Repo.SaveAssetToLibrary(ctx, input.AccountID, asset.ID, "", s.now()); err != nil {
			return nil, err
		}
	}
	return asset, nil
}
