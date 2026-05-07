package worldMap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
)

type AssetLayer int

const (
	SkyboxTile       = -1
	AssetLayerWalls AssetLayer = iota
	AssetLayerFloors
	AssetLayerCeilings
)

type AssetCatalog struct {
	Walls    []string `json:"walls"`
	Floors   []string `json:"floors"`
	Ceilings []string `json:"ceilings"`
	Skybox   string   `json:"skybox"`
}

type FloorData struct {
	Walls    [][]int `json:"walls"`
	Floors   [][]int `json:"floors"`
	Ceilings [][]int `json:"ceilings"`
}

type Level struct {
	Assets      AssetCatalog `json:"assets"`
	Walls       [][]int      `json:"walls"`
	Floors      [][]int      `json:"floors"`
	Ceilings    [][]int      `json:"ceilings"`
	FloorLayers []FloorData  `json:"floorLayers"`
}

type TextureSet struct {
	Walls    [][]uint32
	Floors   [][]uint32
	Ceilings [][]uint32
	Skybox   []uint32
}

func DefaultAssets() AssetCatalog {
	return AssetCatalog{
		Walls: []string{
			"assets/wall1.png",
			"assets/wall2.png",
			"assets/wall3.png",
			"assets/wall4.png",
		},
		Floors: []string{
			"assets/floor1.png",
			"assets/floor2.png",
			"assets/floor3.png",
			"assets/floor4.png",
		},
		Ceilings: []string{
			"assets/ceiling1.png",
			"assets/ceiling2.png",
			"assets/ceiling3.png",
			"assets/ceiling4.png",
		},
		Skybox: "assets/sky.png",
	}
}

func DefaultLevel() Level {
	return Level{
		Assets: DefaultAssets(),
		Walls: [][]int{
			{1, 1, 1, 1, 1, 1, 1, 1},
			{1, 0, 0, 0, 0, 0, 0, 1},
			{1, 0, 0, 0, 0, 0, 0, 1},
			{1, 0, 0, 0, 0, 0, 0, 1},
			{1, 1, 1, 0, 0, 1, 1, 1},
			{1, 0, 0, 0, 0, 0, 0, 1},
			{1, 0, 2, 0, 0, 3, 0, 1},
			{1, 0, 0, 0, 0, 0, 0, 1},
			{1, 0, 0, 0, 0, 0, 0, 1},
			{1, 0, 0, 0, 0, 0, 0, 1},
			{1, 0, 0, 0, 0, 0, 0, 1},
			{1, 1, 1, 1, 1, 1, 1, 1},
		},
		Floors: [][]int{
			{4, 4, 4, 4, 4, 4, 4, 4},
			{4, 4, 4, 4, 4, 4, 4, 4},
			{4, 4, 4, 4, 4, 4, 4, 4},
			{4, 4, 4, 4, 4, 4, 4, 4},
			{1, 2, 2, 2, 2, 2, 2, 1},
			{1, 2, 2, 2, 2, 2, 2, 1},
			{1, 2, 3, 3, 3, 3, 2, 1},
			{1, 2, 3, 3, 3, 3, 2, 1},
			{1, 1, 1, 3, 3, 1, 1, 1},
			{1, 1, 1, 1, 1, 1, 1, 1},
			{1, 1, 1, 1, 1, 1, 1, 1},
			{1, 1, 1, 1, 1, 1, 1, 1},
		},
		Ceilings: [][]int{
			{SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile},
			{SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile},
			{SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile},
			{SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile},
			{SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile},
			{SkyboxTile, 1, 1, 1, 1, 1, 1, SkyboxTile},
			{SkyboxTile, 2, 3, 3, 3, 4, 2, SkyboxTile},
			{SkyboxTile, 2, 3, 3, 3, 4, 2, SkyboxTile},
			{SkyboxTile, 1, 1, 3, 3, 1, 1, SkyboxTile},
			{SkyboxTile, 1, 1, 1, 1, 1, 1, SkyboxTile},
			{SkyboxTile, 1, 1, 1, 1, 1, 1, SkyboxTile},
			{SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile, SkyboxTile},
		},
	}
}

