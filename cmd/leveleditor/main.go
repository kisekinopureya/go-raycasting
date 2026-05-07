package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"kisekinopureya.com.tr/go-raycasting/worldMap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	initialWindowWidth  = 1560
	initialWindowHeight = 920
	minimumWindowWidth  = 1320
	minimumWindowHeight = 760
	gridOffsetX         = 28
	gridOffsetY         = 124
	cellSize            = 56
	panelWidth          = 520
	panelGap            = 28
	thumbnailSize       = 88
	thumbnailGap        = 16
	panelColumns        = 3
)

var (
	uiFont            rl.Font
	hasCustomLoadedUI bool
)

type dialogMode int

const (
	dialogNone     dialogMode = iota
	dialogCreate              // create a new package at typed path
	dialogOpen                // open an existing package
	dialogDelete              // delete a package
	dialogMetadata            // edit package metadata
)

type levelDialog struct {
	mode              dialogMode
	inputPath         string // typed by user (for create/open/delete)
	errorText         string
	metadataName      string // metadata being edited
	metadataAuthor    string
	metadataDesc      string
	metadataVersion   string
	metadataEditField int // 0=name, 1=author, 2=desc, 3=version
}

const (
	metaFieldName    = 0
	metaFieldAuthor  = 1
	metaFieldDesc    = 2
	metaFieldVersion = 3
)

type editorState struct {
	level               worldMap.Level
	packageMetadata     worldMap.LevelPackageMetadata // current package metadata
	packagePath         string                        // empty when no level is open
	activeLayer         worldMap.AssetLayer
	selectedTiles       map[worldMap.AssetLayer]int
	selectedLibraryPath string
	availableAssets     []string
	previews            map[string]rl.Texture2D
	statusText          string
	statusColor         rl.Color
	panelScroll         float32
	gridOffsetX         float32
	gridOffsetY         float32
	assetsPanelOpen     bool
	panelOpen           bool
	dialog              levelDialog
}

type cellCoord struct {
	x int
	y int
}

type layoutMetrics struct {
	screenWidth    int
	screenHeight   int
	panelRect      rl.Rectangle
	gridRect       rl.Rectangle
	scrollViewport rl.Rectangle
}

type contentMetrics struct {
	paletteTitleY   float32
	paletteGridY    float32
	actionsTitleY   float32
	actionsButtonsY float32
	browserTitleY   float32
	browserGridY    float32
	totalHeight     float32
}

func main() {
	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(initialWindowWidth, initialWindowHeight, "go-raycasting level editor")
	defer rl.CloseWindow()
	rl.SetWindowMinSize(minimumWindowWidth, minimumWindowHeight)
	rl.SetTargetFPS(60)

	if _, err := os.Stat("assets/font.ttf"); err != nil {
		log.Fatalf("font file missing: %v", err)
	}

	uiFont = rl.LoadFont("assets/font.ttf")
	if uiFont.Texture.ID == 0 {
		log.Fatalf("failed to load assets/font.ttf")
	}
	hasCustomLoadedUI = true
	defer func() {
		if hasCustomLoadedUI {
			rl.UnloadFont(uiFont)
		}
	}()

	state := editorState{
		activeLayer: worldMap.AssetLayerWalls,
		selectedTiles: map[worldMap.AssetLayer]int{
			worldMap.AssetLayerWalls:    1,
			worldMap.AssetLayerFloors:   1,
			worldMap.AssetLayerCeilings: worldMap.SkyboxTile,
		},
		previews:        make(map[string]rl.Texture2D),
		statusText:      "No level open. Use New Level, Open Level, or Delete Level buttons.",
		statusColor:     rl.DarkGray,
		gridOffsetX:     gridOffsetX,
		gridOffsetY:     gridOffsetY,
		assetsPanelOpen: true,
		panelOpen:       true,
	}

	defer unloadPreviews(state.previews)

	for !rl.WindowShouldClose() {
		layout := currentLayout(state.level, state.gridOffsetX, state.gridOffsetY)

		if state.dialog.mode != dialogNone {
			handleDialogInput(&state)
		} else {
			handleShortcuts(&state)
			handlePanelScroll(&state, layout)
			handlePanelClick(&state, layout)
			handleGridPaint(&state, layout)
			handleGridPan(&state, layout)
		}

		rl.BeginDrawing()
		rl.ClearBackground(rl.NewColor(247, 244, 237, 255))

		drawHeader(state, layout)
		drawGrid(&state, layout)
		drawPanel(&state, layout)

		if state.dialog.mode != dialogNone {
			if state.dialog.mode == dialogMetadata {
				drawMetadataDialog(&state, layout)
			} else {
				drawLevelDialog(&state, layout)
			}
		}

		rl.EndDrawing()
	}

}

func currentLayout(level worldMap.Level, gridOffsetX, gridOffsetY float32) layoutMetrics {
	screenWidth := rl.GetScreenWidth()
	screenHeight := rl.GetScreenHeight()
	gridWidth := float32(level.Width() * cellSize)
	gridHeight := float32(level.Height() * cellSize)
	panelX := float32(screenWidth) - panelWidth - 24
	panelY := float32(24)
	panelHeight := float32(screenHeight) - 48

	return layoutMetrics{
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
		gridRect: rl.NewRectangle(
			gridOffsetX,
			gridOffsetY,
			gridWidth,
			gridHeight,
		),
		panelRect:      rl.NewRectangle(panelX, panelY, panelWidth, panelHeight),
		scrollViewport: rl.NewRectangle(panelX+18, panelY+475, panelWidth-36, panelHeight-493),
	}
}

func panelContentMetrics(state editorState) contentMetrics {
	paletteRows := rowCount(len(paletteTiles(state)))
	browserRows := rowCount(len(state.availableAssets))

	paletteTitleY := float32(8)
	paletteGridY := paletteTitleY + 44
	actionsTitleY := paletteGridY + float32(paletteRows)*(thumbnailSize+thumbnailGap) + 18
	actionsButtonsY := actionsTitleY + 38
	browserTitleY := actionsButtonsY + 66
	browserGridY := browserTitleY + 44
	totalHeight := browserGridY + float32(browserRows)*(thumbnailSize+thumbnailGap)

	return contentMetrics{
		paletteTitleY:   paletteTitleY,
		paletteGridY:    paletteGridY,
		actionsTitleY:   actionsTitleY,
		actionsButtonsY: actionsButtonsY,
		browserTitleY:   browserTitleY,
		browserGridY:    browserGridY,
		totalHeight:     totalHeight,
	}
}

func handleShortcuts(state *editorState) {
	if rl.IsKeyPressed(rl.KeyOne) {
		setActiveLayer(state, worldMap.AssetLayerWalls)
	}

	if rl.IsKeyPressed(rl.KeyTwo) {
		setActiveLayer(state, worldMap.AssetLayerFloors)
	}

	if rl.IsKeyPressed(rl.KeyThree) {
		setActiveLayer(state, worldMap.AssetLayerCeilings)
	}

	if ctrlPressed() && rl.IsKeyPressed(rl.KeyS) {
		saveState(state)
	}

	if rl.IsKeyPressed(rl.KeyR) {
		reloadState(state)
	}

	if rl.IsKeyPressed(rl.KeyF5) {
		refreshAssetBrowser(state)
	}
}

func handlePanelScroll(state *editorState, layout layoutMetrics) {
	if !rl.CheckCollisionPointRec(rl.GetMousePosition(), layout.scrollViewport) {
		return
	}

	content := panelContentMetrics(*state)
	maxScroll := maxFloat32(0, content.totalHeight-layout.scrollViewport.Height+8)
	if maxScroll == 0 {
		state.panelScroll = 0
		return
	}

	wheel := rl.GetMouseWheelMove()
	if wheel == 0 {
		return
	}

	state.panelScroll = clampFloat32(state.panelScroll-wheel*48, 0, maxScroll)
}

