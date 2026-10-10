package db

import (
	"os"
	"testing"
)

func TestReleases(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nexedar-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	database, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer database.Close()

	p := Project{
		ID:   "nexronda",
		Name: "NexRonda",
	}
	if err := database.UpsertProject(p); err != nil {
		t.Fatalf("UpsertProject failed: %v", err)
	}

	rel := Release{
		ProjectID:     "nexronda",
		VersionName:   "1.0.0",
		VersionCode:   2,
		FilePath:      "apps/nexronda/v1.0.0-2.apk",
		FileSizeBytes: 40000000,
		SHA256:        "dummy_sha256",
		Changelog:     "Initial release",
		IsMandatory:   false,
		IsActive:      true,
	}

	id, err := database.InsertRelease(rel)
	if err != nil {
		t.Fatalf("InsertRelease failed: %v", err)
	}
	t.Logf("Inserted release id: %d", id)

	latest, err := database.GetLatestRelease("nexronda")
	if err != nil {
		t.Fatalf("GetLatestRelease failed: %v", err)
	}
	if latest.VersionCode != 2 {
		t.Fatalf("expected version_code 2, got %d", latest.VersionCode)
	}
	t.Logf("Got latest release: %+v", latest)

	list, err := database.ListReleases("nexronda")
	if err != nil {
		t.Fatalf("ListReleases failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 release, got %d", len(list))
	}
	t.Logf("Got %d releases in list", len(list))

	fetched, err := database.GetRelease(id)
	if err != nil {
		t.Fatalf("GetRelease failed: %v", err)
	}
	if fetched.ID != id {
		t.Fatalf("expected release id %d, got %d", id, fetched.ID)
	}

	err = database.IncrementDownloadCount(id)
	if err != nil {
		t.Fatalf("IncrementDownloadCount failed: %v", err)
	}

	updated, _ := database.GetRelease(id)
	if updated.DownloadCount != 1 {
		t.Fatalf("expected download_count 1, got %d", updated.DownloadCount)
	}
}
