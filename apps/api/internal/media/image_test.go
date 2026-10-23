package media

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"strings"
	"testing"

	"github.com/gen2brain/webp"

	"github.com/atcreations/atcreations/apps/api/internal/platform/httpx"
)

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func solid(w, h int, c color.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := range w {
		for y := range h {
			img.Set(x, y, c)
		}
	}
	return img
}

func TestProcessOutputsWebP(t *testing.T) {
	p, err := Process(context.Background(), encodePNG(t, solid(40, 20, color.RGBA{R: 200, A: 255})), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if p.ContentType != "image/webp" || p.Ext != "webp" || p.Width != 40 || p.Height != 20 {
		t.Fatalf("unexpected result: %s %s %dx%d", p.ContentType, p.Ext, p.Width, p.Height)
	}
	cfg, err := webp.DecodeConfig(bytes.NewReader(p.Data))
	if err != nil || cfg.Width != 40 || cfg.Height != 20 {
		t.Fatalf("output is not a valid 40x20 WebP: %v %+v", err, cfg)
	}
}

func TestProcessKeepsTransparency(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 10, 10)) // fully transparent
	p, err := Process(context.Background(), encodePNG(t, img), Options{})
	if err != nil {
		t.Fatal(err)
	}
	out, err := webp.Decode(bytes.NewReader(p.Data))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, a := out.At(5, 5).RGBA(); a != 0 {
		t.Fatalf("alpha = %d, want 0", a)
	}
}

func TestProcessDownscales(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 4800, 1200))
	p, err := Process(context.Background(), encodePNG(t, img), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if p.Width != 2400 || p.Height != 600 {
		t.Fatalf("got %dx%d, want 2400x600", p.Width, p.Height)
	}
	p, err = Process(context.Background(), encodePNG(t, img), Options{MaxEdge: 1000})
	if err != nil || p.Width != 1000 || p.Height != 250 {
		t.Fatalf("MaxEdge 1000: got %dx%d, %v", p.Width, p.Height, err)
	}
}

func TestProcessRejectsNonImages(t *testing.T) {
	for _, body := range []string{"", "<svg xmlns='http://www.w3.org/2000/svg'/>", "GIF89a....", "<?php echo 1; ?>"} {
		_, err := Process(context.Background(), []byte(body), Options{})
		if !errors.Is(err, ErrUnsupported) {
			t.Errorf("Process(%q) err = %v, want ErrUnsupported", body, err)
		}
	}
}

func TestProcessRejectsOversizedHeader(t *testing.T) {
	// A tiny PNG claiming huge dimensions (decompression bomb) must be refused from its header.
	data := encodePNG(t, image.NewGray(image.Rect(0, 0, 1, 1)))
	copy(data[16:24], []byte{0x00, 0x00, 0x75, 0x30, 0x00, 0x00, 0x75, 0x30}) // 30000x30000
	_, err := Process(context.Background(), data, Options{})
	var e *httpx.Error
	if !errors.As(err, &e) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestProcessRejects16BitBombs(t *testing.T) {
	// 16-bit images decode to 8 bytes/pixel: 6000x6000 = 288 MB, over the cap.
	img := image.NewNRGBA64(image.Rect(0, 0, 6000, 6000))
	if _, err := Process(context.Background(), encodePNG(t, img), Options{}); !errors.Is(err, ErrDimensions) {
		t.Fatalf("err = %v, want ErrDimensions", err)
	}
}

func TestInspectRejectsTooLarge(t *testing.T) {
	if _, _, err := Inspect(make([]byte, MaxUploadBytes+1)); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v, want ErrTooLarge", err)
	}
}