func handlePanelClick(state *editorState, layout layoutMetrics) {
	mouse := rl.GetMousePosition()
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		if rl.CheckCollisionPointRec(mouse, newLevelButtonRect(layout)) {
			state.dialog = levelDialog{mode: dialogCreate}
			return
		}
		if rl.CheckCollisionPointRec(mouse, openLevelButtonRect(layout)) {
			state.dialog = levelDialog{mode: dialogOpen}
			return
		}
		if rl.CheckCollisionPointRec(mouse, deleteLevelButtonRect(layout)) {
			state.dialog = levelDialog{mode: dialogDelete, inputPath: state.packagePath}
			return
		}
		if rl.CheckCollisionPointRec(mouse, metadataButtonRect(layout)) {
			if state.packagePath != "" {
				state.dialog = levelDialog{
					mode:              dialogMetadata,
					metadataName:      state.packageMetadata.Name,
					metadataAuthor:    state.packageMetadata.Author,
					metadataDesc:      state.packageMetadata.Description,
					metadataVersion:   state.packageMetadata.Version,
					metadataEditField: metaFieldName,
				}
			}
			return
		}

		for index, layer := range []worldMap.AssetLayer{worldMap.AssetLayerWalls, worldMap.AssetLayerFloors, worldMap.AssetLayerCeilings} {
			if rl.CheckCollisionPointRec(mouse, layerButtonRect(layout, index)) {
				setActiveLayer(state, layer)
				return
			}
		}

		if rl.CheckCollisionPointRec(mouse, saveButtonRect(layout)) {
			saveState(state)
			return
		}

		if rl.CheckCollisionPointRec(mouse, reloadButtonRect(layout)) {
			reloadState(state)
			return
		}

		if rl.CheckCollisionPointRec(mouse, rescanButtonRect(layout)) {
			refreshAssetBrowser(state)
			return
		}

		if rl.CheckCollisionPointRec(mouse, addColButtonRect(layout)) {
			state.level.AddColumn()
			state.statusText = fmt.Sprintf("Map is now %d×%d", state.level.Width(), state.level.Height())
			state.statusColor = rl.DarkGreen
			return
		}

		if rl.CheckCollisionPointRec(mouse, remColButtonRect(layout)) {
			if err := state.level.RemoveColumn(); err != nil {
				state.statusText = err.Error()
				state.statusColor = rl.Maroon
			} else {
				state.statusText = fmt.Sprintf("Map is now %d×%d", state.level.Width(), state.level.Height())
				state.statusColor = rl.DarkGreen
			}
			return
		}

		if rl.CheckCollisionPointRec(mouse, addRowButtonRect(layout)) {
			state.level.AddRow()
			state.statusText = fmt.Sprintf("Map is now %d×%d", state.level.Width(), state.level.Height())
			state.statusColor = rl.DarkGreen
			return
		}

		if rl.CheckCollisionPointRec(mouse, remRowButtonRect(layout)) {
			if err := state.level.RemoveRow(); err != nil {
				state.statusText = err.Error()
				state.statusColor = rl.Maroon
			} else {
				state.statusText = fmt.Sprintf("Map is now %d×%d", state.level.Width(), state.level.Height())
				state.statusColor = rl.DarkGreen
			}
			return
		}

		if rl.CheckCollisionPointRec(mouse, togglePanelRect(layout)) {
			state.panelOpen = !state.panelOpen
			state.panelScroll = 0
			if state.panelOpen {
				state.statusText = "Panel expanded"
			} else {
				state.statusText = "Panel collapsed"
			}
			state.statusColor = rl.DarkGray
			return
		}

		if rl.CheckCollisionPointRec(mouse, hidePanelButtonRect(layout)) {
			state.panelOpen = false
			state.statusText = "Panel collapsed"
			state.statusColor = rl.DarkGray
			return
		}

		if rl.CheckCollisionPointRec(mouse, toggleAssetsPanelRect(layout)) {
			state.assetsPanelOpen = !state.assetsPanelOpen
			if state.assetsPanelOpen {
				state.statusText = "Assets panel expanded"
			} else {
				state.statusText = "Assets panel collapsed"
			}
			state.statusColor = rl.DarkGray
			return
		}

		if !state.assetsPanelOpen {
			return
		}

		content := panelContentMetrics(*state)

		for index, tile := range paletteTiles(*state) {
			if rl.CheckCollisionPointRec(mouse, paletteTileRect(layout, content, state.panelScroll, index)) {
				state.selectedTiles[state.activeLayer] = tile
				state.statusText = fmt.Sprintf("Selected %s", tileLabel(state.activeLayer, tile))
				state.statusColor = rl.DarkGray
				return
			}
		}

		if rl.CheckCollisionPointRec(mouse, addButtonRect(layout, content, state.panelScroll)) {
			addSelectedAsset(state)
			return
		}

		if rl.CheckCollisionPointRec(mouse, replaceButtonRect(layout, content, state.panelScroll)) {
			replaceSelectedAsset(state)
			return
		}

		if rl.CheckCollisionPointRec(mouse, removeButtonRect(layout, content, state.panelScroll)) {
			removeSelectedAsset(state)
			return
		}

		for index, path := range state.availableAssets {
			if rl.CheckCollisionPointRec(mouse, browserTileRect(layout, content, state.panelScroll, index)) {
				state.selectedLibraryPath = path
				state.statusText = fmt.Sprintf("Library asset selected: %s", filepath.Base(path))
				state.statusColor = rl.DarkGray
				return
			}
		}
	}
}

func handleGridPaint(state *editorState, layout layoutMetrics) {
	mouse := rl.GetMousePosition()
	if !rl.CheckCollisionPointRec(mouse, layout.gridRect) {
		return
	}

	x := int((mouse.X - layout.gridRect.X) / cellSize)
	y := int((mouse.Y - layout.gridRect.Y) / cellSize)

	if x < 0 || x >= state.level.Width() || y < 0 || y >= state.level.Height() {
		return
	}

	if rl.IsMouseButtonDown(rl.MouseLeftButton) {
		switch state.activeLayer {
		case worldMap.AssetLayerWalls:
			state.level.Walls[x][y] = state.selectedTiles[state.activeLayer]
		case worldMap.AssetLayerFloors:
			state.level.Floors[x][y] = state.selectedTiles[state.activeLayer]
		default:
			state.level.Ceilings[x][y] = state.selectedTiles[state.activeLayer]
		}
	}

	if rl.IsMouseButtonDown(rl.MouseRightButton) {
		switch state.activeLayer {
		case worldMap.AssetLayerWalls:
			state.level.Walls[x][y] = 0
		case worldMap.AssetLayerFloors:
			state.level.Floors[x][y] = 0
		default:
			state.level.Ceilings[x][y] = 0
		}
	}
}

