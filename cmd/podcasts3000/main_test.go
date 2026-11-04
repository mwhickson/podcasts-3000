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
	if !strings.Contains(string(index), `id="active-podcast"`) || !strings.Contains(string(index), `class="player"`) {
		t.Fatal("frontend is missing artwork header or fixed player regions")
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
}
