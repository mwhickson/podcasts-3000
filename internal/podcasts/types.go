package podcasts

import "time"

const DefaultUserAgent = "Podcasts3000/1.0 (+https://github.com/user/podcasts3000)"

type Podcast struct {
	ID               int64      `json:"id"`
	FeedURL          string     `json:"feedUrl,omitempty"`
	Title            string     `json:"title,omitempty"`
	Description      string     `json:"description,omitempty"`
	SiteURL          string     `json:"siteUrl,omitempty"`
	ArtworkURL       string     `json:"artworkUrl,omitempty"`
	ArtworkLocalPath string     `json:"artworkLocalPath,omitempty"`
	CustomUserAgent  string     `json:"customUserAgent,omitempty"`
	LastRefreshedAt  *time.Time `json:"lastRefreshedAt,omitempty"`
}

type Episode struct {
	ID, PodcastID                                      int64
	GUID, Title, Description, AudioURL, LocalCachePath string
	PubDate                                            *time.Time
	IsDownloaded                                       bool
	DurationSeconds                                    int
	PositionSeconds                                    int
	IsCompleted                                        bool
	IsFavorite                                         bool
	Notes                                              string
}

type SearchResult struct {
	FeedURL     string `json:"feedUrl,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	ArtworkURL  string `json:"artworkUrl,omitempty"`
}