func GetLevel() Level {
	return DefaultLevel()
}

func (assets AssetCatalog) isZero() bool {
	return len(assets.Walls) == 0 && len(assets.Floors) == 0 && len(assets.Ceilings) == 0 && assets.Skybox == ""
}

func (level Level) Width() int {
	return len(level.Walls)
}

func (level Level) Height() int {
	if len(level.Walls) == 0 {
		return 0
	}

	return len(level.Walls[0])
}

func (level Level) GetFloorCount() int {
	return 1 + len(level.FloorLayers)
}

func (level Level) GetFloor(floorIndex int) (FloorData, error) {
	if floorIndex < 0 || floorIndex >= level.GetFloorCount() {
		return FloorData{}, fmt.Errorf("floor index %d out of range", floorIndex)
	}
	if floorIndex == 0 {
		return FloorData{
			Walls:    level.Walls,
			Floors:   level.Floors,
			Ceilings: level.Ceilings,
		}, nil
	}
	return level.FloorLayers[floorIndex-1], nil
}

func (level *Level) SetFloor(floorIndex int, floor FloorData) error {
	if floorIndex < 0 || floorIndex >= level.GetFloorCount() {
		return fmt.Errorf("floor index %d out of range", floorIndex)
	}
	if floorIndex == 0 {
		level.Walls = floor.Walls
		level.Floors = floor.Floors
		level.Ceilings = floor.Ceilings
		return nil
	}
	level.FloorLayers[floorIndex-1] = floor
	return nil
}

func (level *Level) AddFloor() {
	h := level.Height()
	w := level.Width()
	newFloor := FloorData{
		Walls:    make([][]int, w),
		Floors:   make([][]int, w),
		Ceilings: make([][]int, w),
	}
	for x := 0; x < w; x++ {
		newFloor.Walls[x] = make([]int, h)
		newFloor.Floors[x] = make([]int, h)
		newFloor.Ceilings[x] = make([]int, h)
	}
	level.FloorLayers = append(level.FloorLayers, newFloor)
}

func (level *Level) RemoveFloor(floorIndex int) error {
	if floorIndex <= 0 || floorIndex >= level.GetFloorCount() {
		return fmt.Errorf("cannot remove floor %d (min: 1, max: %d)", floorIndex, level.GetFloorCount()-1)
	}
	level.FloorLayers = append(level.FloorLayers[:floorIndex-1], level.FloorLayers[floorIndex:]...)
	return nil
}

func (level Level) IsWall(tile int) bool {
	return tile > 0 && tile <= len(level.Assets.Walls)
}

func (level Level) IsFloorTile(tile int) bool {
	return tile > 0 && tile <= len(level.Assets.Floors)
}

func (level Level) IsCeilingTile(tile int) bool {
	return tile > 0 && tile <= len(level.Assets.Ceilings)
}

func (level Level) IsSkybox(tile int) bool {
	return tile == SkyboxTile
}

func (level Level) IsWalkable(tile int) bool {
	return !level.IsWall(tile)
}

func (level Level) FloorTextureSlot(tile int) int {
	if len(level.Assets.Floors) == 0 {
		return -1
	}

	if level.IsFloorTile(tile) {
		return tile - 1
	}

	return 0
}

func (level Level) CeilingTextureSlot(tile int) int {
	if len(level.Assets.Ceilings) == 0 {
		return -1
	}

	if level.IsCeilingTile(tile) {
		return tile - 1
	}

	return 0
}

func (level Level) WallTextureSlot(tile int) int {
	if !level.IsWall(tile) {
		return -1
	}

	return tile - 1
}

