package application

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

const maxManualTextAssetBytes = 1 << 20

type CreateManualTextAssetInput struct {
	AccountID string
	SessionID string
	Name      string
	Content   string
}

// CreateManualTextAsset records a user-authored Markdown document as a real
// session asset. It intentionally does not create a run: this is a Studio
// capability, not a workflow side effect.
func (s *Service) CreateManualTextAsset(ctx context.Context, input CreateManualTextAssetInput, blobs blob.Store) (*domain.Asset, error) {
	if s == nil || s.Repo == nil || blobs == nil {
		return nil, fmt.Errorf("studio: asset service is not configured")
	}
	input.AccountID = strings.TrimSpace(input.AccountID)
	input.SessionID = strings.TrimSpace(input.SessionID)
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		input.Name = "未命名文档.md"
	}
	if !strings.HasSuffix(strings.ToLower(input.Name), ".md") {
		input.Name += ".md"
	}
	if input.AccountID == "" || input.SessionID == "" || strings.TrimSpace(input.Content) == "" {
		return nil, fmt.Errorf("%w: session and asset content are required", domain.ErrInvalid)
	}
	if len([]byte(input.Content)) > maxManualTextAssetBytes {
		return nil, fmt.Errorf("%w: text asset exceeds 1 MiB", domain.ErrInvalid)
	}
	if _, err := s.Repo.GetSession(ctx, input.AccountID, input.SessionID); err != nil {
		return nil, err
	}
	now := s.now()
	asset, err := domain.NewAsset(s.nextID(), input.SessionID, input.AccountID, input.Name, domain.AssetDocument, domain.AssetOriginUser, now)
	if err != nil {
		return nil, err
	}
	key := filepath.ToSlash(filepath.Join("studio", input.AccountID, input.SessionID, asset.ID, "v1.md"))
	ref, err := blobs.Put(ctx, key, bytes.NewReader([]byte(input.Content)), blob.PutOptions{MIME: "text/markdown"})
	if err != nil {
		return nil, fmt.Errorf("studio: save text asset: %w", err)
	}
	if _, err := asset.AppendVersion(s.nextID(), "text/markdown", ref.Key, ref.Size, now); err != nil {
		return nil, err
	}
	if err := s.Repo.CreateAsset(ctx, asset); err != nil {
		return nil, err
	}
	return asset, nil
}
