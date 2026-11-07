//go:build darwin || windows

package clipboard

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

// Touches the real clipboard, so it only runs when asked to
// (CROSSDESK_REAL_CLIPBOARD=1). The previous content is restored.
func TestSystemClipboard(t *testing.T) {
	if os.Getenv("CROSSDESK_REAL_CLIPBOARD") != "1" {
		t.Skip("set CROSSDESK_REAL_CLIPBOARD=1 to test the real clipboard")
	}
	saved, had := System.Read()
	defer func() {
		if had {
			System.Write(saved)
		}
	}()

	text := "Crossdesk ✓ tiếng Việt 🖱️"
	if err := System.Write(Content{Kind: Text, Data: []byte(text)}); err != nil {
		t.Fatal(err)
	}
	got, ok := System.Read()
	if !ok || got.Kind != Text || string(got.Data) != text {
		t.Fatalf("text round trip: %v %q", ok, got.Data)
	}

	img := image.NewNRGBA(image.Rect(0, 0, 4, 3))
	img.SetNRGBA(1, 1, color.NRGBA{200, 10, 30, 255})
	var buf bytes.Buffer
	png.Encode(&buf, img)
	if err := System.Write(Content{Kind: PNG, Data: buf.Bytes()}); err != nil {
		t.Fatal(err)
	}
	got, ok = System.Read()
	if !ok || got.Kind != PNG {
		t.Fatalf("image round trip: ok=%v kind=%v", ok, got.Kind)
	}
	back, err := png.Decode(bytes.NewReader(got.Data))
	if err != nil {
		t.Fatal(err)
	}
	if back.Bounds().Dx() != 4 || back.Bounds().Dy() != 3 {
		t.Fatalf("image size %v", back.Bounds())
	}
	if c := color.NRGBAModel.Convert(back.At(1, 1)).(color.NRGBA); c.R != 200 || c.G != 10 || c.B != 30 {
		t.Fatalf("pixel %v", c)
	}
}
