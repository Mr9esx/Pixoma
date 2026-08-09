package domain

import "fmt"

// Flatten converts a nested menu tree into a flat list of items with ParentID set.
func Flatten(nodes []MenuNode) []MenuItem {
	var out []MenuItem
	flattenNodes(nodes, "", &out)
	return out
}

func flattenNodes(nodes []MenuNode, parentID string, out *[]MenuItem) {
	for _, n := range nodes {
		*out = append(*out, nodeToItem(n, parentID))
		if len(n.Children) > 0 {
			flattenNodes(n.Children, n.ID, out)
		}
	}
}

func nodeToItem(n MenuNode, parentID string) MenuItem {
	return MenuItem{
		ID:              n.ID,
		ParentID:        parentID,
		Label:           n.Label,
		Row:             n.Row,
		Col:             n.Col,
		Enabled:         n.Enabled,
		Kind:            n.Kind,
		CaseIDs:         append([]string(nil), n.CaseIDs...),
		Tag:             n.Tag,
		PlaceholderText: n.PlaceholderText,
		IntroText:       n.IntroText,
		Reply:           n.Reply,
	}
}

// BuildTree assembles a nested menu from a flat item list.
func BuildTree(items []MenuItem) ([]MenuNode, error) {
	if len(items) == 0 {
		return nil, nil
	}

	byID := make(map[string]MenuItem, len(items))
	childrenOf := make(map[string][]string)
	var roots []string

	for _, it := range items {
		id := it.ID
		if _, ok := byID[id]; ok {
			return nil, fmt.Errorf("%w: duplicate id %q", ErrValidation, id)
		}
		byID[id] = it

		parent := it.ParentID
		if parent == "" {
			roots = append(roots, id)
			continue
		}
		if _, ok := byID[parent]; !ok {
			// Parent may appear later in the slice; defer check.
		}
		childrenOf[parent] = append(childrenOf[parent], id)
	}

	for _, it := range items {
		if it.ParentID != "" {
			if _, ok := byID[it.ParentID]; !ok {
				return nil, fmt.Errorf("%w: unknown parent_id %q for item %q", ErrValidation, it.ParentID, it.ID)
			}
		}
	}

	var build func(id string) (MenuNode, error)
	build = func(id string) (MenuNode, error) {
		it := byID[id]
		node := itemToNode(it)
		for _, childID := range childrenOf[id] {
			child, err := build(childID)
			if err != nil {
				return MenuNode{}, err
			}
			node.Children = append(node.Children, child)
		}
		return node, nil
	}

	nodes := make([]MenuNode, 0, len(roots))
	for _, rootID := range roots {
		node, err := build(rootID)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func itemToNode(it MenuItem) MenuNode {
	return MenuNode{
		ID:              it.ID,
		ParentID:        it.ParentID,
		Label:           it.Label,
		Row:             it.Row,
		Col:             it.Col,
		Enabled:         it.Enabled,
		Kind:            it.Kind,
		CaseIDs:         append([]string(nil), it.CaseIDs...),
		Tag:             it.Tag,
		PlaceholderText: it.PlaceholderText,
		IntroText:       it.IntroText,
		Reply:           it.Reply,
	}
}

// DepthOf returns the maximum depth of nodes (root depth = 1).
func DepthOf(nodes []MenuNode) int {
	max := 0
	var walk func([]MenuNode, int)
	walk = func(ns []MenuNode, depth int) {
		if depth > max {
			max = depth
		}
		for _, n := range ns {
			if len(n.Children) > 0 {
				walk(n.Children, depth+1)
			}
		}
	}
	walk(nodes, 1)
	return max
}
