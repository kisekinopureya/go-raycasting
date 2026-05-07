package worldMap

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultPackagePath  = "levels/level.level"
	defaultPackageFile  = "level.level"
	bundleFormatVersion = "go-raycasting-level-package-v1"
	PackageMimeType     = "application/vnd.go-raycasting.level-package"
	binaryFormatVersion = uint32(1)
)

func init() {
	// Register types for gob encoding
	gob.Register(&singleFileBundle{})
	gob.Register(LevelPackageMetadata{})
	gob.Register(Level{})
	gob.Register(AssetCatalog{})
}

// LevelPackageMetadata contains information about a level package
type LevelPackageMetadata struct {
	Name        string `json:"name"`
	Author      string `json:"author"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

// LevelPackage represents a complete, self-contained level with all its assets
type LevelPackage struct {
	Metadata LevelPackageMetadata `json:"metadata"`
	Level    Level                `json:"level"`
	BasePath string               `json:"-"` // Directory or file path used to load/save package
}

type singleFileBundle struct {
	Format   string               `json:"format"`
	Metadata LevelPackageMetadata `json:"metadata"`
	Level    Level                `json:"level"`
	Assets   map[string]string    `json:"assets"`
}

type packageMetadataOnly struct {
	Format   string               `json:"format"`
	Metadata LevelPackageMetadata `json:"metadata"`
}

var magicBytes = [5]byte{'L', 'E', 'V', 'E','L'}

func isBinaryFormat(file *os.File) (bool, error) {
	header := make([]byte, 5)
	n, err := file.Read(header)
	if err != nil && err != io.EOF {
		return false, err
	}
	if n < 5 {
		return false, nil
	}

	_, err = file.Seek(0, 0)
	if err != nil {
		return false, err
	}

	for i := 0; i < 5; i++ {
		if header[i] != magicBytes[i] {
			return false, nil
		}
	}
	return true, nil
}

func encodeBinaryBundle(bundle *singleFileBundle) ([]byte, error) {
	var buf bytes.Buffer

	buf.Write(magicBytes[:])

	if err := binary.Write(&buf, binary.LittleEndian, binaryFormatVersion); err != nil {
		return nil, fmt.Errorf("write version: %w", err)
	}

	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(bundle); err != nil {
		return nil, fmt.Errorf("gob encode: %w", err)
	}

	return buf.Bytes(), nil
}

func decodeBinaryBundle(data []byte) (*singleFileBundle, error) {
	if len(data) < 9 {
		return nil, errors.New("invalid binary format: too small")
	}

	for i := 0; i < 5; i++ {
		if data[i] != magicBytes[i] {
			return nil, errors.New("invalid binary format: bad magic number")
		}
	}

	version := binary.LittleEndian.Uint32(data[5:9])
	if version != binaryFormatVersion {
		return nil, fmt.Errorf("unsupported binary format version: %d", version)
	}

	var bundle singleFileBundle
	decoder := gob.NewDecoder(bytes.NewReader(data[9:]))
	if err := decoder.Decode(&bundle); err != nil {
		return nil, fmt.Errorf("gob decode: %w", err)
	}

	return &bundle, nil
}

func LoadLevelPackageMetadata(packagePath string) (LevelPackageMetadata, error) {
	absPath, err := filepath.Abs(packagePath)
	if err != nil {
		return LevelPackageMetadata{}, fmt.Errorf("invalid package path: %w", err)
	}

	_ , err = os.Stat(absPath)
	if err != nil {
		return LevelPackageMetadata{}, fmt.Errorf("package path stat failed: %w", err)
	}

	return loadSingleFileMetadata(absPath)
}

func loadSingleFileMetadata(filePath string) (LevelPackageMetadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return LevelPackageMetadata{}, fmt.Errorf("open package file: %w", err)
	}
	defer file.Close()

	binary, err := isBinaryFormat(file)
	if err != nil {
		return LevelPackageMetadata{}, fmt.Errorf("check file format: %w", err)
	}

	if binary {
		data, err := io.ReadAll(file)
		if err != nil {
			return LevelPackageMetadata{}, fmt.Errorf("read package file: %w", err)
		}

		bundle, err := decodeBinaryBundle(data)
		if err != nil {
			return LevelPackageMetadata{}, fmt.Errorf("parse binary package: %w", err)
		}

		return bundle.Metadata, nil
	}

	// Fall back to JSON format (legacy)
	var meta packageMetadataOnly
	if err := json.NewDecoder(file).Decode(&meta); err != nil {
		return LevelPackageMetadata{}, fmt.Errorf("parse package metadata: %w", err)
	}
	if meta.Format != bundleFormatVersion {
		return LevelPackageMetadata{}, fmt.Errorf("unsupported package format: %s", meta.Format)
	}
	return meta.Metadata, nil
}

// LoadLevelPackage loads a level package.
// Supported formats:
//  1. Single-file bundle (*.level) with metadata + level + embedded assets
//  2. Legacy directory package (metadata.json + level.json + assets/)
func LoadLevelPackage(packagePath string) (*LevelPackage, error) {
	absPath, err := filepath.Abs(packagePath)
	if err != nil {
		return nil, fmt.Errorf("invalid package path: %w", err)
	}

	_ , err = os.Stat(absPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("package path stat failed: %w", err)
		}

		return nil, fmt.Errorf("package path not found: %w", err)
	}

	return loadSingleFilePackage(absPath)
}

func loadSingleFilePackage(filePath string) (*LevelPackage, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open package file: %w", err)
	}

	binary, err := isBinaryFormat(file)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("check file format: %w", err)
	}

	// Read entire file
	data, err := io.ReadAll(file)
	file.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read package file: %w", err)
	}

	var bundle *singleFileBundle

	if binary {
		parsedBundle, err := decodeBinaryBundle(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse binary package file: %w", err)
		}
		bundle = parsedBundle
	} else {
		// Fall back to JSON format (legacy)
		if err := json.Unmarshal(data, &bundle); err != nil {
			return nil, fmt.Errorf("failed to parse package file: %w", err)
		}
		if bundle.Format != bundleFormatVersion {
			return nil, fmt.Errorf("unsupported package format: %s", bundle.Format)
		}
	}

	assetsDir, err := os.MkdirTemp("", "go-raycasting-level-assets-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp asset dir: %w", err)
	}

	for name, encoded := range bundle.Assets {
		safeName := filepath.Base(name)
		if safeName == "" || safeName == "." || safeName == string(filepath.Separator) {
			return nil, fmt.Errorf("invalid embedded asset name: %q", name)
		}
		bytes, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("decode embedded asset %s: %w", safeName, err)
		}
		if err := os.WriteFile(filepath.Join(assetsDir, safeName), bytes, 0o644); err != nil {
			return nil, fmt.Errorf("write embedded asset %s: %w", safeName, err)
		}
	}

	resolveFromEmbedded := func(path string) string {
		if path == "" {
			return ""
		}
		return filepath.Join(assetsDir, filepath.Base(path))
	}

	level := bundle.Level
	for i := range level.Assets.Walls {
		level.Assets.Walls[i] = resolveFromEmbedded(level.Assets.Walls[i])
	}
	for i := range level.Assets.Floors {
		level.Assets.Floors[i] = resolveFromEmbedded(level.Assets.Floors[i])
	}
	for i := range level.Assets.Ceilings {
		level.Assets.Ceilings[i] = resolveFromEmbedded(level.Assets.Ceilings[i])
	}
	level.Assets.Skybox = resolveFromEmbedded(level.Assets.Skybox)

	return &LevelPackage{
		Metadata: bundle.Metadata,
		Level:    level,
		BasePath: filePath,
	}, nil
}

func SaveLevelPackage(pkg *LevelPackage, packagePath string) error {
	targetPath, err := resolvePackageFilePath(packagePath)
	if err != nil {
		return fmt.Errorf("resolve package file path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("failed to create package parent dir: %w", err)
	}

	levelCopy := pkg.Level
	assetsData := make(map[string]string)

	encodeList := func(prefix string, paths []string) ([]string, error) {
		result := make([]string, len(paths))
		for i, srcPath := range paths {
			if srcPath == "" {
				continue
			}

			assetName := fmt.Sprintf("%s_%03d_%s", prefix, i, filepath.Base(srcPath))
			content, err := os.ReadFile(srcPath)
			if err != nil {
				return nil, fmt.Errorf("read asset %s: %w", srcPath, err)
			}
			assetsData[assetName] = base64.StdEncoding.EncodeToString(content)
			result[i] = assetName
		}
		return result, nil
	}

	walls, err := encodeList("wall", levelCopy.Assets.Walls)
	if err != nil {
		return err
	}
	floors, err := encodeList("floor", levelCopy.Assets.Floors)
	if err != nil {
		return err
	}
	ceilings, err := encodeList("ceiling", levelCopy.Assets.Ceilings)
	if err != nil {
		return err
	}
	skybox := ""
	if levelCopy.Assets.Skybox != "" {
		skybox = "skybox_000_" + filepath.Base(levelCopy.Assets.Skybox)
		content, err := os.ReadFile(levelCopy.Assets.Skybox)
		if err != nil {
			return fmt.Errorf("read skybox asset %s: %w", levelCopy.Assets.Skybox, err)
		}
		assetsData[skybox] = base64.StdEncoding.EncodeToString(content)
	}

	levelCopy.Assets = AssetCatalog{
		Walls:    walls,
		Floors:   floors,
		Ceilings: ceilings,
		Skybox:   skybox,
	}

	bundle := singleFileBundle{
		Format:   bundleFormatVersion,
		Metadata: pkg.Metadata,
		Level:    levelCopy,
		Assets:   assetsData,
	}

	outputFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create package file: %w", err)
	}
	defer outputFile.Close()

	binaryData, err := encodeBinaryBundle(&bundle)
	if err != nil {
		return fmt.Errorf("encode binary bundle: %w", err)
	}

	if _, err := outputFile.Write(binaryData); err != nil {
		return fmt.Errorf("failed to write package file: %w", err)
	}

	return nil
}

func resolvePackageFilePath(packagePath string) (string, error) {
	absPath, err := filepath.Abs(packagePath)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(absPath)
	if err == nil {
		if info.IsDir() {
			return filepath.Join(absPath, defaultPackageFile), nil
		}
		return absPath, nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	if filepath.Ext(absPath) == "" {
		return filepath.Join(absPath, defaultPackageFile), nil
	}

	return absPath, nil
}

func ExportLevelPackage(level Level, metadata LevelPackageMetadata, exportPath string) error {
	pkg := &LevelPackage{
		Metadata: metadata,
		Level:    level,
		BasePath: exportPath,
	}
	return SaveLevelPackage(pkg, exportPath)
}

func resolveAssetPaths(assets *AssetCatalog, assetsDir string) error {
	resolvePath := func(relPath string) string {
		if relPath == "" {
			return ""
		}
		if filepath.IsAbs(relPath) {
			return relPath
		}
		candidate := filepath.Join(assetsDir, filepath.Base(relPath))
		if fileExists(candidate) {
			return candidate
		}
		if abs, err := filepath.Abs(relPath); err == nil && fileExists(abs) {
			return abs
		}
		return candidate
	}

	for i := range assets.Walls {
		assets.Walls[i] = resolvePath(assets.Walls[i])
	}
	for i := range assets.Floors {
		assets.Floors[i] = resolvePath(assets.Floors[i])
	}
	for i := range assets.Ceilings {
		assets.Ceilings[i] = resolvePath(assets.Ceilings[i])
	}
	assets.Skybox = resolvePath(assets.Skybox)

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func ListLevelPackages(packagesDir string) ([]string, error) {
	entries, err := os.ReadDir(packagesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read packages directory: %w", err)
	}

	var packages []string
	for _, entry := range entries {
		if strings.EqualFold(filepath.Ext(entry.Name()), ".level") {
			packages = append(packages, entry.Name())
		}
	}

	return packages, nil
}

func ImportLevelPackage(packagePath string, copyTo string) (*LevelPackage, error) {
	pkg, err := LoadLevelPackage(packagePath)
	if err != nil {
		return nil, err
	}

	if copyTo != "" {
		if err := SaveLevelPackage(pkg, copyTo); err != nil {
			return nil, fmt.Errorf("failed to copy package: %w", err)
		}
		pkg.BasePath = copyTo
	}

	return pkg, nil
}

func GetLevelPackagePath(levelName string) string {
	if _, err := os.Stat(levelName); err == nil {
		return levelName
	}

	if filepath.Ext(levelName) == "" {
		singleFile := levelName + ".level"
		if _, err := os.Stat(singleFile); err == nil {
			return singleFile
		}
	}

	// Check levels directory
	levelsPath := filepath.Join("levels", levelName)
	if _, err := os.Stat(levelsPath); err == nil {
		return levelsPath
	}
	if filepath.Ext(levelName) == "" {
		levelsSingle := filepath.Join("levels", levelName+".level")
		if _, err := os.Stat(levelsSingle); err == nil {
			return levelsSingle
		}
	}

	return levelName
}

func PackageInfo(pkg *LevelPackage) string {
	var sb strings.Builder

	if pkg.Metadata.Name != "" {
		sb.WriteString(fmt.Sprintf("Name: %s\n", pkg.Metadata.Name))
	}
	if pkg.Metadata.Author != "" {
		sb.WriteString(fmt.Sprintf("Author: %s\n", pkg.Metadata.Author))
	}
	if pkg.Metadata.Description != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", pkg.Metadata.Description))
	}
	if pkg.Metadata.Version != "" {
		sb.WriteString(fmt.Sprintf("Version: %s\n", pkg.Metadata.Version))
	}

	sb.WriteString(fmt.Sprintf("Dimensions: %dx%d\n", pkg.Level.Width(), pkg.Level.Height()))
	sb.WriteString(fmt.Sprintf("Floors: %d\n", pkg.Level.GetFloorCount()))
	sb.WriteString(fmt.Sprintf("Walls: %d\n", len(pkg.Level.Assets.Walls)))
	sb.WriteString(fmt.Sprintf("Floors: %d\n", len(pkg.Level.Assets.Floors)))
	sb.WriteString(fmt.Sprintf("Ceilings: %d\n", len(pkg.Level.Assets.Ceilings)))

	return sb.String()
}