// withOrientation inserts an EXIF APP1 segment carrying the given orientation into a JPEG.
func withOrientation(t *testing.T, jpg []byte, o uint16) []byte {
	t.Helper()
	tiff := make([]byte, 8+2+12+4)
	copy(tiff, "MM")
	binary.BigEndian.PutUint16(tiff[2:], 42)
	binary.BigEndian.PutUint32(tiff[4:], 8)
	binary.BigEndian.PutUint16(tiff[8:], 1)            // one IFD entry
	binary.BigEndian.PutUint16(tiff[10:], 0x0112)      // Orientation
	binary.BigEndian.PutUint16(tiff[12:], 3)           // SHORT
	binary.BigEndian.PutUint32(tiff[14:], 1)           // count
	binary.BigEndian.PutUint16(tiff[18:], o)           // value
	payload := append([]byte("Exif\x00\x00"), tiff...) //nolint:gocritic // fresh slice
	seg := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2))
	out := append([]byte{}, jpg[:2]...)
	out = append(out, seg...)
	out = append(out, payload...)
	return append(out, jpg[2:]...)
}

func TestProcessAppliesEXIFOrientation(t *testing.T) {
	var b bytes.Buffer
	if err := jpeg.Encode(&b, solid(60, 20, color.RGBA{G: 180, A: 255}), nil); err != nil {
		t.Fatal(err)
	}
	if got := jpegOrientation(withOrientation(t, b.Bytes(), 6)); got != 6 {
		t.Fatalf("jpegOrientation = %d, want 6", got)
	}
	p, err := Process(context.Background(), withOrientation(t, b.Bytes(), 6), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if p.Width != 20 || p.Height != 60 {
		t.Fatalf("rotated size = %dx%d, want 20x60", p.Width, p.Height)
	}
	if got := jpegOrientation(b.Bytes()); got != 1 {
		t.Fatalf("no EXIF: orientation = %d, want 1", got)
	}
}

func TestApplyOrientationMapsPixels(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	src.Set(0, 0, color.NRGBA{R: 255, A: 255}) // red left
	src.Set(1, 0, color.NRGBA{B: 255, A: 255}) // blue right
	rot := applyOrientation(src, 6)            // 90° clockwise: red ends on top
	if rot.Bounds().Dx() != 1 || rot.Bounds().Dy() != 2 {
		t.Fatalf("bounds = %v", rot.Bounds())
	}
	if r, _, _, _ := rot.At(0, 0).RGBA(); r == 0 {
		t.Fatal("top pixel should be red after rotating 90° clockwise")
	}
}

func TestProcessHEIC(t *testing.T) {
	raw, err := os.ReadFile("testdata/sample.heic")
	if err != nil {
		t.Fatal(err)
	}
	format, cfg, err := Inspect(raw)
	if err != nil || format != "heic" || cfg.Width == 0 {
		t.Fatalf("Inspect = %q %+v %v", format, cfg, err)
	}
	p, err := Process(context.Background(), raw, Options{MaxEdge: 64})
	if err != nil {
		t.Fatal(err)
	}
	if p.ContentType != "image/webp" || max(p.Width, p.Height) > 64 {
		t.Fatalf("got %s %dx%d", p.ContentType, p.Width, p.Height)
	}
}

func TestNewKeyShape(t *testing.T) {
	k := NewKey("products/abc/", "webp")
	if !strings.HasPrefix(k, "products/abc/") || !strings.HasSuffix(k, ".webp") || len(k) != len("products/abc/")+32+5 {
		t.Fatalf("unexpected key %q", k)
	}
}

func TestProcessReportsDamagedImages(t *testing.T) {
	var b bytes.Buffer
	if err := jpeg.Encode(&b, image.NewGray(image.Rect(0, 0, 400, 400)), nil); err != nil {
		t.Fatal(err)
	}
	// A valid header followed by truncated data: the format is fine, the file is not.
	_, err := Process(context.Background(), b.Bytes()[:b.Len()/3], Options{})
	if !errors.Is(err, ErrUnreadable) {
		t.Fatalf("err = %v, want ErrUnreadable", err)
	}
	if code := ErrorCode(err); code != CodeUnreadable {
		t.Fatalf("ErrorCode = %q, want %q", code, CodeUnreadable)
	}
}

func TestErrorCode(t *testing.T) {
	for err, want := range map[error]string{
		ErrUnsupported:             CodeUnsupported,
		ErrTooLarge:                CodeTooLarge,
		ErrDimensions:              CodeDimensions,
		errors.New("disk on fire"): "",
	} {
		if got := ErrorCode(err); got != want {
			t.Errorf("ErrorCode(%v) = %q, want %q", err, got, want)
		}
	}
}
