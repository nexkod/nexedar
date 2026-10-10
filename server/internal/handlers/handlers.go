package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/skip2/go-qrcode"
	"github.com/nexkod/nexedar/internal/config"
	"github.com/nexkod/nexedar/internal/db"
	"github.com/nexkod/nexedar/internal/storage"
)

type Server struct {
	cfg     *config.Config
	db      *db.DB
	storage *storage.Storage
}

func NewServer(cfg *config.Config, database *db.DB, store *storage.Storage) *Server {
	return &Server{
		cfg:     cfg,
		db:      database,
		storage: store,
	}
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	// Health check
	mux.HandleFunc("GET /healthz", s.handleHealthz)

	// Public Update APIs (Client SDK / OTA)
	mux.HandleFunc("GET /api/v1/apps/{project_id}/latest", s.handleGetLatestRelease)
	mux.HandleFunc("GET /api/v1/apps/{project_id}/releases", s.handleListReleasesAPI)

	// Public Download, QR & Project Icon
	mux.HandleFunc("GET /p/{project_id}", s.handlePortalPage)
	mux.HandleFunc("GET /p/{project_id}/download", s.handleDirectDownloadLatest)
	mux.HandleFunc("GET /p/{project_id}/download/{release_id}", s.handleDownloadRelease)
	mux.HandleFunc("GET /p/{project_id}/qr", s.handleProjectQR)
	mux.HandleFunc("GET /p/{project_id}/icon.png", s.handleProjectIcon)

	// Authenticated Upload & Release Management
	mux.HandleFunc("POST /api/v1/apps/{project_id}/releases", s.authMiddleware(s.handleUploadRelease))
	mux.HandleFunc("POST /api/v1/apps/{project_id}/icon", s.authMiddleware(s.handleUploadIcon))
	mux.HandleFunc("DELETE /api/v1/apps/{project_id}/releases/{release_id}", s.authMiddleware(s.handleDeleteRelease))
	mux.HandleFunc("POST /api/v1/projects", s.authMiddleware(s.handleCreateProject))

	// Branding & Logo
	mux.HandleFunc("GET /logo.png", s.handleLogo)
	mux.HandleFunc("GET /favicon.ico", s.handleLogo)

	// Admin Authentication (Form login)
	mux.HandleFunc("GET /login", s.handleLoginPage)
	mux.HandleFunc("POST /login", s.handleLoginSubmit)
	mux.HandleFunc("POST /logout", s.handleLogout)

	// Admin Dashboard
	mux.HandleFunc("GET /admin", s.adminAuthMiddleware(s.handleAdminDashboard))
	mux.HandleFunc("POST /admin/purge", s.adminAuthMiddleware(s.handleAdminPurgeOld))
	mux.HandleFunc("POST /admin/delete-release", s.adminAuthMiddleware(s.handleAdminDeleteRelease))
	mux.HandleFunc("POST /admin/create-project", s.adminAuthMiddleware(s.handleAdminCreateProject))
	mux.HandleFunc("POST /admin/upload-release", s.adminAuthMiddleware(s.handleAdminUploadRelease))
	mux.HandleFunc("POST /admin/upload-icon", s.adminAuthMiddleware(s.handleAdminUploadIcon))
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"service": "nexedar",
	})
}

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		expectedToken := "Bearer " + s.cfg.APISecret
		if authHeader != expectedToken {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (s *Server) adminAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// First check cookie session
		if s.isAuthenticatedSession(r) {
			next(w, r)
			return
		}

		// Fallback to HTTP Basic Auth if provided
		if user, pass, ok := r.BasicAuth(); ok && pass == s.cfg.AdminPassword {
			_ = user
			next(w, r)
			return
		}

		// Redirect to proper login page for browsers
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func (s *Server) handleGetLatestRelease(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project_id")
	if projectID == "" {
		http.Error(w, `{"error":"missing project_id"}`, http.StatusBadRequest)
		return
	}

	rel, err := s.db.GetLatestRelease(projectID)
	if err != nil {
		http.Error(w, `{"error":"no release found for project"}`, http.StatusNotFound)
		return
	}

	downloadURL := fmt.Sprintf("%s/p/%s/download/%d", s.cfg.PublicURL, projectID, rel.ID)
	resp := map[string]any{
		"project_id":     rel.ProjectID,
		"version_name":   rel.VersionName,
		"version_code":   rel.VersionCode,
		"download_url":   downloadURL,
		"file_size":      rel.FileSizeBytes,
		"sha256":         rel.SHA256,
		"changelog":      rel.Changelog,
		"is_mandatory":   rel.IsMandatory,
		"download_count": rel.DownloadCount,
		"published_at":   rel.CreatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleListReleasesAPI(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project_id")
	releases, err := s.db.ListReleases(projectID)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch releases"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"project_id": projectID,
		"releases":   releases,
	})
}

func (s *Server) handleUploadRelease(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project_id")
	if projectID == "" {
		http.Error(w, `{"error":"missing project_id"}`, http.StatusBadRequest)
		return
	}

	// 100MB limit for multipart
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to parse multipart: %v"}`, err), http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"error":"file field is required"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	versionName := r.FormValue("version_name")
	if versionName == "" {
		versionName = r.FormValue("version")
	}
	if versionName == "" {
		http.Error(w, `{"error":"version_name is required"}`, http.StatusBadRequest)
		return
	}

	versionCodeStr := r.FormValue("version_code")
	versionCode, _ := strconv.Atoi(versionCodeStr)
	if versionCode <= 0 {
		versionCode = 1
	}

	changelog := r.FormValue("changelog")
	isMandatory := r.FormValue("is_mandatory") == "true"
	purgePrevious := r.FormValue("purge_previous") == "true"
	forceUpload := r.FormValue("force") == "true"

	// 1. Version validation check against current latest release
	existingLatest, err := s.db.GetLatestRelease(projectID)
	if err == nil && existingLatest != nil {
		if !forceUpload {
			// Check if version_code is lower than or equal to current latest
			if versionCode < existingLatest.VersionCode {
				http.Error(w, fmt.Sprintf(`{"error":"version check failed: uploaded version_code (%d) is lower than current latest (%d). Use force=true to override."}`, versionCode, existingLatest.VersionCode), http.StatusConflict)
				return
			}
			if versionCode == existingLatest.VersionCode && versionName == existingLatest.VersionName {
				http.Error(w, fmt.Sprintf(`{"error":"version check failed: build v%s (#%d) already exists. Bump version in pubspec.yaml or pass force=true."}`, versionName, versionCode), http.StatusConflict)
				return
			}
		}
	}

	// Ensure project exists
	proj, err := s.db.GetProject(projectID)
	if err != nil || proj == nil {
		name := r.FormValue("project_name")
		if name == "" {
			name = strings.ToUpper(projectID[:1]) + projectID[1:]
		}
		_ = s.db.UpsertProject(db.Project{
			ID:   projectID,
			Name: name,
		})
	}

	// Save APK
	filePath, size, sha, err := s.storage.SaveAPK(projectID, versionName, versionCode, file)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to save apk: %v"}`, err), http.StatusInternalServerError)
		return
	}

	relID, err := s.db.InsertRelease(db.Release{
		ProjectID:     projectID,
		VersionName:   versionName,
		VersionCode:   versionCode,
		FilePath:      filePath,
		FileSizeBytes: size,
		SHA256:        sha,
		Changelog:     changelog,
		IsMandatory:   isMandatory,
		IsActive:      true,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to insert release record: %v"}`, err), http.StatusInternalServerError)
		return
	}

	// Retention cleanup
	if purgePrevious {
		s.purgeExcessReleases(projectID, 1)
	} else if s.cfg.MaxBuildsPerApp > 0 {
		s.purgeExcessReleases(projectID, s.cfg.MaxBuildsPerApp)
	}

	portalURL := fmt.Sprintf("%s/p/%s", s.cfg.PublicURL, projectID)
	downloadURL := fmt.Sprintf("%s/p/%s/download/%d", s.cfg.PublicURL, projectID, relID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"message":      "Release published successfully",
		"release_id":   relID,
		"project_id":   projectID,
		"version_name": versionName,
		"version_code": versionCode,
		"file_size":    size,
		"sha256":       sha,
		"portal_url":   portalURL,
		"download_url": downloadURL,
	})
}

