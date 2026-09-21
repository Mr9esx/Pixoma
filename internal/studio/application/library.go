package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type CreateLibraryFolderInput struct {
	AccountID string
	ParentID  string
	Name      string
}

func (s *Service) CreateLibraryFolder(ctx context.Context, input CreateLibraryFolderInput) (*domain.LibraryFolder, error) {
	if s == nil || s.Repo == nil {
		return nil, fmt.Errorf("studio: library service is not configured")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || strings.TrimSpace(input.AccountID) == "" {
		return nil, fmt.Errorf("%w: folder name is required", domain.ErrInvalid)
	}
	folder, err := domain.NewLibraryFolder(s.nextID(), input.AccountID, input.ParentID, name, s.now())
	if err != nil {
		return nil, err
	}
	if err := s.Repo.CreateLibraryFolder(ctx, folder); err != nil {
		return nil, err
	}
	return folder, nil
}

func (s *Service) ListLibraryFolders(ctx context.Context, accountID string) ([]*domain.LibraryFolder, error) {
	if s == nil || s.Repo == nil {
		return nil, fmt.Errorf("studio: library service is not configured")
	}
	return s.Repo.ListLibraryFolders(ctx, accountID)
}
