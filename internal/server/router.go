package server

import (
	"net/http"
	"path/filepath"
)

func NewMux(api *API) *http.ServeMux {
	mux := http.NewServeMux()

	// API
	mux.HandleFunc("/api/tracks", api.ListTracks)        // GET
	mux.HandleFunc("/api/tracks/", api.TrackGeoJSONByID) // GET /api/tracks/:id.geojson

	// Static (serve the Vite build output if present)
	dist := filepath.Join("web", "dist")
	mux.Handle("/", http.FileServer(http.Dir(dist)))

	return mux
}
