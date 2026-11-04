package podcasts

import (
	"context"
	"path/filepath"
	"testing"
)

func TestListEpisodesIncludesPlaybackAndFavoriteState(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenStore(filepath.Join(dir, "podcasts.db"), filepath.Join(dir, "media"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	result, err := store.DB().Exec(`INSERT INTO podcasts(feed_url,title) VALUES('https://example.test/feed.xml','Example')`)
	if err != nil {
		t.Fatal(err)
	}
	podcastID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	result, err = store.DB().Exec(`INSERT INTO episodes(podcast_id,guid,title,audio_url) VALUES(?,?,?,?)`, podcastID, "episode-1", "Episode 1", "https://example.test/episode.mp3")
	if err != nil {
		t.Fatal(err)
	}
	episodeID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProgress(context.Background(), episodeID, 15, false); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ToggleFavorite(context.Background(), episodeID); err != nil {
		t.Fatal(err)
	}
	episodes, err := store.ListEpisodes(context.Background(), podcastID)
	if err != nil {
		t.Fatal(err)
	}
	if len(episodes) != 1 || episodes[0].PositionSeconds != 15 || !episodes[0].IsFavorite {
		t.Fatalf("unexpected episode state: %#v", episodes)
	}
}

func TestListEpisodesAcceptsTextAndMissingDates(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenStore(filepath.Join(dir, "podcasts.db"), filepath.Join(dir, "media"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	result, err := store.DB().Exec(`INSERT INTO podcasts(feed_url,title) VALUES('https://example.test/dates.xml','Dates')`)
	if err != nil {
		t.Fatal(err)
	}
	podcastID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.DB().Exec(`INSERT INTO episodes(podcast_id,guid,title,audio_url,pub_date) VALUES(?,?,?,?,?)`, podcastID, "dated", "Dated", "https://example.test/dated.mp3", "2024-01-02 03:04:05")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.DB().Exec(`INSERT INTO episodes(podcast_id,guid,title,audio_url,pub_date) VALUES(?,?,?,?,NULL)`, podcastID, "undated", "Undated", "https://example.test/undated.mp3")
	if err != nil {
		t.Fatal(err)
	}
	episodes, err := store.ListEpisodes(context.Background(), podcastID)
	if err != nil {
		t.Fatal(err)
	}
	if len(episodes) != 2 || episodes[0].PubDate == nil || episodes[1].PubDate != nil {
		t.Fatalf("unexpected dates: %#v", episodes)
	}
}
