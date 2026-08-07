package localfs_test

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob/localfs"
)

func TestPutGetRoundTrip(t *testing.T) {
	root := t.TempDir()
	store, err := localfs.New(root)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	ctx := context.Background()
	payload := []byte("hello-blob")
	ref, err := store.Put(ctx, "inputs/task1/prompt.txt", bytes.NewReader(payload), blob.PutOptions{MIME: "text/plain"})
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if ref.Key != "inputs/task1/prompt.txt" {
		t.Fatalf("key=%q", ref.Key)
	}
	if ref.Size != int64(len(payload)) {
		t.Fatalf("size=%d", ref.Size)
	}
	if filepath.IsAbs(ref.Key) {
		t.Fatal("key must stay relative")
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
