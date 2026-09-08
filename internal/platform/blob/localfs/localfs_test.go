package localfs_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/platform/blob/localfs"
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

func TestNewAndPutUsePrivatePermissions(t *testing.T) {
	root := filepath.Join(t.TempDir(), "blob")
	store, err := localfs.New(root)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("root mode = %o, want 0700", info.Mode().Perm())
	}

	_, err = store.Put(context.Background(), "secret.txt", bytes.NewReader([]byte("x")), blob.PutOptions{})
	if err != nil {
		t.Fatal(err)
	}
	info, err = os.Stat(filepath.Join(root, "secret.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("file mode = %o, want 0600", info.Mode().Perm())
	}
}

func TestCheck_Writable(t *testing.T) {
	store, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Check(context.Background()); err != nil {
		t.Fatalf("check: %v", err)
	}
}
