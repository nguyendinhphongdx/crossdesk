package clipboard

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/png"
)

// Windows stores bitmaps on the clipboard as a DIB (BITMAPINFOHEADER +
// pixels). These helpers convert between DIB and PNG.

func dibToPNG(dib []byte) ([]byte, error) {
	if len(dib) < 40 {
		return nil, errors.New("short DIB")
	}
	hdr := binary.LittleEndian.Uint32(dib[0:])
	w := int(int32(binary.LittleEndian.Uint32(dib[4:])))
	h := int(int32(binary.LittleEndian.Uint32(dib[8:])))
	bpp := int(binary.LittleEndian.Uint16(dib[14:]))
	comp := binary.LittleEndian.Uint32(dib[16:])
	if hdr < 40 || w <= 0 || h == 0 || (bpp != 24 && bpp != 32) || (comp != 0 && comp != 3) {
		return nil, errors.New("unsupported DIB format")
	}
	topDown := h < 0
	if topDown {
		h = -h
	}
	off := int(hdr)
	if comp == 3 && hdr == 40 {
		off += 12 // BI_BITFIELDS masks follow a v1 header
	}
	stride := ((w*bpp + 31) / 32) * 4
	if w > 20000 || h > 20000 || off+stride*h > len(dib) {
		return nil, errors.New("truncated DIB")
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	bytesPP := bpp / 8
	anyAlpha := false
	for y := 0; y < h; y++ {
		srcY := h - 1 - y
		if topDown {
			srcY = y
		}
		row := dib[off+srcY*stride:]
		for x := 0; x < w; x++ {
			p := row[x*bytesPP:]
			a := uint8(255)
			if bpp == 32 {
				a = p[3]
				if a != 0 {
					anyAlpha = true
				}
			}
			img.SetNRGBA(x, y, color.NRGBA{R: p[2], G: p[1], B: p[0], A: a})
		}
	}
	if bpp == 32 && !anyAlpha { // alpha channel unused: make opaque
		for i := 3; i < len(img.Pix); i += 4 {
			img.Pix[i] = 255
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func pngToDIB(data []byte) ([]byte, error) {
	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	img := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(img, img.Bounds(), src, b.Min, draw.Src)
	w, h := b.Dx(), b.Dy()
	out := make([]byte, 40+w*h*4)
	binary.LittleEndian.PutUint32(out[0:], 40)
	binary.LittleEndian.PutUint32(out[4:], uint32(w))
	binary.LittleEndian.PutUint32(out[8:], uint32(h)) // bottom-up
	binary.LittleEndian.PutUint16(out[12:], 1)
	binary.LittleEndian.PutUint16(out[14:], 32)
	binary.LittleEndian.PutUint32(out[20:], uint32(w*h*4))
	px := out[40:]
	for y := 0; y < h; y++ {
		row := px[(h-1-y)*w*4:]
		for x := 0; x < w; x++ {
			c := img.NRGBAAt(x, y)
			row[x*4], row[x*4+1], row[x*4+2], row[x*4+3] = c.B, c.G, c.R, c.A
		}
	}
	return out, nil
}
