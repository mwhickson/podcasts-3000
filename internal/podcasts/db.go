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
	episodes := make([]Episode, 0)
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

func (s *Store) Unsubscribe(ctx context.Context, podcastID int64) error {
	rows, err := s.db.QueryContext(ctx, `SELECT local_cache_path FROM episodes WHERE podcast_id=? AND local_cache_path IS NOT NULL AND local_cache_path != ''`, podcastID)
	if err != nil {
		return fmt.Errorf("find podcast cache files: %w", err)
	}
	var paths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			rows.Close()
			return fmt.Errorf("read podcast cache file: %w", err)
		}
		paths = append(paths, path, path+".part")
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("list podcast cache files: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close podcast cache files: %w", err)
	}
	for _, path := range paths {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove podcast cache file %q: %w", path, err)
		}
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM podcasts WHERE id=?`, podcastID)
	if err != nil {
		return fmt.Errorf("delete podcast: %w", err)
	}
	if count, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("check podcast deletion: %w", err)
	} else if count == 0 {
		return fmt.Errorf("podcast %d not found", podcastID)
	}
	return nil
}
