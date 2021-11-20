package main

import (
	"math/rand"
	"time"

	"github.com/JoshCooperr/chip8/pkg/display"
	"github.com/JoshCooperr/chip8/pkg/vm"
	"github.com/faiface/pixel/pixelgl"
)

func RandBool() bool {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(2) == 1
}

// func testDisplay() {
// 	display, err := display.NewDisplay()
// 	if err != nil {
// 		panic(err)
// 	}
// 	for !display.Closed() {
// 		pixels := make([]byte, 64*32)
// 		for i := 0; i < len(pixels); i++ {
// 			if RandBool() {
// 				pixels[i] = 0xFF
// 			}
// 		}
// 		time.Sleep(2 * time.Second)
// 		display.Render(pixels)
// 	}
// }

func test() {
	display, err := display.NewDisplay()
	if err != nil {
		panic(err)
	}
	vm := &vm.VM{}
	vm.Init(*display)
	vm.LoadROM("roms/test_opcode.ch8")
	vm.Run()
}

func main() {
	pixelgl.Run(test)
}
