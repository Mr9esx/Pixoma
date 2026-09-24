package einoagent

import (
	"context"
	"fmt"

	einoskill "github.com/cloudwego/eino/adk/middlewares/skill"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

// runSkillBackend exposes only the selected, enabled Skills for this run.
// The Eino middleware owns discovery and lazy loading; this adapter only
// translates Pixoma's persisted Skill records into its Backend contract.
type runSkillBackend struct {
	skills []domain.Skill
}

func (b runSkillBackend) List(context.Context) ([]einoskill.FrontMatter, error) {
	frontMatters := make([]einoskill.FrontMatter, 0, len(b.skills))
	for _, selected := range b.skills {
		if !selected.Enabled {
			continue
		}
		frontMatters = append(frontMatters, einoskill.FrontMatter{
			Name: selected.ID, Description: selected.Name + "：" + selected.Description,
		})
	}
	return frontMatters, nil
}

func (b runSkillBackend) Get(_ context.Context, name string) (einoskill.Skill, error) {
	for _, selected := range b.skills {
		if selected.Enabled && selected.ID == name {
			return einoskill.Skill{
				FrontMatter: einoskill.FrontMatter{Name: selected.ID, Description: selected.Name + "：" + selected.Description},
				Content:     selected.Prompt,
			}, nil
		}
	}
	return einoskill.Skill{}, fmt.Errorf("studio: selected Skill %q is unavailable", name)
}