func handleGridPan(state *editorState, layout layoutMetrics) {
	mouse := rl.GetMousePosition()

	// Mouse wheel panning (vertical and horizontal)
	if rl.CheckCollisionPointRec(mouse, layout.gridRect) {
		wheel := rl.GetMouseWheelMove()
		if wheel != 0 {
			if ctrlPressed() || rl.IsKeyDown(rl.KeySpace) {
				state.gridOffsetY += wheel * 120
			}
		}

		wheelV := rl.GetMouseWheelMoveV()
		if rl.IsKeyDown(rl.KeyLeftShift) && wheelV.X != 0 {
			state.gridOffsetX += wheelV.X * 120
		}

		// Keyboard panning for horizontal scrolling (arrow keys or A/D)
		if rl.IsKeyDown(rl.KeyLeft) || rl.IsKeyDown(rl.KeyA) {
			state.gridOffsetX += 8
		}
		if rl.IsKeyDown(rl.KeyRight) || rl.IsKeyDown(rl.KeyD) {
			state.gridOffsetX -= 8
		}
		// Keyboard panning for vertical scrolling (arrow keys)
		if rl.IsKeyDown(rl.KeyUp) || rl.IsKeyDown(rl.KeyW) {
			state.gridOffsetY += 8
		}
		if rl.IsKeyDown(rl.KeyDown) || rl.IsKeyDown(rl.KeyS) {
			state.gridOffsetY -= 8
		}
	}

	// Middle mouse button drag panning
	if rl.IsMouseButtonDown(rl.MouseMiddleButton) && rl.CheckCollisionPointRec(mouse, layout.gridRect) {
		delta := rl.GetMouseDelta()
		state.gridOffsetX += delta.X
		state.gridOffsetY += delta.Y
	}

	// Clamp pan offsets to reasonable bounds
	maxGridWidth := float32(state.level.Width() * cellSize)
	maxGridHeight := float32(state.level.Height() * cellSize)
	screenWidth := float32(layout.screenWidth)
	screenHeight := float32(layout.screenHeight)

	// Allow panning so grid can be scrolled up to 80% into view
	minX := -(maxGridWidth * 0.8)
	maxX := screenWidth - 40
	minY := -(maxGridHeight * 0.8)
	maxY := screenHeight - 130

	state.gridOffsetX = clampFloat32(state.gridOffsetX, minX, maxX)
	state.gridOffsetY = clampFloat32(state.gridOffsetY, minY, maxY)
}

func drawHeader(state editorState, layout layoutMetrics) {
	titleFont := float32(34)

	drawTextLine("Raylib Level Editor", rl.NewVector2(gridOffsetX, 24), titleFont, rl.Black)

	// Level management buttons in top-right
	drawActionButton(newLevelButtonRect(layout), "New Level")
	drawActionButton(openLevelButtonRect(layout), "Open Level")
	if state.packagePath != "" {
		drawActionButton(deleteLevelButtonRect(layout), "Delete Level")
		drawActionButton(metadataButtonRect(layout), "Metadata")
	}

	statusLines := wrapTextToWidth(state.statusText, layout.gridRect.Width-20, 18)
	if len(statusLines) == 0 {
		statusLines = []string{""}
	}
	statusHeight := float32(16 + len(statusLines)*22)
	statusBox := rl.NewRectangle(gridOffsetX, 64, layout.gridRect.Width, statusHeight)
	rl.DrawRectangleRounded(statusBox, 0.18, 6, rl.NewColor(255, 250, 240, 255))
	rl.DrawRectangleLinesEx(statusBox, 2, rl.Fade(rl.Black, 0.12))
	for i, line := range statusLines {
		drawTextLine(line, rl.NewVector2(statusBox.X+10, statusBox.Y+8+float32(i*22)), 18, state.statusColor)
	}
}

func drawGrid(state *editorState, layout layoutMetrics) {
	for x := 0; x < state.level.Width(); x++ {
		for y := 0; y < state.level.Height(); y++ {
			drawCell(state, layout, x, y)
		}
	}

	for x := 0; x <= state.level.Width(); x++ {
		lineX := layout.gridRect.X + float32(x*cellSize)
		rl.DrawLineV(rl.NewVector2(lineX, layout.gridRect.Y), rl.NewVector2(lineX, layout.gridRect.Y+layout.gridRect.Height), rl.Fade(rl.Black, 0.18))
	}

	for y := 0; y <= state.level.Height(); y++ {
		lineY := layout.gridRect.Y + float32(y*cellSize)
		rl.DrawLineV(rl.NewVector2(layout.gridRect.X, lineY), rl.NewVector2(layout.gridRect.X+layout.gridRect.Width, lineY), rl.Fade(rl.Black, 0.18))
	}
}

func drawCell(state *editorState, layout layoutMetrics, x, y int) {
	rect := rl.NewRectangle(layout.gridRect.X+float32(x*cellSize), layout.gridRect.Y+float32(y*cellSize), cellSize, cellSize)
	floorTile := worldMap.TileAt(state.level.Floors, x, y)
	ceilingTile := worldMap.TileAt(state.level.Ceilings, x, y)
	wallTile := worldMap.TileAt(state.level.Walls, x, y)

	drawThumbnail(state, floorAssetPath(state.level, floorTile), rect, rl.NewColor(227, 232, 240, 255))
	drawThumbnail(state, ceilingAssetPath(state.level, ceilingTile), rl.NewRectangle(rect.X, rect.Y, rect.Width, 12), rl.NewColor(191, 219, 254, 255))

	if state.level.IsWall(wallTile) {
		inner := rl.NewRectangle(rect.X+10, rect.Y+14, rect.Width-20, rect.Height-24)
		drawThumbnail(state, wallAssetPath(state.level, wallTile), inner, rl.NewColor(203, 213, 225, 255))
	}

	var tile int
	switch state.activeLayer {
	case worldMap.AssetLayerWalls:
		tile = worldMap.TileAt(state.level.Walls, x, y)
	case worldMap.AssetLayerFloors:
		tile = worldMap.TileAt(state.level.Floors, x, y)
	default:
		tile = worldMap.TileAt(state.level.Ceilings, x, y)
	}
	label := tileLabel(state.activeLayer, tile)
	labelBox := rl.NewRectangle(rect.X+4, rect.Y+rect.Height-21, rect.Width-8, 17)
	rl.DrawRectangleRounded(labelBox, 0.22, 4, rl.Fade(rl.RayWhite, 0.88))
	drawCenteredText(trimTextToWidth(label, labelBox.Width-8, 15), labelBox, 15, rl.Black)

	if hover, ok := hoverCell(state.level, layout); ok && hover.x == x && hover.y == y {
		rl.DrawRectangleLinesEx(rect, 3, rl.NewColor(198, 120, 40, 255))
	}
}

