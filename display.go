package main

import (
	"github.com/veandco/go-sdl2/sdl"
)

func initDisplay() (*sdl.Window, *sdl.Renderer, *sdl.Texture) {
	window, _ := sdl.CreateWindow("Raycaster", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, screenWidth, screenHeight, sdl.WINDOW_SHOWN)
	renderer, _ := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	texture, _ := renderer.CreateTexture(uint32(sdl.PIXELFORMAT_ARGB8888), sdl.TEXTUREACCESS_STREAMING, screenWidth, screenHeight)
	return window, renderer, texture
}
