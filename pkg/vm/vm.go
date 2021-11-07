package vm

import (
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
	pixels [64 * 32]byte
}

func (vm *VM) executeCycle() {
	// Fetch next opcode by combining the two successive bytes indicated by the PC.
	// The first byte must be shifted left 8 (eg. 10100110 -> 1010011000000000)
	// then OR'd with the following byte to retrieve the opcode
	vm.opcode = uint16(vm.memory[vm.pc])<<8 | uint16(vm.memory[vm.pc+1])
	vm.pc += 2

	// Extract the various nibbles (half bytes) from the opcode
	instr := vm.opcode & 0xF000 // 1st nibble, the type of instruction
	x := vm.opcode & 0x0F00     // 2nd nibble, used to look up a register (vx) in variables
	y := vm.opcode & 0x00F0     // 3rd nibble, used to look up a register (vy) in variables
	n := vm.opcode & 0x000F     // 4th nibble, a 4-bit number
	nn := vm.opcode & 0x00FF    // 2nd byte, an 8-bit number
	nnn := vm.opcode & 0x0FFF   // 2nd, 3rd & 4th nibbles, a 12-bit memory address

	switch instr {
	case 0x0000:
		switch vm.opcode & 0x00FF {
		case 0x00E0:
			// Clear the screen
			vm.pixels = [64 * 32]byte{}
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
		if vm.variables[x] == vm.variables[y] {
			vm.pc += 2
		}

	case 0xA000:
		// Set the index register to the value in nnn
		vm.index = nnn
	case 0xB000:
		// TODO: Make configurable see (https://tobiasvl.github.io/blog/write-a-chip-8-emulator/#bnnn-jump-with-offset)
	case 0xC000:
		// Generate a random number, r, and set register vx = r AND nn
		r := uint16(rand.Uint32())
		vm.variables[x] = uint8(r & nn)
	case 0xD000:
		// Get the x, y coords from the vx, vy registers as the starting coordinates to draw the
		// sprite from (these coordinates wrap, hence bitwise AND)
		xcoord := vm.variables[x] & 63
		ycoord := vm.variables[y] & 31
		memoryIndex := vm.index
		for i := uint16(0); i < n; i++ {
			spriteRow := vm.memory[memoryIndex+i]
			for b := 0; b < 8; b++ {
				pixelsIndex := (64 * ycoord) + xcoord
				if (spriteRow & 0x80) == 0x80 {
					// The sprite bit is a 1 -> draw the pixel
					if pixelsIndex%64 == 0 {
						// Stop drawing this row as the right edge of the screen has been reached
						break
					}
					if vm.pixels[pixelsIndex] == 0xFF {
						// Set register vf if a pixel is turned ON -> OFF
						vm.vf = 1
					}
					vm.pixels[pixelsIndex] ^= 1 // XOR display pixel with sprite
				}
				spriteRow = spriteRow << 1
				xcoord += 1
			}
			ycoord += 1
			if ycoord%32 == 0 {
				// Stop drawing if it hits the bottom of the screen
				break
			}
		}
	}
}
