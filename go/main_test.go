package thumbhash

import (
	"bytes"
	"image"
	"testing"

	_ "embed"
	_ "image/png"
)

var (
	//go:embed img.png
	sample []byte

	rgba []byte
	h, w int
)

func init() {
	img, _, err := image.Decode(bytes.NewReader(sample))
	if err != nil {
		panic(err)
	}

	bounds := img.Bounds()
	w = bounds.Dx()
	h = bounds.Dy()
	rgba = make([]byte, w*h*4)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			i := (y*w + x) * 4
			rgba[i] = byte(r >> 8)
			rgba[i+1] = byte(g >> 8)
			rgba[i+2] = byte(b >> 8)
			rgba[i+3] = byte(a >> 8)
		}
	}
}

func TestRGBAToThumbhash(t *testing.T) {
	h, err := RGBAToThumbhash(w, h, rgba)
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte{0x2C, 0xF7, 0x05, 0x1F, 0x88, 0xF7, 0xB8, 0x77, 0x77, 0x66, 0xA9, 0x8D, 0x65, 0xC7, 0x67, 0xBA, 0x87, 0x84, 0x79, 0x80, 0x28, 0x08, 0x88, 0x02}

	if !bytes.Equal(h, expected) {
		t.Fatalf("unexpected hash: %x", h)
	}

}