func drawPanel(state *editorState, layout layoutMetrics) {
	content := panelContentMetrics(*state)
	maxScroll := maxFloat32(0, content.totalHeight-layout.scrollViewport.Height+8)
	state.panelScroll = clampFloat32(state.panelScroll, 0, maxScroll)

	if !state.panelOpen {
		toggleBtn := togglePanelRect(layout)
		rl.DrawRectangleRounded(toggleBtn, 0.18, 4, rl.NewColor(230, 240, 255, 255))
		rl.DrawRectangleLinesEx(toggleBtn, 2, rl.Fade(rl.Black, 0.18))
		drawCenteredText("Show Panel", toggleBtn, 20, rl.Black)
		return
	}

	rl.DrawRectangleRounded(layout.panelRect, 0.035, 6, rl.NewColor(242, 238, 231, 255))
	rl.DrawRectangleLinesEx(layout.panelRect, 2, rl.Fade(rl.Black, 0.12))

	for index, layer := range []worldMap.AssetLayer{worldMap.AssetLayerWalls, worldMap.AssetLayerFloors, worldMap.AssetLayerCeilings} {
		rect := layerButtonRect(layout, index)
		color := rl.NewColor(234, 230, 222, 255)
		if state.activeLayer == layer {
			color = rl.NewColor(215, 226, 233, 255)
		}
		rl.DrawRectangleRounded(rect, 0.18, 4, color)
		rl.DrawRectangleLinesEx(rect, 2, rl.Fade(rl.Black, 0.18))
		drawCenteredText(layerLabel(layer), rect, 21, rl.Black)
	}

	drawActionButton(saveButtonRect(layout), "Save")
	drawActionButton(reloadButtonRect(layout), "Reload")
	drawActionButton(rescanButtonRect(layout), "Rescan")

	drawInfoCard(
		rl.NewRectangle(layout.panelRect.X+24, layout.panelRect.Y+158, layout.panelRect.Width-48, 58),
		"Target",
		selectedTargetLabel(*state),
	)

	hoverText := "Move the pointer over the grid"
	if hover, ok := hoverCell(state.level, layout); ok {
		hoverText = fmt.Sprintf(
			"Cell %d,%d  wall:%d floor:%d ceiling:%d",
			hover.x,
			hover.y,
			worldMap.TileAt(state.level.Walls, hover.x, hover.y),
			worldMap.TileAt(state.level.Floors, hover.x, hover.y),
			worldMap.TileAt(state.level.Ceilings, hover.x, hover.y),
		)
	}
	drawInfoCard(
		rl.NewRectangle(layout.panelRect.X+24, layout.panelRect.Y+226, layout.panelRect.Width-48, 62),
		"Tile Info",
		hoverText,
	)

	drawTextLine(
		fmt.Sprintf("Map Size  %d × %d", state.level.Width(), state.level.Height()),
		rl.NewVector2(layout.panelRect.X+24, layout.panelRect.Y+298),
		18, rl.DarkGray,
	)
	drawActionButton(addColButtonRect(layout), "+Col")
	drawActionButton(remColButtonRect(layout), "-Col")
	drawActionButton(addRowButtonRect(layout), "+Row")
	drawActionButton(remRowButtonRect(layout), "-Row")

	toggleBtn := toggleAssetsPanelRect(layout)
	toggleColor := rl.NewColor(240, 248, 255, 255)
	if !state.assetsPanelOpen {
		toggleColor = rl.NewColor(230, 240, 255, 255)
	}
	rl.DrawRectangleRounded(toggleBtn, 0.18, 4, toggleColor)
	rl.DrawRectangleLinesEx(toggleBtn, 2, rl.Fade(rl.Black, 0.18))
	label := "Hide"
	if !state.assetsPanelOpen {
		label = "Show"
	}
	drawCenteredText(label+" Assets", toggleBtn, 18, rl.Black)

	// Draw hide panel button at top-right of panel
	hidePanelBtn := hidePanelButtonRect(layout)
	rl.DrawRectangleRounded(hidePanelBtn, 0.18, 4, rl.NewColor(255, 248, 240, 255))
	rl.DrawRectangleLinesEx(hidePanelBtn, 2, rl.Fade(rl.Black, 0.18))
	drawCenteredText("X", hidePanelBtn, 24, rl.Black)

	if !state.assetsPanelOpen {
		return
	}

	rl.BeginScissorMode(int32(layout.scrollViewport.X), int32(layout.scrollViewport.Y), int32(layout.scrollViewport.Width), int32(layout.scrollViewport.Height))

	drawTextLine("Paint Palette", rl.NewVector2(layout.scrollViewport.X+6, contentY(layout, content.paletteTitleY, state.panelScroll)), 24, rl.Black)
	drawTextLine("Mouse wheel scrolls this area.", rl.NewVector2(layout.scrollViewport.X+6, contentY(layout, content.paletteTitleY+28, state.panelScroll)), 18, rl.DarkGray)

	for index, tile := range paletteTiles(*state) {
		rect := paletteTileRect(layout, content, state.panelScroll, index)
		drawThumbnail(state, paletteAssetPath(*state, tile), rect, rl.NewColor(226, 232, 240, 255))
		border := rl.Fade(rl.Black, 0.18)
		if state.selectedTiles[state.activeLayer] == tile {
			border = rl.Maroon
		}
		rl.DrawRectangleLinesEx(rect, 3, border)
		labelBox := rl.NewRectangle(rect.X+4, rect.Y+rect.Height-19, rect.Width-8, 15)
		rl.DrawRectangleRounded(labelBox, 0.2, 4, rl.Fade(rl.RayWhite, 0.88))
		drawCenteredText(tileLabel(state.activeLayer, tile), labelBox, 14, rl.Black)
	}

	drawTextLine("Asset Actions", rl.NewVector2(layout.scrollViewport.X+6, contentY(layout, content.actionsTitleY, state.panelScroll)), 24, rl.Black)
	drawActionButton(addButtonRect(layout, content, state.panelScroll), "Add")
	drawActionButton(replaceButtonRect(layout, content, state.panelScroll), "Replace")
	drawActionButton(removeButtonRect(layout, content, state.panelScroll), "Remove")

	drawTextLine("Asset Browser", rl.NewVector2(layout.scrollViewport.X+6, contentY(layout, content.browserTitleY, state.panelScroll)), 24, rl.Black)
	drawTextLine("Pick a file, then Add or Replace.", rl.NewVector2(layout.scrollViewport.X+6, contentY(layout, content.browserTitleY+28, state.panelScroll)), 18, rl.DarkGray)

	for index, path := range state.availableAssets {
		rect := browserTileRect(layout, content, state.panelScroll, index)
		drawThumbnail(state, path, rect, rl.NewColor(226, 232, 240, 255))
		border := rl.Fade(rl.Black, 0.18)
		if state.selectedLibraryPath == path {
			border = rl.DarkBlue
		}
		rl.DrawRectangleLinesEx(rect, 3, border)
		labelBox := rl.NewRectangle(rect.X+4, rect.Y+rect.Height-19, rect.Width-8, 15)
		rl.DrawRectangleRounded(labelBox, 0.2, 4, rl.Fade(rl.RayWhite, 0.88))
		drawCenteredText(trimTextToWidth(filepath.Base(path), labelBox.Width-8, 14), labelBox, 14, rl.Black)
	}

	rl.EndScissorMode()

	if maxScroll > 0 {
		drawScrollBar(layout.scrollViewport, state.panelScroll, maxScroll, content.totalHeight)
	}
}

func drawInfoCard(rect rl.Rectangle, title string, value string) {
	rl.DrawRectangleRounded(rect, 0.16, 4, rl.NewColor(252, 249, 243, 255))
	rl.DrawRectangleLinesEx(rect, 2, rl.Fade(rl.Black, 0.14))
	drawTextLine(title, rl.NewVector2(rect.X+10, rect.Y+8), 16, rl.DarkGray)
	drawTextLine(trimTextToWidth(value, rect.Width-20, 19), rl.NewVector2(rect.X+10, rect.Y+28), 19, rl.Black)
}

func drawScrollBar(viewport rl.Rectangle, scroll, maxScroll, contentHeight float32) {
	track := rl.NewRectangle(viewport.X+viewport.Width-8, viewport.Y, 6, viewport.Height)
	rl.DrawRectangleRounded(track, 0.5, 4, rl.Fade(rl.Black, 0.08))

	thumbHeight := maxFloat32(48, viewport.Height*(viewport.Height/contentHeight))
	thumbY := viewport.Y
	if maxScroll > 0 {
		thumbY += (viewport.Height - thumbHeight) * (scroll / maxScroll)
	}

	thumb := rl.NewRectangle(track.X, thumbY, track.Width, thumbHeight)
	rl.DrawRectangleRounded(thumb, 0.5, 4, rl.NewColor(123, 142, 154, 255))
}

func drawThumbnail(state *editorState, path string, rect rl.Rectangle, fallback rl.Color) {
	if path == "" {
		rl.DrawRectangleRounded(rect, 0.12, 4, fallback)
		return
	}

	texture, ok := previewTexture(state, path)
	if !ok {
		rl.DrawRectangleRounded(rect, 0.12, 4, fallback)
		return
	}

	source := rl.NewRectangle(0, 0, float32(texture.Width), float32(texture.Height))
	destination := fitRect(rect, float32(texture.Width), float32(texture.Height))
	rl.DrawRectangleRounded(rect, 0.12, 4, rl.White)
	rl.DrawTexturePro(texture, source, destination, rl.NewVector2(0, 0), 0, rl.White)
}

