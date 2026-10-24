// Package media validates and normalizes uploaded images and stores them in
// S3-compatible object storage (Cloudflare R2).
package media

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg" // register decoders
	_ "image/png"
	"net/http"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

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
	Data          []byte // empty with Options.VariantsOnly
	ContentType   string
	Ext           string
	Width, Height int
	Variants      []Variant
}

// Variant is a smaller copy of an image for responsive loading (srcset).
type Variant struct {
	Edge          int // the requested longest edge
	Data          []byte
	Width, Height int
}

// VariantEdges are the longest edges (px) of the smaller copies stored next to every image,
// so pages can load the size they display. Only edges below the image's own are produced.
var VariantEdges = []int{400, 800, 1600}

// VariantKey is where the edge-px variant of the image stored at key lives ("a/b.webp" →
// "a/b-800.webp"). Deriving it from the key lets any cleanup delete variants by name.
func VariantKey(key string, edge int) string {
	ext := path.Ext(key)
	return strings.TrimSuffix(key, ext) + "-" + strconv.Itoa(edge) + ext
}

// VariantKeys returns the keys of every possible variant of key.
func VariantKeys(key string) []string {
	out := make([]string, len(VariantEdges))
	for i, e := range VariantEdges {
		out[i] = VariantKey(key, e)
	}
	return out
}

// WithVariants returns keys plus the variant keys of every stored image among them (originals
// in the uploads area have none), so deleting an image also deletes its smaller copies.
func WithVariants(keys ...string) []string {
	out := make([]string, 0, len(keys)*(1+len(VariantEdges)))
	for _, k := range keys {
		if k == "" {
			continue
		}
		out = append(out, k)
		if !strings.HasPrefix(k, UploadsPrefix) && BaseKey(k) == k {
			out = append(out, VariantKeys(k)...)
		}
	}
	return out
}

var variantSuffix = regexp.MustCompile(`-(\d+)(\.[a-z0-9]+)$`)

// BaseKey returns the main image key of a variant key, or key itself if it is not a variant.
func BaseKey(key string) string {
	m := variantSuffix.FindStringSubmatchIndex(key)
	if m == nil {
		return key
	}
	if edge, err := strconv.Atoi(key[m[2]:m[3]]); err != nil || !slices.Contains(VariantEdges, edge) {
		return key
	}
	return key[:m[0]] + key[m[4]:m[5]]
}

var (
	ErrTooLarge    = httpx.NewError(http.StatusRequestEntityTooLarge, "image must be 25 MB or smaller")
	ErrUnsupported = httpx.NewError(http.StatusUnsupportedMediaType, "image must be a JPEG, PNG, WebP or HEIC file")
	// ErrUnreadable is a file whose header is a supported image but whose data cannot be decoded
	// (truncated or corrupted).
	ErrUnreadable = httpx.NewError(http.StatusUnprocessableEntity, "image file is damaged or incomplete")
	ErrDimensions = httpx.Validation(httpx.FieldError{Field: "file", Message: "image dimensions are too large (max 50 megapixels, 8-bit color)"})
)

// Stable codes for image errors, reported on upload jobs so clients can show their own wording.
const (
	CodeUnsupported = "unsupported"
	CodeUnreadable  = "unreadable"
	CodeTooLarge    = "too_large"
	CodeDimensions  = "dimensions"
)

// ErrorCode returns the stable code of an image validation error, or "" for any other error.
func ErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrUnsupported):
		return CodeUnsupported
	case errors.Is(err, ErrUnreadable):
		return CodeUnreadable
	case errors.Is(err, ErrTooLarge):
		return CodeTooLarge
	case errors.Is(err, ErrDimensions):
		return CodeDimensions
	}
	return ""
}

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
	// Variants lists longest edges of smaller copies to produce (see VariantEdges).
	Variants []int
	// VariantsOnly skips encoding the main image (adding variants to an existing image).
	VariantsOnly bool
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
		// The header was valid (Inspect passed), so the data itself is damaged.
		return nil, ErrUnreadable
	}
	if format == "jpeg" {
		img = applyOrientation(img, jpegOrientation(raw))
	}
	stage(StageResizing)
	img = downscale(img, opts.MaxEdge)
	b := img.Bounds()

	stage(StageEncoding)
	encode := func(m image.Image) ([]byte, error) {
		var out bytes.Buffer
		if err := webp.Encode(&out, m, webp.Options{Quality: opts.Quality, Method: 4}); err != nil {
			return nil, err
		}
		return out.Bytes(), nil
	}
	p := &Processed{ContentType: "image/webp", Ext: "webp", Width: b.Dx(), Height: b.Dy()}
	if !opts.VariantsOnly {
		if p.Data, err = encode(img); err != nil {
			return nil, err
		}
	}
	// Variants are scaled down from the (already downsized) main image.
	for _, edge := range opts.Variants {
		if edge <= 0 || edge >= max(b.Dx(), b.Dy()) {
			continue
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		small := downscale(img, edge)
		data, err := encode(small)
		if err != nil {
			return nil, err
		}
		sb := small.Bounds()
		p.Variants = append(p.Variants, Variant{Edge: edge, Data: data, Width: sb.Dx(), Height: sb.Dy()})
	}
	return p, nil
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
