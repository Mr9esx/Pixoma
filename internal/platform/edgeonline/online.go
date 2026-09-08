package edgeonline

import (
	"context"

	goredis "github.com/redis/go-redis/v9"

	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

// Key is the Redis heartbeat key written by edge-agent.
func Key(edgeID sharedkernel.EdgeID) string {
	return "edge:online:" + string(edgeID)
}

// Checker returns an Online filter that treats a live TTL key as presence.
func Checker(rdb goredis.Cmdable) func(context.Context, sharedkernel.EdgeID) bool {
	return func(ctx context.Context, id sharedkernel.EdgeID) bool {
		if rdb == nil || id == "" {
			return false
		}
		n, err := rdb.Exists(ctx, Key(id)).Result()
		return err == nil && n > 0
	}
}
