package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// SeamarkLookup queries Overpass for seamark-tagged objects near a coordinate.
func (a *API) SeamarkLookup(w http.ResponseWriter, r *http.Request) {
	latStr := r.URL.Query().Get("lat")
	lonStr := r.URL.Query().Get("lon")
	if latStr == "" || lonStr == "" {
		writeErr(w, http.StatusBadRequest, "missing_params", "lat and lon are required", nil)
		return
	}

	lat, errLat := strconv.ParseFloat(latStr, 64)
	lon, errLon := strconv.ParseFloat(lonStr, 64)
	if errLat != nil || errLon != nil {
		writeErr(w, http.StatusBadRequest, "bad_params", "lat and lon must be numbers", map[string]any{"lat": latStr, "lon": lonStr})
		return
	}

	radius := 400.0
	if r := r.URL.Query().Get("radius"); r != "" {
		if v, err := strconv.ParseFloat(r, 64); err == nil && v > 0 && v <= 1500 {
			radius = v
		}
	}

	const overpassURL = "https://overpass-api.de/api/interpreter"
	query := fmt.Sprintf(`
[out:json][timeout:15];
(
  node["seamark:type"](around:%f,%f,%f);
  way["seamark:type"](around:%f,%f,%f);
  relation["seamark:type"](around:%f,%f,%f);
);
out center tags;
`, radius, lat, lon, radius, lat, lon, radius, lat, lon)

	form := url.Values{}
	form.Set("data", query)

	req, err := http.NewRequest(http.MethodPost, overpassURL, bytes.NewBufferString(form.Encode()))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "request_error", "failed to build upstream request", nil)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "wakemap-dev/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "upstream_error", "overpass request failed", map[string]any{"err": err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		writeErr(w, http.StatusBadGateway, "upstream_status", "unexpected overpass status", map[string]any{"status": resp.StatusCode, "body": string(body)})
		return
	}

	var over struct {
		Elements []struct {
			Type   string                      `json:"type"`
			ID     int64                       `json:"id"`
			Lat    float64                     `json:"lat"`
			Lon    float64                     `json:"lon"`
			Center *struct{ Lat, Lon float64 } `json:"center"`
			Tags   map[string]string           `json:"tags"`
		} `json:"elements"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&over); err != nil {
		writeErr(w, http.StatusBadGateway, "decode_error", "failed to decode overpass response", map[string]any{"err": err.Error()})
		return
	}

	features := make([]map[string]any, 0, len(over.Elements))
	for _, el := range over.Elements {
		var lonLat []float64
		if el.Center != nil {
			lonLat = []float64{el.Center.Lon, el.Center.Lat}
		} else {
			lonLat = []float64{el.Lon, el.Lat}
		}
		if len(lonLat) != 2 || (lonLat[0] == 0 && lonLat[1] == 0) {
			continue
		}
		props := map[string]any{
			"id":   el.ID,
			"type": el.Type,
		}
		for k, v := range el.Tags {
			props[k] = v
		}
		features = append(features, map[string]any{
			"type": "Feature",
			"geometry": map[string]any{
				"type":        "Point",
				"coordinates": lonLat,
			},
			"properties": props,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"type":     "FeatureCollection",
		"features": features,
	})
}
