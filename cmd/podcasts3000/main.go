package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"podcasts3000/internal/podcasts"
)

func main() {
	dataDir := flag.String("data-dir", defaultDataDir(), "directory for database and downloaded media")
	addr := flag.String("addr", "127.0.0.1:3000", "HTTP listen address")
	flag.Parse()
	store, err := podcasts.OpenStore(filepath.Join(*dataDir, "podcasts.db"), filepath.Join(*dataDir, "media"))
	if err != nil {
		fatal(err)
	}
	defer store.Close()
	app := &podcasts.App{Store: store, MediaBaseURL: "http://" + *addr}
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("web")))
	mux.Handle("/media/", http.StripPrefix("/media/", app.MediaHandler()))
	mux.HandleFunc("/api/podcasts", func(w http.ResponseWriter, r *http.Request) {
		podcasts, err := store.ListPodcasts(r.Context())
		writeJSON(w, podcasts, err)
	})
	mux.HandleFunc("/api/subscribe", func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			URL string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, err)
			return
		}
		podcast, err := app.Subscribe(request.URL)
		writeJSON(w, podcast, err)
	})
	mux.HandleFunc("/api/refresh", func(w http.ResponseWriter, r *http.Request) { writeErrorOnly(w, app.RefreshAllFeeds()) })
	mux.HandleFunc("/api/podcasts/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 4 || parts[0] != "api" || parts[1] != "podcasts" || parts[3] != "episodes" {
			http.NotFound(w, r)
			return
		}
		id, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			writeError(w, err)
			return
		}
		rows, err := store.DB().QueryContext(r.Context(), `SELECT e.id,e.podcast_id,e.guid,e.title,COALESCE(e.description,''),e.audio_url,COALESCE(e.local_cache_path,''),e.pub_date,e.is_downloaded,e.duration_seconds,COALESCE(ps.position_seconds,0),COALESCE(ps.is_completed,0),CASE WHEN f.episode_id IS NULL THEN 0 ELSE 1 END,COALESCE(f.notes,'') FROM episodes e LEFT JOIN playback_state ps ON ps.episode_id=e.id LEFT JOIN favorites f ON f.episode_id=e.id WHERE e.podcast_id=? ORDER BY e.pub_date DESC`, id)
		if err != nil {
			writeError(w, err)
			return
		}
		defer rows.Close()
		var episodes []podcasts.Episode
		for rows.Next() {
			var e podcasts.Episode
			var downloaded, completed, favorite int
			if err = rows.Scan(&e.ID, &e.PodcastID, &e.GUID, &e.Title, &e.Description, &e.AudioURL, &e.LocalCachePath, &e.PubDate, &downloaded, &e.DurationSeconds, &e.PositionSeconds, &completed, &favorite, &e.Notes); err != nil {
				writeError(w, err)
				return
			}
			e.IsDownloaded = downloaded != 0
			e.IsCompleted = completed != 0
			e.IsFavorite = favorite != 0
			episodes = append(episodes, e)
		}
		writeJSON(w, episodes, nil)
	})
	mux.HandleFunc("/api/episodes/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 4 || parts[0] != "api" || parts[1] != "episodes" || parts[3] != "source" {
			http.NotFound(w, r)
			return
		}
		id, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			writeError(w, err)
			return
		}
		source, err := app.GetAudioSource(id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, struct {
			Source string `json:"source"`
		}{source}, nil)
	})
	fmt.Printf("Podcasts 3000 listening at http://%s\n", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		fatal(err)
	}
}

func defaultDataDir() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "podcasts3000")
	}
	return ".podcasts3000"
}
func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
func writeJSON(w http.ResponseWriter, value any, err error) {
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(value)
}
func writeErrorOnly(w http.ResponseWriter, err error) {
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	if errors.Is(err, os.ErrInvalid) {
		status = http.StatusBadRequest
	}
	http.Error(w, err.Error(), status)
}