func drawActionButton(rect rl.Rectangle, label string) {
	rl.DrawRectangleRounded(rect, 0.18, 4, rl.NewColor(255, 252, 246, 255))
	rl.DrawRectangleLinesEx(rect, 2, rl.Fade(rl.Black, 0.18))
	drawCenteredText(label, rect, 19, rl.Black)
}

func drawTextLine(text string, position rl.Vector2, size float32, color rl.Color) {
	font := uiFont
	if font.Texture.ID == 0 {
		font = rl.GetFontDefault()
	}
	rl.DrawTextEx(font, text, position, size, 1, color)
}

func drawCenteredText(text string, rect rl.Rectangle, size float32, color rl.Color) {
	font := uiFont
	if font.Texture.ID == 0 {
		font = rl.GetFontDefault()
	}
	width := rl.MeasureTextEx(font, text, size, 1).X
	pos := rl.NewVector2(rect.X+(rect.Width-width)/2, rect.Y+(rect.Height-size)/2-1)
	drawTextLine(text, pos, size, color)
}

func previewTexture(state *editorState, path string) (rl.Texture2D, bool) {
	if texture, exists := state.previews[path]; exists {
		return texture, true
	}

	image := rl.LoadImage(path)
	if image.Width == 0 || image.Height == 0 {
		return rl.Texture2D{}, false
	}
	defer rl.UnloadImage(image)

	texture := rl.LoadTextureFromImage(image)
	if texture.ID == 0 {
		return rl.Texture2D{}, false
	}

	state.previews[path] = texture
	return texture, true
}

func unloadPreviews(previews map[string]rl.Texture2D) {
	for _, texture := range previews {
		if texture.ID != 0 {
			rl.UnloadTexture(texture)
		}
	}
}

func hoverCell(level worldMap.Level, layout layoutMetrics) (cellCoord, bool) {
	mouse := rl.GetMousePosition()
	if !rl.CheckCollisionPointRec(mouse, layout.gridRect) {
		return cellCoord{}, false
	}

	x := int((mouse.X - layout.gridRect.X) / cellSize)
	y := int((mouse.Y - layout.gridRect.Y) / cellSize)
	return cellCoord{x: x, y: y}, true
}

func newLevelButtonRect(layout layoutMetrics) rl.Rectangle {
	return rl.NewRectangle(float32(layout.screenWidth)-panelWidth-406, 24, 116, 44)
}

func openLevelButtonRect(layout layoutMetrics) rl.Rectangle {
	return rl.NewRectangle(float32(layout.screenWidth)-panelWidth-282, 24, 116, 44)
}

func deleteLevelButtonRect(layout layoutMetrics) rl.Rectangle {
	return rl.NewRectangle(float32(layout.screenWidth)-panelWidth-158, 24, 116, 44)
}

func metadataButtonRect(layout layoutMetrics) rl.Rectangle {
	return rl.NewRectangle(float32(layout.screenWidth)-panelWidth-34, 24, 116, 44)
}

func layerButtonRect(layout layoutMetrics, index int) rl.Rectangle {
	return rl.NewRectangle(layout.panelRect.X+24+float32(index)*104, layout.panelRect.Y+24, 96, 52)
}

func saveButtonRect(layout layoutMetrics) rl.Rectangle {
	return rl.NewRectangle(layout.panelRect.X+24, layout.panelRect.Y+94, 120, 44)
}

func reloadButtonRect(layout layoutMetrics) rl.Rectangle {
	return rl.NewRectangle(layout.panelRect.X+160, layout.panelRect.Y+94, 120, 44)
}

func rescanButtonRect(layout layoutMetrics) rl.Rectangle {
	return rl.NewRectangle(layout.panelRect.X+296, layout.panelRect.Y+94, 120, 44)
}

func addColButtonRect(layout layoutMetrics) rl.Rectangle {
	return rl.NewRectangle(layout.panelRect.X+24, layout.panelRect.Y+320, 104, 40)
}

func remColButtonRect(layout layoutMetrics) rl.Rectangle {
	return rl.NewRectangle(layout.panelRect.X+136, layout.panelRect.Y+320, 104, 40)
}

func addRowButtonRect(layout layoutMetrics) rl.Rectangle {
	return rl.NewRectangle(layout.panelRect.X+248, layout.panelRect.Y+320, 104, 40)
}

func remRowButtonRect(layout layoutMetrics) rl.Rectangle {
	return rl.NewRectangle(layout.panelRect.X+360, layout.panelRect.Y+320, 104, 40)
}

func togglePanelRect(layout layoutMetrics) rl.Rectangle {
	return rl.NewRectangle(float32(layout.screenWidth)-120-24, 24, 120, 52)
}

func hidePanelButtonRect(layout layoutMetrics) rl.Rectangle {
	return rl.NewRectangle(layout.panelRect.X+layout.panelRect.Width-44, layout.panelRect.Y+12, 36, 36)
}

func toggleAssetsPanelRect(layout layoutMetrics) rl.Rectangle {
	return rl.NewRectangle(layout.panelRect.X+24, layout.panelRect.Y+430, layout.panelRect.Width-48, 40)
}

func paletteTileRect(layout layoutMetrics, content contentMetrics, scroll float32, index int) rl.Rectangle {
	column := index % panelColumns
	row := index / panelColumns
	x := layout.scrollViewport.X + 6 + float32(column)*(thumbnailSize+thumbnailGap)
	y := contentY(layout, content.paletteGridY+float32(row)*(thumbnailSize+thumbnailGap), scroll)
	return rl.NewRectangle(x, y, thumbnailSize, thumbnailSize)
}

func addButtonRect(layout layoutMetrics, content contentMetrics, scroll float32) rl.Rectangle {
	y := contentY(layout, content.actionsButtonsY, scroll)
	return rl.NewRectangle(layout.scrollViewport.X+6, y, 120, 44)
}

func replaceButtonRect(layout layoutMetrics, content contentMetrics, scroll float32) rl.Rectangle {
	y := contentY(layout, content.actionsButtonsY, scroll)
	return rl.NewRectangle(layout.scrollViewport.X+142, y, 120, 44)
}

func removeButtonRect(layout layoutMetrics, content contentMetrics, scroll float32) rl.Rectangle {
	y := contentY(layout, content.actionsButtonsY, scroll)
	return rl.NewRectangle(layout.scrollViewport.X+278, y, 120, 44)
}

func browserTileRect(layout layoutMetrics, content contentMetrics, scroll float32, index int) rl.Rectangle {
	column := index % panelColumns
	row := index / panelColumns
	x := layout.scrollViewport.X + 6 + float32(column)*(thumbnailSize+thumbnailGap)
	y := contentY(layout, content.browserGridY+float32(row)*(thumbnailSize+thumbnailGap), scroll)
	return rl.NewRectangle(x, y, thumbnailSize, thumbnailSize)
}

func contentY(layout layoutMetrics, relativeY, scroll float32) float32 {
	return layout.scrollViewport.Y + relativeY - scroll
}

func paletteTiles(state editorState) []int {
	paths := state.level.AssetPaths(state.activeLayer)
	tiles := make([]int, 0, len(paths)+2)

	switch state.activeLayer {
	case worldMap.AssetLayerWalls:
		tiles = append(tiles, 0)
	case worldMap.AssetLayerFloors:
		tiles = append(tiles, 0)
	case worldMap.AssetLayerCeilings:
		tiles = append(tiles, worldMap.SkyboxTile, 0)
	}

	for index := range paths {
		tiles = append(tiles, index+1)
	}

	return tiles
}

