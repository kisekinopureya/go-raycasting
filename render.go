package main

import (
	"math"

	"kisekinopureya.com.tr/go-raycasting/worldMap"
)

func renderFrame(state *GameState) {
	for i := range state.pixels {
		state.pixels[i] = 0
	}

	renderSkybox(state)
	renderFloors(state)
	renderCeilings(state)
	renderWalls(state)
}

func renderSkybox(state *GameState) {
	skyTex := state.textures.Skybox
	for x := 0; x < screenWidth; x++ {
		cameraX := 2*float64(x)/float64(screenWidth) - 1
		rayDirX := dirX + planeX*cameraX
		rayDirY := dirY + planeY*cameraX

		angle := math.Atan2(rayDirY, rayDirX)
		u := int((angle/(2*math.Pi) + 0.5) * float64(texWidth))

		for y := 0; y < screenHeight/2; y++ {
			v := int(float64(y) / float64(screenHeight/2) * float64(texHeight))
			color := skyTex[texWidth*v+(u&(texWidth-1))]
			state.pixels[y*screenWidth+x] = color
		}
	}
}

func renderFloors(state *GameState) {
	rayDirX0 := dirX - planeX
	rayDirY0 := dirY - planeY
	rayDirX1 := dirX + planeX
	rayDirY1 := dirY + planeY

	halfHeight := screenHeight / 2
	cameraZ := 0.5
	screenHalf := float64(screenHeight) / 2
	screenHeightF := float64(screenHeight)

	planeDelta := 0.0 - cameraZ
	if planeDelta < 0 {
		for y := halfHeight + 1; y < screenHeight; y++ {
			denominator := screenHalf - float64(y)
			if denominator == 0 {
				continue
			}

			rowDistance := (planeDelta * screenHeightF) / denominator
			if rowDistance <= 0 || math.IsInf(rowDistance, 0) || math.IsNaN(rowDistance) {
				continue
			}

			floorStepX := rowDistance * (rayDirX1 - rayDirX0) / float64(screenWidth)
			floorStepY := rowDistance * (rayDirY1 - rayDirY0) / float64(screenWidth)

			worldX := posX + rowDistance*rayDirX0
			worldY := posY + rowDistance*rayDirY0

			for x := 0; x < screenWidth; x++ {
				cellX := int(worldX)
				cellY := int(worldY)

				if cellX < 0 || cellX >= state.mapWidth || cellY < 0 || cellY >= state.mapHeight {
					worldX += floorStepX
					worldY += floorStepY
					continue
				}

				tx := int(texWidth*(worldX-float64(cellX))) & (texWidth - 1)
				ty := int(texHeight*(worldY-float64(cellY))) & (texHeight - 1)

				worldX += floorStepX
				worldY += floorStepY

				floorTile := worldMap.TileAt(state.floorMap, cellX, cellY)
				if floorTile <= 0 {
					continue
				}
				floorTexSlot := state.level.FloorTextureSlot(floorTile)
				if floorTexSlot < 0 || floorTexSlot >= len(state.textures.Floors) {
					continue
				}
				floorTex := state.textures.Floors[floorTexSlot]
				floorColor := floorTex[texWidth*ty+tx]
				floorColor = (floorColor >> 1) & 8355711
				state.pixels[y*screenWidth+x] = floorColor
			}
		}
	}
}

func renderCeilings(state *GameState) {
	rayDirX0 := dirX - planeX
	rayDirY0 := dirY - planeY
	rayDirX1 := dirX + planeX
	rayDirY1 := dirY + planeY

	halfHeight := screenHeight / 2
	cameraZ := 0.5
	screenHalf := float64(screenHeight) / 2
	screenHeightF := float64(screenHeight)

	planeDelta := 1.0 - cameraZ
	if planeDelta > 0 {
		for y := 0; y < halfHeight; y++ {
			denominator := screenHalf - float64(y)
			if denominator == 0 {
				continue
			}

			rowDistance := (planeDelta * screenHeightF) / denominator
			if rowDistance <= 0 || math.IsInf(rowDistance, 0) || math.IsNaN(rowDistance) {
				continue
			}

			ceilStepX := rowDistance * (rayDirX1 - rayDirX0) / float64(screenWidth)
			ceilStepY := rowDistance * (rayDirY1 - rayDirY0) / float64(screenWidth)

			worldX := posX + rowDistance*rayDirX0
			worldY := posY + rowDistance*rayDirY0

			for x := 0; x < screenWidth; x++ {
				cellX := int(worldX)
				cellY := int(worldY)

				if cellX < 0 || cellX >= state.mapWidth || cellY < 0 || cellY >= state.mapHeight {
					worldX += ceilStepX
					worldY += ceilStepY
					continue
				}

				tx := int(texWidth*(worldX-float64(cellX))) & (texWidth - 1)
				ty := int(texHeight*(worldY-float64(cellY))) & (texHeight - 1)

				worldX += ceilStepX
				worldY += ceilStepY

				ceilingTile := worldMap.TileAt(state.ceilingMap, cellX, cellY)
				if state.level.IsSkybox(ceilingTile) || ceilingTile <= 0 {
					continue
				}

				ceilTexSlot := state.level.CeilingTextureSlot(ceilingTile)
				if ceilTexSlot < 0 || ceilTexSlot >= len(state.textures.Ceilings) {
					continue
				}
				ceilTex := state.textures.Ceilings[ceilTexSlot]
				ceilColor := ceilTex[texWidth*ty+tx]
				ceilColor = (ceilColor >> 1) & 8355711
				state.pixels[y*screenWidth+x] = ceilColor
			}
		}
	}
}

