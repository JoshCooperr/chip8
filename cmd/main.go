package main

import (
	"math/rand"
	"time"

	"github.com/JoshCooperr/chip8/pkg/display"
	"github.com/faiface/pixel/pixelgl"
)

func RandBool() bool {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(2) == 1
}

func test() {
	display, err := display.NewDisplay()
	if err != nil {
		panic(err)
	}
	for !display.Closed() {
		pixels := make([]byte, 64*32)
		for i := 0; i < len(pixels); i++ {
			if RandBool() {
				pixels[i] = 0xFF
			}
		}
		time.Sleep(2 * time.Second)
		display.Render(pixels)
	}
}

func main() {
	pixelgl.Run(test)
}
