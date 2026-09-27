package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type CreateLibraryCategoryInput struct {
	AccountID string
	ParentID  string
	Name      string
}

func (s *Service) CreateLibraryCategory(ctx context.Context, input CreateLibraryCategoryInput) (*domain.LibraryCategory, error) {
	if s == nil || s.Repo == nil {
		return nil, fmt.Errorf("studio: library service is not configured")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || strings.TrimSpace(input.AccountID) == "" {
		return nil, fmt.Errorf("%w: category name is required", domain.ErrInvalid)
	}
	category, err := domain.NewLibraryCategory(s.nextID(), input.AccountID, input.ParentID, name, s.now())
	if err != nil {
		return nil, err
	}
	if err := s.Repo.CreateLibraryCategory(ctx, category); err != nil {
		return nil, err
	}
	return category, nil
}

func (s *Service) ListLibraryCategories(ctx context.Context, accountID string) ([]*domain.LibraryCategory, error) {
	if s == nil || s.Repo == nil {
		return nil, fmt.Errorf("studio: library service is not configured")
	}
	return s.Repo.ListLibraryCategories(ctx, accountID)
}
