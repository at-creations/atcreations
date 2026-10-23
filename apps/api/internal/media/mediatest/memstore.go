// Package mediatest provides an in-memory media.Storage for tests.
package mediatest

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/atcreations/atcreations/apps/api/internal/media"
)

type object struct {
	data     []byte
	meta     map[string]string
	modified time.Time
}

// Store is a concurrency-safe in-memory object store.
type Store struct {
	mu   sync.Mutex
	objs map[string]object
	// FailPuts makes the next n Put/PutObject calls fail (to simulate throttling).
	FailPuts int
}

func New() *Store { return &Store{objs: map[string]object{}} }

func (s *Store) Put(ctx context.Context, key string, data []byte, contentType string) error {
	return s.PutObject(ctx, media.PutInput{Key: key, Data: data, ContentType: contentType})
}

func (s *Store) PutObject(_ context.Context, in media.PutInput) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.FailPuts > 0 {
		s.FailPuts--
		return errThrottled
	}
	s.objs[in.Key] = object{data: append([]byte(nil), in.Data...), meta: in.Metadata, modified: time.Now()}
	return nil
}

func (s *Store) Get(_ context.Context, key string, maxBytes int64) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.objs[key]
	if !ok {
		return nil, media.ErrObjectNotFound
	}
	if int64(len(o.data)) > maxBytes {
		return nil, media.ErrTooLarge
	}
	return append([]byte(nil), o.data...), nil
}

func (s *Store) Head(_ context.Context, key string) (*media.ObjectInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.objs[key]
	if !ok {
		return nil, media.ErrObjectNotFound
	}
	return &media.ObjectInfo{Size: int64(len(o.data)), Metadata: o.meta}, nil
}

func (s *Store) Delete(_ context.Context, keys ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, k := range keys {
		delete(s.objs, k)
	}
	return nil
}

func (s *Store) Walk(_ context.Context, prefix string, fn func(string, time.Time) error) error {
	s.mu.Lock()
	keys := map[string]time.Time{}
	for k, o := range s.objs {
		if strings.HasPrefix(k, prefix) {
			keys[k] = o.modified
		}
	}
	s.mu.Unlock()
	for k, t := range keys {
		if err := fn(k, t); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) URL(key string) string { return "https://cdn.test/" + key }

// Has reports whether key exists.
func (s *Store) Has(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.objs[key]
	return ok
}

// Keys returns all keys under prefix.
func (s *Store) Keys(prefix string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for k := range s.objs {
		if strings.HasPrefix(k, prefix) {
			out = append(out, k)
		}
	}
	return out
}

// Age backdates an object's modification time (for sweep tests).
func (s *Store) Age(key string, by time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if o, ok := s.objs[key]; ok {
		o.modified = o.modified.Add(-by)
		s.objs[key] = o
	}
}

type throttled struct{}

func (throttled) Error() string { return "SlowDown: please reduce your request rate" }

var errThrottled error = throttled{}
