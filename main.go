// main.go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"kisekinopureya.com.tr/go-raycasting/worldMap"

	"github.com/veandco/go-sdl2/sdl"
)

const (
	screenWidth  = 640
	screenHeight = 480
	texWidth     = 64
	texHeight    = 64
)

var posX, posY = 3.0, 3.0
var dirX, dirY = -1.0, 0.0
var planeX, planeY = 0.0, -0.66

type appMode int

const (
	modeMenu appMode = iota
	modePlaying
)

type menuLevel struct {
	path    string
	name    string
	author  string
	desc    string
	version string
}

type menuScreen int

const (
	screenMain menuScreen = iota
	screenOptions
)

type menuAction int

const (
	menuStart menuAction = iota
	menuOptions
	menuRefresh
	menuExit
)

var menuLabels = []string{
	"START GAME",
	"OPTIONS",
	"REFRESH MAP LIST",
	"EXIT",
}

type optionsAction int

const (
	optRebindForward optionsAction = iota
	optRebindBackward
	optRebindLeft
	optRebindRight
	optResolution
	optBack
)

var optionLabels = []string{
	"REMAP FORWARD",
	"REMAP BACKWARD",
	"REMAP TURN LEFT",
	"REMAP TURN RIGHT",
	"RESOLUTION",
	"BACK",
}

type resolutionPreset struct {
	width  int
	height int
	label  string
}

var resolutionPresets = []resolutionPreset{
	{640, 480, "640X480"},
	{800, 600, "800X600"},
	{1024, 768, "1024X768"},
	{1440, 1080, "1440X1080"},
}

func render(renderer *sdl.Renderer, texture *sdl.Texture, pixels []uint32) {
	texture.Update(nil, unsafe.Pointer(&pixels[0]), screenWidth*4)
	renderer.Clear()
	renderer.Copy(texture, nil, nil)
	renderer.Present()
}

func discoverLevelPackages() []menuLevel {
	levelPaths, err := filepath.Glob("levels/*.level")
	if err != nil {
		return nil
	}
	if len(levelPaths) == 0 {
		if _, err := os.Stat(worldMap.DefaultPackagePath); err == nil {
			levelPaths = []string{worldMap.DefaultPackagePath}
		}
	}

	levels := make([]menuLevel, 0, len(levelPaths))
	for _, path := range levelPaths {
		fallbackName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		metadata, err := worldMap.LoadLevelPackageMetadata(path)
		name := fallbackName
		if err == nil && strings.TrimSpace(metadata.Name) != "" {
			name = metadata.Name
		}
		levels = append(levels, menuLevel{
			path:    path,
			name:    name,
			author:  metadata.Author,
			desc:    metadata.Description,
			version: metadata.Version,
		})
	}

	return levels
}

func menuTitle(levels []menuLevel, selected int, cursor int, screen menuScreen) string {
	if len(levels) == 0 {
		return "Go Raycasting - Menu (no maps)"
	}
	name := levels[selected].name
	label := ""
	if screen == screenMain && cursor >= 0 && cursor < len(menuLabels) {
		label = menuLabels[cursor]
	} else if screen == screenOptions && cursor >= 0 && cursor < len(optionLabels) {
		label = optionLabels[cursor]
	}
	return fmt.Sprintf("Go Raycasting - Map: %s | Option: %s", name, label)
}

