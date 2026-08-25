package s3_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	blobs3 "github.com/mr9esx/comfyui_tgbot/internal/platform/blob/s3"
)

func TestPutGetRoundTrip(t *testing.T) {
	backend := s3mem.New()
	faker := gofakes3.New(backend)
	srv := httptest.NewServer(faker.Server())
	t.Cleanup(srv.Close)

	const bucket = "pixoma"
	if err := backend.CreateBucket(bucket); err != nil {
		t.Fatal(err)
	}

	store, err := blobs3.New(blobs3.Options{
		Endpoint:        srv.URL,
		Region:          "us-east-1",
		Bucket:          bucket,
		AccessKeyID:     "AKIA_TEST",
		SecretAccessKey: "testsecret",
		UsePathStyle:    true,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	payload := []byte("hello-s3-blob")
	ref, err := store.Put(ctx, "jobs/t1/job.json", bytes.NewReader(payload), blob.PutOptions{MIME: "application/json"})
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if ref.Key != "jobs/t1/job.json" {
		t.Fatalf("key=%q", ref.Key)
	}
	if ref.Bucket != bucket {
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
		t.Fatalf("got %q", got)
	}
}

func TestCheckAndEnsureBucket(t *testing.T) {
	backend := s3mem.New()
	faker := gofakes3.New(backend)
	srv := httptest.NewServer(faker.Server())
	t.Cleanup(srv.Close)

	store, err := blobs3.New(blobs3.Options{
		Endpoint:        srv.URL,
		Region:          "us-east-1",
		Bucket:          "pixoma-missing",
		AccessKeyID:     "AKIA_TEST",
		SecretAccessKey: "testsecret",
		UsePathStyle:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := store.Check(ctx); !errors.Is(err, blob.ErrBucketNotFound) {
		t.Fatalf("want ErrBucketNotFound, got %v", err)
	}
	if err := store.EnsureBucket(ctx); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if err := store.Check(ctx); err != nil {
		t.Fatalf("check after ensure: %v", err)
	}
}

func TestNew_RequiresBucket(t *testing.T) {
	_, err := blobs3.New(blobs3.Options{Endpoint: "http://127.0.0.1:9000"})
	if err == nil {
		t.Fatal("expected error")
	}
}
