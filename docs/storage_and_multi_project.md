# NexEdar: Multi-Project & Storage Lifecycle Management

## 1. Multi-Project Architecture
NexEdar organizes apps using unique **Project Slugs / App IDs**:

```text
/data/apps/
├── nexronda/              # Project 1: Riding Intercom
│   ├── v1.0.0-1.apk
│   ├── v1.0.1-2.apk
│   └── v1.1.0-3.apk       (active / latest)
├── nexawan-mobile/        # Project 2: Cloud Manager Mobile
│   ├── v0.8.0-1.apk
│   └── v0.9.0-2.apk       (active / latest)
└── custom-app/            # Project N...
```

### URLs per Project:
- **Web Download Page**: `https://edar.nexkod.com/p/nexronda` (renders project-specific QR code, icon, changelog, and download button).
- **Update Check API**: `GET /api/v1/apps/nexronda/latest`
- **Latest APK Direct Link**: `GET /p/nexronda/download` (always redirects to the latest active APK file).

---

## 2. Managing & Cleaning Old APK Files

Because APKs are 20–50 MB each, accumulating hundreds of builds will fill disk space over time. NexEdar handles this with **3 automated cleanup strategies**:

### Strategy A: Retention Rule (Automated "Keep Last N Builds")
- In NexEdar config:
  ```yaml
  storage:
    max_builds_per_app: 3     # Keeps only the 3 newest APKs per project
    cleanup_cron: "0 3 * * *" # Runs daily at 3:00 AM
  ```
- When a new build (`v1.1.0`) is published, builds older than the configured limit (e.g. `v1.0.0`) are automatically deleted from the disk and marked `archived` in SQLite.

### Strategy B: Admin Web Dashboard (One-Click Cleanup)
- Simple password-protected `/admin` interface:
  - Table of all projects and their respective builds.
  - Disk space usage per project (e.g., `NexRonda: 98 MB across 3 builds`).
  - **"Delete"** button next to any build.
  - **"Purge Old Builds"** button to instantly free space across all projects.

### Strategy C: Upload Flag / CI/CD Parameter
- When publishing a new build via `curl` or CI/CD script:
  ```bash
  curl -X POST https://edar.nexkod.com/api/v1/apps/nexronda/releases \
    -H "Authorization: Bearer $NEXEDAR_SECRET" \
    -F "file=@app-release.apk" \
    -F "version=1.1.0" \
    -F "purge_previous=true"  # Instantly cleans up older versions on upload
  ```

---

## 3. Database Schema (SQLite)

```sql
CREATE TABLE projects (
    id TEXT PRIMARY KEY,          -- 'nexronda'
    name TEXT NOT NULL,          -- 'NexRonda'
    description TEXT,
    icon_url TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE releases (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
    version_name TEXT NOT NULL,  -- '1.1.0'
    version_code INTEGER NOT NULL, -- 3
    file_path TEXT NOT NULL,     -- '/data/apps/nexronda/v1.1.0-3.apk'
    file_size_bytes INTEGER NOT NULL,
    sha256 TEXT NOT NULL,
    changelog TEXT,
    is_mandatory BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    download_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```