func drawMenuFrame(
	pixels []uint32,
	levels []menuLevel,
	selected int,
	screen menuScreen,
	mainCursor int,
	optionsCursor int,
	bindings KeyBindings,
	resolutionIdx int,
	fpsIdx int,
	rebinding string,
) {
	bg := uint32(0xFF1B1B1B)
	for i := range pixels {
		pixels[i] = bg
	}

	drawText(pixels, 20, 20, 2, "GO RAYCASTING", 0xFFE8E8E8)
	if screen == screenMain {
		drawText(pixels, 20, 48, 1, "UP/DOWN: OPTION   LEFT/RIGHT: MAP   ENTER: CONFIRM   ESC: QUIT", 0xFFB8B8B8)
	} else {
		drawText(pixels, 20, 48, 1, "UP/DOWN: OPTION   ENTER/LEFT/RIGHT: CHANGE   ESC: BACK", 0xFFB8B8B8)
	}

	if len(levels) == 0 {
		fillRect(pixels, 120, 170, 400, 140, 0xFF3B1F1F)
		drawText(pixels, 138, 214, 2, "NO LEVELS FOUND", 0xFFFFD0D0)
		drawText(pixels, 138, 244, 1, "PLACE .LEVEL FILES IN levels/", 0xFFFFD0D0)
	} else {
		fillRect(pixels, 60, 96, screenWidth-120, 80, 0xFF2A3650)
		drawText(pixels, 74, 108, 2, "CURRENT MAP", 0xFFDDE8FF)
		drawText(pixels, 74, 126, 2, strings.ToUpper(levels[selected].name), 0xFFFFE7B0)

		authorText := "AUTHOR: " + strings.ToUpper(levels[selected].author)
		if strings.TrimSpace(levels[selected].author) == "" {
			authorText = "AUTHOR: UNKNOWN"
		}
		drawText(pixels, 74, 144, 1, authorText, 0xFFB8D8FF)

		descText := strings.ToUpper(levels[selected].desc)
		if len(descText) > 40 {
			descText = descText[:40] + "..."
		}
		drawText(pixels, 74, 158, 1, descText, 0xFFB8D8FF)

		versionText := "V" + strings.ToUpper(levels[selected].version)
		if strings.TrimSpace(levels[selected].version) == "" {
			versionText = "V1.0"
		}
		drawText(pixels, screenWidth-160, 144, 1, versionText, 0xFFB8D8FF)
	}

	startY := 178
	rowHeight := 46
	labels := menuLabels
	cursor := mainCursor
	if screen == screenOptions {
		labels = optionLabels
		cursor = optionsCursor
	}

	for i, label := range labels {
		y0 := startY + i*rowHeight
		color := uint32(0xFF2B3F2B)
		if i == cursor {
			color = 0xFF7A5A20
		}
		fillRect(pixels, 120, y0, 400, 36, color)
		prefix := "  "
		textColor := uint32(0xFFD8E8D8)
		if i == cursor {
			prefix = "> "
			textColor = 0xFFFFE7B0
		}

		rightText := ""
		if screen == screenOptions {
			switch optionsAction(i) {
			case optRebindForward:
				rightText = strings.ToUpper(sdl.GetScancodeName(bindings.Forward))
			case optRebindBackward:
				rightText = strings.ToUpper(sdl.GetScancodeName(bindings.Backward))
			case optRebindLeft:
				rightText = strings.ToUpper(sdl.GetScancodeName(bindings.TurnLeft))
			case optRebindRight:
				rightText = strings.ToUpper(sdl.GetScancodeName(bindings.TurnRight))
			case optResolution:
				rightText = resolutionPresets[resolutionIdx].label
			}
		}

		drawText(pixels, 136, y0+10, 2, prefix+label, textColor)
		if rightText != "" {
			drawText(pixels, 418, y0+10, 2, rightText, textColor)
		}
	}

	if screen == screenOptions && rebinding != "" {
		fillRect(pixels, 74, 402, 492, 54, 0xFF4C2A2A)
		drawText(pixels, 90, 420, 2, "PRESS A KEY FOR "+rebinding, 0xFFFFD0D0)
	}
}

func fillRect(pixels []uint32, x, y, w, h int, color uint32) {
	if w <= 0 || h <= 0 {
		return
	}

	if x < 0 {
		w += x
		x = 0
	}
	if y < 0 {
		h += y
		y = 0
	}
	if x >= screenWidth || y >= screenHeight {
		return
	}
	if x+w > screenWidth {
		w = screenWidth - x
	}
	if y+h > screenHeight {
		h = screenHeight - y
	}
	if w <= 0 || h <= 0 {
		return
	}

	for yy := y; yy < y+h; yy++ {
		row := yy * screenWidth
		for xx := x; xx < x+w; xx++ {
			pixels[row+xx] = color
		}
	}
}

