package persistence_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"gorm.io/gorm"

	"github.com/mr9esx/comfyui_tgbot/internal/identity/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/identity/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	name := "identity_user_" + strings.ReplaceAll(t.Name(), "/", "_")
	gdb, err := db.Open(db.Options{DSN: "file:" + name + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &persistence.UserRow{}, &persistence.UserExternalIdentityRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return gdb
}

func TestUpsertByChannelExternal_IdempotentAndRefresh(t *testing.T) {
	gdb := openTestDB(t)
	repo := persistence.NewUserRepository(gdb)
	ctx := context.Background()

	u1, err := repo.UpsertByChannelExternal(ctx, domain.UpsertFrom{
		ChannelID: "tg-default", ExternalUserID: "42",
		Username: "alice", FirstName: "A", LanguageCode: "zh-hans",
	})
	if err != nil {
		t.Fatal(err)
	}
	u2, err := repo.UpsertByChannelExternal(ctx, domain.UpsertFrom{
		ChannelID: "tg-default", ExternalUserID: "42",
		Username: "alice2", FirstName: "A2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if u1.ID != u2.ID {
		t.Fatalf("id changed: %s vs %s", u1.ID, u2.ID)
	}
	if u2.Username != "alice2" {
		t.Fatalf("username not refreshed: %q", u2.Username)
	}
	if u1.Access != domain.UserAccessDenied {
		t.Fatalf("default access=%q want denied", u1.Access)
	}
}

func TestUpsertByChannelExternal_UsesConfiguredDefaultAccess(t *testing.T) {
	gdb := openTestDB(t)
	repo := persistence.NewUserRepository(gdb)
	repo.SetDefaultAccess(func() domain.UserAccess {
		return domain.UserAccessAlwaysAllowed
	})
	ctx := context.Background()

	created, err := repo.UpsertByChannelExternal(ctx, domain.UpsertFrom{
		ChannelID: "tg-default", ExternalUserID: "77", Username: "newcomer",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Access != domain.UserAccessAlwaysAllowed {
		t.Fatalf("access=%q want always_allowed", created.Access)
	}

	repo.SetDefaultAccess(func() domain.UserAccess {
		return domain.UserAccessPaid
	})
	other, err := repo.UpsertByChannelExternal(ctx, domain.UpsertFrom{
		ChannelID: "tg-default", ExternalUserID: "78", Username: "paid-user",
	})
	if err != nil {
		t.Fatal(err)
	}
	if other.Access != domain.UserAccessPaid {
		t.Fatalf("access=%q want paid", other.Access)
	}

	refreshed, err := repo.UpsertByChannelExternal(ctx, domain.UpsertFrom{
		ChannelID: "tg-default", ExternalUserID: "77", Username: "newcomer",
	})
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.Access != domain.UserAccessAlwaysAllowed {
		t.Fatalf("upsert reset access=%q", refreshed.Access)
	}
}

func TestUserRepository_SetAccess(t *testing.T) {
	gdb := openTestDB(t)
	repo := persistence.NewUserRepository(gdb)
	ctx := context.Background()

	user, err := repo.UpsertByChannelExternal(ctx, domain.UpsertFrom{
		ChannelID: "tg-default", ExternalUserID: "5001", Username: "alice_access",
	})
	if err != nil {
		t.Fatal(err)
	}

	user, err = repo.SetAccess(ctx, user.ID, domain.UserAccessAlwaysAllowed)
	if err != nil {
		t.Fatal(err)
	}
	if user.Access != domain.UserAccessAlwaysAllowed {
		t.Fatalf("access=%q want always_allowed", user.Access)
	}

	refreshed, err := repo.UpsertByChannelExternal(ctx, domain.UpsertFrom{
		ChannelID: "tg-default", ExternalUserID: "5001", Username: "alice_access",
	})
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.Access != domain.UserAccessAlwaysAllowed {
		t.Fatalf("upsert reset access=%q", refreshed.Access)
	}

	if _, err := repo.SetAccess(ctx, "missing", domain.UserAccessPaid); err != domain.ErrNotFound {
		t.Fatalf("missing access error=%v want ErrNotFound", err)
	}
}

func TestUpsert_ConcurrentSameIdentityNoUniqueError(t *testing.T) {
	gdb, err := db.Open(db.Options{
		DSN: "file:identity_user_concurrent_" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_pragma=busy_timeout(5000)",
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &persistence.UserRow{}, &persistence.UserExternalIdentityRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := persistence.NewUserRepository(gdb)
	ctx := context.Background()

	const n = 8
	var wg sync.WaitGroup
	ids := make(chan string, n)
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			u, err := repo.UpsertByChannelExternal(ctx, domain.UpsertFrom{
				ChannelID: "tg-default", ExternalUserID: "42", Username: "alice",
			})
			if err != nil {
				errs <- err
				return
			}
			ids <- u.ID
		}()
	}
	wg.Wait()
	close(errs)
	close(ids)
	for err := range errs {
		t.Fatalf("concurrent upsert: %v", err)
	}
	first := ""
	for id := range ids {
		if first == "" {
			first = id
		}
		if id != first {
			t.Fatalf("concurrent upsert produced distinct users: %s vs %s", first, id)
		}
	}
}

func TestUpsert_DifferentChannelsIsolated(t *testing.T) {
	gdb := openTestDB(t)
	repo := persistence.NewUserRepository(gdb)
	ctx := context.Background()

	tg, err := repo.UpsertByChannelExternal(ctx, domain.UpsertFrom{
		ChannelID: "tg-default", ExternalUserID: "42", Username: "tg-alice",
	})
	if err != nil {
		t.Fatal(err)
	}
	other, err := repo.UpsertByChannelExternal(ctx, domain.UpsertFrom{
		ChannelID: "feishu-1", ExternalUserID: "oc_42", Username: "fs-alice",
	})
	if err != nil {
		t.Fatal(err)
	}
	if tg.ID == other.ID {
		t.Fatalf("channels must map to distinct identities: %s", tg.ID)
	}

	byChannel, err := repo.List(ctx, domain.ListQuery{ChannelID: ptr("tg-default")})
	if err != nil {
		t.Fatal(err)
	}
	if len(byChannel) != 1 || byChannel[0].ID != tg.ID {
		t.Fatalf("channel filter: want 1 tg, got %+v", byChannel)
	}
}

func TestUserListFilters(t *testing.T) {
	gdb := openTestDB(t)
	repo := persistence.NewUserRepository(gdb)
	ctx := context.Background()

	alice, err := repo.UpsertByChannelExternal(ctx, domain.UpsertFrom{
		ChannelID: "tg-default", ExternalUserID: "1001",
		Username: "alice_list", FirstName: "Alice", LastName: "One",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpsertByChannelExternal(ctx, domain.UpsertFrom{
		ChannelID: "tg-default", ExternalUserID: "1002",
		Username: "bob_list", FirstName: "Bob", LastName: "Two",
	}); err != nil {
		t.Fatal(err)
	}

	byQ, err := repo.List(ctx, domain.ListQuery{Q: "alice_list"})
	if err != nil {
		t.Fatal(err)
	}
	if len(byQ) != 1 || byQ[0].Username != "alice_list" {
		t.Fatalf("q username: want 1 alice_list, got %+v", byQ)
	}

	byExt, err := repo.List(ctx, domain.ListQuery{
		ChannelID: ptr("tg-default"), ExternalUserID: ptr("1001"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(byExt) != 1 || byExt[0].ID != alice.ID {
		t.Fatalf("external exact: want alice, got %+v", byExt)
	}

	limited, err := repo.List(ctx, domain.ListQuery{Limit: 1, Offset: 0})
	if err != nil {
		t.Fatal(err)
	}
	if len(limited) != 1 {
		t.Fatalf("limit 1: want len 1, got %d", len(limited))
	}
}

func ptr(s string) *string { return &s }
