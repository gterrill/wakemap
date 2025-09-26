package main

import (
	"net/http"
	"strconv"

	"github.com/gterrill/wakemap/core/internal/sim"
)

func (a *App) simulateBroughtonToNewcastleHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	speedKn := 6.0
	if v := q.Get("speed_kn"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			speedKn = f
		}
	}
	intervalS := 10
	if v := q.Get("interval_s"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			intervalS = n
		}
	}

	id, points, endedAt, err := sim.RunBroughtonToNewcastle(r.Context(), a.q, speedKn, intervalS)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"track_id": id,
		"points":   points,
		"ended_at": endedAt,
		"speed_kn": speedKn,
		"interval": intervalS,
	})
}
