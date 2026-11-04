package podcasts

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type feedDocument struct {
	Channel feedChannel `xml:"channel"`
	Entry   []feedEntry `xml:"entry"`
}
type feedChannel struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Summary     string `xml:"summary"`
	Links       []struct {
		Rel  string `xml:"rel,attr"`
		Href string `xml:"href,attr"`
		Type string `xml:"type,attr"`
	} `xml:"link"`
	Image struct {
		Href string `xml:"href,attr"`
	} `xml:"image"`
	Artwork string     `xml:"-"`
	Items   []feedItem `xml:"item"`
}
type feedItem struct {
	GUID        string `xml:"guid"`
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Summary     string `xml:"summary"`
	PubDate     string `xml:"pubDate"`
	Published   string `xml:"published"`
	Enclosure   struct {
		URL string `xml:"url,attr"`
	} `xml:"enclosure"`
	Links []struct {
		Rel  string `xml:"rel,attr"`
		Href string `xml:"href,attr"`
		Type string `xml:"type,attr"`
	} `xml:"link"`
	Duration string `xml:"duration"`
}
type feedEntry struct {
	ID        string `xml:"id"`
	Title     string `xml:"title"`
	Summary   string `xml:"summary"`
	Published string `xml:"published"`
	Links     []struct {
		Rel  string `xml:"rel,attr"`
		Href string `xml:"href,attr"`
		Type string `xml:"type,attr"`
	} `xml:"link"`
}

func (s *Store) fetch(ctx context.Context, feedURL, customUA string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create feed request: %w", err)
	}
	ua := DefaultUserAgent
	if strings.TrimSpace(customUA) != "" {
		ua = customUA
	}
	req.Header.Set("User-Agent", ua)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch feed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch feed: HTTP %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 16<<20))
}

func parseDate(value string) *time.Time {
	for _, layout := range []string{time.RFC1123Z, time.RFC1123, time.RFC3339, time.RFC822, "Mon, 2 Jan 2006 15:04:05 -0700"} {
		if t, err := time.Parse(layout, strings.TrimSpace(value)); err == nil {
			return &t
		}
	}
	return nil
}

func parseDuration(value string) int {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) == 0 {
		return 0
	}
	total := 0
	for _, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return 0
		}
		total = total*60 + n
	}
	return total
}

func parseFeed(data []byte) (Podcast, []Episode, error) {
	var doc feedDocument
	if err := xml.Unmarshal(data, &doc); err != nil {
		return Podcast{}, nil, fmt.Errorf("parse feed XML: %w", err)
	}
	ch := doc.Channel
	description := ch.Description
	if description == "" {
		description = ch.Summary
	}
	artwork := ch.Image.Href
	for _, link := range ch.Links {
		if artwork == "" && link.Rel == "image" {
			artwork = link.Href
		}
	}
	siteURL := ""
	for _, link := range ch.Links {
		if link.Rel == "" || link.Rel == "alternate" {
			siteURL = link.Href
			break
		}
	}
	p := Podcast{Title: strings.TrimSpace(ch.Title), Description: strings.TrimSpace(description), SiteURL: siteURL, ArtworkURL: artwork}
	var episodes []Episode
	for _, item := range ch.Items {
		audio := item.Enclosure.URL
		if audio == "" {
			for _, link := range item.Links {
				if link.Rel == "enclosure" || strings.HasPrefix(link.Type, "audio/") {
					audio = link.Href
					break
				}
			}
		}
		if audio == "" || strings.TrimSpace(item.Title) == "" {
			continue
		}
		guid := strings.TrimSpace(item.GUID)
		if guid == "" {
			guid = audio
		}
		desc := item.Description
		if desc == "" {
			desc = item.Summary
		}
		date := item.PubDate
		if date == "" {
			date = item.Published
		}
		episodes = append(episodes, Episode{GUID: guid, Title: strings.TrimSpace(item.Title), Description: strings.TrimSpace(desc), AudioURL: audio, PubDate: parseDate(date), DurationSeconds: parseDuration(item.Duration)})
	}
	if len(episodes) == 0 {
		for _, entry := range doc.Entry {
			audio := ""
			for _, link := range entry.Links {
				if link.Rel == "enclosure" || strings.HasPrefix(link.Type, "audio/") {
					audio = link.Href
					break
				}
			}
			if audio != "" {
				episodes = append(episodes, Episode{GUID: entry.ID, Title: entry.Title, Description: entry.Summary, AudioURL: audio, PubDate: parseDate(entry.Published)})
			}
		}
	}
	return p, episodes, nil
}

