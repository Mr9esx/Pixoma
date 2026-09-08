package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

// Options configures an S3-compatible blob store.
type Options struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	UsePathStyle    bool
}

// Store implements blob.Store against S3-compatible object storage.
type Store struct {
	client *s3.Client
	bucket string
	region string
}

// New builds an S3 Store. Endpoint may be empty for real AWS; required for MinIO/fakes.
func New(opts Options) (*Store, error) {
	if strings.TrimSpace(opts.Bucket) == "" {
		return nil, errors.New("blob/s3: empty bucket")
	}
	region := strings.TrimSpace(opts.Region)
	if region == "" {
		region = "us-east-1"
	}
	cfg := aws.Config{
		Region: region,
		Credentials: credentials.NewStaticCredentialsProvider(
			opts.AccessKeyID,
			opts.SecretAccessKey,
			"",
		),
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if ep := strings.TrimSpace(opts.Endpoint); ep != "" {
			o.BaseEndpoint = aws.String(ep)
		}
		o.UsePathStyle = opts.UsePathStyle
		o.HTTPClient = http.DefaultClient
	})
	return &Store{client: client, bucket: opts.Bucket, region: region}, nil
}

// Check verifies the bucket is reachable (HeadBucket).
func (s *Store) Check(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) &&
			(apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "NoSuchBucket") {
			return fmt.Errorf("blob/s3: %w", blob.ErrBucketNotFound)
		}
		return fmt.Errorf("blob/s3: check: %w", err)
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
	in := &s3.CreateBucketInput{Bucket: aws.String(s.bucket)}
	if region := strings.TrimSpace(s.region); region != "" && region != "us-east-1" {
		in.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(region),
		}
	}
	if _, err := s.client.CreateBucket(ctx, in); err != nil {
		return fmt.Errorf("blob/s3: create bucket: %w", err)
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
		return sharedkernel.BlobRef{}, fmt.Errorf("blob/s3: read body: %w", err)
	}
	in := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(rel),
		Body:   bytes.NewReader(data),
	}
	if opts.MIME != "" {
		in.ContentType = aws.String(opts.MIME)
	}
	if _, err := s.client.PutObject(ctx, in); err != nil {
		return sharedkernel.BlobRef{}, fmt.Errorf("blob/s3: put %s: %w", rel, err)
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
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(rel),
	})
	if err != nil {
		return nil, fmt.Errorf("blob/s3: get %s: %w", rel, err)
	}
	return out.Body, nil
}

func cleanKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", errors.New("blob/s3: empty key")
	}
	if path.IsAbs(key) || strings.HasPrefix(key, "/") {
		return "", errors.New("blob/s3: absolute key not allowed")
	}
	clean := path.Clean(key)
	if clean == "." || strings.HasPrefix(clean, "../") || clean == ".." {
		return "", errors.New("blob/s3: invalid key")
	}
	return clean, nil
}

var _ blob.Store = (*Store)(nil)
