package demo

import (
	"context"
	"encoding/json"
	"testing"
)

func TestOpenDatabaseIsMemoryAndSeedsFullDemo(t *testing.T) {
	ctx := context.Background()
	gdb, cleanup, err := OpenDatabase()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cleanup() }()

	var databaseList []struct {
		Seq  int
		Name string
		File string
	}
	if err := gdb.Raw("PRAGMA database_list").Scan(&databaseList).Error; err != nil {
		t.Fatal(err)
	}
	for _, database := range databaseList {
		if database.Name != "main" || database.File != "" {
			t.Fatalf("unexpected attached database: %+v", database)
		}
	}

	if err := Seed(ctx, gdb); err != nil {
		t.Fatal(err)
	}

	var edgeCount, caseCount, channelCount, topicCount, taskCount, sessionCount, userCount, menuCount, statsCount int64
	for _, check := range []struct {
		table string
		count *int64
	}{
		{"edges", &edgeCount},
		{"catalog_cases", &caseCount},
		{"channels", &channelCount},
		{"topics", &topicCount},
		{"tasks", &taskCount},
		{"sessions", &sessionCount},
		{"channel_users", &userCount},
		{"channel_main_menus", &menuCount},
		{"task_daily_stats", &statsCount},
	} {
		if err := gdb.Table(check.table).Count(check.count).Error; err != nil {
			t.Fatalf("count %s: %v", check.table, err)
		}
		if *check.count == 0 {
			t.Fatalf("expected seeded rows in %s", check.table)
		}
	}

	var task struct {
		ID     string
		Status string
		EdgeID string
		CaseID uint64
	}
	if err := gdb.Table("tasks").Select("id, status, edge_id, case_id").Take(&task).Error; err != nil {
		t.Fatal(err)
	}
	if task.ID == "" || task.Status == "" || task.EdgeID == "" || task.CaseID == 0 {
		t.Fatalf("task is not related to demo resources: %+v", task)
	}

	var caseRow struct {
		DocJSON string
	}
	if err := gdb.Table("catalog_cases").Select("doc_json").Take(&caseRow).Error; err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal([]byte(caseRow.DocJSON), &document); err != nil {
		t.Fatal(err)
	}
	if document["name"] != "产品图精修" {
		t.Fatalf("unexpected case document: %v", document["name"])
	}
}
