package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractAPKIcon(t *testing.T) {
	apkPath := "/Users/afiffakhree/StudioProjects/NexRonda/app/build/app/outputs/flutter-apk/app-arm64-v8a-release.apk"
	if _, err := os.Stat(apkPath); os.IsNotExist(err) {
		t.Skip("local APK not found, skipping")
	}

	tmpDir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	s, err := New(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	err = s.ExtractAPKIcon(apkPath, "nexronda")
	if err != nil {
		t.Fatalf("ExtractAPKIcon failed: %v", err)
	}

	iconPath := filepath.Join(s.ProjectDir("nexronda"), "icon.png")
	info, err := os.Stat(iconPath)
	if err != nil {
		t.Fatalf("extracted icon file not found: %v", err)
	}
	t.Logf("Successfully extracted icon: %s (size: %d bytes)", iconPath, info.Size())
}
