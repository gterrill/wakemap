package server

import (
	"encoding/json"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

type Track struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	DistanceM float64   `json:"distance_m"`
}

// TODO: replace with SQLite query (ORDER BY started_at DESC LIMIT ?)
func lastTracks(n int) []Track {
	if n <= 0 || n > 200 {
		n = 50
	}

	// Stub: two demo tracks
	now := time.Now()
	return []Track{
		{ID: "demo-001", Name: "Harbour shake-down", StartedAt: now.Add(-48 * time.Hour), EndedAt: now.Add(-47*time.Hour + 30*time.Minute), DistanceM: 8200},
		{ID: "demo-002", Name: "Evening sail", StartedAt: now.Add(-24 * time.Hour), EndedAt: now.Add(-23*time.Hour + 50*time.Minute), DistanceM: 10500},
	}
}

func ListTracks(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if q := r.URL.Query().Get("limit"); q != "" {
		if v, err := strconv.Atoi(q); err == nil && v > 0 && v <= 200 {
			limit = v
		}
	}
	tracks := lastTracks(limit)
	writeJSON(w, http.StatusOK, map[string]any{"tracks": tracks})
}

func TrackGeoJSONByID(w http.ResponseWriter, r *http.Request) {
	// Expect /api/tracks/:id.geojson
	_, tail := path.Split(r.URL.Path) // e.g., "demo-001.geojson"
	if !strings.HasSuffix(tail, ".geojson") || len(tail) <= len(".geojson") {
		writeErr(w, http.StatusNotFound, "not_found", "invalid track path", nil)
		return
	}
	id := strings.TrimSuffix(tail, ".geojson")

	// TODO: fetch actual geometry/bbox from DB
	// Stub: a small line near Sydney with bbox
	type fc struct {
		Type     string           `json:"type"`
		BBox     [4]float64       `json:"bbox"`
		Features []map[string]any `json:"features"`
	}
	var res fc
	if id == "demo-001" {
		res = fc{
			Type: "FeatureCollection",
			BBox: [4]float64{151.18, -33.89, 151.23, -33.83},
			Features: []map[string]any{{
				"type":       "Feature",
				"properties": map[string]any{"id": id, "name": "Harbour shake-down"},
				"geometry": map[string]any{
					"type": "LineString",
					"coordinates": [][]float64{
						{151.18, -33.86}, {151.20, -33.85}, {151.23, -33.83},
					},
				},
			}},
		}
	} else {
		res = fc{
			Type: "FeatureCollection",
			BBox: [4]float64{151.16, -33.92, 151.21, -33.84},
			Features: []map[string]any{{
				"type":       "Feature",
				"properties": map[string]any{"id": id, "name": "Evening sail"},
				"geometry": map[string]any{
					"type": "LineString",
					"coordinates": [][]float64{
						{151.16, -33.92}, {151.19, -33.89}, {151.21, -33.84},
					},
				},
			}},
		}
	}

	w.Header().Set("Content-Type", "application/geo+json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(res)
}
