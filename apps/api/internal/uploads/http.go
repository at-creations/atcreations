package uploads

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/atcreations/atcreations/apps/api/internal/auth"
	"github.com/atcreations/atcreations/apps/api/internal/media"
	"github.com/atcreations/atcreations/apps/api/internal/platform/httpx"
)

// receiving bounds how many uploads one API instance buffers at once (each up to 25 MB).
var receiving = make(chan struct{}, 4)

const receiveWait = 20 * time.Second

// Upload handles POST /v1/admin/{products|categories}/{id}/uploads (one multipart file per
// request; the admin sends several requests in parallel). Answers 202 with the queued job.
func (s *Service) Upload(kind string) http.HandlerFunc {
	return httpx.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id, err := httpx.ObjectIDParam(r, "id")
		if err != nil {
			return err
		}
		select {
		case receiving <- struct{}{}:
			defer func() { <-receiving }()
		case <-time.After(receiveWait):
			w.Header().Set("Retry-After", "5")
			return httpx.NewError(http.StatusServiceUnavailable, "too many uploads in progress, please retry")
		case <-r.Context().Done():
			return r.Context().Err()
		}

		r.Body = http.MaxBytesReader(w, r.Body, media.MaxUploadBytes+64<<10)
		// Small memory budget: larger parts spill to a temp file that the standard library
		// removes when the request ends. The body is bounded by MaxBytesReader above.
		if err := r.ParseMultipartForm(1 << 20); err != nil { //nolint:gosec // G120: bounded above
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				return media.ErrTooLarge
			}
			return httpx.BadRequest("expected a multipart/form-data body")
		}
		defer func() { _ = r.MultipartForm.RemoveAll() }()
		files := r.MultipartForm.File["file"]
		if len(files) != 1 {
			return httpx.Validation(httpx.FieldError{Field: "file", Message: "exactly one file is required"})
		}
		f, err := files[0].Open()
		if err != nil {
			return err
		}
		data, err := io.ReadAll(io.LimitReader(f, media.MaxUploadBytes+1))
		_ = f.Close()
		if err != nil {
			return err
		}
		job, err := s.Enqueue(r.Context(), actor(r), kind, id, files[0].Filename, data)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusAccepted, job.DTO())
		return nil
	})
}

// List handles GET …/{id}/uploads.
func (s *Service) ListHandler(kind string) http.HandlerFunc {
	return httpx.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id, err := httpx.ObjectIDParam(r, "id")
		if err != nil {
			return err
		}
		jobs, err := s.List(r.Context(), kind, id)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, jobs)
		return nil
	})
}

// CancelHandler handles DELETE …/{id}/uploads/{jobId}.
func (s *Service) CancelHandler(kind string) http.HandlerFunc {
	return httpx.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id, err := httpx.ObjectIDParam(r, "id")
		if err != nil {
			return err
		}
		jobID, err := httpx.ObjectIDParam(r, "jobId")
		if err != nil {
			return err
		}
		if err := s.Cancel(r.Context(), kind, id, jobID); err != nil {
			return err
		}
		httpx.NoContent(w)
		return nil
	})
}

func actor(r *http.Request) string {
	if p := auth.PrincipalFrom(r.Context()); p != nil {
		return p.UserID
	}
	return ""
}
