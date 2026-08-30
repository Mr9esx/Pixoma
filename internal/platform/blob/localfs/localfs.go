package localfs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type Store struct {
	root string
}

func New(root string) (*Store, error) {
	if root == "" {
		return nil, errors.New("blob/localfs: empty root")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("blob/localfs: mkdir root: %w", err)
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return &Store{root: abs}, nil
}

// Check verifies the root directory exists and is writable with a probe file.
func (s *Store) Check(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return fmt.Errorf("blob/localfs: check mkdir: %w", err)
	}
	probe := filepath.Join(s.root, ".pixoma-probe")
	if err := os.WriteFile(probe, []byte("ok"), 0o600); err != nil {
		return fmt.Errorf("blob/localfs: check write: %w", err)
	}
	_ = os.Remove(probe)
	return nil
}

func (s *Store) Put(ctx context.Context, key string, r io.Reader, opts blob.PutOptions) (sharedkernel.BlobRef, error) {
	if err := ctx.Err(); err != nil {
		return sharedkernel.BlobRef{}, err
	}
	rel, err := cleanKey(key)
	if err != nil {
		return sharedkernel.BlobRef{}, err
	}
	full := filepath.Join(s.root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
		return sharedkernel.BlobRef{}, err
	}
	f, err := os.OpenFile(full, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return sharedkernel.BlobRef{}, err
	}
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return sharedkernel.BlobRef{}, err
	}
	defer f.Close()
	n, err := io.Copy(f, r)
	if err != nil {
		return sharedkernel.BlobRef{}, err
	}
	return sharedkernel.BlobRef{Key: rel, MIME: opts.MIME, Size: n}, nil
}

func (s *Store) Get(ctx context.Context, ref sharedkernel.BlobRef) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rel, err := cleanKey(ref.Key)
	if err != nil {
		return nil, err
	}
	full := filepath.Join(s.root, filepath.FromSlash(rel))
	return os.Open(full)
}

func cleanKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", errors.New("blob/localfs: empty key")
	}
	if filepath.IsAbs(key) {
		return "", errors.New("blob/localfs: absolute key not allowed")
	}
	clean := filepath.ToSlash(filepath.Clean(key))
	if clean == "." || strings.HasPrefix(clean, "../") || clean == ".." {
		return "", errors.New("blob/localfs: invalid key")
	}
	return clean, nil
}

var _ blob.Store = (*Store)(nil)
