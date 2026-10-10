package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/nexkod/nexedar/internal/db"
)

var portalTmpl = template.Must(template.New("portal").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.Project.Name}} — Download & Update</title>
  <style>
    :root {
      --bg: #090d16;
      --card-bg: rgba(22, 28, 45, 0.75);
      --border: rgba(255, 255, 255, 0.1);
      --primary: #3b82f6;
      --primary-hover: #2563eb;
      --accent: #10b981;
      --text: #f3f4f6;
      --text-muted: #9ca3af;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background: radial-gradient(circle at 50% 0%, #172554 0%, var(--bg) 70%);
      color: var(--text);
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      min-height: 100vh;
      display: flex;
      flex-direction: column;
      align-items: center;
      padding: 2rem 1rem;
    }
    .container {
      width: 100%;
      max-width: 520px;
      background: var(--card-bg);
      backdrop-filter: blur(16px);
      -webkit-backdrop-filter: blur(16px);
      border: 1px solid var(--border);
      border-radius: 24px;
      padding: 2.25rem 1.75rem;
      box-shadow: 0 20px 40px rgba(0, 0, 0, 0.5);
      text-align: center;
    }
    .badge {
      display: inline-block;
      padding: 4px 12px;
      font-size: 0.75rem;
      font-weight: 700;
      letter-spacing: 0.05em;
      text-transform: uppercase;
      border-radius: 9999px;
      background: rgba(59, 130, 246, 0.15);
      color: #60a5fa;
      border: 1px solid rgba(59, 130, 246, 0.3);
      margin-bottom: 1rem;
    }
    .app-icon {
      width: 88px;
      height: 88px;
      border-radius: 20px;
      background: linear-gradient(135deg, #1e3a8a, #3b82f6);
      display: flex;
      align-items: center;
      justify-content: center;
      margin: 0 auto 1.25rem;
      box-shadow: 0 10px 25px rgba(59, 130, 246, 0.4);
      font-size: 2.5rem;
    }
    .app-title { font-size: 1.75rem; font-weight: 800; margin-bottom: 0.35rem; }
    .app-desc { font-size: 0.95rem; color: var(--text-muted); margin-bottom: 1.5rem; line-height: 1.5; }
    .version-meta {
      background: rgba(0, 0, 0, 0.25);
      border-radius: 14px;
      padding: 0.85rem 1rem;
      display: flex;
      justify-content: space-around;
      margin-bottom: 1.5rem;
      font-size: 0.85rem;
      border: 1px solid rgba(255, 255, 255, 0.05);
    }
    .meta-item strong { display: block; color: var(--text-muted); font-size: 0.75rem; margin-bottom: 2px; }
    .download-btn {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 10px;
      width: 100%;
      background: linear-gradient(135deg, #2563eb, #1d4ed8);
      color: #fff;
      text-decoration: none;
      font-weight: 700;
      font-size: 1.1rem;
      padding: 1rem;
      border-radius: 16px;
      transition: all 0.2s ease;
      box-shadow: 0 8px 20px rgba(37, 99, 235, 0.4);
    }
    .download-btn:hover {
      background: linear-gradient(135deg, #3b82f6, #2563eb);
      transform: translateY(-2px);
      box-shadow: 0 12px 25px rgba(37, 99, 235, 0.6);
    }
    .changelog-box {
      margin-top: 1.5rem;
      text-align: left;
      background: rgba(0, 0, 0, 0.2);
      border-radius: 12px;
      padding: 1rem;
      border: 1px solid rgba(255, 255, 255, 0.05);
    }
    .changelog-title { font-size: 0.85rem; font-weight: 700; color: #93c5fd; margin-bottom: 0.4rem; }
    .changelog-text { font-size: 0.85rem; color: #d1d5db; line-height: 1.4; white-space: pre-wrap; }
    .qr-box {
      margin-top: 1.75rem;
      padding-top: 1.5rem;
      border-top: 1px solid var(--border);
      display: flex;
      flex-direction: column;
      align-items: center;
    }
    .qr-img {
      width: 140px;
      height: 140px;
      border-radius: 12px;
      background: #fff;
      padding: 8px;
      box-shadow: 0 4px 15px rgba(0, 0, 0, 0.3);
    }
    .qr-caption {
      margin-top: 0.75rem;
      font-size: 0.75rem;
      color: var(--text-muted);
    }
    .footer {
      margin-top: 1.5rem;
      font-size: 0.75rem;
      color: #6b7280;
    }
    .no-release {
      padding: 2rem 0;
      color: var(--text-muted);
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="badge">NexKod Distribution</div>
    
    <div class="app-icon">
      {{if .Project.IconURL}}
        <img src="{{.Project.IconURL}}" style="width: 100%; height: 100%; border-radius: 20px; object-fit: cover;" alt="App Icon">
      {{else}}
        📦
      {{end}}
    </div>

    <h1 class="app-title">{{.Project.Name}}</h1>
    <p class="app-desc">{{if .Project.Description}}{{.Project.Description}}{{else}}Fast direct APK distribution & updates.{{end}}</p>

    {{if .LatestRelease}}
      <div class="version-meta">
        <div class="meta-item">
          <strong>VERSION</strong>
          <span>v{{.LatestRelease.VersionName}}</span>
        </div>
        <div class="meta-item">
          <strong>BUILD</strong>
          <span>#{{.LatestRelease.VersionCode}}</span>
        </div>
        <div class="meta-item">
          <strong>SIZE</strong>
          <span>{{.FormattedSize}}</span>
        </div>
        <div class="meta-item">
          <strong>DOWNLOADS</strong>
          <span>{{.LatestRelease.DownloadCount}}</span>
        </div>
      </div>

      <a href="/p/{{.Project.ID}}/download/{{.LatestRelease.ID}}" class="download-btn">
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="7 10 12 15 17 10"></polyline><line x1="12" y1="15" x2="12" y2="3"></line></svg>
        Download APK
      </a>

      {{if .LatestRelease.Changelog}}
      <div class="changelog-box">
        <div class="changelog-title">What's New:</div>
        <div class="changelog-text">{{.LatestRelease.Changelog}}</div>
      </div>
      {{end}}

      <div class="qr-box">
        <img class="qr-img" src="/p/{{.Project.ID}}/qr" alt="Scan to download">
        <p class="qr-caption">Scan with camera to install directly on mobile</p>
      </div>
    {{else}}
      <div class="no-release">
        <p>No releases published yet for this project.</p>
        <p style="font-size: 0.8rem; margin-top: 0.5rem;">Upload an APK using the CI/CD script or the admin dashboard.</p>
      </div>
    {{end}}
  </div>

  <div class="footer">
    Powered by <strong>NexEdar</strong> &bull; NexKod Ecosystem
  </div>
</body>
</html>`))

var adminTmpl = template.Must(template.New("admin").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>NexEdar Admin Dashboard</title>
  <style>
    :root {
      --bg: #0b0f19;
      --card: #151c2e;
      --border: #232d45;
      --primary: #3b82f6;
      --danger: #ef4444;
      --success: #10b981;
      --text: #f9fafb;
      --text-muted: #9ca3af;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background: var(--bg);
      color: var(--text);
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      padding: 2rem 1.5rem;
    }
    .max-w { max-width: 1080px; margin: 0 auto; }
    header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 2rem;
      padding-bottom: 1rem;
      border-bottom: 1px solid var(--border);
    }
    h1 { font-size: 1.5rem; font-weight: 700; display: flex; align-items: center; gap: 8px; }
    .card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 1.5rem;
      margin-bottom: 2rem;
    }
    h2 { font-size: 1.15rem; margin-bottom: 1rem; color: #93c5fd; }
    table { width: 100%; border-collapse: collapse; margin-top: 0.5rem; }
    th, td { text-align: left; padding: 10px 12px; font-size: 0.85rem; border-bottom: 1px solid var(--border); }
    th { color: var(--text-muted); font-weight: 600; text-transform: uppercase; font-size: 0.75rem; }
    .btn {
      display: inline-flex;
      align-items: center;
      padding: 6px 12px;
      border-radius: 6px;
      border: none;
      font-size: 0.8rem;
      font-weight: 600;
      cursor: pointer;
      text-decoration: none;
    }
    .btn-primary { background: var(--primary); color: #fff; }
    .btn-danger { background: var(--danger); color: #fff; }
    .btn-outline { background: transparent; border: 1px solid var(--border); color: var(--text); }
    .form-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
      gap: 1rem;
      margin-bottom: 1rem;
    }
    input, textarea, select {
      width: 100%;
      background: #090d16;
      border: 1px solid var(--border);
      color: #fff;
      padding: 8px 12px;
      border-radius: 6px;
      font-size: 0.85rem;
    }
    .badge {
      padding: 2px 8px;
      border-radius: 4px;
      font-size: 0.75rem;
      font-weight: 600;
    }
    .badge-mandatory { background: rgba(239, 68, 68, 0.2); color: #f87171; }
  </style>
</head>
<body>
  <div class="max-w">
    <header>
      <h1>📦 NexEdar Management Console</h1>
      <div>
        <a href="/healthz" target="_blank" class="btn btn-outline">Health Check</a>
      </div>
    </header>

    <div class="card">
      <h2>Create New Project</h2>
      <form action="/admin/create-project" method="POST">
        <div class="form-grid">
          <div>
            <label style="font-size:0.75rem; color:var(--text-muted);">Project ID / Slug (e.g. nexronda)</label>
            <input type="text" name="id" required placeholder="nexronda">
          </div>
          <div>
            <label style="font-size:0.75rem; color:var(--text-muted);">Display Name</label>
            <input type="text" name="name" required placeholder="NexRonda Intercom">
          </div>
          <div>
            <label style="font-size:0.75rem; color:var(--text-muted);">Description</label>
            <input type="text" name="description" placeholder="Short summary">
          </div>
          <div>
            <label style="font-size:0.75rem; color:var(--text-muted);">Icon URL (optional)</label>
            <input type="text" name="icon_url" placeholder="https://...">
          </div>
        </div>
        <button type="submit" class="btn btn-primary">Create Project</button>
      </form>
    </div>

    {{range .Projects}}
    <div class="card">
      <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom: 1rem;">
        <div>
          <h2>{{.Project.Name}} (<code>{{.Project.ID}}</code>)</h2>
          <span style="font-size:0.8rem; color:var(--text-muted);">
            Disk Used: <strong>{{.FormattedDiskUsage}}</strong> across {{.FileCount}} files &bull;
            <a href="/p/{{.Project.ID}}" target="_blank" style="color:#60a5fa; text-decoration:none;">Open Download Portal &rarr;</a>
          </span>
        </div>
        <form action="/admin/purge" method="POST" onsubmit="return confirm('Purge builds older than retention limit for this project?');">
          <input type="hidden" name="project_id" value="{{.Project.ID}}">
          <button type="submit" class="btn btn-outline">Purge Old Builds</button>
        </form>
      </div>

      {{if .Releases}}
      <table>
        <thead>
          <tr>
            <th>Version</th>
            <th>Code</th>
            <th>Size</th>
            <th>Type</th>
            <th>Downloads</th>
            <th>Uploaded</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {{range .Releases}}
          <tr>
            <td><strong>v{{.VersionName}}</strong></td>
            <td>#{{.VersionCode}}</td>
            <td>{{printf "%.1f MB" (divFloat .FileSizeBytes 1048576.0)}}</td>
            <td>
              {{if .IsMandatory}}<span class="badge badge-mandatory">Mandatory</span>{{else}}Optional{{end}}
            </td>
            <td>{{.DownloadCount}}</td>
            <td>{{.CreatedAt.Format "2006-01-02 15:04"}}</td>
            <td>
              <a href="/p/{{.ProjectID}}/download/{{.ID}}" class="btn btn-outline" style="margin-right:4px;">Download</a>
              <form action="/admin/delete-release" method="POST" style="display:inline;" onsubmit="return confirm('Delete this release permanently?');">
                <input type="hidden" name="release_id" value="{{.ID}}">
                <button type="submit" class="btn btn-danger">Delete</button>
              </form>
            </td>
          </tr>
          {{end}}
        </tbody>
      </table>
      {{else}}
      <p style="font-size:0.85rem; color:var(--text-muted); margin-top:0.5rem;">No releases uploaded yet.</p>
      {{end}}
    </div>
    {{end}}
  </div>
</body>
</html>`))

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func (s *Server) handlePortalPage(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project_id")
	proj, err := s.db.GetProject(projectID)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	latest, _ := s.db.GetLatestRelease(projectID)

	var formattedSize string
	if latest != nil {
		formattedSize = formatBytes(latest.FileSizeBytes)
	}

	data := struct {
		Project       *db.Project
		LatestRelease *db.Release
		FormattedSize string
	}{
		Project:       proj,
		LatestRelease: latest,
		FormattedSize: formattedSize,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = portalTmpl.Execute(w, data)
}

type ProjectAdminData struct {
	Project            db.Project
	Releases           []db.Release
	FormattedDiskUsage string
	FileCount          int
}

func (s *Server) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	projects, err := s.db.ListProjects()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var dataList []ProjectAdminData
	for _, p := range projects {
		rels, _ := s.db.ListReleases(p.ID)
		diskBytes, fileCount, _ := s.storage.GetDiskUsage(p.ID)

		dataList = append(dataList, ProjectAdminData{
			Project:            p,
			Releases:           rels,
			FormattedDiskUsage: formatBytes(diskBytes),
			FileCount:          fileCount,
		})
	}

	funcMap := template.FuncMap{
		"divFloat": func(a int64, b float64) float64 {
			return float64(a) / b
		},
	}

	tmplWithFunc := template.Must(adminTmpl.Clone())
	tmplWithFunc.Funcs(funcMap)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmplWithFunc.Execute(w, map[string]any{
		"Projects": dataList,
	})
}

func (s *Server) handleAdminPurgeOld(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	projectID := r.FormValue("project_id")
	if projectID != "" {
		s.purgeExcessReleases(projectID, s.cfg.MaxBuildsPerApp)
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (s *Server) handleAdminDeleteRelease(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	relIDStr := r.FormValue("release_id")
	relID, _ := strconv.ParseInt(relIDStr, 10, 64)
	if relID > 0 {
		rel, err := s.db.DeleteRelease(relID)
		if err == nil && rel != nil {
			_ = s.storage.DeleteFile(rel.FilePath)
		}
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (s *Server) handleAdminCreateProject(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id := r.FormValue("id")
	name := r.FormValue("name")
	desc := r.FormValue("description")
	icon := r.FormValue("icon_url")

	if id != "" && name != "" {
		_ = s.db.UpsertProject(db.Project{
			ID:          id,
			Name:        name,
			Description: desc,
			IconURL:     icon,
		})
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
