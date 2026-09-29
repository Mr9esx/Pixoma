package localfs_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/platform/blob/localfs"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
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

func TestDeleteRemovesBlob(t *testing.T) {
	store, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	ref, err := store.Put(ctx, "assets/example.txt", bytes.NewBufferString("content"), blob.PutOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, ref); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.Get(ctx, ref); !os.IsNotExist(err) {
		t.Fatalf("get after delete: %v", err)
	}
	if err := store.Delete(ctx, ref); err != nil {
		t.Fatalf("delete missing object: %v", err)
	}
}

func TestDeleteRejectsInvalidKeyAndCanceledContext(t *testing.T) {
	store, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"", "/abs", "../outside"} {
		if err := store.Delete(context.Background(), sharedkernel.BlobRef{Key: key}); err == nil {
			t.Fatalf("expected invalid key error for %q", key)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := store.Delete(ctx, sharedkernel.BlobRef{Key: "valid"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled delete: %v", err)
	}
}

func TestDeleteCannotFollowSymlinkOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	store, err := localfs.New(root)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(outside, "keep.txt")
	if err := os.WriteFile(path, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "outside")); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(context.Background(), sharedkernel.BlobRef{Key: "outside/keep.txt"}); err == nil {
		t.Fatal("delete followed symlink outside root")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("outside file was removed: %v", err)
	}
}
