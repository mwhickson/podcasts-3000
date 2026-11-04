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
}
