package podcasts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestParseFeed(t *testing.T) {
	data := []byte(`<rss xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd"><channel><title>Example</title><description><![CDATA[<p>Desc &amp; details</p><p>More</p>]]></description><item><guid>one</guid><title>Episode one</title><description><![CDATA[<p>Shownotes</p><ul><li>One</li><li>Two</li></ul>]]></description><pubDate>Mon, 02 Jan 2023 15:04:05 +0000</pubDate><itunes:duration>1:02:03</itunes:duration><enclosure url="https://example.test/one.mp3"/></item></channel></rss>`)
	podcast, episodes, err := parseFeed(data)
	if err != nil || podcast.Title != "Example" || podcast.Description != "Desc & details\nMore" || len(episodes) != 1 || episodes[0].AudioURL != "https://example.test/one.mp3" || episodes[0].Description != "Shownotes\nOne\nTwo" || episodes[0].DurationSeconds != 3723 {
		t.Fatalf("unexpected parse result: %#v %#v %v", podcast, episodes, err)
	}
}

func TestHTMLToTextPreservesContentWithoutMarkup(t *testing.T) {
	got := htmlToText(`<div>Hello <strong>world</strong><br>5 &lt; 6</div>`)
	if got != "Hello world\n5 < 6" {
		t.Fatalf("htmlToText() = %q", got)
	}
}

func TestSubscribeUsesExplicitUserAgentAndPersistsState(t *testing.T) {
	var gotUA string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<rss><channel><title>Test</title><item><guid>g</guid><title>Episode</title><enclosure url="https://example.test/audio.mp3"/></item></channel></rss>`))
	}))
	defer server.Close()
	dir := t.TempDir()
	store, err := OpenStore(filepath.Join(dir, "podcasts.db"), filepath.Join(dir, "media"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	podcast, err := store.Subscribe(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if gotUA != DefaultUserAgent {
		t.Fatalf("User-Agent = %q", gotUA)
	}
	if podcast.ID == 0 {
		t.Fatal("expected persisted podcast ID")
	}
	if err = store.SaveProgress(context.Background(), 1, 42, false); err != nil {
		t.Fatal(err)
	}
	last, err := store.GetLastPlayed(context.Background())
	if err != nil || last.PositionSeconds != 42 {
		t.Fatalf("last played = %#v, err=%v", last, err)
	}
}

func TestOPMLRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenStore(filepath.Join(dir, "podcasts.db"), filepath.Join(dir, "media"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<rss><channel><title>OPML podcast</title></channel></rss>`))
	}))
	defer server.Close()
	if _, err = store.Subscribe(context.Background(), server.URL); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "feeds.opml")
	if err = store.ExportOPML(context.Background(), path); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestListPodcastsReturnsEmptySliceWhenUnsubscribed(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenStore(filepath.Join(dir, "podcasts.db"), filepath.Join(dir, "media"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	podcasts, err := store.ListPodcasts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if podcasts == nil {
		t.Fatal("ListPodcasts returned nil; Wails would serialize this as null")
	}
	if len(podcasts) != 0 {
		t.Fatalf("podcast count = %d, want 0", len(podcasts))
	}
}
