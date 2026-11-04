package podcasts

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type App struct {
	Store        *Store
	MediaBaseURL string
}

func (a *App) ImportOPML(path string) error { return a.Store.ImportOPML(context.Background(), path) }
func (a *App) ExportOPML(path string) error { return a.Store.ExportOPML(context.Background(), path) }
func (a *App) Subscribe(url string) (Podcast, error) {
	return a.Store.Subscribe(context.Background(), url)
}
func (a *App) Unsubscribe(id int64) error {
	_, err := a.Store.DB().Exec(`DELETE FROM podcasts WHERE id=?`, id)
	return err
}
func (a *App) RefreshFeeds(id int64) error { return a.Store.RefreshFeeds(context.Background(), id) }
func (a *App) RefreshAllFeeds() error {
	ps, err := a.Store.ListPodcasts(context.Background())
	if err != nil {
		return err
	}
	for _, p := range ps {
		if err = a.RefreshFeeds(p.ID); err != nil {
			return err
		}
	}
	return nil
}
func (a *App) SetPodcastUserAgent(id int64, ua string) error {
	return a.Store.SetPodcastUserAgent(context.Background(), id, ua)
}
func (a *App) GetAudioSource(id int64) (string, error) {
	return a.Store.GetAudioSource(context.Background(), id, a.MediaBaseURL)
}
func (a *App) PurgeCache(scope string, id int64) error {
	return a.Store.PurgeCache(context.Background(), scope, id)
}
func (a *App) SaveProgress(id int64, seconds int, completed bool) error {
	return a.Store.SaveProgress(context.Background(), id, seconds, completed)
}
func (a *App) GetLastPlayed() (Episode, error) { return a.Store.GetLastPlayed(context.Background()) }
func (a *App) ToggleFavorite(id int64) (bool, error) {
	return a.Store.ToggleFavorite(context.Background(), id)
}
func (a *App) SaveNote(id int64, note string) error {
	return a.Store.SaveNote(context.Background(), id, note)
}
func (a *App) MediaHandler() http.Handler { return http.HandlerFunc(a.Store.ServeMedia) }

func (a *App) SearchITunes(query string) ([]SearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}
	endpoint := "https://itunes.apple.com/search?media=podcast&entity=podcast&limit=25&term=" + url.QueryEscape(query)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create iTunes request: %w", err)
	}
	req.Header.Set("User-Agent", DefaultUserAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search iTunes: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("search iTunes: HTTP %s", resp.Status)
	}
	var result struct {
		Results []struct {
			FeedURL     string `json:"feedUrl"`
			Collection  string `json:"collectionName"`
			Description string `json:"collectionDescription"`
			Artwork     string `json:"artworkUrl600"`
		} `json:"results"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode iTunes response: %w", err)
	}
	results := make([]SearchResult, 0, len(result.Results))
	for _, item := range result.Results {
		if item.FeedURL == "" {
			continue
		}
		results = append(results, SearchResult{FeedURL: item.FeedURL, Title: item.Collection, Description: item.Description, ArtworkURL: item.Artwork})
	}
	return results, nil
}