func renderWalls(state *GameState) {
	for x := 0; x < screenWidth; x++ {
		renderWallColumn(state, x)
	}
}

func renderWallColumn(state *GameState, x int) {
	cameraX := 2*float64(x)/float64(screenWidth) - 1
	rayDirX := dirX + planeX*cameraX
	rayDirY := dirY + planeY*cameraX

	mapX := int(posX)
	mapY := int(posY)

	deltaDistX := math.Abs(1 / rayDirX)
	deltaDistY := math.Abs(1 / rayDirY)

	var sideDistX, sideDistY float64
	var stepX, stepY int
	var side int

	if rayDirX < 0 {
		stepX = -1
		sideDistX = (posX - float64(mapX)) * deltaDistX
	} else {
		stepX = 1
		sideDistX = (float64(mapX) + 1 - posX) * deltaDistX
	}

	if rayDirY < 0 {
		stepY = -1
		sideDistY = (posY - float64(mapY)) * deltaDistY
	} else {
		stepY = 1
		sideDistY = (float64(mapY) + 1 - posY) * deltaDistY
	}

	hit := false
	for !hit {
		if sideDistX < sideDistY {
			sideDistX += deltaDistX
			mapX += stepX
			side = 0
		} else {
			sideDistY += deltaDistY
			mapY += stepY
			side = 1
		}

		if mapX < 0 || mapX >= state.mapWidth || mapY < 0 || mapY >= state.mapHeight {
			return
		}

		if state.level.IsWall(worldMap.TileAt(state.wallMap, mapX, mapY)) {
			hit = true
		}
	}

	perpWallDist := calculatePerpDistance(side, sideDistX, sideDistY, deltaDistX, deltaDistY)
	if perpWallDist <= 0 {
		return
	}

	texX := calculateTexX(side, rayDirX, rayDirY, mapX, mapY, perpWallDist)
	drawColumn(state, x, side, mapX, mapY, perpWallDist, texX)
}

func calculatePerpDistance(side int, sideDistX, sideDistY, deltaDistX, deltaDistY float64) float64 {
	var perpWallDist float64
	if side == 0 {
		perpWallDist = sideDistX - deltaDistX
	} else {
		perpWallDist = sideDistY - deltaDistY
	}

	const minDist = 0.3
	if perpWallDist < minDist {
		perpWallDist = minDist
	}
	return perpWallDist
}

func calculateTexX(side int, rayDirX, rayDirY float64, mapX, mapY int, perpWallDist float64) int {
	var wallX float64
	if side == 0 {
		wallX = posY + perpWallDist*rayDirY
	} else {
		wallX = posX + perpWallDist*rayDirX
	}
	wallX -= math.Floor(wallX)

	texX := int(wallX * float64(texWidth))
	if side == 0 && rayDirX > 0 {
		texX = texWidth - texX - 1
	}
	if side == 1 && rayDirY < 0 {
		texX = texWidth - texX - 1
	}
	return texX
}

func drawColumn(state *GameState, x int, side int, mapX, mapY int, perpWallDist float64, texX int) {
	wallTile := worldMap.TileAt(state.wallMap, mapX, mapY)
	if !state.level.IsWall(wallTile) {
		return
	}

	cameraZ := 0.5
	unclippedStart := int(float64(screenHeight)/2 - (1.0-cameraZ)/perpWallDist*float64(screenHeight))
	unclippedEnd := int(float64(screenHeight)/2 - (0.0-cameraZ)/perpWallDist*float64(screenHeight))

	if unclippedEnd < 0 || unclippedStart >= screenHeight {
		return
	}

	drawStart := unclippedStart
	drawEnd := unclippedEnd
	if drawStart < 0 {
		drawStart = 0
	}
	if drawEnd >= screenHeight {
		drawEnd = screenHeight - 1
	}
	if drawEnd <= drawStart {
		return
	}

	texNum := state.level.WallTextureSlot(wallTile)
	texPixels := state.textures.Walls[texNum]
	projectedHeight := unclippedEnd - unclippedStart + 1
	if projectedHeight <= 0 {
		return
	}
	if texX < 0 {
		texX = 0
	}
	if texX >= texWidth {
		texX = texWidth - 1
	}

	step := float64(texHeight) / float64(projectedHeight)
	texPos := float64(drawStart-unclippedStart) * step

	for y := drawStart; y <= drawEnd; y++ {
		texY := int(texPos)
		texPos += step
		if texY < 0 {
			texY = 0
		}
		if texY >= texHeight {
			texY = texHeight - 1
		}

		color := texPixels[texHeight*texY+texX]
		if side == 1 {
			color = (color >> 1) & 8355711
		}

		state.pixels[y*screenWidth+x] = color
	}
}
