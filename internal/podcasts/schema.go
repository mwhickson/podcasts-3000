package podcasts

const schema = `
PRAGMA foreign_keys = ON;
CREATE TABLE IF NOT EXISTS podcasts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    feed_url TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT,
    site_url TEXT,
    artwork_url TEXT,
    artwork_local_path TEXT,
    custom_user_agent TEXT,
    last_refreshed_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS episodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    podcast_id INTEGER NOT NULL,
    guid TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    pub_date DATETIME,
    audio_url TEXT NOT NULL,
    local_cache_path TEXT,
    is_downloaded INTEGER NOT NULL DEFAULT 0,
    duration_seconds INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (podcast_id) REFERENCES podcasts(id) ON DELETE CASCADE,
    UNIQUE(podcast_id, guid)
);
CREATE TABLE IF NOT EXISTS playback_state (
    episode_id INTEGER PRIMARY KEY,
    position_seconds INTEGER NOT NULL DEFAULT 0,
    is_completed INTEGER NOT NULL DEFAULT 0,
    last_played_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (episode_id) REFERENCES episodes(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS favorites (
    episode_id INTEGER PRIMARY KEY,
    starred_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    notes TEXT,
    FOREIGN KEY (episode_id) REFERENCES episodes(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_episodes_podcast_id ON episodes(podcast_id);
CREATE INDEX IF NOT EXISTS idx_episodes_pub_date ON episodes(pub_date DESC);
`
