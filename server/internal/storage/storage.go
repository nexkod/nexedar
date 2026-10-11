package storage

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Storage struct {
	BaseDir string
}

func New(baseDir string) (*Storage, error) {
	appsDir := filepath.Join(baseDir, "apps")
	if err := os.MkdirAll(appsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create apps storage dir: %w", err)
	}
	return &Storage{BaseDir: baseDir}, nil
}

func (s *Storage) ProjectDir(projectID string) string {
	return filepath.Join(s.BaseDir, "apps", projectID)
}

func (s *Storage) SaveAPK(projectID, versionName string, versionCode int, src io.Reader) (string, int64, string, error) {
	projDir := s.ProjectDir(projectID)
	if err := os.MkdirAll(projDir, 0755); err != nil {
		return "", 0, "", fmt.Errorf("failed to create project storage dir: %w", err)
	}

	fileName := fmt.Sprintf("v%s-%d.apk", versionName, versionCode)
	destPath := filepath.Join(projDir, fileName)

	tmpPath := destPath + ".tmp"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return "", 0, "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpPath)

	hasher := sha256.New()
	multiWriter := io.MultiWriter(tmpFile, hasher)

	written, err := io.Copy(multiWriter, src)
	if err != nil {
		tmpFile.Close()
		return "", 0, "", fmt.Errorf("failed to copy file contents: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return "", 0, "", fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		return "", 0, "", fmt.Errorf("failed to move temp file to dest: %w", err)
	}

	sha256Hex := hex.EncodeToString(hasher.Sum(nil))

	// Attempt to auto-extract app icon from APK if possible
	_ = s.ExtractAPKIcon(destPath, projectID)

	return destPath, written, sha256Hex, nil
}

func (s *Storage) SaveIcon(projectID string, src io.Reader) (string, error) {
	projDir := s.ProjectDir(projectID)
	if err := os.MkdirAll(projDir, 0755); err != nil {
		return "", err
	}
	iconPath := filepath.Join(projDir, "icon.png")
	out, err := os.Create(iconPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, src); err != nil {
		return "", err
	}
	return iconPath, nil
}

func (s *Storage) ExtractAPKIcon(apkPath, projectID string) error {
	r, err := zip.OpenReader(apkPath)
	if err != nil {
		return err
	}
	defer r.Close()

	// 1. Explicit high-priority named paths
	primaryNamed := []string{
		"res/mipmap-xxxhdpi-v4/ic_launcher.png",
		"res/mipmap-xxhdpi-v4/ic_launcher.png",
		"res/mipmap-xhdpi-v4/ic_launcher.png",
		"res/mipmap-hdpi-v4/ic_launcher.png",
		"res/mipmap-mdpi-v4/ic_launcher.png",
		"assets/flutter_assets/assets/images/logo.png",
		"assets/flutter_assets/assets/images/app_icon.png",
		"assets/flutter_assets/assets/icon.png",
		"assets/flutter_assets/assets/logo.png",
	}

	for _, cand := range primaryNamed {
		for _, f := range r.File {
			if strings.EqualFold(f.Name, cand) {
				rc, err := f.Open()
				if err != nil {
					continue
				}
				_, err = s.SaveIcon(projectID, rc)
				rc.Close()
				if err == nil {
					return nil
				}
			}
		}
	}

	// 2. Look for any flutter assets that are images matching icon/logo
	type candidate struct {
		score int
		size  int64
		file  *zip.File
	}
	var flutterCands []candidate
	for _, f := range r.File {
		name := strings.ToLower(f.Name)
		if strings.Contains(name, "assets/flutter_assets/") &&
			(strings.HasSuffix(name, ".png") || strings.HasSuffix(name, ".webp") || strings.HasSuffix(name, ".jpg") || strings.HasSuffix(name, ".jpeg")) {
			score := 0
			if strings.Contains(name, "launcher") || strings.Contains(name, "app_icon") {
				score = 500
			} else if strings.Contains(name, "logo") {
				score = 300
			} else if strings.Contains(name, "icon") {
				score = 100
			}
			if score > 0 {
				flutterCands = append(flutterCands, candidate{score: score, size: int64(f.UncompressedSize64), file: f})
			}
		}
	}
	if len(flutterCands) > 0 {
		sort.Slice(flutterCands, func(i, j int) bool {
			if flutterCands[i].score != flutterCands[j].score {
				return flutterCands[i].score > flutterCands[j].score
			}
			return flutterCands[i].size > flutterCands[j].size
		})
		rc, err := flutterCands[0].file.Open()
		if err == nil {
			_, err = s.SaveIcon(projectID, rc)
			rc.Close()
			if err == nil {
				return nil
			}
		}
	}

	// 3. Look for Android launcher icons in res/ (handles obfuscated R8 names like res/o-.png)
	var resCands []candidate
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "res/") && strings.HasSuffix(f.Name, ".png") && !strings.HasSuffix(f.Name, ".9.png") {
			// Read first 24 bytes to inspect PNG dimensions
			rc, err := f.Open()
			if err != nil {
				continue
			}
			header := make([]byte, 24)
			n, _ := io.ReadFull(rc, header)
			rc.Close()
			if n == 24 && bytes.HasPrefix(header, []byte("\x89PNG\r\n\x1a\n")) {
				w := binary.BigEndian.Uint32(header[16:20])
				h := binary.BigEndian.Uint32(header[20:24])
				if w == h && w >= 48 && w <= 512 {
					score := int(w)
					// Prioritize standard Android launcher densities
					if w == 192 || w == 144 || w == 96 || w == 72 || w == 48 {
						score += 1000
					}
					resCands = append(resCands, candidate{score: score, size: int64(f.UncompressedSize64), file: f})
				}
			}
		}
	}
	if len(resCands) > 0 {
		sort.Slice(resCands, func(i, j int) bool {
			return resCands[i].score > resCands[j].score
		})
		rc, err := resCands[0].file.Open()
		if err == nil {
			_, err = s.SaveIcon(projectID, rc)
			rc.Close()
			if err == nil {
				return nil
			}
		}
	}

	return fmt.Errorf("no launcher icon found")
}

func (s *Storage) DeleteFile(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(filePath)
}

func (s *Storage) GetDiskUsage(projectID string) (int64, int, error) {
	projDir := s.ProjectDir(projectID)
	var totalSize int64
	var fileCount int

	entries, err := os.ReadDir(projDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			name := entry.Name()
			if strings.HasSuffix(name, ".apk") || strings.HasSuffix(name, ".ipa") {
				info, err := entry.Info()
				if err == nil {
					totalSize += info.Size()
					fileCount++
				}
			}
		}
	}

	return totalSize, fileCount, nil
}