func paletteAssetPath(state editorState, tile int) string {
	switch state.activeLayer {
	case worldMap.AssetLayerWalls:
		return wallAssetPath(state.level, tile)
	case worldMap.AssetLayerFloors:
		return floorAssetPath(state.level, tile)
	default:
		return ceilingAssetPath(state.level, tile)
	}
}

func wallAssetPath(level worldMap.Level, tile int) string {
	if tile <= 0 || tile > len(level.Assets.Walls) {
		return ""
	}

	return level.Assets.Walls[tile-1]
}

func floorAssetPath(level worldMap.Level, tile int) string {
	if len(level.Assets.Floors) == 0 {
		return ""
	}

	if tile > 0 && tile <= len(level.Assets.Floors) {
		return level.Assets.Floors[tile-1]
	}

	return level.Assets.Floors[0]
}

func ceilingAssetPath(level worldMap.Level, tile int) string {
	if tile == worldMap.SkyboxTile {
		return level.Assets.Skybox
	}

	if len(level.Assets.Ceilings) == 0 {
		return ""
	}

	if tile > 0 && tile <= len(level.Assets.Ceilings) {
		return level.Assets.Ceilings[tile-1]
	}

	return level.Assets.Ceilings[0]
}

func tileLabel(layer worldMap.AssetLayer, tile int) string {
	switch layer {
	case worldMap.AssetLayerWalls:
		if tile == 0 {
			return "Empty"
		}
		return fmt.Sprintf("W%d", tile)
	case worldMap.AssetLayerFloors:
		if tile == 0 {
			return "Default"
		}
		return fmt.Sprintf("F%d", tile)
	default:
		if tile == worldMap.SkyboxTile {
			return "Sky"
		}
		if tile == 0 {
			return "Default"
		}
		return fmt.Sprintf("C%d", tile)
	}
}

func layerLabel(layer worldMap.AssetLayer) string {
	switch layer {
	case worldMap.AssetLayerWalls:
		return "Walls"
	case worldMap.AssetLayerFloors:
		return "Floors"
	default:
		return "Ceilings"
	}
}

func selectedTargetLabel(state editorState) string {
	selectedTile := state.selectedTiles[state.activeLayer]
	if state.activeLayer == worldMap.AssetLayerCeilings && selectedTile == worldMap.SkyboxTile {
		return "Skybox"
	}

	if selectedTile <= 0 {
		return "No asset slot selected"
	}

	return tileLabel(state.activeLayer, selectedTile)
}

func addSelectedAsset(state *editorState) {
	if state.selectedLibraryPath == "" {
		state.statusText = "Select an asset from the browser first."
		state.statusColor = rl.Maroon
		return
	}

	newTile := state.level.AddAsset(state.activeLayer, state.selectedLibraryPath)
	state.selectedTiles[state.activeLayer] = newTile
	state.statusText = fmt.Sprintf("Added %s to %s", filepath.Base(state.selectedLibraryPath), layerLabel(state.activeLayer))
	state.statusColor = rl.DarkGreen
}

func replaceSelectedAsset(state *editorState) {
	if state.selectedLibraryPath == "" {
		state.statusText = "Select an asset from the browser first."
		state.statusColor = rl.Maroon
		return
	}

	selectedTile := state.selectedTiles[state.activeLayer]
	if state.activeLayer == worldMap.AssetLayerCeilings && selectedTile == worldMap.SkyboxTile {
		state.level.SetSkybox(state.selectedLibraryPath)
		state.statusText = fmt.Sprintf("Skybox replaced with %s", filepath.Base(state.selectedLibraryPath))
		state.statusColor = rl.DarkGreen
		return
	}

	if selectedTile <= 0 {
		state.statusText = "Select a concrete asset slot from the palette to replace."
		state.statusColor = rl.Maroon
		return
	}

	if err := state.level.ReplaceAsset(state.activeLayer, selectedTile-1, state.selectedLibraryPath); err != nil {
		state.statusText = err.Error()
		state.statusColor = rl.Maroon
		return
	}

	state.statusText = fmt.Sprintf("Replaced %s with %s", tileLabel(state.activeLayer, selectedTile), filepath.Base(state.selectedLibraryPath))
	state.statusColor = rl.DarkGreen
}

func removeSelectedAsset(state *editorState) {
	selectedTile := state.selectedTiles[state.activeLayer]
	if selectedTile <= 0 {
		state.statusText = "Select a concrete asset slot from the palette to remove."
		state.statusColor = rl.Maroon
		return
	}

	if err := state.level.RemoveAsset(state.activeLayer, selectedTile-1); err != nil {
		state.statusText = err.Error()
		state.statusColor = rl.Maroon
		return
	}

	state.selectedTiles[state.activeLayer] = 0
	state.statusText = fmt.Sprintf("Removed %s asset slot", layerLabel(state.activeLayer))
	state.statusColor = rl.DarkGreen
}

func saveState(state *editorState) {
	pkg := &worldMap.LevelPackage{
		Metadata: state.packageMetadata,
		Level:    state.level,
		BasePath: state.packagePath,
	}
	if err := worldMap.SaveLevelPackage(pkg, state.packagePath); err != nil {
		state.statusText = fmt.Sprintf("Save failed: %v", err)
		state.statusColor = rl.Maroon
		return
	}

	state.statusText = fmt.Sprintf("Saved package at %s", state.packagePath)
	state.statusColor = rl.DarkGreen
}

func reloadState(state *editorState) {
	pkg, err := worldMap.LoadLevelPackage(state.packagePath)
	if err != nil {
		state.statusText = fmt.Sprintf("Reload failed: %v", err)
		state.statusColor = rl.Maroon
		return
	}

	state.level = pkg.Level
	state.panelScroll = 0
	state.statusText = fmt.Sprintf("Reloaded package at %s", state.packagePath)
	state.statusColor = rl.DarkBlue
}

func refreshAssetBrowser(state *editorState) {
	assets, err := listAssetFiles("assets")
	if err != nil {
		state.statusText = fmt.Sprintf("Asset scan failed: %v", err)
		state.statusColor = rl.Maroon
		return
	}

	state.availableAssets = assets
	if len(assets) == 0 {
		state.selectedLibraryPath = ""
		state.statusText = "No image assets found."
		state.statusColor = rl.Maroon
		return
	}

	if !containsString(assets, state.selectedLibraryPath) {
		state.selectedLibraryPath = assets[0]
	}

	state.statusText = fmt.Sprintf("Scanned %d asset files", len(assets))
	state.statusColor = rl.DarkBlue
}

func handleDialogInput(state *editorState) {
	if state.dialog.mode == dialogMetadata {
		handleMetadataDialogInput(state)
		return
	}

	// Append typed characters
	for {
		ch := rl.GetCharPressed()
		if ch == 0 {
			break
		}
		if ch >= 32 {
			state.dialog.inputPath += string(rune(ch))
		}
	}

	// Backspace
	if rl.IsKeyPressed(rl.KeyBackspace) || rl.IsKeyPressedRepeat(rl.KeyBackspace) {
		if len(state.dialog.inputPath) > 0 {
			state.dialog.inputPath = state.dialog.inputPath[:len(state.dialog.inputPath)-1]
		}
	}

	// Escape cancels
	if rl.IsKeyPressed(rl.KeyEscape) {
		state.dialog = levelDialog{}
		return
	}

	// Enter confirms
	if rl.IsKeyPressed(rl.KeyEnter) {
		confirmDialog(state)
	}
}