func (s *Server) handleDeleteRelease(w http.ResponseWriter, r *http.Request) {
	relIDStr := r.PathValue("release_id")
	relID, err := strconv.ParseInt(relIDStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid release_id"}`, http.StatusBadRequest)
		return
	}

	rel, err := s.db.DeleteRelease(relID)
	if err != nil {
		http.Error(w, `{"error":"release not found"}`, http.StatusNotFound)
		return
	}

	_ = s.storage.DeleteFile(rel.FilePath)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true})
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		IconURL     string `json:"icon_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	if body.ID == "" || body.Name == "" {
		http.Error(w, `{"error":"id and name required"}`, http.StatusBadRequest)
		return
	}

	err := s.db.UpsertProject(db.Project{
		ID:          body.ID,
		Name:        body.Name,
		Description: body.Description,
		IconURL:     body.IconURL,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"success": true, "project_id": body.ID})
}

func (s *Server) handleDirectDownloadLatest(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project_id")
	rel, err := s.db.GetLatestRelease(projectID)
	if err != nil {
		http.Error(w, "No release available for download", http.StatusNotFound)
		return
	}

	_ = s.db.IncrementDownloadCount(rel.ID)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-%s.apk"`, projectID, rel.VersionName))
	w.Header().Set("Content-Type", "application/vnd.android.package-archive")
	http.ServeFile(w, r, rel.FilePath)
}

