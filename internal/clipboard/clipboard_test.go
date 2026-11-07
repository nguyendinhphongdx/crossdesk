package clipboard

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestDIBRoundTrip(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	src.SetNRGBA(0, 0, color.NRGBA{255, 0, 0, 255})
	src.SetNRGBA(2, 1, color.NRGBA{0, 0, 255, 128})
	var buf bytes.Buffer
	png.Encode(&buf, src)

	dib, err := pngToDIB(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	back, err := dibToPNG(dib)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(back))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 3 || img.Bounds().Dy() != 2 {
		t.Fatalf("size %v", img.Bounds())
	}
	if c := color.NRGBAModel.Convert(img.At(0, 0)).(color.NRGBA); c != (color.NRGBA{255, 0, 0, 255}) {
		t.Fatalf("pixel (0,0) = %v", c)
	}
	if c := color.NRGBAModel.Convert(img.At(2, 1)).(color.NRGBA); c != (color.NRGBA{0, 0, 255, 128}) {
		t.Fatalf("pixel (2,1) = %v", c)
	}
}

func TestDIBOpaqueWhenAlphaUnused(t *testing.T) {
	// 1x1 32bpp DIB with alpha 0 (common for screenshots): must become opaque.
	dib := make([]byte, 44)
	dib[0], dib[4], dib[8], dib[12], dib[14] = 40, 1, 1, 1, 32
	dib[40], dib[41], dib[42], dib[43] = 10, 20, 30, 0
	out, err := dibToPNG(dib)
	if err != nil {
		t.Fatal(err)
	}
	img, _ := png.Decode(bytes.NewReader(out))
	if c := color.NRGBAModel.Convert(img.At(0, 0)).(color.NRGBA); c != (color.NRGBA{30, 20, 10, 255}) {
		t.Fatalf("got %v", c)
	}
}

func TestDIBRejectsGarbage(t *testing.T) {
	if _, err := dibToPNG([]byte{1, 2, 3}); err == nil {
		t.Fatal("expected error")
	}
	bad := make([]byte, 40)
	bad[0], bad[4], bad[8], bad[14] = 40, 100, 100, 32 // claims 100x100 with no pixels
	if _, err := dibToPNG(bad); err == nil {
		t.Fatal("expected truncated error")
	}
}
