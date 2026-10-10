package storage

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

	// Prioritize highest quality mipmap or flutter asset icons
	candidates := []string{
		"res/mipmap-xxxhdpi-v4/ic_launcher.png",
		"res/mipmap-xxhdpi-v4/ic_launcher.png",
		"res/mipmap-xhdpi-v4/ic_launcher.png",
		"res/mipmap-hdpi-v4/ic_launcher.png",
		"assets/flutter_assets/assets/images/logo.png",
	}

	for _, cand := range candidates {
		for _, f := range r.File {
			if f.Name == cand {
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
