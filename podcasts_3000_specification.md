# Podcasts 3000 Specification & Agent Execution Blueprint

## 1. Overview & Architectural Principles

**Podcasts 3000** is a lightweight, cross-platform desktop podcast player designed for Linux and Windows. It follows a minimalist, offline-first philosophy ("suckless" ethos): simple, self-contained, and free from bloat.

### Technology Stack

* **Language & Runtime:** Go

* **GUI Desktop Framework:** Wails v2 (or latest)

* **Frontend:** Standard HTML5 / Vanilla JavaScript / CSS (no heavy framework overhead, node-free build target)

* **Audio Engine:** HTML5 `<audio>` media engine (native browser rendering inside Wails web view for progressive HTTP streaming, seek control, and local audio file playback)

* **Storage & State:** SQLite via a CGO-free pure-Go driver (`modernc.org/sqlite`)

* **Distribution:** Single standalone binary output with local file system allocation for configuration, database, and cached media.

---

## 2. Low-Level Technical Architecture & Engine Rules

### A. SQLite Connection & Concurrency Strategy
Because SQLite will be accessed by concurrent background processes (feed refresh workers, episode media downloaders) alongside user-driven UI operations, all database connection initializations **must** execute the following Pragmas immediately upon connection:

```sql
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;
PRAGMA busy_timeout = 5000;
```

* **WAL Mode:** Write-Ahead Logging prevents database lock errors during simultaneous background writes and foreground reads.
* **Pure-Go Driver:** Must use `modernc.org/sqlite` to guarantee zero CGO dependencies during cross-compilation with `gcc`/`mingw`.

### B. Local Media Asset Server & Streaming Handler
To avoid CORS, webview cross-origin security restrictions, or broken range-request byte headers across WebKitGTK (Linux) and WebView2 (Windows), local cached media files must not be loaded directly via local file path URIs (`file://`).

* The Go backend must spin up a lightweight, local HTTP loopback media handler bound to localhost or integrated directly into Wails' asset server handler.
* The local media handler must explicitly support HTTP Range Requests (`Accept-Ranges: bytes`, `206 Partial Content`) to ensure smooth HTML5 timeline scrubbing and duration parsing in the web frontend.

### C. Download Lifecycle & Handoff Strategy
1. **Streaming Phase:** When a user selects an un-cached episode, playback begins immediately via the external audio enclosure URL while a background Go routine initiates the download.
2. **Partial File Safety:** Background downloads must stream into temporary files with a `.part` extension in the cache directory (e.g., `episode_123.mp3.part`).
3. **Atomic Handoff:** Once the download completes, the temporary file is renamed atomically to its final extension (e.g., `episode_123.mp3`). The backend signals the frontend, which seamlessly updates the `<audio>` source URL to point to the local asset server endpoint without losing current playback position.

---

## 3. Functional Requirements

### A. Feed & Subscription Management

* **Import/Export:** Full OPML import and export capabilities.

* **Subscription Sources:** Direct RSS/Atom feed URL ingestion and built-in iTunes Podcast Search API lookup.

* **Offline-First Caching:** Feeds, metadata, and images are stored locally in SQLite. The application strictly avoids hitting remote feeds during standard navigation and reading.

* **User-Agent Management:**

  * Default HTTP requests must use an explicit custom User-Agent identifying the app (e.g., `Podcasts3000/1.0 (+https://github.com/user/podcasts3000)`). Default HTTP client strings (e.g., `Go-http-client/1.1`) are strictly forbidden.

  * Per-podcast override setting: Support an optional custom User-Agent string per subscription to handle aggressive feed blocking (e.g., spoofing a browser agent with obvious synthetic versioning such as `Mozilla/5.0 (Windows NT 10.0; Win64; x64) Podcasts3000/9999.999.99`).

* **Feed Refresh Strategy:**

  * Auto-refresh check at application startup.

  * Manual refresh trigger available for **All Feeds** (global) or an **Individual Feed**.

### B. Playback & Media Handling

* **Direct Selection:** Explicit episode selection (no queueing mechanism; pick, listen, complete).

* **Seek / Scrubbing:** Full timeline seeking support via standard HTML5 player controls. Speed modification features are explicitly omitted.

* **Progressive Stream to Cache Transition:** Handled as detailed in Section 2.C.

### C. Cache & Data Lifecycle

* **No Auto-Pruning:** Cached files accumulate indefinitely unless deleted by the user.

* **Purge Controls:**

  * Purge all downloaded media across all subscriptions.

  * Purge downloaded media for a specific podcast subscription.

  * Purge downloaded media for a single episode.

### D. User Annotations & Sharing

* **Favorites System:** Ability to toggle a starred state on any episode.

* **User Notes:** Text commentary notes tied directly to starred/favorited episodes.

* **URL Extraction:** Simple one-click clipboard copying for podcast feed URLs and episode audio enclosure URLs.

---

## 4. Database Schema (SQLite)

The database schema is strictly normalized and explicit. Dynamic JSON strings or arbitrary BLOB buckets are forbidden.

