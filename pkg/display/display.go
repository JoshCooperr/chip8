package display

import (
	"image/color"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/imdraw"

	"github.com/faiface/pixel/pixelgl"
)

var FontSet = [80]byte{
	0xF0, 0x90, 0x90, 0x90, 0xF0, // 0
	0x20, 0x60, 0x20, 0x20, 0x70, // 1
	0xF0, 0x10, 0xF0, 0x80, 0xF0, // 2
	0xF0, 0x10, 0xF0, 0x10, 0xF0, // 3
	0x90, 0x90, 0xF0, 0x10, 0x10, // 4
	0xF0, 0x80, 0xF0, 0x10, 0xF0, // 5
	0xF0, 0x80, 0xF0, 0x90, 0xF0, // 6
	0xF0, 0x10, 0x20, 0x40, 0x40, // 7
	0xF0, 0x90, 0xF0, 0x90, 0xF0, // 8
	0xF0, 0x90, 0xF0, 0x10, 0xF0, // 9
	0xF0, 0x90, 0xF0, 0x90, 0x90, // A
	0xE0, 0x90, 0xe0, 0x90, 0xE0, // B
	0xF0, 0x80, 0x80, 0x80, 0x80, // C
	0xF0, 0x90, 0x90, 0x90, 0xE0, // D
	0xF0, 0x80, 0xF0, 0x80, 0xF0, // E
	0xF0, 0x80, 0xF0, 0x80, 0x80, // F
}

const (
	width     float64 = 64
	height    float64 = 32
	pixelSize float64 = 16
)

type Display struct {
	*pixelgl.Window
}

func NewDisplay() (*Display, error) {
	cfg := pixelgl.WindowConfig{
		Title:  "Chip8",
		Bounds: pixel.R(0, 0, width*pixelSize, height*pixelSize),
		VSync:  true,
	}
	win, err := pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}
	return &Display{
		win,
	}, nil
}

func (d *Display) Render(pixels [64][32]byte) {
	d.Clear(color.Black)
	imd := imdraw.New(nil)
	imd.Color = pixel.RGB(1, 1, 1)

	// Draw pixels from top left -> bottom right
	for x := 0; x < int(width); x++ {
		for y := 0; y < int(height); y++ {
			if pixels[x][31-y] != 0 {
				imd.Push(pixel.V(pixelSize*float64(x), pixelSize*float64(y)))
				imd.Push(pixel.V(pixelSize*float64(x)+pixelSize, pixelSize*float64(y)+pixelSize))
				imd.Rectangle(0)
			}
		}
	}

	imd.Draw(d)
	d.Update()
}
