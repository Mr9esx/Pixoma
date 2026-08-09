package domain

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// CaseExistsFunc checks whether a catalog case id exists.
type CaseExistsFunc func(ctx context.Context, caseID string) (bool, error)

// Validate checks a menu tree before persistence.
func Validate(ctx context.Context, tree MenuTree, caseExists CaseExistsFunc) error {
	if len(tree.Items) == 0 {
		return fmt.Errorf("%w: items must not be empty", ErrValidation)
	}

	flat := Flatten(tree.Items)
	if len(flat) == 0 {
		return fmt.Errorf("%w: items must not be empty", ErrValidation)
	}

	seenIDs := make(map[string]struct{}, len(flat))
	labelsByParent := make(map[string]map[string]struct{})
	enabledRoots := 0

	for i, it := range flat {
		id := strings.TrimSpace(it.ID)
		if id == "" {
			return fmt.Errorf("%w: item[%d] id empty", ErrValidation, i)
		}
		if _, ok := seenIDs[id]; ok {
			return fmt.Errorf("%w: duplicate id %q", ErrValidation, id)
		}
		seenIDs[id] = struct{}{}

		parentKey := it.ParentID
		label := strings.TrimSpace(it.Label)
		if label == "" {
			return fmt.Errorf("%w: item[%d] label empty", ErrValidation, i)
		}
		if labelsByParent[parentKey] == nil {
			labelsByParent[parentKey] = make(map[string]struct{})
		}
		if _, ok := labelsByParent[parentKey][label]; ok {
			return fmt.Errorf("%w: duplicate label %q under parent %q", ErrValidation, label, parentKey)
		}
		labelsByParent[parentKey][label] = struct{}{}

		if it.ParentID == "" && it.Enabled {
			enabledRoots++
		}

		if err := validateItemKind(ctx, it, caseExists); err != nil {
			return err
		}
	}

	allIDs := make(map[string]struct{}, len(flat))
	for _, it := range flat {
		allIDs[it.ID] = struct{}{}
	}
	for _, it := range flat {
		if it.ParentID != "" {
			if _, ok := allIDs[it.ParentID]; !ok {
				return fmt.Errorf("%w: unknown parent_id %q for item %q", ErrValidation, it.ParentID, it.ID)
			}
		}
	}

	if enabledRoots == 0 {
		return fmt.Errorf("%w: at least one root item must be enabled", ErrValidation)
	}

	if DepthOf(tree.Items) > MaxTreeDepth {
		return fmt.Errorf("%w: tree depth exceeds %d", ErrValidation, MaxTreeDepth)
	}

	return validateNodeChildren(tree.Items)
}

func validateNodeChildren(nodes []MenuNode) error {
	for _, n := range nodes {
		if n.Kind == KindOpenCase && len(n.Children) > 0 {
			return fmt.Errorf("%w: item %q open_case must not have children", ErrValidation, n.ID)
		}
		if err := validateNodeChildren(n.Children); err != nil {
			return err
		}
	}
	return nil
}

func validateItemKind(ctx context.Context, it MenuItem, caseExists CaseExistsFunc) error {
	switch it.Kind {
	case KindFolder:
		for _, caseID := range it.CaseIDs {
			caseID = strings.TrimSpace(caseID)
			if caseID == "" {
				return fmt.Errorf("%w: item %q folder has empty case_id", ErrValidation, it.ID)
			}
			if caseExists == nil {
				return fmt.Errorf("%w: item %q folder requires caseExists", ErrValidation, it.ID)
			}
			ok, err := caseExists(ctx, caseID)
			if err != nil {
				return fmt.Errorf("item %q case lookup: %w", it.ID, err)
			}
			if !ok {
				return fmt.Errorf("%w: item %q case_id %q not found", ErrValidation, it.ID, caseID)
			}
		}
	case KindOpenCase:
		if len(it.CaseIDs) != 1 || strings.TrimSpace(it.CaseIDs[0]) == "" {
			return fmt.Errorf("%w: item %q open_case requires exactly one case_id", ErrValidation, it.ID)
		}
		if caseExists == nil {
			return fmt.Errorf("%w: item %q open_case requires caseExists", ErrValidation, it.ID)
		}
		ok, err := caseExists(ctx, it.CaseIDs[0])
		if err != nil {
			return fmt.Errorf("item %q case lookup: %w", it.ID, err)
		}
		if !ok {
			return fmt.Errorf("%w: item %q case_id %q not found", ErrValidation, it.ID, it.CaseIDs[0])
		}
	case KindListCasesByTag:
		if strings.TrimSpace(it.Tag) == "" {
			return fmt.Errorf("%w: item %q list_cases_by_tag requires tag", ErrValidation, it.ID)
		}
		if len(it.CaseIDs) > 0 {
			return fmt.Errorf("%w: item %q list_cases_by_tag must not have case_ids", ErrValidation, it.ID)
		}
	case KindPlaceholder:
		if len(it.CaseIDs) > 0 {
			return fmt.Errorf("%w: item %q placeholder must not have case_ids", ErrValidation, it.ID)
		}
	case KindReplyMedia:
		if len(it.CaseIDs) > 0 {
			return fmt.Errorf("%w: item %q reply_media must not have case_ids", ErrValidation, it.ID)
		}
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
		return fmt.Errorf("%w: item %q unknown kind %q", ErrValidation, it.ID, it.Kind)
	}
	return nil
}
