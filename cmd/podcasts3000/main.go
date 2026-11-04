package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"podcasts3000/internal/podcasts"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed frontend/*
var frontend embed.FS

func main() {
	dataDir := defaultDataDir()
	if value := os.Getenv("PODCASTS3000_DATA_DIR"); value != "" {
		dataDir = value
	}
	store, err := podcasts.OpenStore(filepath.Join(dataDir, "podcasts.db"), filepath.Join(dataDir, "media"))
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	mediaURL, mediaServer, err := startMediaServer(store)
	if err != nil {
		log.Fatal(err)
	}
	defer mediaServer.Close()

	app := &podcasts.App{Store: store, MediaBaseURL: mediaURL}
	assets, err := fs.Sub(frontend, "frontend")
	if err != nil {
		log.Fatal(err)
	}

	err = wails.Run(&options.App{
		Title:            "Podcasts 3000",
		Width:            1200,
		Height:           800,
		MinWidth:         900,
		MinHeight:        600,
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 17, G: 17, B: 17, A: 1},
		OnStartup:        app.Startup,
		OnShutdown:       func(_ context.Context) { mediaServer.Close() },
		Bind:             []interface{}{app},
		Windows:          &windows.Options{DisableWindowIcon: true},
		Mac:              &mac.Options{},
		Linux:            &linux.Options{WebviewGpuPolicy: linux.WebviewGpuPolicyOnDemand},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func startMediaServer(store *podcasts.Store) (string, *http.Server, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, fmt.Errorf("listen for media server: %w", err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if len(r.URL.Path) < len("/media/") || r.URL.Path[:len("/media/")] != "/media/" {
			http.NotFound(w, r)
			return
		}
		store.ServeMedia(w, r)
	})}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("podcasts3000 media server: %v", err)
		}
	}()
	return "http://" + listener.Addr().String(), server, nil
}

func defaultDataDir() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "podcasts3000")
	}
	return ".podcasts3000"
}
