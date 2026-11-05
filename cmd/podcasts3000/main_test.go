package main

import (
	"strings"
	"testing"
)

func TestEmbeddedFrontendUsesBoundPackageNamespace(t *testing.T) {
	index, err := frontend.ReadFile("frontend/index.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), `src="/app.js"`) {
		t.Fatal("embedded index does not load app.js")
	}
	app, err := frontend.ReadFile("frontend/wailsjs/go/podcasts/App.js")
	if err != nil {
		t.Fatal(err)
	}
	source := string(app)
	if !strings.Contains(source, "globalThis.go?.podcasts?.App") {
		t.Fatal("frontend binding does not use the podcasts.App Wails namespace")
	}
	if strings.Contains(source, "globalThis.go?.main?.App") {
		t.Fatal("frontend binding still references the incorrect main.App namespace")
	}
	if !strings.Contains(string(index), `id="close-search"`) || !strings.Contains(string(index), `id="search-view"`) {
		t.Fatal("frontend is missing isolated search view controls")
	}
	if strings.Contains(string(index), `id="opml-path"`) {
		t.Fatal("frontend still exposes a raw OPML path input")
	}
	if !strings.Contains(string(index), `id="active-podcast"`) || !strings.Contains(string(index), `class="player"`) || !strings.Contains(string(index), `id="status-bar"`) {
		t.Fatal("frontend is missing artwork header, player, or status regions")
	}
	if !strings.Contains(string(index), `id="episode-toolbar"`) || !strings.Contains(string(index), `id="unplayed-only"`) || !strings.Contains(string(index), `id="episode-sort"`) {
		t.Fatal("frontend is missing episode filtering and sorting toolbar controls")
	}
	if !strings.Contains(string(index), `id="pane-resizer"`) {
		t.Fatal("frontend is missing the resizable pane divider")
	}
	if !strings.Contains(string(index), `id="import-opml"`) || !strings.Contains(string(index), `id="export-opml"`) {
		t.Fatal("frontend is missing OPML controls")
	}
	app, err = frontend.ReadFile("frontend/app.js")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(app), `data-unsubscribe`) {
		t.Fatal("frontend is missing the podcast unsubscribe action")
	}
	if !strings.Contains(string(app), `(await App.ListPodcasts()) || []`) {
		t.Fatal("frontend does not normalize an empty podcast response")
	}
	if strings.Contains(string(app), "alert(") || strings.Contains(string(app), "confirm(") {
		t.Fatal("frontend still uses browser messageboxes")
	}
	for _, method := range []string{"Confirm"} {
		if !strings.Contains(string(app), "App."+method) {
			t.Fatalf("frontend does not use native App.%s dialog binding", method)
		}
	}
	if !strings.Contains(string(app), "App.GetLastPlayed") || !strings.Contains(string(app), "lastPlayed.positionSeconds") || !strings.Contains(string(app), "episode.positionSeconds") {
		t.Fatal("frontend does not restore saved episode playback positions")
	}
	if !strings.Contains(string(app), "App.GetAudioSource") || !strings.Contains(string(app), "loadedmetadata") {
		t.Fatal("frontend does not load episode audio and restore its position")
	}
	if !strings.Contains(string(app), "titleMatch") || !strings.Contains(string(app), "episode.isCompleted") || !strings.Contains(string(app), "pub-desc") {
		t.Fatal("frontend does not implement episode search, unplayed filtering, and publication sorting")
	}
	if !strings.Contains(string(app), "setStatus") || !strings.Contains(string(app), "Buffering episode") || !strings.Contains(string(app), "Loading episodes") {
		t.Fatal("frontend does not report action progress through the status bar")
	}
	if !strings.Contains(string(app), "data-played") || !strings.Contains(string(app), "formatDate") || !strings.Contains(string(app), "formatDuration") {
		t.Fatal("frontend does not display episode metadata or provide played-state controls")
	}
	if !strings.Contains(string(app), "App.UpdateDuration") || !strings.Contains(string(app), "audio.duration") {
		t.Fatal("frontend does not persist confirmed playback duration")
	}
	if !strings.Contains(string(app), "window.addEventListener(\"resize\"") || !strings.Contains(string(app), "setSidebarWidth") {
		t.Fatal("frontend does not update layout on resize or support pane resizing")
	}
	if !strings.Contains(string(app), "play(lastPlayed.id, Math.max(0, lastPlayed.positionSeconds || 0), false)") {
		t.Fatal("frontend restores startup playback without auto-starting")
	}
	if !strings.Contains(string(app), `if (!(await confirmAction("Unsubscribe and remove all downloaded episodes for this podcast?"))) return;`) {
		t.Fatal("unsubscribe action does not gate deletion on native confirmation")
	}
}
