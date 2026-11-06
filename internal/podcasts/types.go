package podcasts

import "time"

const DefaultUserAgent = "Podcasts3000/1.0 (+https://github.com/user/podcasts3000)"

type Podcast struct {
	ID                int64      `json:"id"`
	FeedURL           string     `json:"feedUrl,omitempty"`
	Title             string     `json:"title,omitempty"`
	Description       string     `json:"description,omitempty"`
	SiteURL           string     `json:"siteUrl,omitempty"`
	ArtworkURL        string     `json:"artworkUrl,omitempty"`
	ArtworkLocalPath  string     `json:"artworkLocalPath,omitempty"`
	CustomUserAgent   string     `json:"customUserAgent,omitempty"`
	LastRefreshedAt   *time.Time `json:"lastRefreshedAt,omitempty"`
	HasRecentUnplayed bool       `json:"hasRecentUnplayed"`
	HasCachedEpisodes bool       `json:"hasCachedEpisodes"`
}

type Episode struct {
	ID              int64      `json:"id"`
	PodcastID       int64      `json:"podcastId"`
	GUID            string     `json:"guid,omitempty"`
	Title           string     `json:"title,omitempty"`
	Description     string     `json:"description,omitempty"`
	AudioURL        string     `json:"audioUrl,omitempty"`
	LocalCachePath  string     `json:"localCachePath,omitempty"`
	PubDate         *time.Time `json:"pubDate,omitempty"`
	IsDownloaded    bool       `json:"isDownloaded"`
	DurationSeconds int        `json:"durationSeconds"`
	PositionSeconds int        `json:"positionSeconds"`
	IsCompleted     bool       `json:"isCompleted"`
	IsFavorite      bool       `json:"isFavorite"`
	Notes           string     `json:"notes,omitempty"`
}

type SearchResult struct {
	FeedURL     string `json:"feedUrl,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	ArtworkURL  string `json:"artworkUrl,omitempty"`
}
