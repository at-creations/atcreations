package media

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"

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

func TestProcessOpaqueBecomesJPEG(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for x := range 40 {
		for y := range 20 {
			img.Set(x, y, color.RGBA{R: 200, A: 255})
		}
	}
	p, err := Process(context.Background(), bytes.NewReader(encodePNG(t, img)))
	if err != nil {
		t.Fatal(err)
	}
	if p.ContentType != "image/jpeg" || p.Width != 40 || p.Height != 20 {
		t.Fatalf("unexpected result: %s %dx%d", p.ContentType, p.Width, p.Height)
	}
	if _, err := jpeg.Decode(bytes.NewReader(p.Data)); err != nil {
		t.Fatalf("output is not a valid JPEG: %v", err)
	}
}

func TestProcessTransparentStaysPNG(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	p, err := Process(context.Background(), bytes.NewReader(encodePNG(t, img)))
	if err != nil {
		t.Fatal(err)
	}
	if p.ContentType != "image/png" {
		t.Fatalf("got %s, want image/png", p.ContentType)
	}
}

func TestProcessDownscales(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 4800, 1200))
	p, err := Process(context.Background(), bytes.NewReader(encodePNG(t, img)))
	if err != nil {
		t.Fatal(err)
	}
	if p.Width != 2400 || p.Height != 600 {
		t.Fatalf("got %dx%d, want 2400x600", p.Width, p.Height)
	}
}

func TestProcessRejectsNonImages(t *testing.T) {
	for _, body := range []string{"", "<svg xmlns='http://www.w3.org/2000/svg'/>", "GIF89a....", "<?php echo 1; ?>"} {
		_, err := Process(context.Background(), strings.NewReader(body))
		if !errors.Is(err, ErrUnsupported) {
			t.Errorf("Process(%q) err = %v, want ErrUnsupported", body, err)
		}
	}
}

func TestProcessRejectsOversizedHeader(t *testing.T) {
	// A tiny PNG claiming huge dimensions (decompression bomb) must be refused
	// from its header alone.
	img := image.NewGray(image.Rect(0, 0, 1, 1))
	data := encodePNG(t, img)
	// IHDR width/height live at bytes 16..24.
	copy(data[16:24], []byte{0x00, 0x00, 0x75, 0x30, 0x00, 0x00, 0x75, 0x30}) // 30000x30000
	_, err := Process(context.Background(), bytes.NewReader(data))
	var e *httpx.Error
	if !errors.As(err, &e) || err == nil {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestNewKeyShape(t *testing.T) {
	k := NewKey("products/abc/", "jpg")
	if !strings.HasPrefix(k, "products/abc/") || !strings.HasSuffix(k, ".jpg") || len(k) != len("products/abc/")+32+4 {
		t.Fatalf("unexpected key %q", k)
	}
}

func TestProcessRejects16BitBombs(t *testing.T) {
	// 16-bit images decode to 8 bytes/pixel: 5000x4000 = 160 MB, over the cap.
	img := image.NewNRGBA64(image.Rect(0, 0, 5000, 4000))
	if _, err := Process(context.Background(), bytes.NewReader(encodePNG(t, img))); !errors.Is(err, ErrDimensions) {
		t.Fatalf("err = %v, want ErrDimensions", err)
	}
}
