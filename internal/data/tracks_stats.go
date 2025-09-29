package data

import (
	"context"
	"database/sql"
	"math"
)

type TrackStats struct {
	Name       string
	StartedAt  int64
	EndedAt    int64
	DistanceM  float64
	MinX, MinY float64
	MaxX, MaxY float64
	Coords     [][2]float64
	SOGms      []float64 // one per coordinate; NaN when unknown
}

func haversineMeters(aLon, aLat, bLon, bLat float64) float64 {
	const R = 6371000.0
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(bLat - aLat)
	dLon := toRad(bLon - aLon)
	la1 := toRad(aLat)
	la2 := toRad(bLat)
	sin1 := math.Sin(dLat / 2)
	sin2 := math.Sin(dLon / 2)
	h := sin1*sin1 + math.Cos(la1)*math.Cos(la2)*sin2*sin2
	return 2 * R * math.Asin(math.Min(1, math.Sqrt(h)))
}

func (s *Store) ComputeTrackStats(ctx context.Context, id int64) (*TrackStats, error) {
	ts := &TrackStats{
		MinX: math.Inf(1), MinY: math.Inf(1),
		MaxX: math.Inf(-1), MaxY: math.Inf(-1),
	}

	// Track name
	if err := s.DB.QueryRowContext(ctx, `SELECT name FROM tracks WHERE id = ?`, id).Scan(&ts.Name); err != nil {
		return nil, err
	}

	rows, err := s.DB.QueryContext(ctx, `
		SELECT lon, lat, t, sog_ms
		FROM positions
		WHERE track_id = ?
		ORDER BY t ASC
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prevLon, prevLat float64
	var prevT int64
	var hasPrev bool

	for rows.Next() {
		var lon, lat float64
		var t int64
		var sog sql.NullFloat64
		if err := rows.Scan(&lon, &lat, &t, &sog); err != nil {
			return nil, err
		}

		// set start/end
		if !hasPrev {
			ts.StartedAt = t
			hasPrev = true
			prevLon, prevLat, prevT = lon, lat, t
			// first point SOG unknown if not provided
			if sog.Valid {
				ts.SOGms = append(ts.SOGms, sog.Float64)
			} else {
				ts.SOGms = append(ts.SOGms, math.NaN())
			}
		} else {
			// accumulate distance
			seg := haversineMeters(prevLon, prevLat, lon, lat)
			ts.DistanceM += seg

			// compute SOG if missing
			if sog.Valid {
				ts.SOGms = append(ts.SOGms, sog.Float64)
			} else {
				dt := float64(t - prevT)
				if dt > 0 {
					ts.SOGms = append(ts.SOGms, seg/dt) // m/s
				} else {
					ts.SOGms = append(ts.SOGms, math.NaN())
				}
			}

			prevLon, prevLat, prevT = lon, lat, t
		}
		ts.EndedAt = t

		// bbox
		if lon < ts.MinX {
			ts.MinX = lon
		}
		if lon > ts.MaxX {
			ts.MaxX = lon
		}
		if lat < ts.MinY {
			ts.MinY = lat
		}
		if lat > ts.MaxY {
			ts.MaxY = lat
		}

		ts.Coords = append(ts.Coords, [2]float64{lon, lat})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(ts.Coords) == 0 {
		ts.MinX, ts.MaxX = 0, 0
		ts.MinY, ts.MaxY = 0, 0
	}
	return ts, nil
}
