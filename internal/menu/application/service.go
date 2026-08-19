package application

import (
	"context"
	"fmt"

	"github.com/mr9esx/comfyui_tgbot/internal/menu/domain"
)

type CaseChecker interface {
	CaseExists(ctx context.Context, caseID string) (bool, error)
}

// CapabilityChecker provides capability existence and params validation.
type CapabilityChecker interface {
	CapabilityExists(ctx context.Context, capabilityID string) (bool, error)
	ValidateParams(ctx context.Context, capabilityID string, params map[string]any) error
}

type Store interface {
	domain.Repository
	EnsureDefault(ctx context.Context, channelID string, listImageCaseIDs func(context.Context) ([]string, error)) (domain.MenuTree, error)
}

type Service struct {
	Store            Store
	Cases            CaseChecker
	Capabilities     CapabilityChecker
	ListImageCaseIDs func(context.Context) ([]string, error)
}

func (s *Service) Get(ctx context.Context, channelID string) (domain.MenuTree, error) {
	if s == nil || s.Store == nil {
		return domain.MenuTree{}, fmt.Errorf("tgmenu service: nil store")
	}
	return s.Store.EnsureDefault(ctx, channelID, s.ListImageCaseIDs)
}

func (s *Service) Replace(ctx context.Context, channelID string, tree domain.MenuTree) (domain.MenuTree, error) {
	if s == nil || s.Store == nil {
		return domain.MenuTree{}, fmt.Errorf("tgmenu service: nil store")
	}

	tree.ChannelID = channelID

	var caseExists domain.CaseExistsFunc
	if s.Cases != nil {
		caseExists = s.Cases.CaseExists
	}
	var capExists domain.CapabilityExistsFunc
	var validateParams domain.ParamsValidatorFunc
	if s.Capabilities != nil {
		capExists = s.Capabilities.CapabilityExists
		validateParams = s.Capabilities.ValidateParams
	}
	if err := domain.Validate(ctx, tree, caseExists, capExists, validateParams); err != nil {
		return domain.MenuTree{}, err
	}
	if err := s.Store.ReplaceTree(ctx, tree); err != nil {
		return domain.MenuTree{}, err
	}
	return s.Store.GetTree(ctx, channelID)
}

func (s *Service) ListPlacementsByCase(ctx context.Context, caseID string) ([]domain.MenuPlacement, error) {
	if s == nil || s.Store == nil {
		return nil, fmt.Errorf("tgmenu service: nil store")
	}
	return s.Store.ListPlacementsByCase(ctx, caseID)
}

func (s *Service) ListExtras(ctx context.Context, channelID string) (map[string][]domain.Extra, error) {
	if s == nil || s.Store == nil {
		return nil, fmt.Errorf("tgmenu service: nil store")
	}
	return s.Store.ListExtras(ctx, channelID)
}

func (s *Service) SaveExtras(ctx context.Context, channelID string, extras map[string][]domain.Extra) error {
	if s == nil || s.Store == nil {
		return fmt.Errorf("tgmenu service: nil store")
	}
	return s.Store.SaveExtras(ctx, channelID, extras)
}