func (s *Store) Subscribe(ctx context.Context, feedURL string) (Podcast, error) {
	if _, err := url.ParseRequestURI(feedURL); err != nil {
		return Podcast{}, fmt.Errorf("invalid feed URL: %w", err)
	}
	data, err := s.fetch(ctx, feedURL, "")
	if err != nil {
		return Podcast{}, err
	}
	p, episodes, err := parseFeed(data)
	if err != nil {
		return Podcast{}, err
	}
	p.FeedURL = feedURL
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Podcast{}, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO podcasts(feed_url,title,description,site_url,artwork_url,last_refreshed_at) VALUES(?,?,?,?,?,CURRENT_TIMESTAMP) ON CONFLICT(feed_url) DO UPDATE SET title=excluded.title,description=excluded.description,site_url=excluded.site_url,artwork_url=excluded.artwork_url,last_refreshed_at=CURRENT_TIMESTAMP`, p.FeedURL, p.Title, p.Description, p.SiteURL, p.ArtworkURL)
	if err != nil {
		return Podcast{}, fmt.Errorf("save podcast: %w", err)
	}
	var customUserAgent sql.NullString
	if err = tx.QueryRowContext(ctx, `SELECT id, custom_user_agent FROM podcasts WHERE feed_url=?`, feedURL).Scan(&p.ID, &customUserAgent); err != nil {
		return Podcast{}, err
	}
	p.CustomUserAgent = customUserAgent.String
	for _, ep := range episodes {
		_, err = tx.ExecContext(ctx, `INSERT INTO episodes(podcast_id,guid,title,description,pub_date,audio_url,duration_seconds) VALUES(?,?,?,?,?,?,?) ON CONFLICT(podcast_id,guid) DO UPDATE SET title=excluded.title,description=excluded.description,pub_date=excluded.pub_date,audio_url=excluded.audio_url,duration_seconds=excluded.duration_seconds`, p.ID, ep.GUID, ep.Title, ep.Description, ep.PubDate, ep.AudioURL, ep.DurationSeconds)
		if err != nil {
			return Podcast{}, fmt.Errorf("save episode: %w", err)
		}
	}
	if err = tx.Commit(); err != nil {
		return Podcast{}, err
	}
	return p, nil
}

func (s *Store) RefreshFeeds(ctx context.Context, podcastID int64) error {
	var p Podcast
	var customUserAgent sql.NullString
	if err := s.db.QueryRowContext(ctx, `SELECT id,feed_url,title,description,site_url,artwork_url,custom_user_agent FROM podcasts WHERE id=?`, podcastID).Scan(&p.ID, &p.FeedURL, &p.Title, &p.Description, &p.SiteURL, &p.ArtworkURL, &customUserAgent); err != nil {
		return err
	}
	p.CustomUserAgent = customUserAgent.String
	data, err := s.fetch(ctx, p.FeedURL, p.CustomUserAgent)
	if err != nil {
		return err
	}
	parsed, episodes, err := parseFeed(data)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `UPDATE podcasts SET title=?,description=?,site_url=?,artwork_url=?,last_refreshed_at=CURRENT_TIMESTAMP WHERE id=?`, parsed.Title, parsed.Description, parsed.SiteURL, parsed.ArtworkURL, podcastID)
	if err != nil {
		return err
	}
	for _, ep := range episodes {
		if _, err = tx.ExecContext(ctx, `INSERT INTO episodes(podcast_id,guid,title,description,pub_date,audio_url,duration_seconds) VALUES(?,?,?,?,?,?,?) ON CONFLICT(podcast_id,guid) DO UPDATE SET title=excluded.title,description=excluded.description,pub_date=excluded.pub_date,audio_url=excluded.audio_url,duration_seconds=excluded.duration_seconds`, podcastID, ep.GUID, ep.Title, ep.Description, ep.PubDate, ep.AudioURL, ep.DurationSeconds); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) SetPodcastUserAgent(ctx context.Context, id int64, ua string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE podcasts SET custom_user_agent=? WHERE id=?`, strings.TrimSpace(ua), id)
	return err
}
func (s *Store) ListPodcasts(ctx context.Context) ([]Podcast, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,feed_url,title,COALESCE(description,''),COALESCE(site_url,''),COALESCE(artwork_url,''),COALESCE(artwork_local_path,''),COALESCE(custom_user_agent,''),last_refreshed_at FROM podcasts ORDER BY title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]Podcast, 0)
	for rows.Next() {
		var p Podcast
		if err := rows.Scan(&p.ID, &p.FeedURL, &p.Title, &p.Description, &p.SiteURL, &p.ArtworkURL, &p.ArtworkLocalPath, &p.CustomUserAgent, &p.LastRefreshedAt); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}