func (level Level) AssetPaths(layer AssetLayer) []string {
	var source []string

	switch layer {
	case AssetLayerWalls:
		source = level.Assets.Walls
	case AssetLayerFloors:
		source = level.Assets.Floors
	case AssetLayerCeilings:
		source = level.Assets.Ceilings
	default:
		return nil
	}

	paths := make([]string, len(source))
	copy(paths, source)
	return paths
}

func (level *Level) AddAsset(layer AssetLayer, path string) int {
	switch layer {
	case AssetLayerWalls:
		level.Assets.Walls = append(level.Assets.Walls, path)
		return len(level.Assets.Walls)
	case AssetLayerFloors:
		level.Assets.Floors = append(level.Assets.Floors, path)
		return len(level.Assets.Floors)
	case AssetLayerCeilings:
		level.Assets.Ceilings = append(level.Assets.Ceilings, path)
		return len(level.Assets.Ceilings)
	default:
		return 0
	}
}

func (level *Level) ReplaceAsset(layer AssetLayer, slot int, path string) error {
	assets := level.assetSlice(layer)
	if slot < 0 || slot >= len(*assets) {
		return fmt.Errorf("asset slot %d out of range", slot)
	}

	(*assets)[slot] = path
	return nil
}

func (level *Level) SetSkybox(path string) {
	level.Assets.Skybox = path
}

func (level *Level) AddColumn() {
	h := level.Height()
	wallCol := make([]int, h)
	floorCol := make([]int, h)
	ceilingCol := make([]int, h)
	for y := 0; y < h; y++ {
		floorCol[y] = 1
		ceilingCol[y] = SkyboxTile
	}
	level.Walls = append(level.Walls, wallCol)
	level.Floors = append(level.Floors, floorCol)
	level.Ceilings = append(level.Ceilings, ceilingCol)

	for i := range level.FloorLayers {
		level.FloorLayers[i].Walls = append(level.FloorLayers[i].Walls, make([]int, h))
		level.FloorLayers[i].Floors = append(level.FloorLayers[i].Floors, make([]int, h))
		level.FloorLayers[i].Ceilings = append(level.FloorLayers[i].Ceilings, make([]int, h))
	}
}

func (level *Level) RemoveColumn() error {
	if level.Width() <= 3 {
		return fmt.Errorf("map must be at least 3 columns wide")
	}
	n := len(level.Walls) - 1
	level.Walls = level.Walls[:n]
	level.Floors = level.Floors[:n]
	level.Ceilings = level.Ceilings[:n]

	for i := range level.FloorLayers {
		level.FloorLayers[i].Walls = level.FloorLayers[i].Walls[:n]
		level.FloorLayers[i].Floors = level.FloorLayers[i].Floors[:n]
		level.FloorLayers[i].Ceilings = level.FloorLayers[i].Ceilings[:n]
	}
	return nil
}

func (level *Level) AddRow() {
	for x := range level.Walls {
		level.Walls[x] = append(level.Walls[x], 0)
		level.Floors[x] = append(level.Floors[x], 1)
		level.Ceilings[x] = append(level.Ceilings[x], SkyboxTile)
	}

	for i := range level.FloorLayers {
		for x := range level.FloorLayers[i].Walls {
			level.FloorLayers[i].Walls[x] = append(level.FloorLayers[i].Walls[x], 0)
			level.FloorLayers[i].Floors[x] = append(level.FloorLayers[i].Floors[x], 0)
			level.FloorLayers[i].Ceilings[x] = append(level.FloorLayers[i].Ceilings[x], 0)
		}
	}
}

func (level *Level) RemoveRow() error {
	if level.Height() <= 3 {
		return fmt.Errorf("map must be at least 3 rows tall")
	}
	for x := range level.Walls {
		h := len(level.Walls[x]) - 1
		level.Walls[x] = level.Walls[x][:h]
		level.Floors[x] = level.Floors[x][:h]
		level.Ceilings[x] = level.Ceilings[x][:h]
	}

	for i := range level.FloorLayers {
		for x := range level.FloorLayers[i].Walls {
			h := len(level.FloorLayers[i].Walls[x]) - 1
			level.FloorLayers[i].Walls[x] = level.FloorLayers[i].Walls[x][:h]
			level.FloorLayers[i].Floors[x] = level.FloorLayers[i].Floors[x][:h]
			level.FloorLayers[i].Ceilings[x] = level.FloorLayers[i].Ceilings[x][:h]
		}
	}
	return nil
}

