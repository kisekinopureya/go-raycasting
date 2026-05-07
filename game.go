package main

import (
	"kisekinopureya.com.tr/go-raycasting/worldMap"
)

type GameState struct {
	level      worldMap.Level
	wallMap    [][]int
	floorMap   [][]int
	ceilingMap [][]int
	mapWidth   int
	mapHeight  int
	textures   *worldMap.TextureSet
	pixels     []uint32
}

func initGame(levelPath string) (*GameState, error) {
	pkg, err := worldMap.LoadLevelPackage(levelPath)
	if err != nil {
		return nil, err
	}
	level := pkg.Level

	baseFloor, err := level.GetFloor(0)
	if err != nil {
		return nil, err
	}

	textures, err := worldMap.LoadTextures(level)
	if err != nil {
		return nil, err
	}

	return &GameState{
		level:      level,
		wallMap:    baseFloor.Walls,
		floorMap:   baseFloor.Floors,
		ceilingMap: baseFloor.Ceilings,
		mapWidth:   level.Width(),
		mapHeight:  level.Height(),
		textures:   &textures,
		pixels:     make([]uint32, screenWidth*screenHeight),
	}, nil
}

func cleanupGame(state *GameState) {
	if state == nil {
		return
	}
}

func createWalkabilityChecker(state *GameState) func(float64, float64) bool {
	return func(x, y float64) bool {
		cellX := int(x)
		cellY := int(y)
		if cellX < 0 || cellX >= state.mapWidth || cellY < 0 || cellY >= state.mapHeight {
			return false
		}
		return state.level.IsWalkable(worldMap.TileAt(state.wallMap, cellX, cellY))
	}
}
