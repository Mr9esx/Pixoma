package persistence_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

func TestProjectMembershipAndSessionPages(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	project := &domain.Project{ID: "project-1", AccountID: "account-a", Name: "产品图", CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	for index, id := range []string{"one", "two", "three"} {
		session, err := domain.NewSession(id, "account-a", now.Add(time.Duration(index)*time.Minute))
		if err != nil {
			t.Fatal(err)
		}
		if err := repo.CreateSession(ctx, session); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.MoveSessionToProject(ctx, "account-a", "one", project.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.MoveSessionToProject(ctx, "account-a", "two", project.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.MoveSessionToProject(ctx, "account-b", "three", project.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-account move = %v", err)
	}
	recent := ""
	page, err := repo.ListSessions(ctx, "account-a", domain.SessionListQuery{ProjectID: &recent, Limit: 1})
	if err != nil || len(page) != 1 || page[0].ID != "three" {
		t.Fatalf("recent page = %v, %v", page, err)
	}
	page, err = repo.ListSessions(ctx, "account-a", domain.SessionListQuery{ProjectID: &project.ID, Limit: 1, Offset: 1})
	if err != nil || len(page) != 1 || page[0].ID != "one" {
		t.Fatalf("project page = %v, %v", page, err)
	}
	if err := repo.DeleteProject(ctx, "account-a", project.ID); err != nil {
		t.Fatal(err)
	}
	page, err = repo.ListSessions(ctx, "account-a", domain.SessionListQuery{ProjectID: &recent, Limit: 30})
	if err != nil || len(page) != 3 {
		t.Fatalf("recent after delete = %v, %v", page, err)
	}
}

func TestRecentAndProjectSessionPagination(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	project, err := domain.NewProject("project-1", "account-a", "长篇", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	for index := range 38 {
		session, err := domain.NewSession(fmt.Sprintf("session-%02d", index), "account-a", now.Add(time.Duration(index)*time.Minute))
		if err != nil {
			t.Fatal(err)
		}
		if err := repo.CreateSession(ctx, session); err != nil {
			t.Fatal(err)
		}
		if index < 7 {
			if err := repo.MoveSessionToProject(ctx, "account-a", session.ID, project.ID); err != nil {
				t.Fatal(err)
			}
		}
	}
	recent := ""
	first, err := repo.ListSessions(ctx, "account-a", domain.SessionListQuery{ProjectID: &recent, Limit: 30})
	if err != nil || len(first) != 30 {
		t.Fatalf("recent first page = %d, %v", len(first), err)
	}
	second, err := repo.ListSessions(ctx, "account-a", domain.SessionListQuery{ProjectID: &recent, Limit: 30, Offset: 30})
	if err != nil || len(second) != 1 || second[0].ID != "session-07" {
		t.Fatalf("recent second page = %v, %v", second, err)
	}
	projectFirst, err := repo.ListSessions(ctx, "account-a", domain.SessionListQuery{ProjectID: &project.ID, Limit: 5})
	if err != nil || len(projectFirst) != 5 {
		t.Fatalf("project first page = %d, %v", len(projectFirst), err)
	}
	projectSecond, err := repo.ListSessions(ctx, "account-a", domain.SessionListQuery{ProjectID: &project.ID, Limit: 5, Offset: 5})
	if err != nil || len(projectSecond) != 2 || projectSecond[0].ID != "session-01" {
		t.Fatalf("project second page = %v, %v", projectSecond, err)
	}
}