func confirmDialog(state *editorState) {
	path := strings.TrimSpace(state.dialog.inputPath)
	if path == "" {
		state.dialog.errorText = "Path cannot be empty"
		return
	}

	switch state.dialog.mode {
	case dialogCreate:
		pkg := &worldMap.LevelPackage{
			Level:    worldMap.DefaultLevel(),
			BasePath: path,
		}
		if err := worldMap.SaveLevelPackage(pkg, path); err != nil {
			state.dialog.errorText = fmt.Sprintf("Create failed: %v", err)
			return
		}
		loadPackageIntoState(state, path)

	case dialogOpen:
		loadPackageIntoState(state, path)
		if state.dialog.mode != dialogNone {
			// loadPackageIntoState sets dialog to none on success; if still set, error occurred
			return
		}

	case dialogDelete:
		if err := os.RemoveAll(path); err != nil {
			state.dialog.errorText = fmt.Sprintf("Delete failed: %v", err)
			return
		}
		unloadPreviews(state.previews)
		state.previews = make(map[string]rl.Texture2D)
		state.level = worldMap.Level{}
		state.packagePath = ""
		state.availableAssets = nil
		state.selectedLibraryPath = ""
		state.statusText = fmt.Sprintf("Deleted package at %s", path)
		state.statusColor = rl.Maroon
		state.dialog = levelDialog{}
	}
}

func loadPackageIntoState(state *editorState, path string) {
	pkg, err := worldMap.LoadLevelPackage(path)
	if err != nil {
		state.dialog.errorText = fmt.Sprintf("Open failed: %v", err)
		return
	}

	assets, err := listAssetFiles("assets")
	if err != nil {
		state.dialog.errorText = fmt.Sprintf("Asset scan failed: %v", err)
		return
	}

	unloadPreviews(state.previews)
	state.previews = make(map[string]rl.Texture2D)
	state.level = pkg.Level
	state.packageMetadata = pkg.Metadata
	state.packagePath = pkg.BasePath
	state.availableAssets = assets
	if len(assets) > 0 {
		state.selectedLibraryPath = assets[0]
	} else {
		state.selectedLibraryPath = ""
	}
	state.panelScroll = 0
	state.statusText = fmt.Sprintf("Opened %s. 1/2/3 switches layer, Ctrl+S saves.", pkg.BasePath)
	state.statusColor = rl.DarkGreen
	state.dialog = levelDialog{}
}

func handleMetadataDialogInput(state *editorState) {
	// Tab or arrow keys to switch fields
	if rl.IsKeyPressed(rl.KeyTab) {
		state.dialog.metadataEditField = (state.dialog.metadataEditField + 1) % 4
	}
	if rl.IsKeyPressed(rl.KeyUp) {
		state.dialog.metadataEditField = (state.dialog.metadataEditField - 1 + 4) % 4
	}
	if rl.IsKeyPressed(rl.KeyDown) {
		state.dialog.metadataEditField = (state.dialog.metadataEditField + 1) % 4
	}

	// Append typed characters to current field
	for {
		ch := rl.GetCharPressed()
		if ch == 0 {
			break
		}
		if ch >= 32 {
			switch state.dialog.metadataEditField {
			case metaFieldName:
				if len(state.dialog.metadataName) < 60 {
					state.dialog.metadataName += string(rune(ch))
				}
			case metaFieldAuthor:
				if len(state.dialog.metadataAuthor) < 60 {
					state.dialog.metadataAuthor += string(rune(ch))
				}
			case metaFieldDesc:
				if len(state.dialog.metadataDesc) < 200 {
					state.dialog.metadataDesc += string(rune(ch))
				}
			case metaFieldVersion:
				if len(state.dialog.metadataVersion) < 20 {
					state.dialog.metadataVersion += string(rune(ch))
				}
			}
		}
	}

	// Backspace
	if rl.IsKeyPressed(rl.KeyBackspace) || rl.IsKeyPressedRepeat(rl.KeyBackspace) {
		switch state.dialog.metadataEditField {
		case metaFieldName:
			if len(state.dialog.metadataName) > 0 {
				state.dialog.metadataName = state.dialog.metadataName[:len(state.dialog.metadataName)-1]
			}
		case metaFieldAuthor:
			if len(state.dialog.metadataAuthor) > 0 {
				state.dialog.metadataAuthor = state.dialog.metadataAuthor[:len(state.dialog.metadataAuthor)-1]
			}
		case metaFieldDesc:
			if len(state.dialog.metadataDesc) > 0 {
				state.dialog.metadataDesc = state.dialog.metadataDesc[:len(state.dialog.metadataDesc)-1]
			}
		case metaFieldVersion:
			if len(state.dialog.metadataVersion) > 0 {
				state.dialog.metadataVersion = state.dialog.metadataVersion[:len(state.dialog.metadataVersion)-1]
			}
		}
	}

	// Escape cancels
	if rl.IsKeyPressed(rl.KeyEscape) {
		state.dialog = levelDialog{}
		return
	}

	// Enter confirms
	if rl.IsKeyPressed(rl.KeyEnter) {
		state.packageMetadata = worldMap.LevelPackageMetadata{
			Name:        state.dialog.metadataName,
			Author:      state.dialog.metadataAuthor,
			Description: state.dialog.metadataDesc,
			Version:     state.dialog.metadataVersion,
		}
		state.dialog = levelDialog{}
		state.statusText = "Metadata updated. Use Ctrl+S to save."
		state.statusColor = rl.DarkGreen
	}
}

func drawLevelDialog(state *editorState, layout layoutMetrics) {
	// Dimmed overlay
	rl.DrawRectangle(0, 0, int32(layout.screenWidth), int32(layout.screenHeight), rl.Fade(rl.Black, 0.45))

	// Dialog box
	dialogW := float32(620)
	dialogH := float32(220)
	dialogX := (float32(layout.screenWidth) - dialogW) / 2
	dialogY := (float32(layout.screenHeight) - dialogH) / 2
	box := rl.NewRectangle(dialogX, dialogY, dialogW, dialogH)

	rl.DrawRectangleRounded(box, 0.05, 6, rl.NewColor(245, 242, 235, 255))
	rl.DrawRectangleLinesEx(box, 2, rl.Fade(rl.Black, 0.3))

	var title string
	switch state.dialog.mode {
	case dialogCreate:
		title = "New Level — Enter directory path"
	case dialogOpen:
		title = "Open Level — Enter directory path"
	case dialogDelete:
		title = "Delete Level — Confirm directory path"
	}

	drawTextLine(title, rl.NewVector2(dialogX+20, dialogY+18), 22, rl.Black)

	// Input field
	inputRect := rl.NewRectangle(dialogX+20, dialogY+62, dialogW-40, 44)
	rl.DrawRectangleRounded(inputRect, 0.1, 4, rl.White)
	rl.DrawRectangleLinesEx(inputRect, 2, rl.Fade(rl.Black, 0.4))
	cursor := "|"
	if int(rl.GetTime()*2)%2 == 0 {
		cursor = ""
	}
	drawTextLine(state.dialog.inputPath+cursor, rl.NewVector2(inputRect.X+10, inputRect.Y+10), 20, rl.Black)

	if state.dialog.errorText != "" {
		drawTextLine(state.dialog.errorText, rl.NewVector2(dialogX+20, dialogY+118), 18, rl.Maroon)
	}

	// Confirm / Cancel buttons
	confirmRect := rl.NewRectangle(dialogX+dialogW-220, dialogY+dialogH-58, 100, 38)
	cancelRect := rl.NewRectangle(dialogX+dialogW-112, dialogY+dialogH-58, 100, 38)

	confirmColor := rl.NewColor(200, 240, 200, 255)
	if state.dialog.mode == dialogDelete {
		confirmColor = rl.NewColor(255, 210, 210, 255)
	}
	rl.DrawRectangleRounded(confirmRect, 0.18, 4, confirmColor)
	rl.DrawRectangleLinesEx(confirmRect, 2, rl.Fade(rl.Black, 0.2))
	confirmLabel := "Create"
	if state.dialog.mode == dialogOpen {
		confirmLabel = "Open"
	} else if state.dialog.mode == dialogDelete {
		confirmLabel = "Delete"
	}
	drawCenteredText(confirmLabel, confirmRect, 19, rl.Black)

	rl.DrawRectangleRounded(cancelRect, 0.18, 4, rl.NewColor(255, 248, 240, 255))
	rl.DrawRectangleLinesEx(cancelRect, 2, rl.Fade(rl.Black, 0.2))
	drawCenteredText("Cancel", cancelRect, 19, rl.Black)

	drawTextLine("Enter to confirm • Esc to cancel", rl.NewVector2(dialogX+20, dialogY+dialogH-44), 16, rl.DarkGray)

	// Handle button clicks
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		mouse := rl.GetMousePosition()
		if rl.CheckCollisionPointRec(mouse, confirmRect) {
			confirmDialog(state)
		}
		if rl.CheckCollisionPointRec(mouse, cancelRect) {
			state.dialog = levelDialog{}
		}
	}
}

