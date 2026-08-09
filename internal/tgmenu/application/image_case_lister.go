package application

import (
	"context"

	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
)

// CatalogImageCaseIDs adapts catalog List(Tag:"image") for menu seeding.
func CatalogImageCaseIDs(repo catalogdomain.Repository) func(context.Context) ([]string, error) {
	return func(ctx context.Context) ([]string, error) {
		if repo == nil {
			return nil, nil
		}
		cases, err := repo.List(ctx, catalogdomain.ListQuery{Tag: "image"})
		if err != nil {
			return nil, err
		}
		ids := make([]string, 0, len(cases))
		for _, c := range cases {
			if c == nil {
				continue
			}
			ids = append(ids, string(c.Document.ID))
		}
		return ids, nil
	}
}
