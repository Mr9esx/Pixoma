package domain

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// CaseExistsFunc checks whether a catalog case id exists.
type CaseExistsFunc func(ctx context.Context, caseID string) (bool, error)

// CapabilityExistsFunc checks whether a capability id is registered.
type CapabilityExistsFunc func(ctx context.Context, capabilityID string) (bool, error)

// ParamsValidatorFunc validates capability params against its schema.
type ParamsValidatorFunc func(ctx context.Context, capabilityID string, params map[string]any) error

// Validate checks a menu tree before persistence.
func Validate(
	ctx context.Context,
	tree MenuTree,
	caseExists CaseExistsFunc,
	capabilityExists CapabilityExistsFunc,
	validateParams ParamsValidatorFunc,
) error {
	if len(tree.Items) == 0 {
		return fmt.Errorf("%w: items must not be empty", ErrValidation)
	}

	flat := Flatten(tree.Items)
	if len(flat) == 0 {
		return fmt.Errorf("%w: items must not be empty", ErrValidation)
	}

	seenIDs := make(map[string]struct{}, len(flat))
	labelsByParent := make(map[string]map[string]struct{})
	ordersByParent := make(map[string]map[int]struct{})
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

		if ordersByParent[parentKey] == nil {
			ordersByParent[parentKey] = make(map[int]struct{})
		}
		if _, ok := ordersByParent[parentKey][it.Order]; ok {
			return fmt.Errorf("%w: duplicate order %d under parent %q", ErrValidation, it.Order, parentKey)
		}
		ordersByParent[parentKey][it.Order] = struct{}{}

		if it.ParentID == "" && it.Enabled {
			enabledRoots++
		}

		if err := validateItem(ctx, it, caseExists, capabilityExists, validateParams); err != nil {
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
	if enabledRoots > MaxRootEntries {
		return fmt.Errorf("%w: root entries exceed %d", ErrValidation, MaxRootEntries)
	}
	if DepthOf(tree.Items) > MaxTreeDepth {
		return fmt.Errorf("%w: tree depth exceeds %d", ErrValidation, MaxTreeDepth)
	}

	return validateNodeChildren(tree.Items)
}

// validateNodeChildren enforces one-level grouping: a node with children is a
// pure group (no capability, no display fields), and its children must not nest.
func validateNodeChildren(nodes []MenuNode) error {
	for _, n := range nodes {
		if len(n.Children) > 0 {
			if n.CapabilityID != "" {
				return fmt.Errorf("%w: item %q with children must be a pure group (no capability)", ErrValidation, n.ID)
			}
			for _, child := range n.Children {
				if len(child.Children) > 0 {
					return fmt.Errorf("%w: group %q children must not nest groups (one level only)", ErrValidation, n.ID)
				}
			}
		}
		if err := validateNodeChildren(n.Children); err != nil {
			return err
		}
	}
	return nil
}

func validateItem(
	ctx context.Context,
	it MenuItem,
	caseExists CaseExistsFunc,
	capabilityExists CapabilityExistsFunc,
	validateParams ParamsValidatorFunc,
) error {
	if it.CapabilityID != "" {
		if capabilityExists == nil {
			return fmt.Errorf("%w: item %q capability check not configured", ErrValidation, it.ID)
		}
		ok, err := capabilityExists(ctx, it.CapabilityID)
		if err != nil {
			return fmt.Errorf("item %q capability lookup: %w", it.ID, err)
		}
		if !ok {
			return fmt.Errorf("%w: item %q unknown capability %q", ErrValidation, it.ID, it.CapabilityID)
		}
		if validateParams != nil {
			if err := validateParams(ctx, it.CapabilityID, it.Params); err != nil {
				return fmt.Errorf("item %q params: %w", it.ID, err)
			}
		}
		if it.CapabilityID == "open_case" {
			caseIDs := CaseIDsOf(MenuNode{CapabilityID: it.CapabilityID, Params: it.Params})
			if len(caseIDs) == 0 {
				return fmt.Errorf("%w: item %q open_case requires case_ids", ErrValidation, it.ID)
			}
			for _, caseID := range caseIDs {
				caseID = strings.TrimSpace(caseID)
				if caseID == "" {
					return fmt.Errorf("%w: item %q open_case has empty case_id", ErrValidation, it.ID)
				}
				if caseExists == nil {
					return fmt.Errorf("%w: item %q open_case requires caseExists", ErrValidation, it.ID)
				}
				ok, err := caseExists(ctx, caseID)
				if err != nil {
					return fmt.Errorf("item %q case lookup: %w", it.ID, err)
				}
				if !ok {
					return fmt.Errorf("%w: item %q case_id %q not found", ErrValidation, it.ID, caseID)
				}
			}
		}
	}

	if it.Reply != nil {
		textOK := strings.TrimSpace(it.Reply.Text) != ""
		imagesOK := len(it.Reply.Images) > 0
		if !textOK && !imagesOK {
			return fmt.Errorf("%w: item %q reply needs text or images", ErrValidation, it.ID)
		}
		for j, raw := range it.Reply.Images {
			u, err := url.Parse(raw)
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
				return fmt.Errorf("%w: item %q images[%d] must be http(s) URL", ErrValidation, it.ID, j)
			}
		}
	}
	return nil
}
