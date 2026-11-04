package podcasts

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPurgeCacheScopes(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenStore(filepath.Join(dir, "podcasts.db"), filepath.Join(dir, "media"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	result, err := store.DB().Exec(`INSERT INTO podcasts(feed_url,title) VALUES('https://example.test/feed','Example')`)
	if err != nil {
		t.Fatal(err)
	}
	podcastID, _ := result.LastInsertId()
	result, err = store.DB().Exec(`INSERT INTO episodes(podcast_id,guid,title,audio_url) VALUES(?,?,?,?)`, podcastID, "one", "One", "https://example.test/one.mp3")
	if err != nil {
		t.Fatal(err)
	}
	episodeID, _ := result.LastInsertId()
	cachePath := filepath.Join(dir, "media", "one.mp3")
	if err := os.WriteFile(cachePath, []byte("audio"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB().Exec(`UPDATE episodes SET local_cache_path=?,is_downloaded=1 WHERE id=?`, cachePath, episodeID); err != nil {
		t.Fatal(err)
	}
	if err := store.PurgeCache(context.Background(), "episode", episodeID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Fatalf("cache file still exists, stat error: %v", err)
	}
	var downloaded int
	if err := store.DB().QueryRow(`SELECT is_downloaded FROM episodes WHERE id=?`, episodeID).Scan(&downloaded); err != nil {
		t.Fatal(err)
	}
	if downloaded != 0 {
		t.Fatalf("is_downloaded = %d, want 0", downloaded)
	}
}

func TestUnsubscribeRemovesPodcastAndCacheFiles(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenStore(filepath.Join(dir, "podcasts.db"), filepath.Join(dir, "media"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	result, err := store.DB().Exec(`INSERT INTO podcasts(feed_url,title) VALUES('https://example.test/unsubscribe','Unsubscribe me')`)
	if err != nil {
		t.Fatal(err)
	}
	podcastID, _ := result.LastInsertId()
	result, err = store.DB().Exec(`INSERT INTO episodes(podcast_id,guid,title,audio_url) VALUES(?,?,?,?)`, podcastID, "episode", "Episode", "https://example.test/audio.mp3")
	if err != nil {
		t.Fatal(err)
	}
	episodeID, _ := result.LastInsertId()
	cachePath := filepath.Join(dir, "media", "episode.mp3")
	partialPath := cachePath + ".part"
	for _, path := range []string{cachePath, partialPath} {
		if err := os.WriteFile(path, []byte("audio"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.DB().Exec(`UPDATE episodes SET local_cache_path=?,is_downloaded=1 WHERE id=?`, cachePath, episodeID); err != nil {
		t.Fatal(err)
	}
	if err := store.Unsubscribe(context.Background(), podcastID); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{cachePath, partialPath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("cache file %q still exists, stat error: %v", path, err)
		}
	}
	var count int
	if err := store.DB().QueryRow(`SELECT COUNT(*) FROM podcasts WHERE id=?`, podcastID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("podcast count = %d, want 0", count)
	}
	if err := store.DB().QueryRow(`SELECT COUNT(*) FROM episodes WHERE id=?`, episodeID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("episode count = %d, want 0", count)
	}
}
