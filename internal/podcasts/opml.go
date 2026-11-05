package podcasts

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
)

type opmlDocument struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Body    struct {
		Outlines []opmlOutline `xml:"outline"`
	} `xml:"body"`
}
type opmlOutline struct {
	Text     string        `xml:"text,attr"`
	Title    string        `xml:"title,attr"`
	Type     string        `xml:"type,attr"`
	XMLURL   string        `xml:"xmlUrl,attr"`
	HTMLURL  string        `xml:"htmlUrl,attr"`
	Children []opmlOutline `xml:"outline"`
}
type OPMLImportResult struct {
	FeedURL string `json:"feedUrl"`
	Error   string `json:"error,omitempty"`
}

func flattenOutlines(outlines []opmlOutline) []opmlOutline {
	var result []opmlOutline
	for _, outline := range outlines {
		if outline.XMLURL != "" {
			result = append(result, outline)
		}
		result = append(result, flattenOutlines(outline.Children)...)
	}
	return result
}

func (s *Store) ImportOPML(ctx context.Context, path string) error {
	_, err := s.ImportOPMLResults(ctx, path)
	return err
}

func (s *Store) ImportOPMLResults(ctx context.Context, path string) ([]OPMLImportResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read OPML: %w", err)
	}
	var doc opmlDocument
	if err = xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse OPML: %w", err)
	}
	results := make([]OPMLImportResult, 0)
	for _, outline := range flattenOutlines(doc.Body.Outlines) {
		if _, err = s.Subscribe(ctx, outline.XMLURL); err != nil {
			results = append(results, OPMLImportResult{FeedURL: outline.XMLURL, Error: err.Error()})
			continue
		}
		results = append(results, OPMLImportResult{FeedURL: outline.XMLURL})
	}
	return results, nil
}

func (s *Store) ExportOPML(ctx context.Context, path string) error {
	podcasts, err := s.ListPodcasts(ctx)
	if err != nil {
		return err
	}
	doc := opmlDocument{Version: "2.0"}
	for _, p := range podcasts {
		doc.Body.Outlines = append(doc.Body.Outlines, opmlOutline{Text: p.Title, Title: p.Title, Type: "rss", XMLURL: p.FeedURL, HTMLURL: p.SiteURL})
	}
	data, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append([]byte(xml.Header), data...), 0o644)
}
