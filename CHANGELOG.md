# Changelog

## Unreleased

- Added pure-Go SQLite storage with WAL, foreign keys, busy timeout, and normalized podcast schema.
- Added RSS/Atom subscription refresh, OPML import/export, explicit User-Agent handling, and iTunes search.
- Added safe media downloads with `.part` files and atomic handoff, HTTP range serving, cache purge, and playback persistence.
- Added favorites, notes, a small HTTP bridge, and a vanilla HTML/CSS/JavaScript interface.
- Fixed Wails IPC binding resolution for the `podcasts.App` package namespace and added tagged runtime smoke checks.
- Added iTunes podcast search and direct subscribe results.
- Added OPML import/export controls, episode starring and notes, and all/podcast/episode cache purge controls.
- Refined the Wails UI with isolated search results, compact episode actions, native OPML dialogs, artwork headers, and a fixed player bar.
- Added podcast unsubscribe actions that remove the subscription and all associated cached and partial media files.
- Fixed the empty-library state after removing the final podcast.
- Replaced browser alerts and confirmations with native Wails message dialogs.
- Fixed affirmative responses from native confirmation dialogs on Linux.
- Hardened feed loading for missing fields and SQLite date values.
- Converted HTML podcast and episode descriptions to readable plain text.
- Restored the last played episode and saved playback position when starting the application or switching episodes.
- Added sticky episode search, unplayed filtering, and name/publication-date sorting controls.
