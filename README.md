# Podcasts 3000

Minimal, offline-first podcast player built with Go, pure-Go SQLite, and vanilla HTML5.

## Run

```sh
go run ./cmd/podcasts3000
```

Open `http://127.0.0.1:3000`. Data is stored under the platform user configuration directory. Use `-data-dir` to choose another location.

## Verify

```sh
go test ./...
go vet ./...
```

The backend is organized around `internal/podcasts.App`, whose methods are suitable for binding to a Wails v2 frontend. The included executable uses the same methods through a small local HTTP bridge so the application can be exercised without a Node toolchain.