var tinyFont = map[rune][7]uint8{
	' ': {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	'-': {0x00, 0x00, 0x00, 0x1F, 0x00, 0x00, 0x00},
	'_': {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x1F},
	'.': {0x00, 0x00, 0x00, 0x00, 0x00, 0x06, 0x06},
	'/': {0x01, 0x02, 0x04, 0x08, 0x10, 0x00, 0x00},
	':': {0x00, 0x06, 0x06, 0x00, 0x06, 0x06, 0x00},
	'>': {0x10, 0x08, 0x04, 0x02, 0x04, 0x08, 0x10},
	'0': {0x0E, 0x11, 0x13, 0x15, 0x19, 0x11, 0x0E},
	'1': {0x04, 0x0C, 0x04, 0x04, 0x04, 0x04, 0x0E},
	'2': {0x0E, 0x11, 0x01, 0x02, 0x04, 0x08, 0x1F},
	'3': {0x1E, 0x01, 0x01, 0x0E, 0x01, 0x01, 0x1E},
	'4': {0x02, 0x06, 0x0A, 0x12, 0x1F, 0x02, 0x02},
	'5': {0x1F, 0x10, 0x10, 0x1E, 0x01, 0x01, 0x1E},
	'6': {0x06, 0x08, 0x10, 0x1E, 0x11, 0x11, 0x0E},
	'7': {0x1F, 0x01, 0x02, 0x04, 0x08, 0x08, 0x08},
	'8': {0x0E, 0x11, 0x11, 0x0E, 0x11, 0x11, 0x0E},
	'9': {0x0E, 0x11, 0x11, 0x0F, 0x01, 0x02, 0x0C},
	'A': {0x0E, 0x11, 0x11, 0x1F, 0x11, 0x11, 0x11},
	'B': {0x1E, 0x11, 0x11, 0x1E, 0x11, 0x11, 0x1E},
	'C': {0x0E, 0x11, 0x10, 0x10, 0x10, 0x11, 0x0E},
	'D': {0x1C, 0x12, 0x11, 0x11, 0x11, 0x12, 0x1C},
	'E': {0x1F, 0x10, 0x10, 0x1E, 0x10, 0x10, 0x1F},
	'F': {0x1F, 0x10, 0x10, 0x1E, 0x10, 0x10, 0x10},
	'G': {0x0E, 0x11, 0x10, 0x17, 0x11, 0x11, 0x0E},
	'H': {0x11, 0x11, 0x11, 0x1F, 0x11, 0x11, 0x11},
	'I': {0x0E, 0x04, 0x04, 0x04, 0x04, 0x04, 0x0E},
	'J': {0x01, 0x01, 0x01, 0x01, 0x11, 0x11, 0x0E},
	'K': {0x11, 0x12, 0x14, 0x18, 0x14, 0x12, 0x11},
	'L': {0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x1F},
	'M': {0x11, 0x1B, 0x15, 0x15, 0x11, 0x11, 0x11},
	'N': {0x11, 0x11, 0x19, 0x15, 0x13, 0x11, 0x11},
	'O': {0x0E, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0E},
	'P': {0x1E, 0x11, 0x11, 0x1E, 0x10, 0x10, 0x10},
	'Q': {0x0E, 0x11, 0x11, 0x11, 0x15, 0x12, 0x0D},
	'R': {0x1E, 0x11, 0x11, 0x1E, 0x14, 0x12, 0x11},
	'S': {0x0F, 0x10, 0x10, 0x0E, 0x01, 0x01, 0x1E},
	'T': {0x1F, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04},
	'U': {0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0E},
	'V': {0x11, 0x11, 0x11, 0x11, 0x11, 0x0A, 0x04},
	'W': {0x11, 0x11, 0x11, 0x15, 0x15, 0x15, 0x0A},
	'X': {0x11, 0x11, 0x0A, 0x04, 0x0A, 0x11, 0x11},
	'Y': {0x11, 0x11, 0x0A, 0x04, 0x04, 0x04, 0x04},
	'Z': {0x1F, 0x01, 0x02, 0x04, 0x08, 0x10, 0x1F},
}

func drawText(pixels []uint32, x, y, scale int, text string, color uint32) {
	px := x
	for _, ch := range text {
		glyph, ok := tinyFont[ch]
		if !ok {
			glyph = [7]uint8{0x1F, 0x11, 0x11, 0x11, 0x11, 0x11, 0x1F}
		}
		drawGlyph(pixels, px, y, scale, glyph, color)
		px += 6 * scale
	}
}

func drawGlyph(pixels []uint32, x, y, scale int, glyph [7]uint8, color uint32) {
	for row := 0; row < 7; row++ {
		bits := glyph[row]
		for col := 0; col < 5; col++ {
			if (bits & (1 << (4 - col))) == 0 {
				continue
			}
			for sy := 0; sy < scale; sy++ {
				for sx := 0; sx < scale; sx++ {
					px := x + col*scale + sx
					py := y + row*scale + sy
					if px < 0 || px >= screenWidth || py < 0 || py >= screenHeight {
						continue
					}
					pixels[py*screenWidth+px] = color
				}
			}
		}
	}
}

func resetPlayerTransform() {
	posX, posY = 3.0, 3.0
	dirX, dirY = -1.0, 0.0
	planeX, planeY = 0.0, -0.66
}

func main() {
	sdl.Init(uint32(sdl.INIT_VIDEO))
	defer sdl.Quit()

	window, renderer, texture := initDisplay()
	defer window.Destroy()
	defer renderer.Destroy()
	defer texture.Destroy()

	levels := discoverLevelPackages()
	selectedLevel := 0
	menuCursor := 0
	optionsCursor := 0
	activeScreen := screenMain
	bindings := DefaultKeyBindings()
	resolutionIdx := 0
	fpsIdx := 2
	rebinding := ""
	mode := modeMenu
	menuPixels := make([]uint32, screenWidth*screenHeight)
	var state *GameState
	var isWalkableAt func(float64, float64) bool

	window.SetTitle(menuTitle(levels, selectedLevel, menuCursor, activeScreen))

	running := true
	var time, oldTime float64
	for running {
		oldTime = time
		time = float64(sdl.GetTicks64())
		frameTime := (time - oldTime) / 1000.0

		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			if _, ok := event.(*sdl.QuitEvent); ok {
				running = false
			}

			if keyEvent, ok := event.(*sdl.KeyboardEvent); ok && keyEvent.Type == sdl.KEYDOWN && keyEvent.Repeat == 0 {
				switch mode {
				case modeMenu:
					if rebinding != "" {
						sc := keyEvent.Keysym.Scancode
						switch rebinding {
						case "FORWARD":
							bindings.Forward = sc
						case "BACKWARD":
							bindings.Backward = sc
						case "TURN LEFT":
							bindings.TurnLeft = sc
						case "TURN RIGHT":
							bindings.TurnRight = sc
						}
						rebinding = ""
						window.SetTitle(menuTitle(levels, selectedLevel, menuCursor, activeScreen))
						continue
					}

					switch keyEvent.Keysym.Sym {
					case sdl.K_ESCAPE:
						if activeScreen == screenOptions {
							activeScreen = screenMain
							window.SetTitle(menuTitle(levels, selectedLevel, menuCursor, activeScreen))
						} else {
							running = false
						}
					case sdl.K_UP:
						if activeScreen == screenMain {
							menuCursor--
							if menuCursor < 0 {
								menuCursor = len(menuLabels) - 1
							}
						} else {
							optionsCursor--
							if optionsCursor < 0 {
								optionsCursor = len(optionLabels) - 1
							}
						}
						window.SetTitle(menuTitle(levels, selectedLevel, menuCursor, activeScreen))
					case sdl.K_DOWN:
						if activeScreen == screenMain {
							menuCursor++
							if menuCursor >= len(menuLabels) {
								menuCursor = 0
							}
						} else {
							optionsCursor++
							if optionsCursor >= len(optionLabels) {
								optionsCursor = 0
							}
						}
						window.SetTitle(menuTitle(levels, selectedLevel, menuCursor, activeScreen))
					case sdl.K_LEFT:
						if activeScreen == screenMain {
							if len(levels) > 0 {
								selectedLevel--
								if selectedLevel < 0 {
									selectedLevel = len(levels) - 1
								}
							}
						} else {
							switch optionsAction(optionsCursor) {
							case optResolution:
								resolutionIdx--
								if resolutionIdx < 0 {
									resolutionIdx = len(resolutionPresets) - 1
								}
								window.SetSize(int32(resolutionPresets[resolutionIdx].width), int32(resolutionPresets[resolutionIdx].height))
							}
						}
						window.SetTitle(menuTitle(levels, selectedLevel, menuCursor, activeScreen))
					case sdl.K_RIGHT:
						if activeScreen == screenMain {
							if len(levels) > 0 {
								selectedLevel++
								if selectedLevel >= len(levels) {
									selectedLevel = 0
								}
							}
						} else {
							switch optionsAction(optionsCursor) {
							case optResolution:
								resolutionIdx++
								if resolutionIdx >= len(resolutionPresets) {
									resolutionIdx = 0
								}
								window.SetSize(int32(resolutionPresets[resolutionIdx].width), int32(resolutionPresets[resolutionIdx].height))
							}
						}
						window.SetTitle(menuTitle(levels, selectedLevel, menuCursor, activeScreen))
					case sdl.K_RETURN, sdl.K_KP_ENTER:
						if activeScreen == screenMain {
							switch menuAction(menuCursor) {
							case menuStart:
								if len(levels) > 0 {
									resetPlayerTransform()
									loaded, err := initGame(levels[selectedLevel].path)
									if err == nil {
										cleanupGame(state)
										state = loaded
										isWalkableAt = createWalkabilityChecker(state)
										mode = modePlaying
										window.SetTitle("Go Raycasting - Playing (Esc to menu)")
									} else {
										window.SetTitle(fmt.Sprintf("Go Raycasting - Load failed: %v", err))
									}
								}
							case menuOptions:
								activeScreen = screenOptions
								window.SetTitle(menuTitle(levels, selectedLevel, menuCursor, activeScreen))
							case menuRefresh:
								levels = discoverLevelPackages()
								if len(levels) == 0 {
									selectedLevel = 0
								} else if selectedLevel >= len(levels) {
									selectedLevel = 0
								}
								window.SetTitle(menuTitle(levels, selectedLevel, menuCursor, activeScreen))
							case menuExit:
								running = false
							}
						} else {
							switch optionsAction(optionsCursor) {
							case optRebindForward:
								rebinding = "FORWARD"
							case optRebindBackward:
								rebinding = "BACKWARD"
							case optRebindLeft:
								rebinding = "TURN LEFT"
							case optRebindRight:
								rebinding = "TURN RIGHT"
							case optResolution:
								resolutionIdx++
								if resolutionIdx >= len(resolutionPresets) {
									resolutionIdx = 0
								}
								window.SetSize(int32(resolutionPresets[resolutionIdx].width), int32(resolutionPresets[resolutionIdx].height))
							case optBack:
								activeScreen = screenMain
							}
							window.SetTitle(menuTitle(levels, selectedLevel, menuCursor, activeScreen))
						}
					}
				case modePlaying:
					if keyEvent.Keysym.Sym == sdl.K_ESCAPE {
						mode = modeMenu
						levels = discoverLevelPackages()
						if len(levels) == 0 {
							selectedLevel = 0
						} else if selectedLevel >= len(levels) {
							selectedLevel = 0
						}
						window.SetTitle(menuTitle(levels, selectedLevel, menuCursor, activeScreen))
					}
				}
			}
		}

		if mode == modePlaying && state != nil {
			keys := sdl.GetKeyboardState()
			handleInput(keys, frameTime, isWalkableAt, bindings)
			renderFrame(state)
			render(renderer, texture, state.pixels)
		} else {
			drawMenuFrame(menuPixels, levels, selectedLevel, activeScreen, menuCursor, optionsCursor, bindings, resolutionIdx, fpsIdx, rebinding)
			render(renderer, texture, menuPixels)
		}
	}

	cleanupGame(state)
}
