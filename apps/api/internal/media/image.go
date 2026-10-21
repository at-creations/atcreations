// Package media validates and normalizes uploaded images and stores them in
// S3-compatible object storage (Cloudflare R2).
package media

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register the WebP decoder

	"github.com/atcreations/atcreations/apps/api/internal/platform/httpx"
)

const (
	MaxUploadBytes = 10 << 20 // 10 MiB
	maxPixels      = 32_000_000
	maxDimension   = 12_000
	// Upper bound on the decoded bitmap. A small, highly compressible file can
	// expand to hundreds of MB (16-bit PNG = 8 bytes/pixel), so this is checked
	// from the header before decoding.
	maxDecodedBytes = 128 << 20
	maxOutputEdge   = 2400
	jpegQuality     = 85
)

// decodeSlots bounds concurrent decodes so parallel uploads cannot exhaust
// memory; the remaining requests wait (or give up when the client leaves).
var decodeSlots = make(chan struct{}, 2)

// bytesPerPixel estimates the decoded size for a color model.
func bytesPerPixel(m color.Model) int {
	switch m {
	case color.GrayModel, color.AlphaModel:
		return 1
	case color.Gray16Model, color.Alpha16Model:
		return 2
	case color.RGBA64Model, color.NRGBA64Model:
		return 8
	default:
		return 4
	}
}

var allowedFormats = map[string]bool{"jpeg": true, "png": true, "webp": true}

// Processed is a re-encoded image ready to store.
type Processed struct {
	Data          []byte
	ContentType   string
	Ext           string
	Width, Height int
}

var (
	ErrTooLarge    = httpx.NewError(http.StatusRequestEntityTooLarge, "image must be 10 MB or smaller")
	ErrUnsupported = httpx.NewError(http.StatusUnsupportedMediaType, "image must be a JPEG, PNG or WebP file")
	ErrDimensions  = httpx.Validation(httpx.FieldError{Field: "file", Message: "image dimensions are too large (max 32 megapixels, 8-bit color)"})
)

// Process sniffs the real format from the bytes (never trusting the client's
// filename or Content-Type), guards against decompression bombs, downsizes
// very large images and re-encodes them. Re-encoding strips EXIF/GPS metadata
// and neutralizes polyglot files.
func Process(ctx context.Context, r io.Reader) (*Processed, error) {
	raw, err := io.ReadAll(io.LimitReader(r, MaxUploadBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > MaxUploadBytes {
		return nil, ErrTooLarge
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || !allowedFormats[format] {
		return nil, ErrUnsupported
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > maxDimension || cfg.Height > maxDimension ||
		cfg.Width*cfg.Height > maxPixels || cfg.Width*cfg.Height*bytesPerPixel(cfg.ColorModel) > maxDecodedBytes {
		return nil, ErrDimensions
	}
	select {
	case decodeSlots <- struct{}{}:
		defer func() { <-decodeSlots }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, ErrUnsupported
	}
	img = downscale(img, maxOutputEdge)
	b := img.Bounds()

	var out bytes.Buffer
	if hasAlpha(img) {
		enc := png.Encoder{CompressionLevel: png.BestCompression}
		if err := enc.Encode(&out, img); err != nil {
			return nil, err
		}
		return &Processed{Data: out.Bytes(), ContentType: "image/png", Ext: "png", Width: b.Dx(), Height: b.Dy()}, nil
	}
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, err
	}
	return &Processed{Data: out.Bytes(), ContentType: "image/jpeg", Ext: "jpg", Width: b.Dx(), Height: b.Dy()}, nil
}

func downscale(img image.Image, maxEdge int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxEdge && h <= maxEdge {
		return img
	}
	if w >= h {
		h, w = h*maxEdge/w, maxEdge
	} else {
		w, h = w*maxEdge/h, maxEdge
	}
	dst := image.NewNRGBA(image.Rect(0, 0, max(w, 1), max(h, 1)))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
	return dst
}

// hasAlpha reports whether any pixel is not fully opaque.
func hasAlpha(img image.Image) bool {
	if o, ok := img.(interface{ Opaque() bool }); ok {
		return !o.Opaque()
	}
	return true
}