func drawMetadataDialog(state *editorState, layout layoutMetrics) {
	// Dimmed overlay
	rl.DrawRectangle(0, 0, int32(layout.screenWidth), int32(layout.screenHeight), rl.Fade(rl.Black, 0.45))

	// Dialog box
	dialogW := float32(620)
	dialogH := float32(380)
	dialogX := (float32(layout.screenWidth) - dialogW) / 2
	dialogY := (float32(layout.screenHeight) - dialogH) / 2
	box := rl.NewRectangle(dialogX, dialogY, dialogW, dialogH)

	rl.DrawRectangleRounded(box, 0.05, 6, rl.NewColor(245, 242, 235, 255))
	rl.DrawRectangleLinesEx(box, 2, rl.Fade(rl.Black, 0.3))

	drawTextLine("Edit Package Metadata", rl.NewVector2(dialogX+20, dialogY+18), 22, rl.Black)

	// Input fields
	fieldY := dialogY + 60
	fieldHeight := float32(50)
	fieldSpacing := float32(5)

	fields := []struct {
		label string
		value *string
		index int
	}{
		{"Name:", &state.dialog.metadataName, metaFieldName},
		{"Author:", &state.dialog.metadataAuthor, metaFieldAuthor},
		{"Description:", &state.dialog.metadataDesc, metaFieldDesc},
		{"Version:", &state.dialog.metadataVersion, metaFieldVersion},
	}

	for i, field := range fields {
		y := fieldY + float32(i)*(fieldHeight+fieldSpacing)

		// Label
		drawTextLine(field.label, rl.NewVector2(dialogX+20, y), 16, rl.DarkGray)

		// Input field
		inputRect := rl.NewRectangle(dialogX+120, y, dialogW-140, 32)

		// Highlight if selected
		inputColor := rl.White
		if state.dialog.metadataEditField == field.index {
			inputColor = rl.NewColor(220, 240, 255, 255)
		}

		rl.DrawRectangleRounded(inputRect, 0.08, 4, inputColor)
		rl.DrawRectangleLinesEx(inputRect, 2, rl.Fade(rl.Black, 0.3))

		// Text with cursor
		displayText := *field.value
		if state.dialog.metadataEditField == field.index {
			if int(rl.GetTime()*2)%2 == 0 {
				displayText += "|"
			}
		}

		drawTextLine(displayText, rl.NewVector2(inputRect.X+8, inputRect.Y+6), 14, rl.Black)
	}

	drawTextLine("Tab/↑↓ to switch fields • Enter to save • Esc to cancel", rl.NewVector2(dialogX+20, dialogY+dialogH-44), 14, rl.DarkGray)

	// Confirm / Cancel buttons
	confirmRect := rl.NewRectangle(dialogX+dialogW-220, dialogY+dialogH-48, 100, 36)
	cancelRect := rl.NewRectangle(dialogX+dialogW-112, dialogY+dialogH-48, 100, 36)

	rl.DrawRectangleRounded(confirmRect, 0.18, 4, rl.NewColor(200, 240, 200, 255))
	rl.DrawRectangleLinesEx(confirmRect, 2, rl.Fade(rl.Black, 0.2))
	drawCenteredText("Save", confirmRect, 18, rl.Black)

	rl.DrawRectangleRounded(cancelRect, 0.18, 4, rl.NewColor(255, 248, 240, 255))
	rl.DrawRectangleLinesEx(cancelRect, 2, rl.Fade(rl.Black, 0.2))
	drawCenteredText("Cancel", cancelRect, 18, rl.Black)

	// Handle button clicks
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		mouse := rl.GetMousePosition()
		if rl.CheckCollisionPointRec(mouse, confirmRect) {
			state.packageMetadata = worldMap.LevelPackageMetadata{
				Name:        state.dialog.metadataName,
				Author:      state.dialog.metadataAuthor,
				Description: state.dialog.metadataDesc,
				Version:     state.dialog.metadataVersion,
			}
			state.dialog = levelDialog{}
			state.statusText = "Metadata updated. Use Ctrl+S to save."
			state.statusColor = rl.DarkGreen
		}
		if rl.CheckCollisionPointRec(mouse, cancelRect) {
			state.dialog = levelDialog{}
		}
	}
}

func setActiveLayer(state *editorState, layer worldMap.AssetLayer) {
	state.activeLayer = layer
	state.statusText = fmt.Sprintf("Editing %s", strings.ToLower(layerLabel(layer)))
	state.statusColor = rl.DarkGray
	state.panelScroll = 0
}

func listAssetFiles(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		switch ext {
		case ".png", ".jpg", ".jpeg", ".bmp", ".gif", ".webp":
			files = append(files, filepath.ToSlash(filepath.Join(root, entry.Name())))
		}
	}

	sort.Strings(files)
	return files, nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}

func fitRect(bounds rl.Rectangle, sourceWidth, sourceHeight float32) rl.Rectangle {
	if sourceWidth == 0 || sourceHeight == 0 {
		return bounds
	}

	scale := bounds.Width / sourceWidth
	if candidate := bounds.Height / sourceHeight; candidate < scale {
		scale = candidate
	}

	width := sourceWidth * scale
	height := sourceHeight * scale
	x := bounds.X + (bounds.Width-width)/2
	y := bounds.Y + (bounds.Height-height)/2

	return rl.NewRectangle(x, y, width, height)
}

func trimTextToWidth(text string, width float32, size float32) string {
	font := uiFont
	if font.Texture.ID == 0 {
		font = rl.GetFontDefault()
	}

	if rl.MeasureTextEx(font, text, size, 1).X <= width {
		return text
	}

	trimmed := text
	for len(trimmed) > 3 {
		trimmed = trimmed[:len(trimmed)-1]
		candidate := trimmed + "..."
		if rl.MeasureTextEx(font, candidate, size, 1).X <= width {
			return candidate
		}
	}

	return text
}

func wrapTextToWidth(text string, width float32, size float32) []string {
	font := uiFont
	if font.Texture.ID == 0 {
		font = rl.GetFontDefault()
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	lines := make([]string, 0, 2)
	current := words[0]

	for _, word := range words[1:] {
		candidate := current + " " + word
		if rl.MeasureTextEx(font, candidate, size, 1).X <= width {
			current = candidate
			continue
		}

		lines = append(lines, current)
		current = word
	}

	lines = append(lines, current)
	return lines
}

func rowCount(items int) int {
	if items <= 0 {
		return 0
	}

	rows := items / panelColumns
	if items%panelColumns != 0 {
		rows++
	}

	return rows
}

func clampFloat32(value, minimum, maximum float32) float32 {
	if value < minimum {
		return minimum
	}

	if value > maximum {
		return maximum
	}

	return value
}

func maxFloat32(left, right float32) float32 {
	if left > right {
		return left
	}

	return right
}

func ctrlPressed() bool {
	return rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
}
