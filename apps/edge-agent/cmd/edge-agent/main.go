package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob/s3"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edgeonline"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue"
	queueredis "github.com/mr9esx/comfyui_tgbot/internal/platform/queue/redis"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/actuator"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		slog.Error("edge-agent failed", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	instID := envOr("INSTANCE_ID", "local")
	redisAddr := envOr("REDIS_ADDR", "127.0.0.1:6379")
	comfyMock := envBool("COMFY_MOCK", true)
	comfyURL := envOr("COMFYUI_BASE_URL", "http://127.0.0.1:8188")

	rdb := goredis.NewClient(&goredis.Options{Addr: redisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		return err
	}
	defer func() { _ = rdb.Close() }()

	bus, err := queueredis.New(queueredis.Options{
		Client:        rdb,
		ConsumerGroup: envOr("QUEUE_GROUP", "edge"),
		ConsumerName:  envOr("QUEUE_CONSUMER", "edge-"+instID),
	})
	if err != nil {
		return err
	}
	defer func() { _ = bus.Close() }()

	blobStore, err := s3.New(s3.Options{
		Endpoint:        envOr("S3_ENDPOINT", ""),
		Region:          envOr("S3_REGION", "us-east-1"),
		Bucket:          envOr("S3_BUCKET", "pixoma"),
		AccessKeyID:     envOr("S3_ACCESS_KEY", ""),
		SecretAccessKey: envOr("S3_SECRET_KEY", ""),
		UsePathStyle:    envBool("S3_PATH_STYLE", true),
	})
	if err != nil {
		return err
	}

	var comfy comfyui.Client
	if comfyMock {
		comfy = &comfyui.Mock{}
	} else {
		comfy, err = comfyui.NewClient(comfyui.Options{Mock: false, BaseURL: comfyURL})
		if err != nil {
			return err
		}
	}

	worker := &actuator.Worker{
		InstanceID: sharedkernel.InstanceID(instID),
		Comfy:      comfy,
		Blob:       blobStore,
		Status:     bus,
	}

	topic := envOr("DISPATCH_TOPIC", sharedkernel.TopicDispatch(sharedkernel.InstanceID(instID)))
	if err := bus.Subscribe(ctx, topic, func(ctx context.Context, msg queue.Message) error {
		var cmd sharedkernel.DispatchCommand
		if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
			return err
		}
		return worker.HandleDispatch(ctx, cmd)
	}); err != nil {
		return err
	}

	go heartbeat(ctx, rdb, instID)
	slog.Info("edge-agent running", "instance_id", instID, "topic", topic, "mock", comfyMock)
	<-ctx.Done()
	return nil
}

func heartbeat(ctx context.Context, rdb *goredis.Client, instanceID string) {
	key := edgeonline.Key(sharedkernel.InstanceID(instanceID))
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	_ = rdb.Set(ctx, key, "1", 30*time.Second).Err()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_ = rdb.Set(ctx, key, "1", 30*time.Second).Err()
		}
	}
}

func envOr(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func envBool(k string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}
