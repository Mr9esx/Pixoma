package tos

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	volctos "github.com/volcengine/ve-tos-golang-sdk/v2/tos"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// Options configures a Volcengine TOS blob store.
type Options struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
}

// Store implements blob.Store against Volcengine TOS.
type Store struct {
	client *volctos.ClientV2
	bucket string
}

// New builds a TOS Store. Endpoint, region, and bucket are required.
func New(opts Options) (*Store, error) {
	if strings.TrimSpace(opts.Bucket) == "" {
		return nil, errors.New("blob/tos: empty bucket")
	}
	endpoint := strings.TrimSpace(opts.Endpoint)
	if endpoint == "" {
		return nil, errors.New("blob/tos: empty endpoint")
	}
	region := strings.TrimSpace(opts.Region)
	if region == "" {
		return nil, errors.New("blob/tos: empty region")
	}
	client, err := volctos.NewClientV2(endpoint,
		volctos.WithRegion(region),
		volctos.WithCredentials(volctos.NewStaticCredentials(opts.AccessKeyID, opts.SecretAccessKey)),
	)
	if err != nil {
		return nil, fmt.Errorf("blob/tos: new client: %w", err)
	}
	return &Store{client: client, bucket: opts.Bucket}, nil
}

// Check verifies the bucket is reachable (HeadBucket).
func (s *Store) Check(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := s.client.HeadBucket(ctx, &volctos.HeadBucketInput{Bucket: s.bucket})
	if err != nil {
		var serr *volctos.TosServerError
		if errors.As(err, &serr) && serr.StatusCode == http.StatusNotFound {
			return fmt.Errorf("blob/tos: %w", blob.ErrBucketNotFound)
		}
		return fmt.Errorf("blob/tos: check: %w", err)
	}
	return nil
}

// EnsureBucket creates the bucket when missing, then re-checks reachability.
func (s *Store) EnsureBucket(ctx context.Context) error {
	if err := s.Check(ctx); err == nil {
		return nil
	} else if !errors.Is(err, blob.ErrBucketNotFound) {
		return err
	}
	if _, err := s.client.CreateBucketV2(ctx, &volctos.CreateBucketV2Input{Bucket: s.bucket}); err != nil {
		return fmt.Errorf("blob/tos: create bucket: %w", err)
	}
	return s.Check(ctx)
}

func (s *Store) Put(ctx context.Context, key string, r io.Reader, opts blob.PutOptions) (sharedkernel.BlobRef, error) {
	if err := ctx.Err(); err != nil {
		return sharedkernel.BlobRef{}, err
	}
	rel, err := cleanKey(key)
	if err != nil {
		return sharedkernel.BlobRef{}, err
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return sharedkernel.BlobRef{}, fmt.Errorf("blob/tos: read body: %w", err)
	}
	in := &volctos.PutObjectV2Input{
		PutObjectBasicInput: volctos.PutObjectBasicInput{
			Bucket: s.bucket,
			Key:    rel,
		},
		Content: bytes.NewReader(data),
	}
	if opts.MIME != "" {
		in.ContentType = opts.MIME
	}
	if _, err := s.client.PutObjectV2(ctx, in); err != nil {
		return sharedkernel.BlobRef{}, fmt.Errorf("blob/tos: put %s: %w", rel, err)
	}
	return sharedkernel.BlobRef{
		Bucket: s.bucket,
		Key:    rel,
		MIME:   opts.MIME,
		Size:   int64(len(data)),
	}, nil
}

func (s *Store) Get(ctx context.Context, ref sharedkernel.BlobRef) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rel, err := cleanKey(ref.Key)
	if err != nil {
		return nil, err
	}
	bucket := ref.Bucket
	if bucket == "" {
		bucket = s.bucket
	}
	out, err := s.client.GetObjectV2(ctx, &volctos.GetObjectV2Input{
		Bucket: bucket,
		Key:    rel,
	})
	if err != nil {
		return nil, fmt.Errorf("blob/tos: get %s: %w", rel, err)
	}
	return out.Content, nil
}

func cleanKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", errors.New("blob/tos: empty key")
	}
	if path.IsAbs(key) || strings.HasPrefix(key, "/") {
		return "", errors.New("blob/tos: absolute key not allowed")
	}
	clean := path.Clean(key)
	if clean == "." || strings.HasPrefix(clean, "../") || clean == ".." {
		return "", errors.New("blob/tos: invalid key")
	}
	return clean, nil
}

var _ blob.Store = (*Store)(nil)
