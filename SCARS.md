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

## [2026-11-03] Desktop Shell Integration
- Issue: The first executable exposed the application through a local HTTP server, which did not satisfy the required Wails desktop architecture.
- Root Cause: The initial implementation prioritized quickly exercising the backend and browser UI before wiring the desktop lifecycle.
- Prevention Rule: Treat the Wails entrypoint, embedded assets, bound methods, startup lifecycle, and platform options as required deliverables rather than optional packaging work.

## [2026-11-03] Wails Linux Build Tags
- Issue: Running the application without build tags produced `Wails applications will not build without the correct build tags`.
- Root Cause: Wails intentionally uses a guard implementation when neither `dev` nor `production` is selected; Linux WebKitGTK integration also requires the matching `webkit2_41` tag.
- Prevention Rule: Run Linux Wails applications with `go run -tags 'production webkit2_41' ./cmd/podcasts3000` or use the repository `make run` target. Use the relative `./cmd/podcasts3000` package path, and ensure `/usr/local/go/bin` is on `PATH`.

## [2026-11-03] Wails Binding Initialization Race
- Issue: The frontend called `window.go.main.App` immediately during module evaluation and displayed `TypeError: undefined is not an object` when Wails had not injected bindings yet.
- Root Cause: The bound value is `podcasts.App`, so Wails exposes it at `window.go.podcasts.App`; the frontend also needed to tolerate asynchronous runtime initialization.
- Prevention Rule: Resolve Wails bindings lazily and wait for `globalThis.go.podcasts.App` before invoking methods. Validate the embedded frontend namespace with `make runtime-test`. Treat `go run` as a temporary executable workflow; use `make build` for a persistent binary.

## [2026-11-03] Wails Frontend Source Location
- Issue: The root `web/` directory appeared to be the active frontend location even though Wails embedded `cmd/podcasts3000/frontend`.
- Root Cause: The initial HTTP proof of concept used root-level web assets before the Wails migration; the directory was left empty after assets moved.
- Prevention Rule: Keep Wails HTML, CSS, JavaScript, and generated binding wrappers under `cmd/podcasts3000/frontend`; do not maintain a second root-level frontend tree.

## [2026-11-04] Native Wails File Dialogs
- Issue: The UI initially requested raw OPML filesystem paths in a text input, which bypassed native desktop file selection.
- Root Cause: Wails v2.10.2 exposes `OpenFileDialog` and `SaveFileDialog` through the Go runtime package, not as JavaScript `window.runtime` methods.
- Prevention Rule: Bind Go methods that invoke `github.com/wailsapp/wails/v2/pkg/runtime` dialogs and keep OPML path entry out of the frontend. Native dialogs provide platform path handling, including home-directory navigation.

## [2026-11-04] Podcast Unsubscribe Cache Cleanup
- Issue: The initial unsubscribe method deleted only the podcast database row and left downloaded media on disk.
- Root Cause: Unsubscribe bypassed the cache lifecycle helpers and issued a direct SQL delete.
- Prevention Rule: Podcast deletion must enumerate and remove final and `.part` episode cache files before deleting the subscription, then verify both database cascade and filesystem cleanup in tests.

## [2026-11-04] Empty Podcast List Serialization
- Issue: After unsubscribing the final podcast, the frontend received `null` and failed with `TypeError: null is not an object (evaluating 'podcasts.map')`.
- Root Cause: A nil Go slice serializes through Wails as JSON `null` rather than an empty array.
- Prevention Rule: Collection-returning backend methods must initialize empty slices, and frontend collection consumers must normalize unexpected null responses to `[]`.

## [2026-11-04] Native Wails Messageboxes
- Issue: Browser `alert()` and `confirm()` calls produced inconsistent webview messageboxes instead of native desktop dialogs.
- Root Cause: Frontend code used browser globals even though Wails provides platform-native message dialogs through `runtime.MessageDialog`.
- Prevention Rule: Use Wails-native UI APIs whenever possible. Route confirmations, errors, and informational messages through bound Go methods backed by `runtime.MessageDialog`; do not add browser `alert`, `confirm`, or `prompt` calls.

## [2026-11-04] Native Confirmation Response Labels
- Issue: Unsubscribe stopped after replacing browser `confirm()` with a native Wails question dialog.
- Root Cause: Linux Wails question dialogs return `"Yes"`/`"No"` regardless of the custom button labels, while the binding only treated `"OK"` as affirmative.
- Prevention Rule: For native question dialogs, use platform-standard Yes/No response handling and accept the documented affirmative response (`"Yes"`; retain `"OK"` compatibility where applicable).