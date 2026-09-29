//go:build live_s3

package s3_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	blobs3 "github.com/Mr9esx/Pixoma/internal/platform/blob/s3"
)

func TestRealS3_Delete(t *testing.T) {
	for _, name := range []string{"S3_ENDPOINT", "S3_REGION", "S3_BUCKET", "S3_ACCESS_KEY", "S3_SECRET_KEY"} {
		if os.Getenv(name) == "" {
			t.Fatalf("real S3 test requires %s", name)
		}
	}
	store, err := blobs3.New(blobs3.Options{
		Endpoint:        os.Getenv("S3_ENDPOINT"),
		Region:          os.Getenv("S3_REGION"),
		Bucket:          os.Getenv("S3_BUCKET"),
		AccessKeyID:     os.Getenv("S3_ACCESS_KEY"),
		SecretAccessKey: os.Getenv("S3_SECRET_KEY"),
		UsePathStyle:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	key := fmt.Sprintf("jobs/s3-gate/%d/ping.txt", time.Now().UnixNano())
	ref, err := store.Put(ctx, key, bytes.NewBufferString("s3-delete"), blob.PutOptions{MIME: "text/plain"})
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := store.Delete(ctx, ref); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.Get(ctx, ref); err == nil {
		t.Fatal("get after delete succeeded")
	}
	if err := store.Delete(ctx, ref); err != nil {
		t.Fatalf("delete missing object: %v", err)
	}
}