func (s *Server) handleDownloadRelease(w http.ResponseWriter, r *http.Request) {
	relIDStr := r.PathValue("release_id")
	relID, err := strconv.ParseInt(relIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid release ID", http.StatusBadRequest)
		return
	}

	rel, err := s.db.GetRelease(relID)
	if err != nil {
		http.Error(w, "Release not found", http.StatusNotFound)
		return
	}

	_ = s.db.IncrementDownloadCount(rel.ID)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-%s.apk"`, rel.ProjectID, rel.VersionName))
	w.Header().Set("Content-Type", "application/vnd.android.package-archive")
	http.ServeFile(w, r, rel.FilePath)
}

func (s *Server) handleProjectQR(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project_id")
	targetURL := fmt.Sprintf("%s/p/%s", s.cfg.PublicURL, projectID)

	pngBytes, err := qrcode.Encode(targetURL, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, "Failed to generate QR code", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(pngBytes)
}

func (s *Server) handleProjectIcon(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project_id")
	iconPath := filepath.Join(s.storage.ProjectDir(projectID), "icon.png")
	if _, err := os.Stat(iconPath); err == nil {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		http.ServeFile(w, r, iconPath)
		return
	}
	// Fallback to default NexKod/NexEdar logo
	s.handleLogo(w, r)
}

func (s *Server) handleUploadIcon(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project_id")
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("icon")
	if err != nil {
		file, _, err = r.FormFile("file")
	}
	if err != nil {
		http.Error(w, "Icon file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	_, err = s.storage.SaveIcon(projectID, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update project icon_url if not set
	iconURL := fmt.Sprintf("%s/p/%s/icon.png", s.cfg.PublicURL, projectID)
	proj, _ := s.db.GetProject(projectID)
	if proj != nil {
		proj.IconURL = iconURL
		_ = s.db.UpsertProject(*proj)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success":  true,
		"icon_url": iconURL,
	})
}

func (s *Server) purgeExcessReleases(projectID string, keepCount int) {
	oldReleases, err := s.db.GetReleasesExceedingRetention(projectID, keepCount)
	if err != nil {
		return
	}

	for _, old := range oldReleases {
		_, _ = s.db.DeleteRelease(old.ID)
		_ = s.storage.DeleteFile(old.FilePath)
	}
}
