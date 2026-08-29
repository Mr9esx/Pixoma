// Package migrate 提供菜单 Action 字段迁移的纯函数。
//
// 历史 Action schema（v1）：
//   - workflow_ids: []string  （列表）
//   - mode: "list" | "direct"
//   - direct_id: string
//
// 新 Action schema（v2）：
//   - workflow_id: string  （单数）
//   - mode 字段删除
//   - direct_id 改名为 workflow_id
//
// Action(in) 返回 (out, changed, err)：
//   - out: 迁移后的字段 map（in-place 修改并返回，调用方可决定是否持久化）
//   - changed: 是否真的发生了字段变更
//   - err: 解析失败时返回
package migrate

// Action 迁移 Action 字段 map。
// 规则：
//  1. workflow_ids 非空 → 取 [0] 写 workflow_id，删除 workflow_ids
//  2. mode 字段删除（无论 list/direct）
//  3. direct_id 改名为 workflow_id（若已有 workflow_id 则不覆盖）
//  4. 优先级：workflow_ids[0] > direct_id > 已有 workflow_id
func Action(in map[string]any) (map[string]any, bool, error) {
	if in == nil {
		return in, false, nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	changed := false

	// 1. workflow_ids → workflow_id
	if wids, ok := out["workflow_ids"]; ok {
		switch v := wids.(type) {
		case []any:
			if len(v) > 0 {
				if first, ok := v[0].(string); ok && first != "" {
					if _, exists := out["workflow_id"]; !exists {
						out["workflow_id"] = first
						changed = true
					}
				}
			}
		case []string:
			if len(v) > 0 && v[0] != "" {
				if _, exists := out["workflow_id"]; !exists {
					out["workflow_id"] = v[0]
					changed = true
				}
			}
		}
		delete(out, "workflow_ids")
		changed = true
	}

	// 2. mode 删除
	if _, ok := out["mode"]; ok {
		delete(out, "mode")
		changed = true
	}

	// 3. direct_id → workflow_id
	if did, ok := out["direct_id"]; ok {
		if s, ok := did.(string); ok && s != "" {
			if _, exists := out["workflow_id"]; !exists {
				out["workflow_id"] = s
				changed = true
			}
		}
		delete(out, "direct_id")
		changed = true
	}

	return out, changed, nil
}
