package studio_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/httpapi/apitest"
	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
)

func TestSkillVersionHistoryAPI(t *testing.T) {
	router := skillImportRouter(t)
	created := request(t, router, http.MethodPost, "/skills", map[string]any{
		"name": "storyboard", "description": "Make storyboards", "prompt": "Original prompt", "enabled": true,
	}, "account-a")
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d: %s", created.Code, created.Body.String())
	}
	var initial studioapp.SkillView
	if err := json.Unmarshal(apitest.DataBytes(created), &initial); err != nil {
		t.Fatal(err)
	}
	if initial.Version != "1.0.0" {
		t.Fatalf("initial version = %q", initial.Version)
	}

	saved := request(t, router, http.MethodPatch, "/skills/"+initial.ID, map[string]any{
		"name": "storyboard", "description": "Make storyboards", "prompt": "Revised prompt",
		"version": "1.0.1", "updated_at": initial.UpdatedAt,
	}, "account-a")
	if saved.Code != http.StatusOK {
		t.Fatalf("save status = %d: %s", saved.Code, saved.Body.String())
	}
	var current studioapp.SkillView
	if err := json.Unmarshal(apitest.DataBytes(saved), &current); err != nil {
		t.Fatal(err)
	}
	if current.Version != "1.0.1" || current.Prompt != "Revised prompt" {
		t.Fatalf("saved Skill = %#v", current)
	}
	disabled := request(t, router, http.MethodPatch, "/skills/"+initial.ID+"/enabled", map[string]any{"enabled": false}, "account-a")
	if disabled.Code != http.StatusOK {
		t.Fatalf("disable status = %d: %s", disabled.Code, disabled.Body.String())
	}

	history := request(t, router, http.MethodGet, "/skills/"+initial.ID+"/versions", nil, "account-a")
	if history.Code != http.StatusOK {
		t.Fatalf("history status = %d: %s", history.Code, history.Body.String())
	}
	var versions []struct {
		Version   string    `json:"version"`
		CreatedAt time.Time `json:"created_at"`
	}
	if err := json.Unmarshal(apitest.DataBytes(history), &versions); err != nil {
		t.Fatal(err)
	}
	if len(versions) != 2 || versions[0].Version != "1.0.1" || versions[1].Version != "1.0.0" || versions[0].CreatedAt.IsZero() {
		t.Fatalf("versions = %#v", versions)
	}

	old := request(t, router, http.MethodGet, "/skills/"+initial.ID+"/versions/1.0.0", nil, "account-a")
	if old.Code != http.StatusOK {
		t.Fatalf("old version status = %d: %s", old.Code, old.Body.String())
	}
	var oldVersion struct {
		Version     string    `json:"version"`
		CreatedAt   time.Time `json:"created_at"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		Prompt      string    `json:"prompt"`
	}
	if err := json.Unmarshal(apitest.DataBytes(old), &oldVersion); err != nil {
		t.Fatal(err)
	}
	if oldVersion.Version != "1.0.0" || oldVersion.Name != "storyboard" || oldVersion.Description != "Make storyboards" || oldVersion.Prompt != "Original prompt" || oldVersion.CreatedAt.IsZero() {
		t.Fatalf("old version = %#v", oldVersion)
	}

	foreign := request(t, router, http.MethodGet, "/skills/"+initial.ID+"/versions/1.0.0", nil, "account-b")
	if foreign.Code != http.StatusNotFound {
		t.Fatalf("foreign status = %d: %s", foreign.Code, foreign.Body.String())
	}
	missing := request(t, router, http.MethodGet, "/skills/"+initial.ID+"/versions/9.9.9", nil, "account-a")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d: %s", missing.Code, missing.Body.String())
	}
}

func TestSkillVersionSaveRejectsInvalidAndStaleVersions(t *testing.T) {
	router := skillImportRouter(t)
	created := request(t, router, http.MethodPost, "/skills", map[string]any{
		"name": "storyboard", "description": "Make storyboards", "prompt": "Original prompt", "enabled": true,
	}, "account-a")
	var skill studioapp.SkillView
	if err := json.Unmarshal(apitest.DataBytes(created), &skill); err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"", "01.0.1", "1.0", "1.0.1-beta", "1.0.0"} {
		response := request(t, router, http.MethodPatch, "/skills/"+skill.ID, map[string]any{
			"name": skill.Name, "description": skill.Description, "prompt": "Changed",
			"version": version, "updated_at": skill.UpdatedAt,
		}, "account-a")
		want := http.StatusBadRequest
		if version == "1.0.0" {
			want = http.StatusConflict
		}
		if response.Code != want {
			t.Fatalf("version %q status = %d, want %d: %s", version, response.Code, want, response.Body.String())
		}
	}

	saved := request(t, router, http.MethodPatch, "/skills/"+skill.ID, map[string]any{
		"name": skill.Name, "description": skill.Description, "prompt": "Changed",
		"version": "2.0.0", "updated_at": skill.UpdatedAt,
	}, "account-a")
	if saved.Code != http.StatusOK {
		t.Fatalf("save status = %d: %s", saved.Code, saved.Body.String())
	}
	stale := request(t, router, http.MethodPatch, "/skills/"+skill.ID, map[string]any{
		"name": skill.Name, "description": skill.Description, "prompt": "Stale",
		"version": "2.0.1", "updated_at": skill.UpdatedAt,
	}, "account-a")
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale status = %d: %s", stale.Code, stale.Body.String())
	}
	history := request(t, router, http.MethodGet, "/skills/"+skill.ID+"/versions", nil, "account-a")
	var versions []struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(apitest.DataBytes(history), &versions); err != nil {
		t.Fatal(err)
	}
	if len(versions) != 2 || versions[0].Version != "2.0.0" {
		t.Fatalf("versions = %#v", versions)
	}
}
