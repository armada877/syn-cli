package images

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/charmbracelet/x/ansi/kitty"
	"golang.org/x/image/draw"
)

const MaxRows = 256

func Rows(width, height, columns int, cellAspect float64) int {
	if cellAspect <= 0 {
		cellAspect = 0.5
	}
	return max(1, min(MaxRows, int(math.Ceil(float64(height)*float64(columns)*cellAspect/float64(max(1, width))))))
}

func Size(width, height, columns int, cellAspect float64) (int, int) {
	if cellAspect <= 0 {
		cellAspect = 0.5
	}
	ratio := float64(height) * cellAspect / float64(max(1, width))
	if ratio*float64(columns) > MaxRows {
		columns = max(1, int(float64(MaxRows)/ratio))
	}
	return columns, Rows(width, height, columns, cellAspect)
}

func Scale(source image.Image, width, height int) *image.RGBA {
	target := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(target, target.Bounds(), source, source.Bounds(), draw.Src, nil)
	return target
}

func Blocks(source image.Image, columns, rows int) []string {
	scaled := Scale(source, columns, rows*2)
	lines := make([]string, rows)
	for y := range rows {
		var out strings.Builder
		for x := range columns {
			top := color.RGBAModel.Convert(scaled.At(x, y*2)).(color.RGBA)
			bottom := color.RGBAModel.Convert(scaled.At(x, y*2+1)).(color.RGBA)
			fmt.Fprintf(&out, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀", top.R, top.G, top.B, bottom.R, bottom.G, bottom.B)
		}
		out.WriteString("\x1b[0m")
		lines[y] = out.String()
	}
	return lines
}

func Upload(source image.Image, id, placement, columns, rows int) (string, error) {
	var out bytes.Buffer
	err := kitty.EncodeGraphics(&out, source, &kitty.Options{Transmission: kitty.Direct, Action: kitty.TransmitAndPut, ID: id, PlacementID: placement, Format: kitty.PNG, Quiet: 2, VirtualPlacement: true, Columns: columns, Rows: rows, Chunk: true})
	return out.String(), err
}

func Placeholders(id, placement, columns, rows int) []string {
	columns, rows = min(256, max(1, columns)), min(MaxRows, max(1, rows))
	lines := make([]string, rows)
	for row := range rows {
		var out strings.Builder
		fmt.Fprintf(&out, "\x1b[38;2;%d;%d;%dm\x1b[58;2;%d;%d;%dm", (id>>16)&255, (id>>8)&255, id&255, (placement>>16)&255, (placement>>8)&255, placement&255)
		for col := range columns {
			out.WriteRune(kitty.Placeholder)
			out.WriteRune(kitty.Diacritic(row))
			out.WriteRune(kitty.Diacritic(col))
			out.WriteRune(kitty.Diacritic((id >> 24) & 255))
		}
		out.WriteString("\x1b[39;59m")
		lines[row] = out.String()
	}
	return lines
}

func Delete(id int) string { return fmt.Sprintf("\x1b_Ga=d,d=I,i=%d,q=2\x1b\\", id) }
func Query(id int) string  { return fmt.Sprintf("\x1b_Ga=q,i=%d,s=1,v=1,f=24;AAAA\x1b\\", id) }
