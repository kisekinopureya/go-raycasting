package main

import (
	"math"

	"github.com/veandco/go-sdl2/sdl"
)

type KeyBindings struct {
	Forward   sdl.Scancode
	Backward  sdl.Scancode
	TurnLeft  sdl.Scancode
	TurnRight sdl.Scancode
}

func DefaultKeyBindings() KeyBindings {
	return KeyBindings{
		Forward:   sdl.SCANCODE_W,
		Backward:  sdl.SCANCODE_S,
		TurnLeft:  sdl.SCANCODE_A,
		TurnRight: sdl.SCANCODE_D,
	}
}

func handleInput(keys []uint8, frameTime float64, isWalkable func(float64, float64) bool, bindings KeyBindings) {
	moveSpeed := frameTime * 5.0
	rotSpeed := frameTime * 3.0

	// Movement
	if keys[bindings.Forward] != 0 {
		if isWalkable(posX+dirX*moveSpeed, posY) {
			posX += dirX * moveSpeed
		}
		if isWalkable(posX, posY+dirY*moveSpeed) {
			posY += dirY * moveSpeed
		}
	}

	if keys[bindings.Backward] != 0 {
		if isWalkable(posX-dirX*moveSpeed, posY) {
			posX -= dirX * moveSpeed
		}
		if isWalkable(posX, posY-dirY*moveSpeed) {
			posY -= dirY * moveSpeed
		}
	}

	// Rotation
	if keys[bindings.TurnRight] != 0 {
		rotateCamera(rotSpeed)
	}

	if keys[bindings.TurnLeft] != 0 {
		rotateCamera(-rotSpeed)
	}
}

func rotateCamera(angle float64) {
	oldDirX := dirX
	dirX = dirX*math.Cos(angle) - dirY*math.Sin(angle)
	dirY = oldDirX*math.Sin(angle) + dirY*math.Cos(angle)
	oldPlaneX := planeX
	planeX = planeX*math.Cos(angle) - planeY*math.Sin(angle)
	planeY = oldPlaneX*math.Sin(angle) + planeY*math.Cos(angle)
}
