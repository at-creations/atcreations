package media

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// UploadsPrefix is the private staging area for originals waiting to be processed. Objects
// there are never referenced by the catalogue and are removed once processed (or by the sweep).
const UploadsPrefix = "uploads/"

// ErrObjectNotFound is returned by Get and Head for a missing key.
var ErrObjectNotFound = errors.New("object not found")

// Storage is the subset of object-storage operations the API and the image worker need.
type Storage interface {
	Put(ctx context.Context, key string, data []byte, contentType string) error
	PutObject(ctx context.Context, in PutInput) error
	// Get reads an object, refusing anything larger than maxBytes.
	Get(ctx context.Context, key string, maxBytes int64) ([]byte, error)
	Head(ctx context.Context, key string) (*ObjectInfo, error)
	Delete(ctx context.Context, keys ...string) error
	// Walk calls fn for every object under prefix.
	Walk(ctx context.Context, prefix string, fn func(key string, modified time.Time) error) error
	URL(key string) string
}

type PutInput struct {
	Key         string
	Data        []byte
	ContentType string
	// Metadata is stored as x-amz-meta-* headers (lowercase keys).
	Metadata map[string]string
}

type ObjectInfo struct {
	Size     int64
	Metadata map[string]string
}

type S3Config struct {
	Endpoint, Region, Bucket, AccessKeyID, SecretAccessKey, PublicBaseURL string
	// MaxAttempts per S3 call including the first (default 8).
	MaxAttempts int
}

type S3Storage struct {
	client  *s3.Client
	bucket  string
	baseURL string
}

// Throttling responses from S3-compatible stores (R2, MinIO, RustFS) are not always tagged with
// AWS error codes, so status 429 and 503 also count as throttling: the adaptive retryer then
// backs off with jitter and slows this client's request rate instead of failing the upload.
var throttleStatus = retry.IsErrorThrottleFunc(func(err error) aws.Ternary {
	var re *smithyhttp.ResponseError
	if errors.As(err, &re) {
		switch re.HTTPStatusCode() {
		case http.StatusTooManyRequests, http.StatusServiceUnavailable:
			return aws.TrueTernary
		}
	}
	return aws.UnknownTernary
})

func newRetryer(maxAttempts int) aws.Retryer {
	if maxAttempts <= 0 {
		maxAttempts = 8
	}
	return retry.NewAdaptiveMode(func(o *retry.AdaptiveModeOptions) {
		o.StandardOptions = append(o.StandardOptions, func(so *retry.StandardOptions) {
			so.MaxAttempts = maxAttempts
			so.MaxBackoff = 20 * time.Second
			so.Retryables = append(so.Retryables,
				retry.RetryableHTTPStatusCode{Codes: map[int]struct{}{http.StatusTooManyRequests: {}}})
			so.Throttles = append(so.Throttles, throttleStatus)
		})
	})
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
		Retryer:                    newRetryer(cfg.MaxAttempts),
	})
	return &S3Storage{client: client, bucket: cfg.Bucket, baseURL: strings.TrimRight(cfg.PublicBaseURL, "/")}
}

func (s *S3Storage) Put(ctx context.Context, key string, data []byte, contentType string) error {
	return s.PutObject(ctx, PutInput{Key: key, Data: data, ContentType: contentType})
}

func (s *S3Storage) PutObject(ctx context.Context, in PutInput) error {
	// Published keys are content-unique (random), so they can be cached forever. Originals in
	// the staging area are private and short-lived.
	cache := "public, max-age=31536000, immutable"
	if strings.HasPrefix(in.Key, UploadsPrefix) {
		cache = "private, no-store"
	}
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(in.Key),
		Body:          bytes.NewReader(in.Data),
		ContentLength: aws.Int64(int64(len(in.Data))),
		ContentType:   aws.String(in.ContentType),
		CacheControl:  aws.String(cache),
		Metadata:      in.Metadata,
	})
	if err != nil {
		return fmt.Errorf("put object %s: %w", in.Key, err)
	}
	return nil
}

func (s *S3Storage) Get(ctx context.Context, key string, maxBytes int64) ([]byte, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		if isNotFound(err) {
			return nil, ErrObjectNotFound
		}
		return nil, fmt.Errorf("get object %s: %w", key, err)
	}
	defer func() { _ = out.Body.Close() }()
	if out.ContentLength != nil && *out.ContentLength > maxBytes {
		return nil, ErrTooLarge
	}
	data, err := io.ReadAll(io.LimitReader(out.Body, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read object %s: %w", key, err)
	}
	if int64(len(data)) > maxBytes {
		return nil, ErrTooLarge
	}
	return data, nil
}

func (s *S3Storage) Head(ctx context.Context, key string) (*ObjectInfo, error) {
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		if isNotFound(err) {
			return nil, ErrObjectNotFound
		}
		return nil, fmt.Errorf("head object %s: %w", key, err)
	}
	return &ObjectInfo{Size: aws.ToInt64(out.ContentLength), Metadata: out.Metadata}, nil
}

func isNotFound(err error) bool {
	var nsk *types.NoSuchKey
	var nf *types.NotFound
	if errors.As(err, &nsk) || errors.As(err, &nf) {
		return true
	}
	var re *smithyhttp.ResponseError
	return errors.As(err, &re) && re.HTTPStatusCode() == http.StatusNotFound
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
