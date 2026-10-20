package images

import (
	"image"
	"image/color"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
)

func TestVirtualPlacementAndChunks(t *testing.T) {
	pixels := image.NewRGBA(image.Rect(0, 0, 128, 128))
	for y := range 128 {
		for x := range 128 {
			pixels.SetRGBA(x, y, color.RGBA{byte(rand.IntN(256)), byte(rand.IntN(256)), byte(rand.IntN(256)), 255})
		}
	}
	output, err := Upload(pixels, 0x12345678, 7, 60, 30)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"a=T", "U=1", "q=2", "i=305419896", "p=7", "c=60", "r=30", "f=100", "m=1", "m=0"} {
		if !strings.Contains(output, field) {
			t.Fatal(field)
		}
	}
	for _, chunk := range strings.Split(output, "\x1b_G")[1:] {
		payload := strings.SplitN(chunk, ";", 2)
		if len(payload) != 2 || len(strings.TrimSuffix(payload[1], "\x1b\\")) > 4096 {
			t.Fatal("invalid chunk")
		}
	}
	lines := Placeholders(0x12345678, 7, 4, 3)
	expected := string([]rune{kitty.Placeholder, kitty.Diacritic(2), kitty.Diacritic(0), kitty.Diacritic(0x12)})
	if !strings.Contains(lines[2], expected) || ansi.StringWidth(lines[2]) != 4 || !strings.Contains(lines[2], "38;2;52;86;120") || !strings.Contains(lines[2], "58;2;0;0;7") {
		t.Fatal(lines)
	}
	clipped := ansi.Cut(lines[2], 1, 3)
	if !strings.Contains(clipped, string([]rune{kitty.Placeholder, kitty.Diacritic(2), kitty.Diacritic(1)})) {
		t.Fatal(clipped)
	}
	if Delete(42) != "\x1b_Ga=d,d=I,i=42,q=2\x1b\\" {
		t.Fatal(Delete(42))
	}
	if !strings.Contains(Query(42), "a=q,i=42") {
		t.Fatal(Query(42))
	}
}

func TestBlockAspectAndTransparency(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 1, 2))
	source.SetNRGBA(0, 0, color.NRGBA{255, 0, 0, 128})
	source.SetNRGBA(0, 1, color.NRGBA{0, 255, 0, 255})
	lines := Blocks(source, 1, 1)
	if len(lines) != 1 || !strings.Contains(lines[0], "38;2;128;0;0") || !strings.Contains(lines[0], "48;2;0;255;0") || ansi.StringWidth(lines[0]) != 1 {
		t.Fatal(lines)
	}
	if Rows(100, 100, 60, 0) != 30 || Rows(1, 8192, 60, 0.5) != 256 {
		t.Fatal("incorrect aspect")
	}
}

func TestTallImageSize(t *testing.T) {
	columns, rows := Size(100, 8000, 60, 0.5)
	if columns != 6 || rows != 240 {
		t.Fatal(columns, rows)
	}
}
