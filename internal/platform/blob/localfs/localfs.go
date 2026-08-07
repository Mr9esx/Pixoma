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
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("blob/localfs: mkdir root: %w", err)
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return &Store{root: abs}, nil
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
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return sharedkernel.BlobRef{}, err
	}
	f, err := os.Create(full)
	if err != nil {
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
