//go:build live_tos

package tos_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	blobtos "github.com/mr9esx/comfyui_tgbot/internal/platform/blob/tos"
)

func requireTOSEnv(t *testing.T) blobtos.Options {
	t.Helper()
	opts := blobtos.Options{
		Endpoint:        os.Getenv("TOS_ENDPOINT"),
		Region:          os.Getenv("TOS_REGION"),
		Bucket:          os.Getenv("TOS_BUCKET"),
		AccessKeyID:     os.Getenv("TOS_ACCESS_KEY"),
		SecretAccessKey: os.Getenv("TOS_SECRET_KEY"),
	}
	var missing []string
	if opts.Endpoint == "" {
		missing = append(missing, "TOS_ENDPOINT")
	}
	if opts.Region == "" {
		missing = append(missing, "TOS_REGION")
	}
	if opts.Bucket == "" {
		missing = append(missing, "TOS_BUCKET")
	}
	if opts.AccessKeyID == "" {
		missing = append(missing, "TOS_ACCESS_KEY")
	}
	if opts.SecretAccessKey == "" {
		missing = append(missing, "TOS_SECRET_KEY")
	}
	if len(missing) > 0 {
		t.Fatalf("real TOS gate requires env %v; load with: set -a; source .env.tos.local; set +a", missing)
	}
	return opts
}

func TestRealTOS_PutGetRoundTrip(t *testing.T) {
	opts := requireTOSEnv(t)
	store, err := blobtos.New(opts)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	key := fmt.Sprintf("jobs/tos-gate/%d/ping.txt", time.Now().UnixNano())
	payload := []byte("tos-live-roundtrip")

	ref, err := store.Put(ctx, key, bytes.NewReader(payload), blob.PutOptions{MIME: "text/plain"})
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if ref.Key != key {
		t.Fatalf("key=%q", ref.Key)
	}
	if ref.Bucket != opts.Bucket {
		t.Fatalf("bucket=%q", ref.Bucket)
	}

	rc, err := store.Get(ctx, ref)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer rc.Close()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("got %q want %q", got, payload)
	}
}
