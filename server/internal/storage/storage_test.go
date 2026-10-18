package storage

import (
	"bytes"
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

func TestSaveArtifactGenericBinary(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage-artifact-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	s, err := New(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	content := []byte("\x7fELF\x02\x01\x01\x00custom_linux_agent_binary")
	destPath, written, sha, err := s.SaveArtifact("nexawan-agent", "1.2.0", 12, "nexawan-agent-linux-amd64", bytes.NewReader(content))
	if err != nil {
		t.Fatalf("SaveArtifact failed: %v", err)
	}

	if written != int64(len(content)) {
		t.Fatalf("expected written %d, got %d", len(content), written)
	}
	if sha == "" {
		t.Fatal("expected sha256 to be computed")
	}

	expectedName := "nexawan-agent-linux-amd64-v1.2.0-12"
	if filepath.Base(destPath) != expectedName {
		t.Fatalf("expected filename %s, got %s", expectedName, filepath.Base(destPath))
	}
}
