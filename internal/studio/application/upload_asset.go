package application

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/assetmedia"
)

type UploadAssetInput struct {
	AccountID        string
	SessionID        string
	ProjectID        string
	RequestID        string
	Name             string
	MIMEType         string
	Content          io.Reader
	SourceModifiedAt *time.Time
}

func (s *Service) UploadAsset(ctx context.Context, input UploadAssetInput, blobs blob.Store) (*domain.Asset, error) {
	if s == nil || s.Repo == nil || blobs == nil || input.Content == nil {
		return nil, fmt.Errorf("studio: upload asset service is not configured")
	}
	if strings.TrimSpace(input.AccountID) == "" || strings.TrimSpace(input.Name) == "" {
		return nil, fmt.Errorf("%w: asset name is required", domain.ErrInvalid)
	}
	projectID := strings.TrimSpace(input.ProjectID)
	if input.SessionID != "" {
		session, err := s.Repo.GetSession(ctx, input.AccountID, input.SessionID)
		if err != nil {
			return nil, err
		}
		if projectID != "" && projectID != session.ProjectID {
			return nil, fmt.Errorf("%w: session and project differ", domain.ErrInvalid)
		}
		projectID = session.ProjectID
	} else if projectID != "" {
		if _, err := s.Repo.GetProject(ctx, input.AccountID, projectID); err != nil {
			return nil, err
		}
	}
	content, err := io.ReadAll(io.LimitReader(input.Content, (50<<20)+1))
	if err != nil {
		return nil, fmt.Errorf("studio: read uploaded asset: %w", err)
	}
	if len(content) > 50<<20 {
		return nil, fmt.Errorf("%w: uploaded asset exceeds 50 MiB", domain.ErrInvalid)
	}
	info, err := assetmedia.InspectFile(ctx, content, input.MIMEType)
	if err != nil {
		return nil, err
	}
	reuseExisting := func(existing *domain.Asset) (*domain.Asset, error) {
		if len(existing.Versions) == 0 || existing.Versions[0].SHA256 != info.SHA256 || existing.Name != filepath.Base(input.Name) {
			return nil, domain.ErrConflict
		}
		if _, err := s.Repo.GetProjectAssetByAsset(ctx, input.AccountID, projectID, existing.ID); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, domain.ErrConflict
			}
			return nil, err
		}
		if input.SessionID != "" {
			usages, err := s.Repo.ListSessionAssetUsages(ctx, input.AccountID, input.SessionID)
			if err != nil {
				return nil, err
			}
			for _, usage := range usages {
				if usage.AssetID == existing.ID && usage.UsageKind == "uploaded" {
					return existing, nil
				}
			}
			return nil, domain.ErrConflict
		}
		return existing, nil
	}
	kind := domain.AssetFile
	switch {
	case strings.HasPrefix(info.MIMEType, "image/"):
		kind = domain.AssetImage
	case strings.HasPrefix(info.MIMEType, "video/"):
		kind = domain.AssetVideo
	case strings.HasPrefix(info.MIMEType, "audio/"):
		kind = domain.AssetAudio
	case info.MIMEType == "text/markdown" || strings.HasPrefix(info.MIMEType, "text/"):
		kind = domain.AssetDocument
	}
	assetID := s.nextID()
	requestID := strings.TrimSpace(input.RequestID)
	if requestID != "" {
		if _, err := uuid.Parse(requestID); err != nil {
			return nil, fmt.Errorf("%w: request_id must be a UUID", domain.ErrInvalid)
		}
		assetID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(input.AccountID+"\x00upload\x00"+requestID)).String()
		if existing, err := s.Repo.GetAsset(ctx, input.AccountID, assetID); err == nil {
			return reuseExisting(existing)
		} else if !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
	}
	now := s.now()
	asset, err := domain.NewAsset(assetID, input.AccountID, filepath.Base(input.Name), kind, domain.AssetOriginUser, now)
	if err != nil {
		return nil, err
	}
	asset.CreationKey = requestID
	key := filepath.ToSlash(filepath.Join("studio", input.AccountID, "uploads", asset.ID, uuid.NewString(), asset.Name))
	ref, err := putOwnedBlob(ctx, s.Repo, blobs, input.AccountID, key, info.MIMEType, bytes.NewReader(content), now)
	if err != nil {
		return nil, fmt.Errorf("studio: save uploaded asset: %w", err)
	}
	version, err := asset.AppendVersion(s.nextID(), info.MIMEType, ref.Key, ref.Size, now)
	if err != nil {
		_ = cleanupOwnedBlob(ctx, s.Repo, blobs, input.AccountID, ref)
		return nil, err
	}
	version.Format = info.Format
	version.SHA256 = info.SHA256
	version.ContentOrigin = "upload"
	version.OperationKey = requestID
	if version.OperationKey == "" {
		version.OperationKey = asset.ID
	}
	version.SourceModifiedAt = input.SourceModifiedAt
	if info.WidthPx > 0 && info.HeightPx > 0 {
		version.WidthPx = &info.WidthPx
		version.HeightPx = &info.HeightPx
	}
	asset.Versions[0] = version
	placement := &domain.ProjectAsset{
		ID: s.nextID(), AccountID: input.AccountID, ProjectID: projectID,
		AssetID: asset.ID, AssetVersionID: version.ID, DisplayName: asset.Name,
		AddedAt: now, UpdatedAt: now,
	}
	var usage *domain.SessionAssetUsage
	if input.SessionID != "" {
		usage = &domain.SessionAssetUsage{
			ID: s.nextID(), AccountID: input.AccountID, SessionID: input.SessionID,
			AssetID: asset.ID, AssetVersionID: version.ID, UsageKind: "uploaded",
			OperationKey: asset.ID, CreatedAt: now,
		}
	}
	if err := s.Repo.CreateAssetWithPlacement(ctx, asset, placement, usage); err != nil {
		_ = cleanupOwnedBlob(ctx, s.Repo, blobs, input.AccountID, ref)
		if requestID != "" && errors.Is(err, domain.ErrAlreadyExists) {
			if existing, getErr := s.Repo.GetAsset(ctx, input.AccountID, asset.ID); getErr == nil {
				return reuseExisting(existing)
			}
		}
		return nil, err
	}
	return asset, nil
}
