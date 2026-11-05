package podcasts

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	Store        *Store
	MediaBaseURL string
	ctx          context.Context
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	go func() {
		if err := a.RefreshAllFeeds(); err != nil {
			fmt.Printf("podcasts3000: startup refresh: %v\n", err)
		}
	}()
}

func (a *App) ListPodcasts() ([]Podcast, error) {
	return a.Store.ListPodcasts(context.Background())
}

func (a *App) ListEpisodes(podcastID int64) ([]Episode, error) {
	return a.Store.ListEpisodes(context.Background(), podcastID)
}

func (a *App) ImportOPML(path string) error { return a.Store.ImportOPML(context.Background(), path) }
func (a *App) ImportOPMLResults(path string) ([]OPMLImportResult, error) {
	return a.Store.ImportOPMLResults(context.Background(), path)
}
func (a *App) ExportOPML(path string) error { return a.Store.ExportOPML(context.Background(), path) }
func (a *App) ImportOPMLDialog() ([]OPMLImportResult, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Import podcast subscriptions",
		Filters: []runtime.FileFilter{{DisplayName: "OPML files", Pattern: "*.opml;*.xml"}},
	})
	if err != nil || path == "" {
		return nil, err
	}
	return a.ImportOPMLResults(path)
}
func (a *App) ExportOPMLDialog() error {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export podcast subscriptions",
		DefaultFilename: "podcasts-3000.opml",
		Filters:         []runtime.FileFilter{{DisplayName: "OPML files", Pattern: "*.opml"}},
	})
	if err != nil || path == "" {
		return err
	}
	return a.ExportOPML(path)
}
func (a *App) Confirm(message string) (bool, error) {
	response, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:          runtime.QuestionDialog,
		Title:         "Podcasts 3000",
		Message:       message,
		Buttons:       []string{"Yes", "No"},
		DefaultButton: "Yes",
		CancelButton:  "No",
	})
	return isAffirmativeDialogResponse(response), err
}

func isAffirmativeDialogResponse(response string) bool {
	switch strings.ToLower(strings.TrimSpace(response)) {
	case "yes", "ok":
		return true
	default:
		return false
	}
}
func (a *App) ShowError(message string) error {
	_, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.ErrorDialog,
		Title:   "Podcasts 3000",
		Message: message,
		Buttons: []string{"OK"},
	})
	return err
}
func (a *App) ShowInfo(message string) error {
	_, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.InfoDialog,
		Title:   "Podcasts 3000",
		Message: message,
		Buttons: []string{"OK"},
	})
	return err
}
func (a *App) Subscribe(url string) (Podcast, error) {
	return a.Store.Subscribe(context.Background(), url)
}
func (a *App) Unsubscribe(id int64) error {
	return a.Store.Unsubscribe(context.Background(), id)
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
func (a *App) UpdateDuration(id int64, seconds int) error {
	return a.Store.UpdateDuration(context.Background(), id, seconds)
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
