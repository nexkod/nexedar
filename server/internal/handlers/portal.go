package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/nexkod/nexedar/internal/assets"
	"github.com/nexkod/nexedar/internal/db"
)

var portalTmpl = template.Must(template.New("portal").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.Project.Name}} — Download & Update</title>
  <link rel="icon" type="image/png" href="/logo.png">
  <style>
    :root {
      --bg: #090d16;
      --card-bg: rgba(18, 24, 38, 0.85);
      --border: rgba(255, 255, 255, 0.1);
      --primary: #06b6d4;
      --primary-hover: #0891b2;
      --accent: #38bdf8;
      --text: #f3f4f6;
      --text-muted: #94a3b8;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background: radial-gradient(circle at 50% 0%, #0f2744 0%, var(--bg) 70%);
      color: var(--text);
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      min-height: 100vh;
      display: flex;
      flex-direction: column;
      align-items: center;
      padding: 2.5rem 1rem;
    }
    .container {
      width: 100%;
      max-width: 520px;
      background: var(--card-bg);
      backdrop-filter: blur(20px);
      -webkit-backdrop-filter: blur(20px);
      border: 1px solid var(--border);
      border-radius: 28px;
      padding: 2.5rem 2rem;
      box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.7);
      text-align: center;
    }
    .badge {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 5px 14px;
      font-size: 0.75rem;
      font-weight: 700;
      letter-spacing: 0.08em;
      text-transform: uppercase;
      border-radius: 9999px;
      background: rgba(6, 182, 212, 0.12);
      color: #38bdf8;
      border: 1px solid rgba(6, 182, 212, 0.3);
      margin-bottom: 1.5rem;
    }
    .logo-header {
      display: flex;
      justify-content: center;
      margin-bottom: 1.25rem;
    }
    .brand-logo {
      width: 100px;
      height: 100px;
      border-radius: 24px;
      object-fit: cover;
      box-shadow: 0 10px 25px rgba(6, 182, 212, 0.3);
      border: 1px solid rgba(255, 255, 255, 0.15);
    }
    .app-title { font-size: 1.85rem; font-weight: 800; margin-bottom: 0.4rem; letter-spacing: -0.02em; }
    .app-desc { font-size: 0.95rem; color: var(--text-muted); margin-bottom: 1.75rem; line-height: 1.5; }
    .version-meta {
      background: rgba(0, 0, 0, 0.35);
      border-radius: 16px;
      padding: 1rem 1.25rem;
      display: flex;
      justify-content: space-around;
      margin-bottom: 1.75rem;
      font-size: 0.85rem;
      border: 1px solid rgba(255, 255, 255, 0.05);
    }
    .meta-item strong { display: block; color: var(--text-muted); font-size: 0.7rem; margin-bottom: 3px; letter-spacing: 0.05em; }
    .meta-item span { font-weight: 700; font-size: 0.95rem; color: #f8fafc; }
    .download-btn {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 10px;
      width: 100%;
      background: linear-gradient(135deg, #0284c7, #0369a1);
      color: #fff;
      text-decoration: none;
      font-weight: 700;
      font-size: 1.1rem;
      padding: 1.05rem;
      border-radius: 18px;
      transition: all 0.25s ease;
      box-shadow: 0 10px 25px rgba(2, 132, 199, 0.4);
    }
    .download-btn:hover {
      background: linear-gradient(135deg, #0ea5e9, #0284c7);
      transform: translateY(-2px);
      box-shadow: 0 14px 30px rgba(2, 132, 199, 0.6);
    }
    .changelog-box {
      margin-top: 1.75rem;
      text-align: left;
      background: rgba(0, 0, 0, 0.25);
      border-radius: 14px;
      padding: 1.1rem;
      border: 1px solid rgba(255, 255, 255, 0.05);
    }
    .changelog-title { font-size: 0.85rem; font-weight: 700; color: #38bdf8; margin-bottom: 0.5rem; text-transform: uppercase; letter-spacing: 0.05em; }
    .changelog-text { font-size: 0.85rem; color: #cbd5e1; line-height: 1.5; white-space: pre-wrap; }
    .qr-box {
      margin-top: 2rem;
      padding-top: 1.75rem;
      border-top: 1px solid var(--border);
      display: flex;
      flex-direction: column;
      align-items: center;
    }
    .qr-img {
      width: 150px;
      height: 150px;
      border-radius: 16px;
      background: #fff;
      padding: 10px;
      box-shadow: 0 8px 20px rgba(0, 0, 0, 0.4);
    }
    .qr-caption {
      margin-top: 0.85rem;
      font-size: 0.8rem;
      color: var(--text-muted);
    }
    .footer {
      margin-top: 2rem;
      font-size: 0.8rem;
      color: #64748b;
      display: flex;
      align-items: center;
      gap: 6px;
    }
    .no-release {
      padding: 2.5rem 0;
      color: var(--text-muted);
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="badge">NexKod Distribution</div>
    
    <div class="logo-header">
      {{if .Project.IconURL}}
        <img class="brand-logo" src="{{.Project.IconURL}}?t={{.NowUnix}}" alt="App Icon">
      {{else}}
        <img class="brand-logo" src="/p/{{.Project.ID}}/icon.png?t={{.NowUnix}}" alt="App Icon">
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
        {{.DownloadLabel}}
      </a>

      {{if .LatestRelease.Changelog}}
      <div class="changelog-box">
        <div class="changelog-title">Changelog</div>
        <div class="changelog-text">{{.LatestRelease.Changelog}}</div>
      </div>
      {{end}}

      <div class="qr-box">
        <img class="qr-img" src="/p/{{.Project.ID}}/qr" alt="Scan to download">
        <p class="qr-caption">Scan with mobile camera to download directly</p>
      </div>
    {{else}}
      <div class="no-release">
        <p>No releases published yet for this project.</p>
        <p style="font-size: 0.8rem; margin-top: 0.5rem;">Upload an APK using CI/CD or the management console.</p>
      </div>
    {{end}}
  </div>

  <div class="footer">
    Powered by <strong>NexEdar</strong> &bull; NexKod Systems
  </div>
</body>
</html>`))

var loginTmpl = template.Must(template.New("login").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>NexEdar Admin Login</title>
  <link rel="icon" type="image/png" href="/logo.png">
  <style>
    :root {
      --bg: #090d16;
      --card: rgba(18, 24, 38, 0.9);
      --border: rgba(255, 255, 255, 0.12);
      --primary: #0284c7;
      --primary-hover: #0369a1;
      --text: #f8fafc;
      --text-muted: #94a3b8;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background: radial-gradient(circle at 50% 10%, #0c233c 0%, var(--bg) 80%);
      color: var(--text);
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      min-height: 100vh;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 1.5rem;
    }
    .card {
      width: 100%;
      max-width: 420px;
      background: var(--card);
      backdrop-filter: blur(20px);
      -webkit-backdrop-filter: blur(20px);
      border: 1px solid var(--border);
      border-radius: 24px;
      padding: 2.5rem 2rem;
      box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.8);
      text-align: center;
    }
    .logo-img {
      width: 88px;
      height: 88px;
      border-radius: 20px;
      object-fit: cover;
      margin-bottom: 1.25rem;
      box-shadow: 0 8px 24px rgba(6, 182, 212, 0.35);
      border: 1px solid rgba(255, 255, 255, 0.15);
    }
    h1 { font-size: 1.6rem; font-weight: 800; margin-bottom: 0.35rem; }
    p { font-size: 0.85rem; color: var(--text-muted); margin-bottom: 1.75rem; }
    .form-group { text-align: left; margin-bottom: 1.25rem; }
    label { display: block; font-size: 0.75rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-muted); margin-bottom: 0.4rem; }
    input {
      width: 100%;
      background: rgba(0, 0, 0, 0.4);
      border: 1px solid var(--border);
      color: #fff;
      padding: 0.85rem 1rem;
      border-radius: 12px;
      font-size: 0.95rem;
      transition: all 0.2s;
    }
    input:focus {
      outline: none;
      border-color: #38bdf8;
      box-shadow: 0 0 0 3px rgba(56, 189, 248, 0.2);
    }
    .btn {
      width: 100%;
      background: linear-gradient(135deg, var(--primary), var(--primary-hover));
      color: #fff;
      border: none;
      font-weight: 700;
      font-size: 1rem;
      padding: 0.95rem;
      border-radius: 14px;
      cursor: pointer;
      margin-top: 0.5rem;
      transition: all 0.2s;
      box-shadow: 0 8px 20px rgba(2, 132, 199, 0.4);
    }
    .btn:hover {
      transform: translateY(-1px);
      box-shadow: 0 12px 25px rgba(2, 132, 199, 0.6);
    }
    .error-msg {
      background: rgba(239, 68, 68, 0.15);
      border: 1px solid rgba(239, 68, 68, 0.3);
      color: #fca5a5;
      padding: 0.75rem;
      border-radius: 10px;
      font-size: 0.85rem;
      margin-bottom: 1.25rem;
    }
  </style>
</head>
<body>
  <div class="card">
    <img src="/logo.png" alt="NexEdar Logo" class="logo-img">
    <h1>NexEdar Console</h1>
    <p>Sign in to manage projects & application releases</p>

    {{if .Error}}
    <div class="error-msg">{{.Error}}</div>
    {{end}}

    <form method="POST" action="/login">
      <div class="form-group">
        <label>Admin Password</label>
        <input type="password" name="password" required autofocus placeholder="Enter admin password">
      </div>
      <button type="submit" class="btn">Sign In &rarr;</button>
    </form>
  </div>
</body>
</html>`))

var adminFuncMap = template.FuncMap{
	"divFloat": func(a int64, b float64) float64 {
		if b == 0 {
			return 0
		}
		return float64(a) / b
	},
}

var adminTmpl = template.Must(template.New("admin").Funcs(adminFuncMap).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>NexEdar Console — Dashboard</title>
  <link rel="icon" type="image/png" href="/logo.png">
  <style>
    :root {
      --bg: #090d16;
      --card: #121826;
      --border: #1e293b;
      --primary: #0284c7;
      --danger: #ef4444;
      --success: #10b981;
      --text: #f8fafc;
      --text-muted: #94a3b8;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background: var(--bg);
      color: var(--text);
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      padding: 2rem 1.5rem;
    }
    .max-w { max-width: 1100px; margin: 0 auto; }
    header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 2rem;
      padding-bottom: 1.25rem;
      border-bottom: 1px solid var(--border);
    }
    .brand {
      display: flex;
      align-items: center;
      gap: 14px;
    }
    .header-logo {
      width: 44px;
      height: 44px;
      border-radius: 12px;
      object-fit: cover;
      box-shadow: 0 4px 12px rgba(6, 182, 212, 0.4);
    }
    h1 { font-size: 1.5rem; font-weight: 800; letter-spacing: -0.02em; }
    .card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 18px;
      padding: 1.75rem;
      margin-bottom: 2rem;
      box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
    }
    h2 { font-size: 1.2rem; margin-bottom: 1rem; color: #38bdf8; font-weight: 700; }
    table { width: 100%; border-collapse: collapse; margin-top: 0.5rem; }
    th, td { text-align: left; padding: 12px 14px; font-size: 0.85rem; border-bottom: 1px solid var(--border); }
    th { color: var(--text-muted); font-weight: 600; text-transform: uppercase; font-size: 0.75rem; letter-spacing: 0.05em; }
    .btn {
      display: inline-flex;
      align-items: center;
      padding: 7px 14px;
      border-radius: 8px;
      border: none;
      font-size: 0.8rem;
      font-weight: 600;
      cursor: pointer;
      text-decoration: none;
      transition: all 0.2s;
    }
    .btn-primary { background: var(--primary); color: #fff; }
    .btn-primary:hover { background: #0369a1; }
    .btn-danger { background: rgba(239, 68, 68, 0.15); border: 1px solid rgba(239, 68, 68, 0.3); color: #f87171; }
    .btn-danger:hover { background: #ef4444; color: #fff; }
    .btn-outline { background: transparent; border: 1px solid var(--border); color: var(--text); }
    .btn-outline:hover { background: rgba(255, 255, 255, 0.05); }
    .form-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
      gap: 1rem;
      margin-bottom: 1.25rem;
    }
    input, textarea, select {
      width: 100%;
      background: #090d16;
      border: 1px solid var(--border);
      color: #fff;
      padding: 9px 13px;
      border-radius: 8px;
      font-size: 0.85rem;
    }
    input:focus, textarea:focus, select:focus {
      outline: none;
      border-color: #38bdf8;
    }
    .badge {
      padding: 3px 9px;
      border-radius: 6px;
      font-size: 0.75rem;
      font-weight: 600;
    }
    .badge-mandatory { background: rgba(239, 68, 68, 0.2); color: #f87171; }
    .badge-optional { background: rgba(148, 163, 184, 0.15); color: #94a3b8; }
    .btn-device {
      background: linear-gradient(135deg, #10b981, #059669);
      color: #fff;
    }
    .btn-device:hover {
      background: linear-gradient(135deg, #34d399, #10b981);
    }
    /* Device farm modal */
    .modal-backdrop {
      display: none;
      position: fixed;
      inset: 0;
      background: rgba(0, 0, 0, 0.75);
      backdrop-filter: blur(8px);
      z-index: 1000;
      align-items: center;
      justify-content: center;
      padding: 1rem;
    }
    .modal-backdrop.open { display: flex; }
    .modal-box {
      background: #111827;
      border: 1px solid rgba(255, 255, 255, 0.15);
      border-radius: 20px;
      width: 100%;
      max-width: 960px;
      height: 85vh;
      display: flex;
      flex-direction: column;
      overflow: hidden;
      box-shadow: 0 25px 60px rgba(0, 0, 0, 0.9);
    }
    .modal-header {
      padding: 1rem 1.5rem;
      border-bottom: 1px solid var(--border);
      display: flex;
      justify-content: space-between;
      align-items: center;
      background: #0f172a;
    }
    .modal-body {
      flex: 1;
      display: grid;
      grid-template-columns: 360px 1fr;
      overflow: hidden;
      background: #030712;
    }
    @media (max-width: 768px) {
      .modal-body { grid-template-columns: 1fr; }
    }
    .device-controls {
      padding: 1.5rem;
      border-right: 1px solid var(--border);
      display: flex;
      flex-direction: column;
      gap: 1.25rem;
      background: #0b0f19;
      overflow-y: auto;
    }
    .device-screen-container {
      position: relative;
      background: #000;
      display: flex;
      align-items: center;
      justify-content: center;
      overflow: hidden;
    }
    .device-screen-iframe {
      width: 100%;
      height: 100%;
      border: none;
    }
  </style>
</head>
<body>
  <div class="max-w">
    <header>
      <div class="brand">
        <img src="/logo.png" alt="NexEdar" class="header-logo">
        <div>
          <h1>NexEdar Console</h1>
          <span style="font-size: 0.8rem; color: var(--text-muted);">NexKod Ecosystem App Distribution</span>
        </div>
      </div>
      <div style="display: flex; gap: 8px;">
        <a href="/healthz" target="_blank" class="btn btn-outline">Health Check</a>
        <form action="/logout" method="POST" style="display: inline;">
          <button type="submit" class="btn btn-outline">Sign Out</button>
        </form>
      </div>
    </header>

    <div class="card">
      <h2>Create New App Project</h2>
      <form action="/admin/create-project" method="POST">
        <div class="form-grid">
          <div>
            <label style="display:block; font-size:0.75rem; color:var(--text-muted); margin-bottom:4px;">Project ID / Slug (e.g. nexronda)</label>
            <input type="text" name="id" required placeholder="nexronda">
          </div>
          <div>
            <label style="display:block; font-size:0.75rem; color:var(--text-muted); margin-bottom:4px;">Display Name</label>
            <input type="text" name="name" required placeholder="NexRonda Intercom">
          </div>
          <div>
            <label style="display:block; font-size:0.75rem; color:var(--text-muted); margin-bottom:4px;">Description</label>
            <input type="text" name="description" placeholder="App description">
          </div>
          <div>
            <label style="display:block; font-size:0.75rem; color:var(--text-muted); margin-bottom:4px;">Custom Icon URL (optional)</label>
            <input type="text" name="icon_url" placeholder="https://...">
          </div>
        </div>
        <button type="submit" class="btn btn-primary">+ Create Project</button>
      </form>
    </div>

    {{range .Projects}}
    <div class="card">
      <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom: 1.25rem;">
        <div>
          <h2>{{.Project.Name}} <span style="font-size:0.85rem; color:var(--text-muted); font-weight:normal;">({{.Project.ID}})</span></h2>
          <span style="font-size:0.85rem; color:var(--text-muted);">
            Disk Used: <strong style="color:#f8fafc;">{{.FormattedDiskUsage}}</strong> across {{.FileCount}} builds &bull;
            <a href="/p/{{.Project.ID}}" target="_blank" style="color:#38bdf8; text-decoration:none; font-weight:600;">Open Download Page &rarr;</a>
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
            <th>Build Code</th>
            <th>Size</th>
            <th>Update Type</th>
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
              {{if .IsMandatory}}<span class="badge badge-mandatory">Mandatory</span>{{else}}<span class="badge badge-optional">Optional</span>{{end}}
            </td>
            <td>{{.DownloadCount}}</td>
            <td>{{.CreatedAt.Format "2006-01-02 15:04"}}</td>
            <td>
              <a href="/p/{{.ProjectID}}/download/{{.ID}}" class="btn btn-outline" style="margin-right:4px;">Download</a>
              {{if $.DeviceFarmConfigured}}
              <button type="button" class="btn btn-device" style="margin-right:4px;" onclick="openDeviceModal({{.ID}}, '{{.ProjectID}}', '{{.VersionName}}')">📱 Test on Device</button>
              {{end}}
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
      <p style="font-size:0.85rem; color:var(--text-muted); margin-top:0.5rem; margin-bottom:1rem;">No releases uploaded yet.</p>
      {{end}}

      <div style="margin-top:1.5rem; padding-top:1rem; border-top:1px solid var(--border);">
        <h3 style="font-size:0.9rem; margin-bottom:0.75rem; color:#94a3b8;">📤 Upload New Binary / App Build</h3>
        <form action="/admin/upload-release" method="POST" enctype="multipart/form-data">
          <input type="hidden" name="project_id" value="{{.Project.ID}}">
          <div class="form-grid">
            <div>
              <label style="display:block; font-size:0.75rem; color:var(--text-muted); margin-bottom:4px;">Artifact File (APK, binary, tar.gz, zip, etc.)</label>
              <input type="file" name="file" required style="padding:6px;">
            </div>
            <div>
              <label style="display:block; font-size:0.75rem; color:var(--text-muted); margin-bottom:4px;">Version Name (e.g. 1.0.0)</label>
              <input type="text" name="version_name" required placeholder="1.0.0">
            </div>
            <div>
              <label style="display:block; font-size:0.75rem; color:var(--text-muted); margin-bottom:4px;">Version Code (e.g. 1)</label>
              <input type="number" name="version_code" required value="1">
            </div>
            <div>
              <label style="display:block; font-size:0.75rem; color:var(--text-muted); margin-bottom:4px;">Changelog</label>
              <input type="text" name="changelog" placeholder="Bug fixes & performance improvements">
            </div>
            <div>
              <label style="display:block; font-size:0.75rem; color:var(--text-muted); margin-bottom:4px;">App Icon (optional, auto-extracted if empty)</label>
              <input type="file" name="icon" accept="image/png,image/jpeg,image/webp" style="padding:4px; font-size:0.75rem;">
            </div>
          </div>
          <button type="submit" class="btn btn-primary">Upload & Publish Release</button>
        </form>
      </div>

      <div style="margin-top:1.25rem; padding-top:1rem; border-top:1px solid var(--border); display:flex; align-items:center; justify-content:space-between;">
        <div style="display:flex; align-items:center; gap:12px;">
          <img src="/p/{{.Project.ID}}/icon.png?t={{$.NowUnix}}" alt="Icon" style="width:36px; height:36px; border-radius:8px; object-fit:cover; border:1px solid var(--border);">
          <span style="font-size:0.8rem; color:var(--text-muted);">Current App Icon</span>
        </div>
        <form action="/admin/upload-icon" method="POST" enctype="multipart/form-data" style="display:flex; gap:8px; align-items:center;">
          <input type="hidden" name="project_id" value="{{.Project.ID}}">
          <input type="file" name="icon" accept="image/png,image/jpeg" required style="font-size:0.75rem; padding:4px;">
          <button type="submit" class="btn btn-outline">Update Icon</button>
        </form>
      </div>
    </div>
    {{end}}
  </div>

  <!-- Device Farm Testing Modal -->
  <div id="deviceModal" class="modal-backdrop">
    <div class="modal-box">
      <div class="modal-header">
        <div style="display:flex; align-items:center; gap:10px;">
          <span style="font-size:1.2rem;">📱</span>
          <div>
            <strong style="font-size:1rem; color:#fff;" id="modalAppTitle">Android Device Farm</strong>
            <div style="font-size:0.75rem; color:var(--text-muted);" id="modalAppSubtitle">Stream and control physical test device</div>
          </div>
        </div>
        <button type="button" class="btn btn-outline" onclick="closeDeviceModal()">✕ Close</button>
      </div>
      <div class="modal-body">
        <div class="device-controls">
          <div>
            <label style="display:block; font-size:0.75rem; font-weight:700; color:var(--text-muted); margin-bottom:6px; text-transform:uppercase;">Select Connected Device</label>
            <select id="deviceSelect" onchange="onDeviceSelected()">
              <option value="">Loading devices...</option>
            </select>
          </div>
          <div>
            <button id="installBtn" class="btn btn-primary" onclick="installOnDevice()" style="width:100%; display:flex; justify-content:center; align-items:center; gap:8px;">
              <span>⚡ Install APK on Device</span>
            </button>
            <div id="installStatus" style="font-size:0.8rem; margin-top:8px; color:var(--text-muted); line-height:1.4;"></div>
          </div>
          <div style="margin-top:auto; font-size:0.75rem; color:var(--text-muted); border-top:1px solid var(--border); padding-top:1rem;">
            <div><strong>Stream Host:</strong> <span id="streamHost">{{.DeviceFarmURL}}</span></div>
            <div style="margin-top:4px;">Low-latency screen streaming with interactive touch and keyboard control via ADB &amp; scrcpy.</div>
          </div>
        </div>
        <div class="device-screen-container">
          <div id="noDeviceOverlay" style="text-align:center; padding:2rem; color:var(--text-muted);">
            <div style="font-size:2.5rem; margin-bottom:1rem;">📱</div>
            <p>Select a device to start live interactive stream</p>
          </div>
          <iframe id="deviceStreamIframe" class="device-screen-iframe" style="display:none;" allow="autoplay; fullscreen"></iframe>
        </div>
      </div>
    </div>
  </div>

  <script>
    let activeReleaseId = null;
    let activeProjectId = null;
    let deviceFarmBase = '{{.DeviceFarmURL}}';

    async function openDeviceModal(releaseId, projectId, versionName) {
      activeReleaseId = releaseId;
      activeProjectId = projectId;
      document.getElementById('modalAppTitle').innerText = 'Test ' + projectId + ' (v' + versionName + ')';
      document.getElementById('modalAppSubtitle').innerText = 'Physical Android device live stream & ADB install';
      document.getElementById('installStatus').innerText = '';
      document.getElementById('deviceModal').classList.add('open');
      await fetchDevices();
    }

    function closeDeviceModal() {
      document.getElementById('deviceModal').classList.remove('open');
      const iframe = document.getElementById('deviceStreamIframe');
      iframe.src = 'about:blank';
      iframe.style.display = 'none';
      document.getElementById('noDeviceOverlay').style.display = 'block';
    }

    async function fetchDevices() {
      const select = document.getElementById('deviceSelect');
      select.innerHTML = '<option value="">Scanning devices...</option>';
      try {
        const res = await fetch('/admin/devices');
        const data = await res.json();
        if (!data.devices || data.devices.length === 0) {
          select.innerHTML = '<option value="">No devices connected</option>';
          return;
        }
        select.innerHTML = '';
        data.devices.forEach((dev, idx) => {
          const opt = document.createElement('option');
          opt.value = dev.id;
          opt.innerText = (dev.model || 'Android Device') + ' (' + dev.id + ') - ' + (dev.status || 'online');
          if (dev.stream_url) {
            opt.dataset.stream = dev.stream_url;
          }
          select.appendChild(opt);
        });
        onDeviceSelected();
      } catch (e) {
        select.innerHTML = '<option value="">Failed to fetch devices: ' + e.message + '</option>';
      }
    }

    function onDeviceSelected() {
      const select = document.getElementById('deviceSelect');
      const deviceId = select.value;
      const iframe = document.getElementById('deviceStreamIframe');
      const overlay = document.getElementById('noDeviceOverlay');
      if (!deviceId) {
        iframe.style.display = 'none';
        overlay.style.display = 'block';
        return;
      }
      const selectedOpt = select.options[select.selectedIndex];
      let streamUrl = selectedOpt ? selectedOpt.dataset.stream : '';
      if (!streamUrl && deviceFarmBase) {
        streamUrl = deviceFarmBase.replace(/\/+$/, '') + '/stream?device=' + encodeURIComponent(deviceId);
      }
      if (streamUrl) {
        iframe.src = streamUrl;
        iframe.style.display = 'block';
        overlay.style.display = 'none';
      }
    }

    async function installOnDevice() {
      const select = document.getElementById('deviceSelect');
      const deviceId = select.value;
      const statusEl = document.getElementById('installStatus');
      const btn = document.getElementById('installBtn');
      if (!deviceId) {
        statusEl.innerHTML = '<span style="color:#ef4444;">Please select a connected device first.</span>';
        return;
      }
      btn.disabled = true;
      statusEl.innerHTML = '<span style="color:#38bdf8;">⏳ Forwarding APK and installing on device ' + deviceId + '...</span>';
      try {
        const res = await fetch('/admin/install-on-device', {
          method: 'POST',
          headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
          body: new URLSearchParams({
            release_id: activeReleaseId,
            device_id: deviceId
          })
        });
        const result = await res.json();
        if (res.ok && result.success) {
          statusEl.innerHTML = '<span style="color:#10b981;">✅ Installed successfully! Check device screen.</span>';
        } else {
          statusEl.innerHTML = '<span style="color:#ef4444;">❌ Installation error: ' + (result.error || 'Failed') + '</span>';
        }
      } catch (err) {
        statusEl.innerHTML = '<span style="color:#ef4444;">❌ Request failed: ' + err.message + '</span>';
      } finally {
        btn.disabled = false;
      }
    }
  </script>
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

func (s *Server) handleLogo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(assets.LogoBytes)
}

func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	if s.isAuthenticatedSession(r) {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = loginTmpl.Execute(w, map[string]any{"Error": ""})
}

func (s *Server) handleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	pass := r.FormValue("password")
	if pass != s.cfg.AdminPassword {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = loginTmpl.Execute(w, map[string]any{"Error": "Invalid admin password"})
		return
	}

	cookieVal := s.generateSessionCookie()
	http.SetCookie(w, &http.Cookie{
		Name:     "nexedar_session",
		Value:    cookieVal,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 7, // 7 days
	})

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "nexedar_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) generateSessionCookie() string {
	mac := hmac.New(sha256.New, []byte(s.cfg.APISecret))
	mac.Write([]byte(s.cfg.AdminPassword))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Server) isAuthenticatedSession(r *http.Request) bool {
	cookie, err := r.Cookie("nexedar_session")
	if err != nil {
		return false
	}
	expected := s.generateSessionCookie()
	return hmac.Equal([]byte(cookie.Value), []byte(expected))
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

	downloadLabel := "Download"
	if latest != nil {
		ext := strings.ToLower(filepath.Ext(latest.FilePath))
		switch ext {
		case ".apk":
			downloadLabel = "Download APK"
		case ".ipa":
			downloadLabel = "Download IPA"
		case ".dmg":
			downloadLabel = "Download DMG"
		case ".exe":
			downloadLabel = "Download EXE"
		case ".zip":
			downloadLabel = "Download ZIP"
		case ".gz", ".tar":
			downloadLabel = "Download Tarball"
		default:
			downloadLabel = "Download Binary"
		}
	}

	nowUnix := time.Now().Unix()
	data := struct {
		Project       *db.Project
		LatestRelease *db.Release
		FormattedSize string
		DownloadLabel string
		NowUnix       int64
	}{
		Project:       proj,
		LatestRelease: latest,
		FormattedSize: formattedSize,
		DownloadLabel: downloadLabel,
		NowUnix:       nowUnix,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = portalTmpl.Execute(w, data)
}

type ProjectAdminData struct {
	Project            db.Project
	Releases           []db.Release
	FormattedDiskUsage string
	FileCount          int
	NowUnix            int64
}

func (s *Server) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	projects, err := s.db.ListProjects()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nowUnix := time.Now().Unix()
	var dataList []ProjectAdminData
	for _, p := range projects {
		rels, _ := s.db.ListReleases(p.ID)
		diskBytes, fileCount, _ := s.storage.GetDiskUsage(p.ID)

		dataList = append(dataList, ProjectAdminData{
			Project:            p,
			Releases:           rels,
			FormattedDiskUsage: formatBytes(diskBytes),
			FileCount:          fileCount,
			NowUnix:            nowUnix,
		})
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = adminTmpl.Execute(w, map[string]any{
		"Projects":             dataList,
		"DeviceFarmURL":        s.cfg.DeviceFarmURL,
		"DeviceFarmConfigured": s.cfg.DeviceFarmURL != "",
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
	id := strings.TrimSpace(r.FormValue("id"))
	name := strings.TrimSpace(r.FormValue("name"))
	desc := strings.TrimSpace(r.FormValue("description"))
	icon := strings.TrimSpace(r.FormValue("icon_url"))

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

func (s *Server) handleAdminUploadRelease(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	projectID := r.FormValue("project_id")
	versionName := r.FormValue("version_name")
	versionCode, _ := strconv.Atoi(r.FormValue("version_code"))
	changelog := r.FormValue("changelog")

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if versionCode <= 0 {
		versionCode = 1
	}

	// Save binary artifact
	filePath, size, sha, err := s.storage.SaveArtifact(projectID, versionName, versionCode, header.Filename, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if explicit icon was provided
	if iconFile, _, err := r.FormFile("icon"); err == nil {
		_, _ = s.storage.SaveIcon(projectID, iconFile)
		iconFile.Close()
		iconURL := fmt.Sprintf("%s/p/%s/icon.png", s.cfg.PublicURL, projectID)
		if p, _ := s.db.GetProject(projectID); p != nil {
			p.IconURL = iconURL
			_ = s.db.UpsertProject(*p)
		}
	}

	_, err = s.db.InsertRelease(db.Release{
		ProjectID:     projectID,
		VersionName:   versionName,
		VersionCode:   versionCode,
		FilePath:      filePath,
		FileSizeBytes: size,
		SHA256:        sha,
		Changelog:     changelog,
		IsMandatory:   false,
		IsActive:      true,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to record release in database: %v", err), http.StatusInternalServerError)
		return
	}

	if s.cfg.MaxBuildsPerApp > 0 {
		s.purgeExcessReleases(projectID, s.cfg.MaxBuildsPerApp)
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (s *Server) handleAdminUploadIcon(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	projectID := r.FormValue("project_id")
	file, _, err := r.FormFile("icon")
	if err != nil {
		http.Error(w, "Icon is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	_, err = s.storage.SaveIcon(projectID, file)
	if err == nil {
		iconURL := fmt.Sprintf("%s/p/%s/icon.png", s.cfg.PublicURL, projectID)
		proj, _ := s.db.GetProject(projectID)
		if proj != nil {
			proj.IconURL = iconURL
			_ = s.db.UpsertProject(*proj)
		}
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (s *Server) handleAdminListDevices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.cfg.DeviceFarmURL == "" {
		json.NewEncoder(w).Encode(map[string]any{
			"devices": []any{},
			"error":   "Device Farm URL not configured",
		})
		return
	}

	targetURL := strings.TrimRight(s.cfg.DeviceFarmURL, "/") + "/api/v1/devices"
	req, err := http.NewRequestWithContext(r.Context(), "GET", targetURL, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{"error": err.Error(), "devices": []any{}})
		return
	}
	if s.cfg.DeviceFarmToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.DeviceFarmToken)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]any{"error": "Failed to reach device farm: " + err.Error(), "devices": []any{}})
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (s *Server) handleAdminInstallOnDevice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.cfg.DeviceFarmURL == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   "Device Farm is not configured (DEVICE_FARM_URL is empty)",
		})
		return
	}

	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "Invalid form request"})
		return
	}

	releaseIDStr := r.FormValue("release_id")
	deviceID := strings.TrimSpace(r.FormValue("device_id"))
	if releaseIDStr == "" || deviceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "release_id and device_id are required"})
		return
	}

	releaseID, _ := strconv.ParseInt(releaseIDStr, 10, 64)
	rel, err := s.db.GetRelease(releaseID)
	if err != nil || rel == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "Release not found"})
		return
	}

	// Read file from storage
	file, err := os.Open(rel.FilePath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "Failed to open release binary: " + err.Error()})
		return
	}
	defer file.Close()

	targetURL := fmt.Sprintf("%s/api/v1/devices/%s/install", strings.TrimRight(s.cfg.DeviceFarmURL, "/"), deviceID)

	// Create pipe to stream multipart form data
	pr, pw := io.Pipe()
	multipartWriter := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		part, err := multipartWriter.CreateFormFile("file", filepath.Base(rel.FilePath))
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		if _, err := io.Copy(part, file); err != nil {
			pw.CloseWithError(err)
			return
		}
		_ = multipartWriter.WriteField("project_id", rel.ProjectID)
		_ = multipartWriter.WriteField("version_name", rel.VersionName)
		_ = multipartWriter.Close()
	}()

	req, err := http.NewRequestWithContext(r.Context(), "POST", targetURL, pr)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": err.Error()})
		return
	}
	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	if s.cfg.DeviceFarmToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.DeviceFarmToken)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "Device farm connection error: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

