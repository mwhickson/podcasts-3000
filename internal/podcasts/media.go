package podcasts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func cacheName(id int64, source string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d:%s", id, source)))
	return hex.EncodeToString(sum[:]) + ".mp3"
}

func (s *Store) GetAudioSource(ctx context.Context, episodeID int64, baseURL string) (string, error) {
	var audio, local string
	if err := s.db.QueryRowContext(ctx, `SELECT audio_url,COALESCE(local_cache_path,'') FROM episodes WHERE id=?`, episodeID).Scan(&audio, &local); err != nil {
		return "", err
	}
	if local != "" {
		if _, err := os.Stat(local); err == nil {
			return strings.TrimRight(baseURL, "/") + "/media/" + filepath.Base(local), nil
		}
	}
	go func() {
		if err := s.Download(context.Background(), episodeID); err != nil {
			log.Printf("podcasts3000: cache episode %d: %v", episodeID, err)
			return
		}
	}()
	return audio, nil
}

func (s *Store) Download(ctx context.Context, episodeID int64) error {
	var audio, customUserAgent string
	if err := s.db.QueryRowContext(ctx, `SELECT e.audio_url,COALESCE(p.custom_user_agent,'') FROM episodes e JOIN podcasts p ON p.id=e.podcast_id WHERE e.id=?`, episodeID).Scan(&audio, &customUserAgent); err != nil {
		return err
	}
	name := cacheName(episodeID, audio)
	final := filepath.Join(s.cacheDir, name)
	part := final + ".part"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, audio, nil)
	if err != nil {
		return err
	}
	if strings.TrimSpace(customUserAgent) == "" {
		customUserAgent = DefaultUserAgent
	}
	req.Header.Set("User-Agent", customUserAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download episode: HTTP %s", resp.Status)
	}
	file, err := os.Create(part)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(file, resp.Body)
	closeErr := file.Close()
	if copyErr != nil {
		os.Remove(part)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(part)
		return closeErr
	}
	if err = os.Rename(part, final); err != nil {
		return fmt.Errorf("complete download: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `UPDATE episodes SET local_cache_path=?,is_downloaded=1 WHERE id=?`, final, episodeID)
	return err
}

func (s *Store) ServeMedia(w http.ResponseWriter, r *http.Request) {
	name := filepath.Base(r.URL.Path)
	if name == "." || name == ".." || strings.Contains(name, string(filepath.Separator)) {
		http.Error(w, "invalid media path", http.StatusBadRequest)
		return
	}
	http.ServeFile(w, r, filepath.Join(s.cacheDir, name))
}

func (s *Store) PurgeCache(ctx context.Context, scope string, targetID int64) error {
	query := `SELECT id,local_cache_path FROM episodes WHERE local_cache_path IS NOT NULL AND local_cache_path != ''`
	args := []any{}
	switch scope {
	case "episode":
		query += ` AND id=?`
		args = append(args, targetID)
	case "podcast":
		query += ` AND podcast_id=?`
		args = append(args, targetID)
	case "all":
	default:
		return fmt.Errorf("invalid purge scope %q", scope)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		var path string
		if err = rows.Scan(&id, &path); err != nil {
			return err
		}
		if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
			return removeErr
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		return err
	}
	for _, id := range ids {
		if _, err = s.db.ExecContext(ctx, `UPDATE episodes SET local_cache_path=NULL,is_downloaded=0 WHERE id=?`, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) SaveProgress(ctx context.Context, episodeID int64, seconds int, completed bool) error {
	if seconds < 0 {
		return fmt.Errorf("position cannot be negative")
	}
	done := 0
	if completed {
		done = 1
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO playback_state(episode_id,position_seconds,is_completed,last_played_at) VALUES(?,?,?,CURRENT_TIMESTAMP) ON CONFLICT(episode_id) DO UPDATE SET position_seconds=excluded.position_seconds,is_completed=excluded.is_completed,last_played_at=CURRENT_TIMESTAMP`, episodeID, seconds, done)
	return err
}

func (s *Store) GetLastPlayed(ctx context.Context) (Episode, error) {
	var ep Episode
	var completed, favorite int
	err := s.db.QueryRowContext(ctx, `SELECT e.id,e.podcast_id,e.guid,e.title,COALESCE(e.description,''),e.audio_url,COALESCE(e.local_cache_path,''),e.pub_date,e.is_downloaded,e.duration_seconds,ps.position_seconds,ps.is_completed,CASE WHEN f.episode_id IS NULL THEN 0 ELSE 1 END,COALESCE(f.notes,'') FROM playback_state ps JOIN episodes e ON e.id=ps.episode_id LEFT JOIN favorites f ON f.episode_id=e.id ORDER BY ps.last_played_at DESC LIMIT 1`).Scan(&ep.ID, &ep.PodcastID, &ep.GUID, &ep.Title, &ep.Description, &ep.AudioURL, &ep.LocalCachePath, &ep.PubDate, &ep.IsDownloaded, &ep.DurationSeconds, &ep.PositionSeconds, &completed, &favorite, &ep.Notes)
	ep.IsCompleted = completed != 0
	ep.IsFavorite = favorite != 0
	return ep, err
}

func (s *Store) ToggleFavorite(ctx context.Context, episodeID int64) (bool, error) {
	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM favorites WHERE episode_id=?`, episodeID).Scan(&exists)
	if err != nil {
		return false, err
	}
	if exists > 0 {
		_, err = s.db.ExecContext(ctx, `DELETE FROM favorites WHERE episode_id=?`, episodeID)
		return false, err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO favorites(episode_id) VALUES(?)`, episodeID)
	return true, err
}
func (s *Store) SaveNote(ctx context.Context, episodeID int64, note string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO favorites(episode_id,notes) VALUES(?,?) ON CONFLICT(episode_id) DO UPDATE SET notes=excluded.notes`, episodeID, note)
	return err
}

func (s *Store) DownloadedPath(episodeID int64) (string, error) {
	var p string
	err := s.db.QueryRow(`SELECT local_cache_path FROM episodes WHERE id=?`, episodeID).Scan(&p)
	return p, err
}
func ContentRange(w http.ResponseWriter, start, end, size int64) {
	w.Header().Set("Content-Range", "bytes "+strconv.FormatInt(start, 10)+"-"+strconv.FormatInt(end, 10)+"/"+strconv.FormatInt(size, 10))
}
