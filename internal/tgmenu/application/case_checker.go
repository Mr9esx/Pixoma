package application

import (
	"context"
	"errors"

	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type CatalogCaseChecker struct {
	Repo catalogdomain.Repository
}

func (c CatalogCaseChecker) CaseExists(ctx context.Context, caseID string) (bool, error) {
	if c.Repo == nil {
		return false, nil
	}
	_, err := c.Repo.Get(ctx, sharedkernel.CaseID(caseID))
	if errors.Is(err, catalogdomain.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
