package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

const maxManualTextAssetBytes = 1 << 20

type CreateManualTextAssetInput struct {
	AccountID string
	SessionID string
	RequestID string
	Name      string
	Content   string
}

type UpdateManualTextAssetInput struct {
	AccountID string
	AssetID   string
	RequestID string
	Content   string
}

// CreateManualTextAsset 保存用户编写的 Markdown 文档及其项目和对话关联。
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
	session, err := s.Repo.GetSession(ctx, input.AccountID, input.SessionID)
	if err != nil {
		return nil, err
	}
	content := []byte(input.Content)
	hash := sha256.Sum256(content)
	requestID := strings.TrimSpace(input.RequestID)
	assetID := s.nextID()
	if requestID != "" {
		if _, err := uuid.Parse(requestID); err != nil {
			return nil, fmt.Errorf("%w: request_id must be a UUID", domain.ErrInvalid)
		}
		assetID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(input.AccountID+"\x00manual\x00"+requestID)).String()
		if existing, err := s.Repo.GetAsset(ctx, input.AccountID, assetID); err == nil {
			if len(existing.Versions) > 0 && existing.Versions[0].SHA256 == hex.EncodeToString(hash[:]) {
				return existing, nil
			}
			return nil, domain.ErrConflict
		} else if !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
	}
	now := s.now()
	asset, err := domain.NewAsset(assetID, input.AccountID, input.Name, domain.AssetDocument, domain.AssetOriginUser, now)
	if err != nil {
		return nil, err
	}
	asset.CreationKey = requestID
	key := filepath.ToSlash(filepath.Join("studio", input.AccountID, "manual", asset.ID, uuid.NewString()+".md"))
	ref, err := putOwnedBlob(ctx, s.Repo, blobs, input.AccountID, key, "text/markdown", bytes.NewReader(content), now)
	if err != nil {
		return nil, fmt.Errorf("studio: save text asset: %w", err)
	}
	version, err := asset.AppendVersion(s.nextID(), "text/markdown", ref.Key, ref.Size, now)
	if err != nil {
		_ = cleanupOwnedBlob(ctx, s.Repo, blobs, input.AccountID, ref)
		return nil, err
	}
	version.Format = "md"
	version.SHA256 = hex.EncodeToString(hash[:])
	version.ContentOrigin = "manual"
	version.OperationKey = requestID
	if version.OperationKey == "" {
		version.OperationKey = asset.ID
	}
	asset.Versions[0] = version
	placement := &domain.ProjectAsset{
		ID: s.nextID(), AccountID: input.AccountID, ProjectID: session.ProjectID,
		AssetID: asset.ID, AssetVersionID: version.ID, DisplayName: asset.Name,
		AddedAt: now, UpdatedAt: now,
	}
	usage := &domain.SessionAssetUsage{
		ID: s.nextID(), AccountID: input.AccountID, SessionID: input.SessionID,
		AssetID: asset.ID, AssetVersionID: version.ID, UsageKind: "created",
		OperationKey: asset.ID, CreatedAt: now,
	}
	if err := s.Repo.CreateAssetWithPlacement(ctx, asset, placement, usage); err != nil {
		_ = cleanupOwnedBlob(ctx, s.Repo, blobs, input.AccountID, ref)
		if requestID != "" && errors.Is(err, domain.ErrAlreadyExists) {
			if existing, getErr := s.Repo.GetAsset(ctx, input.AccountID, asset.ID); getErr == nil && len(existing.Versions) > 0 && existing.Versions[0].SHA256 == hex.EncodeToString(hash[:]) {
				return existing, nil
			}
		}
		return nil, err
	}
	return asset, nil
}

// UpdateManualTextAsset 增加不可变的 Markdown 版本，现有项目条目继续引用原版本。
func (s *Service) UpdateManualTextAsset(ctx context.Context, input UpdateManualTextAssetInput, blobs blob.Store) (*domain.Asset, error) {
	if s == nil || s.Repo == nil || blobs == nil {
		return nil, fmt.Errorf("studio: asset service is not configured")
	}
	input.AccountID = strings.TrimSpace(input.AccountID)
	input.AssetID = strings.TrimSpace(input.AssetID)
	if input.AccountID == "" || input.AssetID == "" || strings.TrimSpace(input.Content) == "" {
		return nil, fmt.Errorf("%w: asset content is required", domain.ErrInvalid)
	}
	if len([]byte(input.Content)) > maxManualTextAssetBytes {
		return nil, fmt.Errorf("%w: text asset exceeds 1 MiB", domain.ErrInvalid)
	}
	asset, err := s.Repo.GetAsset(ctx, input.AccountID, input.AssetID)
	if err != nil {
		return nil, err
	}
	if asset.Kind != domain.AssetDocument || len(asset.Versions) == 0 || asset.Versions[0].ContentOrigin != "manual" {
		return nil, fmt.Errorf("%w: only manual documents can be edited", domain.ErrInvalid)
	}
	hash := sha256.Sum256([]byte(input.Content))
	requestID := strings.TrimSpace(input.RequestID)
	if requestID != "" {
		if _, err := uuid.Parse(requestID); err != nil {
			return nil, fmt.Errorf("%w: request_id must be a UUID", domain.ErrInvalid)
		}
		for _, version := range asset.Versions {
			if version.OperationKey == requestID {
				if version.SHA256 == hex.EncodeToString(hash[:]) {
					return asset, nil
				}
				return nil, domain.ErrConflict
			}
		}
	}
	now := s.now()
	versionNumber := asset.CurrentVersion + 1
	key := filepath.ToSlash(filepath.Join("studio", input.AccountID, "manual", asset.ID, fmt.Sprintf("v%d-%s.md", versionNumber, uuid.NewString())))
	ref, err := putOwnedBlob(ctx, s.Repo, blobs, input.AccountID, key, "text/markdown", bytes.NewReader([]byte(input.Content)), now)
	if err != nil {
		return nil, fmt.Errorf("studio: save updated text asset: %w", err)
	}
	version, err := asset.AppendVersion(s.nextID(), "text/markdown", ref.Key, ref.Size, now)
	if err != nil {
		_ = cleanupOwnedBlob(ctx, s.Repo, blobs, input.AccountID, ref)
		return nil, err
	}
	version.Format = "md"
	version.SHA256 = hex.EncodeToString(hash[:])
	version.ContentOrigin = "manual"
	version.OperationKey = requestID
	if version.OperationKey == "" {
		version.OperationKey = version.ID
	}
	if err := s.Repo.AppendAssetVersion(ctx, asset.ID, input.AccountID, version); err != nil {
		_ = cleanupOwnedBlob(ctx, s.Repo, blobs, input.AccountID, ref)
		if requestID != "" && errors.Is(err, domain.ErrAlreadyExists) {
			return s.Repo.GetAsset(ctx, input.AccountID, asset.ID)
		}
		return nil, err
	}
	asset.Versions[len(asset.Versions)-1] = version
	return asset, nil
}
