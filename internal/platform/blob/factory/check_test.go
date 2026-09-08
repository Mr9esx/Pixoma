package factory_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"

	"github.com/Mr9esx/Pixoma/internal/platform/blob/factory"
	"github.com/Mr9esx/Pixoma/internal/platform/botconfig"
)

func TestCheck_LocalFS(t *testing.T) {
	if err := factory.Check(context.Background(), factory.CheckOptions{
		Driver:    botconfig.BlobDriverLocalFS,
		LocalRoot: t.TempDir(),
	}); err != nil {
		t.Fatalf("check localfs: %v", err)
	}
}

func TestCheck_UnknownDriver(t *testing.T) {
	err := factory.Check(context.Background(), factory.CheckOptions{Driver: "oss"})
	if err == nil || !strings.Contains(err.Error(), "unknown driver") {
		t.Fatalf("want unknown driver error, got %v", err)
	}
}

func TestEnsureBucket_S3(t *testing.T) {
	backend := s3mem.New()
	faker := gofakes3.New(backend)
	srv := httptest.NewServer(faker.Server())
	t.Cleanup(srv.Close)

	opts := factory.CheckOptions{
		Driver:    botconfig.BlobDriverS3,
		Endpoint:  srv.URL,
		Region:    "us-east-1",
		Bucket:    "factory-new-bucket",
		AccessKey: "AKIA_TEST",
		SecretKey: "testsecret",
	}
	ctx := context.Background()
	if err := factory.EnsureBucket(ctx, opts); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if err := factory.Check(ctx, opts); err != nil {
		t.Fatalf("check after ensure: %v", err)
	}
}

func TestCheck_SharedFS(t *testing.T) {
	if err := factory.Check(context.Background(), factory.CheckOptions{
		Driver:    botconfig.BlobDriverSharedFS,
		LocalRoot: t.TempDir(),
	}); err != nil {
		t.Fatalf("sharedfs check: %v", err)
	}
}
