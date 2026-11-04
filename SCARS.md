# Synthetic Scars Log - Podcasts 3000

This document tracks mistakes, edge cases, build failures, and invalid technical decisions encountered during the development of Podcasts 3000. 

All AI agents working on this codebase MUST read this log prior to executing any phase or writing new code, and MUST append new entries whenever a failure or regression occurs.

---

## [2026-11-03] Database Driver Selection & CGO Dependencies
- Issue: Attempting to use `github.com/mattn/go-sqlite3` caused cross-compilation errors on Windows targets lacking a full CGO toolchain environment.
- Root Cause: `mattn/go-sqlite3` relies on C bindings via CGO.
- Prevention Rule: Always use `modernc.org/sqlite` as the pure-Go SQLite driver. Never import standard CGO SQLite packages.

## [2026-11-03] Database Concurrency & File Locking
- Issue: Background feed refreshing threw `database is locked` errors while the UI attempted to read playback history.
- Root Cause: SQLite default journal mode (delete) locks the entire database during write transactions.
- Prevention Rule: Always execute `PRAGMA journal_mode = WAL;` and `PRAGMA busy_timeout = 5000;` on every database connection open event in Go.

## [2026-11-03] HTML5 Audio Media File Paths
- Issue: Loading downloaded audio files via direct local file URIs (`file:///path/to/file.mp3`) was blocked by webview security contexts on Linux WebKitGTK.
- Root Cause: Browser engines restrict local filesystem access from frontend application origins.
- Prevention Rule: Route all local media file playback through the Go internal HTTP asset server with explicit `Accept-Ranges: bytes` headers. Never pass raw `file://` URIs to the frontend HTML5 `<audio>` element.

## [2026-11-03] External HTTP User-Agent Rejections
- Issue: Standard HTTP GET requests to third-party RSS feeds were returning HTTP 403 Forbidden responses during automated parsing tests.
- Root Cause: Standard Go HTTP client headers (`Go-http-client/1.1`) are actively blocked by CDN feed providers (e.g., Cloudflare, Imperva).
- Prevention Rule: Every HTTP client request originating from Go must explicitly define an app-identifying User-Agent string or respect the per-podcast custom User-Agent override stored in SQLite.

## [2026-11-03] Incomplete Audio Download Handoffs
- Issue: Directing the HTML5 player to a partially downloaded local file caused playback to terminate early when the player reached the temporary file's end boundary.
- Root Cause: The app pointed to the cached file before the background HTTP download completed.
- Prevention Rule: Downloads must be saved with a `.part` extension until completed 100%. Only switch the frontend `<audio>` target to the local file URL after an atomic file rename from `.part` to `.mp3` occurs.

## [2026-11-03] Feed XML Model and Null Database Values
- Issue: Initial feed parsing conflicted on repeated XML link elements, and scanning nullable custom User-Agent values into strings failed.
- Root Cause: `encoding/xml` requires distinct field paths for repeated tags, and SQLite NULL cannot scan directly into a Go string.
- Prevention Rule: Model repeated XML elements with one slice and scan nullable database columns through `sql.NullString` or `COALESCE`.

## [2026-11-03] JSON and XML Struct Tags
- Issue: Initial grouped struct tags caused `go vet` failures because one tag was applied to several fields.
- Root Cause: Go struct tags are per-field and cannot be safely grouped when fields have different names.
- Prevention Rule: Declare explicit tags on every serialized field and run `go vet ./...` after model changes.