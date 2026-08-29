package migrate_test

import (
	"testing"

	"github.com/mr9esx/comfyui_tgbot/scripts/migrate-menu-action/migrate"
)

func TestMigrateAction(t *testing.T) {
	cases := []struct {
		name    string
		in      map[string]any
		want    map[string]any
		changed bool
	}{
		{
			name:    "workflow_ids 多元素取首",
			in:      map[string]any{"type": "open_workflow", "workflow_ids": []any{"10", "20"}},
			want:    map[string]any{"type": "open_workflow", "workflow_id": "10"},
			changed: true,
		},
		{
			name:    "workflow_ids 单元素取首",
			in:      map[string]any{"type": "open_workflow", "workflow_ids": []any{"10"}},
			want:    map[string]any{"type": "open_workflow", "workflow_id": "10"},
			changed: true,
		},
		{
			name:    "mode list 字段删除",
			in:      map[string]any{"type": "open_workflow", "workflow_id": "10", "mode": "list"},
			want:    map[string]any{"type": "open_workflow", "workflow_id": "10"},
			changed: true,
		},
		{
			name:    "mode direct 字段删除",
			in:      map[string]any{"type": "open_workflow", "workflow_id": "10", "mode": "direct"},
			want:    map[string]any{"type": "open_workflow", "workflow_id": "10"},
			changed: true,
		},
		{
			name:    "direct_id 改 workflow_id",
			in:      map[string]any{"type": "open_workflow", "direct_id": "42", "mode": "direct"},
			want:    map[string]any{"type": "open_workflow", "workflow_id": "42"},
			changed: true,
		},
		{
			name:    "无变化透传",
			in:      map[string]any{"type": "send_text", "text": "hi"},
			want:    map[string]any{"type": "send_text", "text": "hi"},
			changed: false,
		},
		{
			name:    "open_card 不变",
			in:      map[string]any{"type": "open_card", "card_id": "c1"},
			want:    map[string]any{"type": "open_card", "card_id": "c1"},
			changed: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, changed, err := migrate.Action(c.in)
			if err != nil {
				t.Fatalf("err=%v", err)
			}
			if changed != c.changed {
				t.Errorf("changed: got %v want %v", changed, c.changed)
			}
			if !mapsEqual(got, c.want) {
				t.Errorf("got=%v want=%v", got, c.want)
			}
		})
	}
}

func mapsEqual(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || !valueEqual(v, bv) {
			return false
		}
	}
	return true
}

func valueEqual(a, b any) bool {
	as, ok := a.(string)
	if ok {
		return as == b
	}
	as2, ok2 := a.([]any)
	if ok2 {
		bs, ok3 := b.([]any)
		if !ok3 || len(as2) != len(bs) {
			return false
		}
		for i := range as2 {
			if as2[i] != bs[i] {
				return false
			}
		}
		return true
	}
	return a == b
}
