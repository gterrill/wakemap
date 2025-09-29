package server

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"

	"wakemap/internal/data"
)

type API struct {
	Store *data.Store
}

// RFC3339 layout literal (avoids importing time just for the const)
const timeRFC3339 = "2006-01-02T15:04:05Z07:00"

// at top of handlers_tracks.go (below imports)
func asFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int64:
		return float64(t)
	case []byte: // SQLite may return numeric as text blobs
		if f, err := strconv.ParseFloat(string(t), 64); err == nil {
			return f
		}
	}
	return 0
}

func (api *API) ListTracks(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if q := r.URL.Query().Get("limit"); q != "" {
		if v, err := strconv.Atoi(q); err == nil && v > 0 && v <= 200 {
			limit = v
		}
	}

	items, err := api.Store.ListTracks(r.Context(), limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "failed to list tracks", map[string]any{"err": err.Error()})
		return
	}

	type outTrack struct {
		ID        int64   `json:"id"`
		Name      string  `json:"name"`
		StartedAt string  `json:"started_at"`
		EndedAt   string  `json:"ended_at,omitempty"`
		DistanceM float64 `json:"distance_m,omitempty"`
	}

	resp := struct {
		Tracks []outTrack `json:"tracks"`
	}{Tracks: make([]outTrack, 0, len(items))}

	for _, t := range items {
		ot := outTrack{
			ID:        t.ID,
			Name:      t.Name,
			StartedAt: data.UnixToTime(t.StartedAt).Format(timeRFC3339),
		}
		if t.EndedAt.Valid {
			ot.EndedAt = data.UnixToTime(t.EndedAt.Int64).Format(timeRFC3339)
		}
		if t.DistanceM.Valid {
			ot.DistanceM = t.DistanceM.Float64
		}
		resp.Tracks = append(resp.Tracks, ot)
	}

	writeJSON(w, http.StatusOK, resp)
}

func (a *API) TrackGeoJSONByID(w http.ResponseWriter, r *http.Request) {
	// /api/tracks/:id.geojson
	p := strings.TrimPrefix(r.URL.Path, "/api/tracks/")
	p = strings.TrimSuffix(p, ".geojson")

	id, err := strconv.ParseInt(p, 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad_id", "invalid track id", map[string]any{"id": p})
		return
	}

	ctx := r.Context()
	ts, err := a.Store.ComputeTrackStats(ctx, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "failed to load track", map[string]any{"err": err.Error()})
		return
	}

	props := map[string]any{
		"id":          id,
		"name":        ts.Name,
		"started_at":  ts.StartedAt,
		"ended_at":    ts.EndedAt,
		"distance_m":  ts.DistanceM,
		"distance_nm": ts.DistanceM / 1852.0,
		"duration_s":  max64(0, ts.EndedAt-ts.StartedAt),
	}
	if props["duration_s"].(int64) > 0 {
		props["avg_knots"] = (ts.DistanceM / float64(props["duration_s"].(int64))) * 1.943844492
	} else {
		props["avg_knots"] = 0.0
	}

	coords := make([][]float64, 0, len(ts.Coords))
	for i, xy := range ts.Coords {
		lon, lat := xy[0], xy[1]
		sog := ts.SOGms[i]
		if math.IsNaN(sog) {
			// emit 2D when SOG unknown to keep JSON small and avoid nulls
			coords = append(coords, []float64{lon, lat})
		} else {
			coords = append(coords, []float64{lon, lat, sog}) // sog in m/s
		}
	}

	gj := map[string]any{
		"type":       "Feature",
		"properties": props, // your existing props incl. distance_nm, duration_s, avg_knots
		"geometry": map[string]any{
			"type":        "LineString",
			"coordinates": coords,
		},
		"bbox": []float64{ts.MinX, ts.MinY, ts.MaxX, ts.MaxY},
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(gj)
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
