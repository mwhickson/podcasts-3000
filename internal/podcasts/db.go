package podcasts

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	db       *sql.DB
	cacheDir string
}

func OpenStore(path, cacheDir string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("create cache directory: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(8)
	if _, err = db.Exec(`PRAGMA journal_mode = WAL; PRAGMA foreign_keys = ON; PRAGMA busy_timeout = 5000;`); err != nil {
		db.Close()
		return nil, fmt.Errorf("configure database: %w", err)
	}
	if _, err = db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}
	return &Store{db: db, cacheDir: cacheDir}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) DB() *sql.DB { return s.db }

func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

func (s *Store) ListEpisodes(ctx context.Context, podcastID int64) ([]Episode, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT e.id,e.podcast_id,e.guid,e.title,COALESCE(e.description,''),e.audio_url,COALESCE(e.local_cache_path,''),e.pub_date,e.is_downloaded,e.duration_seconds,COALESCE(ps.position_seconds,0),COALESCE(ps.is_completed,0),CASE WHEN f.episode_id IS NULL THEN 0 ELSE 1 END,COALESCE(f.notes,'') FROM episodes e LEFT JOIN playback_state ps ON ps.episode_id=e.id LEFT JOIN favorites f ON f.episode_id=e.id WHERE e.podcast_id=? ORDER BY e.pub_date DESC`, podcastID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var episodes []Episode
	for rows.Next() {
		var episode Episode
		var downloaded, completed, favorite int
		if err := rows.Scan(&episode.ID, &episode.PodcastID, &episode.GUID, &episode.Title, &episode.Description, &episode.AudioURL, &episode.LocalCachePath, &episode.PubDate, &downloaded, &episode.DurationSeconds, &episode.PositionSeconds, &completed, &favorite, &episode.Notes); err != nil {
			return nil, err
		}
		episode.IsDownloaded = downloaded != 0
		episode.IsCompleted = completed != 0
		episode.IsFavorite = favorite != 0
		episodes = append(episodes, episode)
	}
	return episodes, rows.Err()
}
