package application

import (
	"context"
	"fmt"

	"github.com/mr9esx/comfyui_tgbot/internal/tgmenu/domain"
)

type CaseChecker interface {
	CaseExists(ctx context.Context, caseID string) (bool, error)
}

type Store interface {
	domain.Repository
	EnsureDefault(ctx context.Context) (domain.MenuDocument, error)
}

type Service struct {
	Store Store
	Cases CaseChecker
}

func (s *Service) Get(ctx context.Context) (domain.MenuDocument, error) {
	if s == nil || s.Store == nil {
		return domain.MenuDocument{}, fmt.Errorf("tgmenu service: nil store")
	}
	return s.Store.EnsureDefault(ctx)
}

func (s *Service) Replace(ctx context.Context, items []domain.MenuItem) (domain.MenuDocument, error) {
	if s == nil || s.Store == nil {
		return domain.MenuDocument{}, fmt.Errorf("tgmenu service: nil store")
	}
	doc := domain.MenuDocument{
		ID:    domain.DocumentIDDefault,
		Items: items,
	}
	var caseExists domain.CaseExistsFunc
	if s.Cases != nil {
		caseExists = s.Cases.CaseExists
	}
	if err := domain.Validate(ctx, doc, caseExists); err != nil {
		return domain.MenuDocument{}, err
	}
	if err := s.Store.Replace(ctx, doc); err != nil {
		return domain.MenuDocument{}, err
	}
	return s.Store.Get(ctx, domain.DocumentIDDefault)
}
