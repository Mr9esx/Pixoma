package tos_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	blobtos "github.com/Mr9esx/Pixoma/internal/platform/blob/tos"
)

func TestNew_RequiresBucket(t *testing.T) {
	_, err := blobtos.New(blobtos.Options{
		Endpoint: "https://tos-cn-beijing.volces.com",
		Region:   "cn-beijing",
	})
	if err == nil {
		t.Fatal("expected error for empty bucket")
	}
	if !strings.Contains(err.Error(), "bucket") {
		t.Fatalf("got %v", err)
	}
}

func TestPut_RejectsInvalidKeys(t *testing.T) {
	store, err := blobtos.New(blobtos.Options{
		Endpoint:        "https://tos-cn-beijing.volces.com",
		Region:          "cn-beijing",
		Bucket:          "pixoma-test",
		AccessKeyID:     "unused-in-key-check",
		SecretAccessKey: "unused-in-key-check",
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, key := range []string{"", "/abs", "../x", ".."} {
		_, err := store.Put(ctx, key, bytes.NewReader([]byte("x")), blob.PutOptions{})
		if err == nil {
			t.Fatalf("expected error for key %q", key)
		}
	}
}