```sql
-- Schema Version: 1.0

PRAGMA foreign_keys = ON;

-- Podcast Subscriptions
CREATE TABLE IF NOT EXISTS podcasts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    feed_url TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT,
    site_url TEXT,
    artwork_url TEXT,
    artwork_local_path TEXT,
    custom_user_agent TEXT, -- NULL uses default app agent
    last_refreshed_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Individual Podcast Episodes
CREATE TABLE IF NOT EXISTS episodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    podcast_id INTEGER NOT NULL,
    guid TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    pub_date DATETIME,
    audio_url TEXT NOT NULL,
    local_cache_path TEXT,
    is_downloaded INTEGER NOT NULL DEFAULT 0, -- 0 = False, 1 = True
    duration_seconds INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (podcast_id) REFERENCES podcasts(id) ON DELETE CASCADE,
    UNIQUE(podcast_id, guid)
);

-- Playback State & Progress (Persisted per episode)
CREATE TABLE IF NOT EXISTS playback_state (
    episode_id INTEGER PRIMARY KEY,
    position_seconds INTEGER NOT NULL DEFAULT 0,
    is_completed INTEGER NOT NULL DEFAULT 0, -- 0 = False, 1 = True
    last_played_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (episode_id) REFERENCES episodes(id) ON DELETE CASCADE
);

-- Favorites & User Notes
CREATE TABLE IF NOT EXISTS favorites (
    episode_id INTEGER PRIMARY KEY,
    starred_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    notes TEXT,
    FOREIGN KEY (episode_id) REFERENCES episodes(id) ON DELETE CASCADE
);

-- Indexes for frequent queries
CREATE INDEX IF NOT EXISTS idx_episodes_podcast_id ON episodes(podcast_id);
CREATE INDEX IF NOT EXISTS idx_episodes_pub_date ON episodes(pub_date DESC);
```

---

## 5. Wails Bridge & Backend Contract

The Go application backend should expose a lean service structure (`App` struct) providing thread-safe operations for:

1. **Feed Management:** `ImportOPML(path)`, `ExportOPML(path)`, `SearchITunes(query)`, `Subscribe(url)`, `Unsubscribe(podcastID)`, `RefreshFeeds(podcastID)`, `SetPodcastUserAgent(podcastID, customUA)`.

2. **Media & Cache:** `GetAudioSource(episodeID)` (returns HTTP streaming URL if un-cached, or local asset HTTP URL if downloaded), `PurgeCache(scope, targetID)`.

3. **Playback & State:** `SaveProgress(episodeID, seconds, isCompleted)`, `GetLastPlayed()`.

4. **Favorites & Notes:** `ToggleFavorite(episodeID)`, `SaveNote(episodeID, noteText)`.

---

## 6. Agentic Development & Execution Guidelines

These instructions govern any AI agent (e.g., GitHub Copilot, local LLMs via Koboldcpp) working autonomously or semi-autonomously on this codebase.

### A. Phased Execution Protocol

Development must proceed strictly phase-by-phase. An agent **must not** proceed to a subsequent phase until the current phase's verification criteria are fully satisfied and committed.

* **Phase 1: Core Engine & Storage** (Database migrations with WAL, RSS/Atom parser, OPML import/export, User-Agent handler, unit tests).

* **Phase 2: Media Management & Stream-to-Cache** (Local HTTP range asset handler, `.part` download pipeline, cache manager, state persistence, unit tests).

* **Phase 3: Wails Integration & Bridge Layer** (Exposing Go services to frontend, IPC bounds validation).

* **Phase 4: Frontend UI** (Minimalist HTML/CSS/JS interface, HTML5 player controls, settings modal for User-Agent/Purge).

* **Phase 5: Packaging & Cross-Platform Validation** (Verification on Linux and Windows targets).

### B. Verification & Testing Requirements

* Use standard Go tooling only (`go test ./...`, `go vet`, `golangci-lint` if available).

* Unit tests must cover: RSS feed parsing, schema migrations, state persistence, OPML handling, and HTTP User-Agent header assignment.

* Tests must run and pass cleanly before concluding any execution phase.

* Third-party Go packages must be kept to absolute minimum essential modules (e.g., SQLite driver, Wails runtime). Do not add micro-utility libraries.

### C. Git & Source Control Conventions

* **Isolated Identity:** All agent commits must use a designated bot author:

  ```bash
  git config user.name "Podcasts3000-Builder"
  git config user.email "builder@podcasts3000.local"
  ```

* **Commit Messages:** Follow standard conventional commit format (`feat:`, `fix:`, `test:`, `docs:`, `refactor:`).

* **Clean History:** Atomic commits per task within a phase.

### D. Audit Trail, Documentation & Synthetic Scars

* **Zero Emoji Policy:** Emojis are strictly prohibited in all logs, documentation, source code comments, and commit messages.

* **Audit Documentation:** Maintain the following files in the project root:

  * `CHANGELOG.md`: Human-readable summary of additions and fixes per version.

  * `DEVELOPMENT_LOG.md`: Chronological log of agent actions, decisions, and phase completions.

  * `SCARS.md` (Synthetic Scars Log): Mandatory log where the agent records any failure, unexpected bug, or edge case encountered during build/test iterations. Format:

    ```markdown
    ## [YYYY-MM-DD] Feature/Component Area
    - Issue: Description of what failed or went wrong.
    - Root Cause: Why it failed.
    - Prevention Rule: Concrete rule for the agent to consult in future steps to prevent repeating the mistake.
    ```

* Before beginning any task or phase, the agent **must** read `SCARS.md` to ensure past mistakes are not re-introduced.