package media

import (
	"encoding/binary"
	"image"
	"image/draw"
)

// jpegOrientation returns the EXIF orientation (1-8) of a JPEG, or 1 when absent or unreadable.
// Phones store photos sideways and rely on this tag; decoders ignore it, so without it portrait
// photos come out rotated.
func jpegOrientation(raw []byte) int {
	if len(raw) < 4 || raw[0] != 0xFF || raw[1] != 0xD8 {
		return 1
	}
	i := 2
	for i+4 <= len(raw) {
		if raw[i] != 0xFF {
			return 1
		}
		marker := raw[i+1]
		if marker == 0xDA || marker == 0xD9 { // start of scan / end of image: no EXIF before pixels
			return 1
		}
		size := int(binary.BigEndian.Uint16(raw[i+2 : i+4]))
		if size < 2 || i+2+size > len(raw) {
			return 1
		}
		seg := raw[i+4 : i+2+size]
		if marker == 0xE1 && len(seg) > 14 && string(seg[:6]) == "Exif\x00\x00" {
			return tiffOrientation(seg[6:])
		}
		i += 2 + size
	}
	return 1
}

func tiffOrientation(t []byte) int {
	if len(t) < 8 {
		return 1
	}
	var bo binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 1
	}
	off := int(bo.Uint32(t[4:8]))
	if off < 8 || off+2 > len(t) {
		return 1
	}
	n := int(bo.Uint16(t[off : off+2]))
	for e := 0; e < n; e++ {
		p := off + 2 + e*12
		if p+12 > len(t) {
			return 1
		}
		if bo.Uint16(t[p:p+2]) == 0x0112 { // Orientation, SHORT
			if v := int(bo.Uint16(t[p+8 : p+10])); v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}

// applyOrientation rotates/flips img so it displays upright for EXIF orientation o.
func applyOrientation(img image.Image, o int) image.Image {
	if o <= 1 || o > 8 {
		return img
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	src := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(src, src.Bounds(), img, b.Min, draw.Src)

	dw, dh := w, h
	if o >= 5 { // 5-8 swap width and height
		dw, dh = h, w
	}
	dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy int
			switch o {
			case 2: // mirror horizontal
				dx, dy = w-1-x, y
			case 3: // rotate 180
				dx, dy = w-1-x, h-1-y
			case 4: // mirror vertical
				dx, dy = x, h-1-y
			case 5: // transpose
				dx, dy = y, x
			case 6: // rotate 90 clockwise
				dx, dy = h-1-y, x
			case 7: // transverse
				dx, dy = h-1-y, w-1-x
			case 8: // rotate 90 counter-clockwise
				dx, dy = y, w-1-x
			}
			si := src.PixOffset(x, y)
			di := dst.PixOffset(dx, dy)
			copy(dst.Pix[di:di+4], src.Pix[si:si+4])
		}
	}
	return dst
}
