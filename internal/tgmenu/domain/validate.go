package domain

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// CaseExistsFunc checks whether a catalog case id exists.
type CaseExistsFunc func(ctx context.Context, caseID string) (bool, error)

// Validate checks a menu document before persistence.
func Validate(ctx context.Context, doc MenuDocument, caseExists CaseExistsFunc) error {
	if len(doc.Items) == 0 {
		return fmt.Errorf("%w: items must not be empty", ErrValidation)
	}

	seenLabels := make(map[string]struct{}, len(doc.Items))
	seenIDs := make(map[string]struct{}, len(doc.Items))
	enabledCount := 0

	for i, it := range doc.Items {
		id := strings.TrimSpace(it.ID)
		if id == "" {
			return fmt.Errorf("%w: item[%d] id empty", ErrValidation, i)
		}
		if _, ok := seenIDs[id]; ok {
			return fmt.Errorf("%w: duplicate id %q", ErrValidation, id)
		}
		seenIDs[id] = struct{}{}

		label := strings.TrimSpace(it.Label)
		if label == "" {
			return fmt.Errorf("%w: item[%d] label empty", ErrValidation, i)
		}
		if _, ok := seenLabels[label]; ok {
			return fmt.Errorf("%w: duplicate label %q", ErrValidation, label)
		}
		seenLabels[label] = struct{}{}

		if it.Enabled {
			enabledCount++
		}

		switch it.Action {
		case ActionOpenCase:
			if strings.TrimSpace(it.CaseID) == "" {
				return fmt.Errorf("%w: item %q open_case requires case_id", ErrValidation, it.ID)
			}
			if caseExists == nil {
				return fmt.Errorf("%w: item %q open_case requires caseExists", ErrValidation, it.ID)
			}
			ok, err := caseExists(ctx, it.CaseID)
			if err != nil {
				return fmt.Errorf("item %q case lookup: %w", it.ID, err)
			}
			if !ok {
				return fmt.Errorf("%w: item %q case_id %q not found", ErrValidation, it.ID, it.CaseID)
			}
		case ActionListCasesByTag:
			if strings.TrimSpace(it.Tag) == "" {
				return fmt.Errorf("%w: item %q list_cases_by_tag requires tag", ErrValidation, it.ID)
			}
		case ActionPlaceholder:
			// placeholder_text optional
		case ActionReplyMedia:
			if it.Reply == nil {
				return fmt.Errorf("%w: item %q reply_media requires reply", ErrValidation, it.ID)
			}
			textOK := strings.TrimSpace(it.Reply.Text) != ""
			imagesOK := len(it.Reply.Images) > 0
			if !textOK && !imagesOK {
				return fmt.Errorf("%w: item %q reply_media needs text or images", ErrValidation, it.ID)
			}
			for j, raw := range it.Reply.Images {
				u, err := url.Parse(raw)
				if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
					return fmt.Errorf("%w: item %q images[%d] must be http(s) URL", ErrValidation, it.ID, j)
				}
			}
		default:
			return fmt.Errorf("%w: item %q unknown action %q", ErrValidation, it.ID, it.Action)
		}
	}

	if enabledCount == 0 {
		return fmt.Errorf("%w: at least one item must be enabled", ErrValidation)
	}
	return nil
}