func (level *Level) RemoveAsset(layer AssetLayer, slot int) error {
	assets := level.assetSlice(layer)
	if slot < 0 || slot >= len(*assets) {
		return fmt.Errorf("asset slot %d out of range", slot)
	}

	if len(*assets) == 1 {
		return fmt.Errorf("at least one asset is required for this layer")
	}

	*assets = append((*assets)[:slot], (*assets)[slot+1:]...)
	level.shiftLayerTiles(layer, slot+1)
	return nil
}

func (level *Level) assetSlice(layer AssetLayer) *[]string {
	switch layer {
	case AssetLayerWalls:
		return &level.Assets.Walls
	case AssetLayerFloors:
		return &level.Assets.Floors
	default:
		return &level.Assets.Ceilings
	}
}

func (level *Level) shiftLayerTiles(layer AssetLayer, removedTile int) {
	var tiles [][]int

	switch layer {
	case AssetLayerWalls:
		tiles = level.Walls
	case AssetLayerFloors:
		tiles = level.Floors
	default:
		tiles = level.Ceilings
	}

	for x := range tiles {
		for y := range tiles[x] {
			current := tiles[x][y]
			if current == SkyboxTile {
				continue
			}

			if current == removedTile {
				tiles[x][y] = 0
				continue
			}

			if current > removedTile {
				tiles[x][y] = current - 1
			}
		}
	}
}

func (level Level) Validate() error {
	if len(level.Assets.Walls) == 0 {
		return fmt.Errorf("at least one wall asset is required")
	}

	if len(level.Assets.Floors) == 0 {
		return fmt.Errorf("at least one floor asset is required")
	}

	if len(level.Assets.Ceilings) == 0 {
		return fmt.Errorf("at least one ceiling asset is required")
	}

	if level.Assets.Skybox == "" {
		return fmt.Errorf("a skybox asset is required")
	}

	if len(level.Walls) == 0 || len(level.Floors) == 0 || len(level.Ceilings) == 0 {
		return fmt.Errorf("level layers must not be empty")
	}

	if len(level.Walls) != len(level.Floors) || len(level.Walls) != len(level.Ceilings) {
		return fmt.Errorf("level layers must have the same width")
	}

	for _, path := range level.Assets.Walls {
		if path == "" {
			return fmt.Errorf("wall asset paths must not be empty")
		}
	}

	for _, path := range level.Assets.Floors {
		if path == "" {
			return fmt.Errorf("floor asset paths must not be empty")
		}
	}

	for _, path := range level.Assets.Ceilings {
		if path == "" {
			return fmt.Errorf("ceiling asset paths must not be empty")
		}
	}

	for x := range level.Walls {
		columnHeight := len(level.Walls[x])
		if columnHeight == 0 {
			return fmt.Errorf("level columns must not be empty")
		}

		if len(level.Floors[x]) != columnHeight || len(level.Ceilings[x]) != columnHeight {
			return fmt.Errorf("level layers must have matching column heights")
		}

		for y := 0; y < columnHeight; y++ {
			wallTile := level.Walls[x][y]
			floorTile := level.Floors[x][y]
			ceilingTile := level.Ceilings[x][y]

			if wallTile < 0 || wallTile > len(level.Assets.Walls) {
				return fmt.Errorf("invalid wall tile %d at %d,%d", wallTile, x, y)
			}

			if floorTile < 0 || floorTile > len(level.Assets.Floors) {
				return fmt.Errorf("invalid floor tile %d at %d,%d", floorTile, x, y)
			}

			if ceilingTile != SkyboxTile && (ceilingTile < 0 || ceilingTile > len(level.Assets.Ceilings)) {
				return fmt.Errorf("invalid ceiling tile %d at %d,%d", ceilingTile, x, y)
			}
		}
	}

	return nil
}

