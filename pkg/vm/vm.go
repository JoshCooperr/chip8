package vm

import (
	"fmt"
	"io/ioutil"
	"math/rand"

	"github.com/JoshCooperr/chip8/pkg/display"
)

type VM struct {
	// The current opcode being emulated
	opcode uint16
	// Direct access memory (4kb RAM)
	memory [4096]byte
	// Programme counter
	pc uint16
	// Index register
	index uint16
	// Stack for 16-bit addresses, used to call subroutines/functions and return
	stack [16]uint16
	// Stack pointer
	sp uint16
	// Delay timer, decremented at 60Hz -> 0
	delayTimer uint8
	// Sound timer, decremented at 60Hz -> 0, plays sound if not at 0
	soundTimer uint8
	// Variable registers, 16 general purpose 8-bit registers numbered [0-F]
	variables [16]uint8
	// Flag register, used by instructions (e.g. as a carry flag)
	vf uint8
	// Interface to use to draw the game window
	display *display.Display
	// Current state of the display
	pixels [64][32]byte
}

func (vm *VM) Init(display display.Display) error {
	vm.display = &display
	vm.pc = 0x200
	return nil
}

func (vm *VM) executeCycle() {
	// Fetch next opcode by combining the two successive bytes indicated by the PC.
	// The first byte must be shifted left 8 (eg. 10100110 -> 1010011000000000)
	// then OR'd with the following byte to retrieve the opcode
	vm.opcode = uint16(vm.memory[vm.pc])<<8 | uint16(vm.memory[vm.pc+1])
	vm.pc += 2

	// Extract the various nibbles (half bytes) from the opcode
	instr := vm.opcode & 0xF000  // 1st nibble, the type of instruction
	x := vm.opcode & 0x0F00 >> 8 // 2nd nibble, used to look up a register (vx) in variables
	y := vm.opcode & 0x00F0 >> 4 // 3rd nibble, used to look up a register (vy) in variables
	n := vm.opcode & 0x000F      // 4th nibble, a 4-bit number
	nn := vm.opcode & 0x00FF     // 2nd byte, an 8-bit number
	nnn := vm.opcode & 0x0FFF    // 2nd, 3rd & 4th nibbles, a 12-bit memory address

	switch instr {
	case 0x0000:
		switch vm.opcode & 0x00FF {
		case 0x00E0:
			// Clear the screen
			vm.pixels = [64][32]byte{}
		case 0x00EE:
			// Return from a subroutine, pop address from stack and assign to PC
			vm.pc = vm.stack[vm.sp]
			vm.sp -= 1
		}

	case 0x1000:
		// Jump by setting PC to nnn
		vm.pc = nnn

	case 0x2000:
		// Call the subroutine at nnn in memory, set PC to this after saving current value to
		// the stack so the subroutine can return later
		vm.sp += 1
		vm.stack[vm.sp] = vm.pc
		vm.pc = nnn

	case 0x3000:
		// Skip the next instruction if the value in register vx == nn
		if vm.variables[x] == uint8(nn) {
			vm.pc += 2
		}

	case 0x4000:
		// Skip the next instruction if the value in register vx != nn
		if vm.variables[x] != uint8(nn) {
			vm.pc += 2
		}

	case 0x5000:
		// Skip the next instruction if the values in registers vx == vy
		if vm.variables[x] == vm.variables[y] {
			vm.pc += 2
		}

	case 0x6000:
		// Set register vx to the value in nn
		vm.variables[x] = uint8(nn)

	case 0x7000:
		// Add to register vx the value in nn
		vm.variables[x] += uint8(nn)

	case 0x8000:
		// Bitwise operations
		switch vm.opcode & 0x000F {
		case 0x0000:
			// Set register vx = vy
			vm.variables[x] = vm.variables[y]
		case 0x0001:
			// Set register vx = vx OR vy
			vm.variables[x] = vm.variables[x] | vm.variables[y]
		case 0x0002:
			// Set register vx = vx AND vy
			vm.variables[x] = vm.variables[x] & vm.variables[y]
		case 0x0003:
			// Set register vx = vx XOR vy
			vm.variables[x] = vm.variables[x] ^ vm.variables[y]
		case 0x0004:
			// Set register vx = vx + vy
			vm.variables[x] = vm.variables[x] + vm.variables[y]
		case 0x0005:
			// Set register vx = vx - vy
			vm.variables[x] = vm.variables[x] - vm.variables[y]
		case 0x0006:
			// Set register vx = vy > 1 (if bit shifted out was 1 then set vf = 1)
			if vm.variables[y]&0x01 == 0x1 {
				vm.vf = 1
			}
			vm.variables[x] = vm.variables[y] >> 1
		case 0x0007:
			// Set register vx = vy - vx
			vm.variables[x] = vm.variables[y] - vm.variables[x]
		case 0x000E:
			// Set register vx = vy < 1 (if bit shifted out was 1 then set vf = 1)
			if vm.variables[y]&0x80 == 0x80 {
				vm.vf = 1
			}
			vm.variables[x] = vm.variables[y] << 1
		}

	case 0x9000:
		// Skip the next instruction if the values in registers vx != vy
		if vm.variables[x] != vm.variables[y] {
			vm.pc += 2
		}

	case 0xA000:
		// Set the index register to the value in nnn
		vm.index = nnn

	case 0xB000:
		// TODO: Make configurable see (https://tobiasvl.github.io/blog/write-a-chip-8-emulator/#bnnn-jump-with-offset)
		panic(fmt.Errorf("not implemented: %v", vm.opcode))

	case 0xC000:
		// Generate a random number, r, and set register vx = r AND nn
		r := uint16(rand.Uint32())
		vm.variables[x] = uint8(r & nn)

	case 0xD000:
		// Get the x, y coords from the vx, vy registers as the starting coordinates to draw the
		// sprite from (these coordinates wrap, hence bitwise AND)
		xcoord := vm.variables[x] & 63
		ycoord := vm.variables[y] & 31
		vm.vf = 0
		for y := uint16(0); y < n; y++ {
			spriteRow := vm.memory[vm.index+y]
			for x := 0; x < 8; x++ {
				// Iterate over the bits of the sprite byte
				if (spriteRow & (0x80 >> x)) != 0 {
					if vm.pixels[xcoord+uint8(x)][ycoord+uint8(y)] == 0xFF {
						// Set register vf if a pixel is turned ON -> OFF
						vm.vf = 1
					}
					vm.pixels[xcoord+uint8(x)][ycoord+uint8(y)] ^= 0xFF // XOR display pixel with sprite
				}
			}
		}
		vm.display.Render(vm.pixels)

	case 0xF000:
		// Timer manipulation
		switch vm.opcode & 0x00FF {
		case 0x0007:
			// Set vx to the value of the delay timer
			vm.variables[x] = vm.delayTimer
		case 0x0015:
			// Set delay timer to value in vx
			vm.delayTimer = vm.variables[x]
		case 0x0018:
			// Set sound timer to value in vx
			vm.soundTimer = vm.variables[x]
		case 0x001E:
			// Add the value in vx to the index register
			vm.index += uint16(vm.variables[x])
		case 0x000A:
			// Block and wait for key press. If key is pressed then set vx to its hex value
			panic(fmt.Errorf("not implemented: %x", vm.opcode))
		case 0x0029:
			// Font character
			panic(fmt.Errorf("not implemented: %x", vm.opcode))
		case 0x0033:
			// Binary-coded decimal conversion, get the value in vx and convert to 3 decimal digits
			// (eg. 156 -> 1, 5, 6) and store in memory (addresses determined by index register)

			panic(fmt.Errorf("not implemented: %x", vm.opcode))
		case 0x0055:
			// Save the values in the variable registers into memory (addresses determined by index register)
			for i := 0; i < len(vm.variables); i++ {
				vm.memory[vm.index+uint16(i)] = vm.variables[i]
			}
		case 0x0065:
			// Load values from memory (addresses determined by index register) into the variable registers
			for i := 0; i < len(vm.variables); i++ {
				vm.variables[i] = vm.memory[vm.index+uint16(i)]
			}
		}

	}
}

func (vm *VM) LoadROM(filename string) error {
	// This function loads a given ROM, from the provided filepath, into the memory of the VM
	bytes, err := ioutil.ReadFile(filename)

	if err != nil {
		return err
	}

	// Sanity check the size of the ROM
	if len(bytes) > 4096 {
		return fmt.Errorf("the size of the ROM (%v) exceeds the 4096 byte limit", len(bytes))
	}

	// First 512 bytes of memory are reserved for the CHIP-8 interpreter
	for i, b := range bytes {
		vm.memory[i+512] = b
	}

	fmt.Printf("ROM loaded successfully, size: %v bytes\n", len(bytes))
	return nil
}

func (vm *VM) Run() {
	for {
		vm.executeCycle()
	}
}
