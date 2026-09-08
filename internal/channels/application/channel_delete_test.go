package application_test

import (
	"context"
	"testing"
	"time"

	channelapp "github.com/Mr9esx/Pixoma/internal/channels/application"
	channelpersist "github.com/Mr9esx/Pixoma/internal/channels/infrastructure/persistence"
	mcdomain "github.com/Mr9esx/Pixoma/internal/menus/domain"
	mencardpersist "github.com/Mr9esx/Pixoma/internal/menus/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/platform/db"
	convdomain "github.com/Mr9esx/Pixoma/internal/sessions/domain"
	sesspersist "github.com/Mr9esx/Pixoma/internal/sessions/infrastructure/persistence"
)

func TestDeleteWithCleanup(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:ch_delete_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb,
		&channelpersist.ChannelRow{},
		&sesspersist.SessionRow{},
		&mencardpersist.MainMenuRow{},
		&mencardpersist.CardRow{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Now().UTC()

	chRepo := channelpersist.NewGormRepository(gdb)
	if err := gdb.Create(&channelpersist.ChannelRow{
		ID:                   "tg",
		Platform:             "telegram",
		Name:                 "主",
		CredentialCiphertext: "enc",
		Enabled:              true,
		CreatedAt:            now,
		UpdatedAt:            now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	sessRepo := sesspersist.NewSessionRepository(gdb)
	if err := sessRepo.Save(ctx, convdomain.NewCollecting("s1", "tg:9", 1, []string{"a"}, now)); err != nil {
		t.Fatal(err)
	}
	submitted := convdomain.NewCollecting("s2", "tg:8", 1, []string{"a"}, now)
	submitted.Status = convdomain.StatusSubmitted
	if err := sessRepo.Save(ctx, submitted); err != nil {
		t.Fatal(err)
	}
	menuRepo := mencardpersist.NewGormCardRepository(gdb)
	if err := menuRepo.PutTree(ctx, "tg", mcdomain.MenuTree{
		Columns: 2,
		Items:   []mcdomain.TreeButton{{ID: "a", Label: "A", Action: mcdomain.TreeAction{Type: "list_tasks"}}},
	}); err != nil {
		t.Fatal(err)
	}

	cleanup := channelapp.DeleteWithCleanup(gdb)
	chats, err := cleanup(ctx, "tg")
	if err != nil {
		t.Fatal(err)
	}
	if len(chats) != 1 || chats[0] != "tg:9" {
		t.Fatalf("chats=%+v", chats)
	}
	if _, err := chRepo.Get(ctx, "tg"); err == nil {
		t.Fatal("channel must be deleted")
	}
	var rows []sesspersist.SessionRow
	if err := gdb.Where("channel_id = ?", "tg").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("sessions=%+v", rows)
	}
	byID := map[string]string{}
	for _, row := range rows {
		byID[row.ID] = row.Status
	}
	if byID["s1"] != string(convdomain.StatusExited) {
		t.Fatalf("s1 status=%s want exited", byID["s1"])
	}
	if byID["s2"] != string(convdomain.StatusSubmitted) {
		t.Fatalf("s2 status=%s want submitted (untouched)", byID["s2"])
	}
	if _, err := menuRepo.GetTree(ctx, "tg"); err == nil {
		t.Fatal("menu must be deleted")
	}
}
