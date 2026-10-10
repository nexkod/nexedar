# NexEdar Deployment & Upload Workflow

This guide explains how any project (like **NexRonda**) builds and uploads an APK to **NexEdar**.

---

## 1. Authentication & Security
NexEdar issues an **API Key (Bearer Token)** for your projects. You store this token as an environment variable or in a local `.env` file:
```bash
NEXEDAR_URL="https://edar.nexkod.com"
NEXEDAR_API_KEY="nx_sec_live_99a8b7c6d5e4f3..."
```

---

## 2. Option A: One-Command Local Script (Fastest for Development)

In `NexRonda` (or any app), we add a simple deploy script: `deploy.sh`:

```bash
#!/bin/bash
set -e

# 1. Build release APK
echo "🚀 Building Flutter release APK..."
flutter build apk --release

APK_PATH="build/app/outputs/flutter-apk/app-release.apk"

# 2. Extract version from pubspec.yaml (e.g. 1.0.2+3)
VERSION_NAME=$(grep 'version:' pubspec.yaml | awk '{print $2}' | cut -d'+' -f1)
VERSION_CODE=$(grep 'version:' pubspec.yaml | awk '{print $2}' | cut -d'+' -f2)

echo "📦 Uploading NexRonda v$VERSION_NAME (Build $VERSION_CODE) to NexEdar..."

# 3. Upload to NexEdar
curl -X POST "${NEXEDAR_URL}/api/v1/apps/nexronda/releases" \
  -H "Authorization: Bearer ${NEXEDAR_API_KEY}" \
  -F "file=@${APK_PATH}" \
  -F "version_name=${VERSION_NAME}" \
  -F "version_code=${VERSION_CODE}" \
  -F "is_mandatory=false" \
  -F "changelog=Bug fixes & performance improvements"

echo "✅ Published! Live at ${NEXEDAR_URL}/p/nexronda"
```

To release a new version to all riders, you just run:
```bash
./deploy.sh
```

---

## 3. Option B: GitHub Actions / CI/CD (Fully Automated on Git Tag)

Whenever you tag a release in git (`git tag v1.0.2 && git push --tags`), GitHub Actions builds and uploads automatically without touching your machine:

```yaml
name: Release APK to NexEdar

on:
  push:
    tags:
      - 'v*'

jobs:
  build-and-deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Flutter
        uses: subosito/flutter-action@v2
        with:
          channel: 'stable'

      - name: Build APK
        run: flutter build apk --release

      - name: Publish to NexEdar
        env:
          NEXEDAR_URL: ${{ secrets.NEXEDAR_URL }}
          NEXEDAR_API_KEY: ${{ secrets.NEXEDAR_API_KEY }}
        run: |
          VERSION=${GITHUB_REF_NAME#v}
          curl -f -X POST "${NEXEDAR_URL}/api/v1/apps/nexronda/releases" \
            -H "Authorization: Bearer ${NEXEDAR_API_KEY}" \
            -F "file=@build/app/outputs/flutter-apk/app-release.apk" \
            -F "version_name=${VERSION}" \
            -F "changelog=Automatic build from commit ${{ github.sha }}"
```

---

## 4. Option C: Web Admin UI (Drag-and-Drop)

If you don't want to run command lines:
1. Open `https://edar.nexkod.com/admin` in your browser.
2. Select project **NexRonda**.
3. Drag and drop the `app-release.apk` file.
4. Fill in changelog notes and click **Publish**.
