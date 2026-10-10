# NexEdar 📦

> Self-hosted App Distribution & In-App OTA (Over-The-Air) Auto-Update Service for the NexKod ecosystem.

---

## 🚀 Overview

**NexEdar** (*Edar* = distribute / circulate in Malay) eliminates the friction of public app stores by offering:

1. **Direct Web Portal**: Mobile-responsive download page with QR codes for fast on-the-road APK sharing.
2. **In-App Auto-Update (OTA)**: Built-in update engine for Android apps to detect newer versions, download the update with progress bars, and install without touching the Play Store.
3. **Release Management**: Version tagging, mandatory vs. optional update flags, changelog tracking, and SHA-256 integrity checks.

---

## 📱 Android vs. iOS Reality

- **Android**: Full support. Direct APK sideloading + seamless in-app auto-updating via native `PackageInstaller`.
- **iOS**: Apple restricts direct web-sideloading. Distribution requires Apple TestFlight (up to 10k external testers) or enterprise profile signing.

---

## 🛠️ Stack & Components

- **Backend / API**: Go or Node.js / Fastify microservice
- **Storage**: Local filesystem storage or S3/MinIO bucket
- **Client SDK**: Flutter package (`nexedar_updater`) for instant in-app update dialogs & downloaders
- **Portal**: High-contrast, clean web download UI

---

## 📁 Repository Structure

```text
NexEdar/
├── server/       # Distribution API & Release manager
├── web/          # Public download & QR landing page
└── sdk/flutter/  # Reusable Flutter OTA updater package
```
