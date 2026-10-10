package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IconURL     string    `json:"icon_url"`
	CreatedAt   time.Time `json:"created_at"`
}

type Release struct {
	ID            int64     `json:"id"`
	ProjectID     string    `json:"project_id"`
	VersionName   string    `json:"version_name"`
	VersionCode   int       `json:"version_code"`
	FilePath      string    `json:"file_path"`
	FileSizeBytes int64     `json:"file_size_bytes"`
	SHA256        string    `json:"sha256"`
	Changelog     string    `json:"changelog"`
	IsMandatory   bool      `json:"is_mandatory"`
	IsActive      bool      `json:"is_active"`
	DownloadCount int64     `json:"download_count"`
	CreatedAt     time.Time `json:"created_at"`
}

type DB struct {
	*sql.DB
}

func Init(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "nexedar.db")
	database, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := createTables(database); err != nil {
		return nil, fmt.Errorf("failed to migrate tables: %w", err)
	}

	return &DB{database}, nil
}

func createTables(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		icon_url TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS releases (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		version_name TEXT NOT NULL,
		version_code INTEGER NOT NULL,
		file_path TEXT NOT NULL,
		file_size_bytes INTEGER NOT NULL,
		sha256 TEXT NOT NULL,
		changelog TEXT,
		is_mandatory BOOLEAN DEFAULT FALSE,
		is_active BOOLEAN DEFAULT TRUE,
		download_count INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_releases_project_id ON releases(project_id);
	CREATE INDEX IF NOT EXISTS idx_releases_active ON releases(is_active);
	`
	_, err := db.Exec(query)
	return err
}

func (db *DB) UpsertProject(p Project) error {
	query := `
	INSERT INTO projects (id, name, description, icon_url)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		name = excluded.name,
		description = excluded.description,
		icon_url = excluded.icon_url;
	`
	_, err := db.Exec(query, p.ID, p.Name, p.Description, p.IconURL)
	return err
}

func (db *DB) GetProject(id string) (*Project, error) {
	row := db.QueryRow("SELECT id, name, description, icon_url, created_at FROM projects WHERE id = ?", id)
	var p Project
	var desc, icon sql.NullString
	if err := row.Scan(&p.ID, &p.Name, &desc, &icon, &p.CreatedAt); err != nil {
		return nil, err
	}
	p.Description = desc.String
	p.IconURL = icon.String
	return &p, nil
}

func (db *DB) ListProjects() ([]Project, error) {
	rows, err := db.Query("SELECT id, name, description, icon_url, created_at FROM projects ORDER BY name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Project
	for rows.Next() {
		var p Project
		var desc, icon sql.NullString
		if err := rows.Scan(&p.ID, &p.Name, &desc, &icon, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.Description = desc.String
		p.IconURL = icon.String
		list = append(list, p)
	}
	return list, nil
}

func (db *DB) InsertRelease(r Release) (int64, error) {
	query := `INSERT INTO releases (project_id, version_name, version_code, file_path, file_size_bytes, sha256, changelog, is_mandatory, is_active) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := db.Exec(query, r.ProjectID, r.VersionName, r.VersionCode, r.FilePath, r.FileSizeBytes, r.SHA256, r.Changelog, r.IsMandatory, r.IsActive)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) GetLatestRelease(projectID string) (*Release, error) {
	query := `SELECT id, project_id, version_name, version_code, file_path, file_size_bytes, sha256, changelog, is_mandatory, is_active, download_count, created_at FROM releases WHERE project_id = ? AND is_active = 1 ORDER BY version_code DESC, id DESC LIMIT 1`
	row := db.QueryRow(query, projectID)
	var r Release
	var changelog sql.NullString
	if err := row.Scan(&r.ID, &r.ProjectID, &r.VersionName, &r.VersionCode, &r.FilePath, &r.FileSizeBytes, &r.SHA256, &changelog, &r.IsMandatory, &r.IsActive, &r.DownloadCount, &r.CreatedAt); err != nil {
		return nil, err
	}
	r.Changelog = changelog.String
	return &r, nil
}

func (db *DB) GetRelease(id int64) (*Release, error) {
	query := `SELECT id, project_id, version_name, version_code, file_path, file_size_bytes, sha256, changelog, is_mandatory, is_active, download_count, created_at FROM releases WHERE id = ?`
	row := db.QueryRow(query, id)
	var r Release
	var changelog sql.NullString
	if err := row.Scan(&r.ID, &r.ProjectID, &r.VersionName, &r.VersionCode, &r.FilePath, &r.FileSizeBytes, &r.SHA256, &changelog, &r.IsMandatory, &r.IsActive, &r.DownloadCount, &r.CreatedAt); err != nil {
		return nil, err
	}
	r.Changelog = changelog.String
	return &r, nil
}

func (db *DB) ListReleases(projectID string) ([]Release, error) {
	query := `SELECT id, project_id, version_name, version_code, file_path, file_size_bytes, sha256, changelog, is_mandatory, is_active, download_count, created_at FROM releases WHERE project_id = ? ORDER BY version_code DESC, id DESC`
	rows, err := db.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Release
	for rows.Next() {
		var r Release
		var changelog sql.NullString
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.VersionName, &r.VersionCode, &r.FilePath, &r.FileSizeBytes, &r.SHA256, &changelog, &r.IsMandatory, &r.IsActive, &r.DownloadCount, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Changelog = changelog.String
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (db *DB) IncrementDownloadCount(releaseID int64) error {
	_, err := db.Exec("UPDATE releases SET download_count = download_count + 1 WHERE id = ?", releaseID)
	return err
}

func (db *DB) DeleteRelease(id int64) (*Release, error) {
	rel, err := db.GetRelease(id)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec("DELETE FROM releases WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	return rel, nil
}

func (db *DB) DeleteProject(id string) error {
	_, err := db.Exec("DELETE FROM projects WHERE id = ?", id)
	return err
}

func (db *DB) GetReleasesExceedingRetention(projectID string, keepCount int) ([]Release, error) {
	query := `SELECT id, project_id, version_name, version_code, file_path, file_size_bytes, sha256, changelog, is_mandatory, is_active, download_count, created_at FROM releases WHERE project_id = ? ORDER BY version_code DESC, id DESC LIMIT -1 OFFSET ?`
	rows, err := db.Query(query, projectID, keepCount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Release
	for rows.Next() {
		var r Release
		var changelog sql.NullString
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.VersionName, &r.VersionCode, &r.FilePath, &r.FileSizeBytes, &r.SHA256, &changelog, &r.IsMandatory, &r.IsActive, &r.DownloadCount, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Changelog = changelog.String
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}
