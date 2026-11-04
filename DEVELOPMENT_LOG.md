# Development Log

## 2026-11-03

- Read `SCARS.md` before implementation and followed its storage, media, and User-Agent constraints.
- Implemented Phase 1: schema migrations, RSS/Atom parsing, subscription refresh, OPML support, iTunes lookup, and tests.
- Implemented Phase 2: media cache lifecycle, atomic `.part` downloads, range-capable local serving, playback state, favorites, and notes.
- Implemented Phase 3 and 4 baseline: application bridge methods and a node-free browser interface served by the Go executable.
- Verified with `go test ./...` and `go vet ./...`.
- Replaced the HTTP proof-of-concept entrypoint with a Wails v2 desktop application.
- Embedded the frontend in the binary, bound application methods through Wails IPC, added startup feed refresh, and retained a loopback range-serving media endpoint for cached audio.
- Verified the Wails application with `go build ./cmd/podcasts3000`, `go test ./...`, and `go vet ./...`.
- Confirmed direct Linux execution requires `production webkit2_41` tags and documented the tagged run target.
- Diagnosed subscription binding failures: the bound type is `podcasts.App`, not `main.App`; aligned the frontend wrapper and generated-binding path with that namespace.
- Added `make runtime-test`, an embedded-frontend binding test, and a short graphical Wails startup smoke test.
- Implemented the next application phase: iTunes search UI, OPML path controls, episode favorites and notes, and scoped cache purge controls.
- Refactored the frontend UX: search results now use a closable sidebar sub-view, episode actions are compact pill controls, OPML uses native Wails dialogs, and active podcast artwork plus a fixed player bar are presented consistently.
