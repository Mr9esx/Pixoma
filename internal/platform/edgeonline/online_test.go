package edgeonline_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/edgeonline"
)

func TestChecker_ReportsOnlineWhenKeyPresent(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	ctx := context.Background()
	check := edgeonline.Checker(rdb)
	if check(ctx, "local") {
		t.Fatal("expected offline before heartbeat")
	}
	if err := rdb.Set(ctx, edgeonline.Key("local"), "1", 30*time.Second).Err(); err != nil {
		t.Fatal(err)
	}
	if !check(ctx, "local") {
		t.Fatal("expected online after key set")
	}
	if check(ctx, "other") {
		t.Fatal("other instance must stay offline")
	}
}
