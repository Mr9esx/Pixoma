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
	EnsureDefault(ctx context.Context, listImageCaseIDs func(context.Context) ([]string, error)) (domain.MenuTree, error)
}

type Service struct {
	Store            Store
	Cases            CaseChecker
	ListImageCaseIDs func(context.Context) ([]string, error)
}

func (s *Service) Get(ctx context.Context) (domain.MenuTree, error) {
	if s == nil || s.Store == nil {
		return domain.MenuTree{}, fmt.Errorf("tgmenu service: nil store")
	}
	return s.Store.EnsureDefault(ctx, s.ListImageCaseIDs)
}

func (s *Service) Replace(ctx context.Context, tree domain.MenuTree) (domain.MenuTree, error) {
	if s == nil || s.Store == nil {
		return domain.MenuTree{}, fmt.Errorf("tgmenu service: nil store")
	}

	tree.ID = domain.DocumentIDDefault
	tree.BotID = domain.BotIDDefault

	var caseExists domain.CaseExistsFunc
	if s.Cases != nil {
		caseExists = s.Cases.CaseExists
	}
	if err := domain.Validate(ctx, tree, caseExists); err != nil {
		return domain.MenuTree{}, err
	}
	if err := s.Store.ReplaceTree(ctx, tree); err != nil {
		return domain.MenuTree{}, err
	}
	return s.Store.GetTree(ctx, domain.DocumentIDDefault)
}

func (s *Service) ListPlacementsByCase(ctx context.Context, caseID string) ([]domain.MenuPlacement, error) {
	if s == nil || s.Store == nil {
		return nil, fmt.Errorf("tgmenu service: nil store")
	}
	return s.Store.ListPlacementsByCase(ctx, caseID)
}
