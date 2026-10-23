// Package media validates and normalizes uploaded images and stores them in
// S3-compatible object storage (Cloudflare R2).
package media

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg" // register decoders
	_ "image/png"
	"net/http"

	"github.com/gen2brain/heic"
	"github.com/gen2brain/webp"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register the WebP decoder

	"github.com/atcreations/atcreations/apps/api/internal/platform/httpx"
)

const (
	// MaxUploadBytes caps an original upload. Originals are downscaled by the image worker, so
	// full-resolution phone photos are accepted.
	MaxUploadBytes = 25 << 20 // 25 MiB
	maxPixels      = 50_000_000
	maxDimension   = 16_000
	// Upper bound on the decoded bitmap. A small, highly compressible file can expand to
	// hundreds of MB (16-bit PNG = 8 bytes/pixel), so this is checked from the header before
	// decoding.
	maxDecodedBytes = 256 << 20

	DefaultMaxEdge = 2400
	DefaultQuality = 82
)

// decodeSlots bounds concurrent decodes so parallel jobs cannot exhaust memory; the remaining
// callers wait (or give up when their context ends). The worker sizes it with SetConcurrency.
var decodeSlots = make(chan struct{}, 2)

// SetConcurrency sets how many images may be decoded at once (call before processing starts).
func SetConcurrency(n int) { decodeSlots = make(chan struct{}, max(1, n)) }

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

var allowedFormats = map[string]bool{"jpeg": true, "png": true, "webp": true, "heic": true}

func init() {
	// iPhone photos often declare the generic HEIF brand "mif1"; the heic package only
	// registers the HEVC-specific brands.
	image.RegisterFormat("heic", "????ftypmif1", heic.Decode, heic.DecodeConfig)
}

// Processed is a re-encoded image ready to store.
type Processed struct {
	Data          []byte
	ContentType   string
	Ext           string
	Width, Height int
}

var (
	ErrTooLarge    = httpx.NewError(http.StatusRequestEntityTooLarge, "image must be 25 MB or smaller")
	ErrUnsupported = httpx.NewError(http.StatusUnsupportedMediaType, "image must be a JPEG, PNG, WebP or HEIC file")
	ErrDimensions  = httpx.Validation(httpx.FieldError{Field: "file", Message: "image dimensions are too large (max 50 megapixels, 8-bit color)"})
)

// Inspect checks the real format and dimensions from the file header without decoding the
// pixels (never trusting the client's filename or Content-Type). Uploads are inspected before
// they are queued so obviously bad files fail immediately.
func Inspect(raw []byte) (format string, cfg image.Config, err error) {
	if len(raw) > MaxUploadBytes {
		return "", cfg, ErrTooLarge
	}
	cfg, format, err = image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || !allowedFormats[format] {
		return "", cfg, ErrUnsupported
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > maxDimension || cfg.Height > maxDimension ||
		cfg.Width*cfg.Height > maxPixels || cfg.Width*cfg.Height*bytesPerPixel(cfg.ColorModel) > maxDecodedBytes {
		return "", cfg, ErrDimensions
	}
	return format, cfg, nil
}

type Options struct {
	// MaxEdge is the longest side of the output in pixels (default DefaultMaxEdge).
	MaxEdge int
	// Quality is the WebP quality 1-100 (default DefaultQuality).
	Quality int
	// OnStage, when set, is called as processing enters each stage (for progress reporting).
	OnStage func(Stage)
}

// Stage of Process, reported through Options.OnStage.
type Stage int

const (
	StageDecoding Stage = iota + 1
	StageResizing
	StageEncoding
)

// Process validates, decodes (honouring the JPEG EXIF orientation), downsizes and re-encodes an
// image as WebP. Re-encoding strips EXIF/GPS metadata and neutralizes polyglot files.
func Process(ctx context.Context, raw []byte, opts Options) (*Processed, error) {
	if opts.MaxEdge <= 0 {
		opts.MaxEdge = DefaultMaxEdge
	}
	if opts.Quality <= 0 || opts.Quality > 100 {
		opts.Quality = DefaultQuality
	}
	format, _, err := Inspect(raw)
	if err != nil {
		return nil, err
	}
	stage := func(st Stage) {
		if opts.OnStage != nil {
			opts.OnStage(st)
		}
	}
	select {
	case decodeSlots <- struct{}{}:
		defer func() { <-decodeSlots }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	stage(StageDecoding)
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, ErrUnsupported
	}
	if format == "jpeg" {
		img = applyOrientation(img, jpegOrientation(raw))
	}
	stage(StageResizing)
	img = downscale(img, opts.MaxEdge)
	b := img.Bounds()

	stage(StageEncoding)
	var out bytes.Buffer
	if err := webp.Encode(&out, img, webp.Options{Quality: opts.Quality, Method: 4}); err != nil {
		return nil, err
	}
	return &Processed{Data: out.Bytes(), ContentType: "image/webp", Ext: "webp", Width: b.Dx(), Height: b.Dy()}, nil
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