func LoadLevel(path string) (Level, error) {
	file, err := os.Open(path)
	if err != nil {
		return Level{}, err
	}
	defer file.Close()

	var level Level
	if err := json.NewDecoder(file).Decode(&level); err != nil {
		return Level{}, err
	}

	if level.Assets.isZero() {
		level.Assets = DefaultAssets()
		migrateLegacyTiles(&level)
	}

	if err := level.Validate(); err != nil {
		return Level{}, err
	}

	return level, nil
}

func LoadLevelOrDefault(path string) (Level, error) {
	level, err := LoadLevel(path)
	if err == nil {
		return level, nil
	}

	if os.IsNotExist(err) {
		return DefaultLevel(), nil
	}

	return Level{}, err
}

func SaveLevel(path string, level Level) error {
	if err := level.Validate(); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	return encoder.Encode(level)
}

func TileAt(layer [][]int, x, y int) int {
	if x < 0 || x >= len(layer) {
		return 0
	}

	if y < 0 || y >= len(layer[x]) {
		return 0
	}

	return layer[x][y]
}

func LoadTextures(level Level) (TextureSet, error) {
	textures := TextureSet{
		Walls:    make([][]uint32, len(level.Assets.Walls)),
		Floors:   make([][]uint32, len(level.Assets.Floors)),
		Ceilings: make([][]uint32, len(level.Assets.Ceilings)),
	}

	for index, path := range level.Assets.Walls {
		pixels, err := loadTexturePixels(path)
		if err != nil {
			return TextureSet{}, err
		}
		textures.Walls[index] = pixels
	}

	for index, path := range level.Assets.Floors {
		pixels, err := loadTexturePixels(path)
		if err != nil {
			return TextureSet{}, err
		}
		textures.Floors[index] = pixels
	}

	for index, path := range level.Assets.Ceilings {
		pixels, err := loadTexturePixels(path)
		if err != nil {
			return TextureSet{}, err
		}
		textures.Ceilings[index] = pixels
	}

	skybox, err := loadTexturePixels(level.Assets.Skybox)
	if err != nil {
		return TextureSet{}, err
	}
	textures.Skybox = skybox

	return textures, nil
}

func loadTexturePixels(path string) ([]uint32, error) {
	surface, err := img.Load(path)
	if err != nil {
		return nil, err
	}
	defer surface.Free()

	converted, err := surface.ConvertFormat(uint32(sdl.PIXELFORMAT_ARGB8888), 0)
	if err != nil {
		return nil, err
	}
	defer converted.Free()

	width := int(converted.W)
	height := int(converted.H)
	pixels := make([]uint32, width*height)

	data := converted.Pixels()
	pitch := int(converted.Pitch)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := y*pitch + x*4
			pixels[y*width+x] = uint32(data[offset]) | uint32(data[offset+1])<<8 | uint32(data[offset+2])<<16 | uint32(data[offset+3])<<24
		}
	}

	return pixels, nil
}

func migrateLegacyTiles(level *Level) {
	for x := range level.Floors {
		for y := range level.Floors[x] {
			if level.Floors[x][y] >= 4 {
				level.Floors[x][y] -= 3
			} else {
				level.Floors[x][y] = 0
			}
		}
	}

	for x := range level.Ceilings {
		for y := range level.Ceilings[x] {
			switch {
			case level.Ceilings[x][y] == 12:
				level.Ceilings[x][y] = SkyboxTile
			case level.Ceilings[x][y] >= 8:
				level.Ceilings[x][y] -= 7
			default:
				level.Ceilings[x][y] = 0
			}
		}
	}
}
