package media

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Storage is the subset of object-storage operations the API needs.
type Storage interface {
	Put(ctx context.Context, key string, data []byte, contentType string) error
	Delete(ctx context.Context, keys ...string) error
	// Walk calls fn for every object under prefix.
	Walk(ctx context.Context, prefix string, fn func(key string, modified time.Time) error) error
	URL(key string) string
}

type S3Config struct {
	Endpoint, Region, Bucket, AccessKeyID, SecretAccessKey, PublicBaseURL string
}

type S3Storage struct {
	client  *s3.Client
	bucket  string
	baseURL string
}

func NewS3Storage(cfg S3Config) *S3Storage {
	client := s3.New(s3.Options{
		Region:       cfg.Region,
		BaseEndpoint: aws.String(cfg.Endpoint),
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		UsePathStyle: true,
		// R2 does not support the newer default checksum trailers on every operation.
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
	})
	return &S3Storage{client: client, bucket: cfg.Bucket, baseURL: strings.TrimRight(cfg.PublicBaseURL, "/")}
}

func (s *S3Storage) Put(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(data),
		ContentLength: aws.Int64(int64(len(data))),
		ContentType:   aws.String(contentType),
		// Keys are content-unique (random), so objects can be cached forever.
		CacheControl: aws.String("public, max-age=31536000, immutable"),
	})
	if err != nil {
		return fmt.Errorf("put object %s: %w", key, err)
	}
	return nil
}

func (s *S3Storage) Delete(ctx context.Context, keys ...string) error {
	for start := 0; start < len(keys); start += 1000 {
		batch := keys[start:min(start+1000, len(keys))]
		objs := make([]types.ObjectIdentifier, len(batch))
		for i, k := range batch {
			objs[i] = types.ObjectIdentifier{Key: aws.String(k)}
		}
		_, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(s.bucket),
			Delete: &types.Delete{Objects: objs, Quiet: aws.Bool(true)},
		})
		if err != nil {
			return fmt.Errorf("delete objects: %w", err)
		}
	}
	return nil
}

func (s *S3Storage) Walk(ctx context.Context, prefix string, fn func(string, time.Time) error) error {
	p := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{Bucket: aws.String(s.bucket), Prefix: aws.String(prefix)})
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("list objects: %w", err)
		}
		for _, o := range page.Contents {
			if err := fn(aws.ToString(o.Key), aws.ToTime(o.LastModified)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *S3Storage) URL(key string) string { return s.baseURL + "/" + key }

// NewKey returns an unguessable object key under prefix. Keys never contain
// user-supplied filenames, which rules out path traversal and overwrites.
func NewKey(prefix, ext string) string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return strings.TrimRight(prefix, "/") + "/" + hex.EncodeToString(b[:]) + "." + ext
}

// NewID returns a short random identifier for embedded image records.
func NewID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
