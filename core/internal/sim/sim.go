// core/internal/sim/sim.go
package sim

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	dbgen "github.com/gterrill/wakemap/core/internal/db"
)

const earthRadiusM = 6371000.0
const nmToMeters = 1852.0

func deg2rad(d float64) float64 { return d * (math.Pi / 180.0) }
func rad2deg(r float64) float64 { return r * (180.0 / math.Pi) }

func destPoint(latDeg, lonDeg, bearingRad, distM float64) (outLatDeg, outLonDeg float64) {
	φ1 := deg2rad(latDeg)
	λ1 := deg2rad(lonDeg)
	δ := distM / earthRadiusM
	θ := bearingRad

	φ2 := math.Asin(math.Sin(φ1)*math.Cos(δ) + math.Cos(φ1)*math.Sin(δ)*math.Cos(θ))
	λ2 := λ1 + math.Atan2(math.Sin(θ)*math.Sin(δ)*math.Cos(φ1), math.Cos(δ)-math.Sin(φ1)*math.Sin(φ2))
	return rad2deg(φ2), rad2deg(λ2)
}

func bearingRad(lat1, lon1, lat2, lon2 float64) float64 {
	φ1, φ2 := deg2rad(lat1), deg2rad(lat2)
	Δλ := deg2rad(lon2 - lon1)
	y := math.Sin(Δλ) * math.Cos(φ2)
	x := math.Cos(φ1)*math.Sin(φ2) - math.Sin(φ1)*math.Cos(φ2)*math.Cos(Δλ)
	θ := math.Atan2(y, x)
	if θ < 0 {
		θ += 2 * math.Pi
	}
	return θ
}

func haversineM(lat1, lon1, lat2, lon2 float64) float64 {
	φ1, φ2 := deg2rad(lat1), deg2rad(lat2)
	dφ := φ2 - φ1
	dλ := deg2rad(lon2 - lon1)
	a := math.Sin(dφ/2)*math.Sin(dφ/2) + math.Cos(φ1)*math.Cos(φ2)*math.Sin(dλ/2)*math.Sin(dλ/2)
	return 2 * earthRadiusM * math.Asin(math.Sqrt(a))
}

// UpsertRTree inserts a point bbox for a position id.
// Note: sqlc generated field names are Minx/Maxx/Miny/Maxy (lowercase x/y).
func UpsertRTree(ctx context.Context, q *dbgen.Queries, posID int64, lon, lat float64) error {
	return q.UpsertRTree(ctx, dbgen.UpsertRTreeParams{
		ID:   posID,
		Minx: lon, Maxx: lon,
		Miny: lat, Maxy: lat,
	})
}

// RunBroughtonToNewcastle simulates the route and returns (trackID, points, endedAt).
func RunBroughtonToNewcastle(ctx context.Context, q *dbgen.Queries, speedKn float64, intervalS int) (int64, int, int64, error) {
	if speedKn <= 0 {
		speedKn = 6
	}
	if intervalS < 1 {
		intervalS = 10
	}

	now := time.Now().Unix()

	// (lat, lon)
	route := [][2]float64{
		{-32.60, 152.40}, // 5 nmi east of Broughton Island
		{-32.60, 152.30}, // abeam Broughton Island
		{-32.71, 152.17}, // Port Stephens entrance
		{-32.79, 152.08}, // Stockton Bight
		{-32.92, 151.79}, // Newcastle Harbour entrance
	}

	tr, err := q.CreateTrack(ctx, dbgen.CreateTrackParams{
		Name:      "Sim: Broughton→Newcastle (SSW)",
		StartedAt: now,
	})
	if err != nil {
		return 0, 0, 0, fmt.Errorf("create track: %w", err)
	}

	sogMS := speedKn * nmToMeters / 3600.0
	dt := float64(intervalS)
	t := now
	points := 0

	lastLon, lastLat := route[0][1], route[0][0]
	firstID, err := q.InsertPositionReturning(ctx, dbgen.InsertPositionReturningParams{
		TrackID: int64(tr.ID),
		T:       t,
		Lon:     lastLon,
		Lat:     lastLat,
		SogMs:   sql.NullFloat64{Float64: sogMS, Valid: true},
		CogRad:  sql.NullFloat64{Float64: 0, Valid: true},
		Src:     sql.NullString{String: "sim", Valid: true},
		Qual:    sql.NullInt64{Int64: 1, Valid: true},
	})
	if err != nil {
		return 0, 0, 0, fmt.Errorf("insert first pos: %w", err)
	}
	if err := UpsertRTree(ctx, q, firstID, lastLon, lastLat); err != nil {
		return 0, 0, 0, fmt.Errorf("rtree first pos: %w", err)
	}
	points++

	for leg := 0; leg < len(route)-1; leg++ {
		lat1, lon1 := route[leg][0], route[leg][1]
		lat2, lon2 := route[leg+1][0], route[leg+1][1]

		brg := bearingRad(lat1, lon1, lat2, lon2)
		dist := haversineM(lat1, lon1, lat2, lon2)

		stepM := sogMS * dt
		steps := int(math.Ceil(dist / stepM))
		if steps < 1 {
			steps = 1
		}

		curLat, curLon := lastLat, lastLon
		for i := 0; i < steps; i++ {
			var nextLat, nextLon float64
			if i == steps-1 {
				nextLat, nextLon = lat2, lon2
			} else {
				nextLat, nextLon = destPoint(curLat, curLon, brg, stepM)
			}

			t += int64(intervalS)
			id, err := q.InsertPositionReturning(ctx, dbgen.InsertPositionReturningParams{
				TrackID: int64(tr.ID),
				T:       t,
				Lon:     nextLon,
				Lat:     nextLat,
				SogMs:   sql.NullFloat64{Float64: sogMS, Valid: true},
				CogRad:  sql.NullFloat64{Float64: brg, Valid: true},
				Src:     sql.NullString{String: "sim", Valid: true},
				Qual:    sql.NullInt64{Int64: 1, Valid: true},
			})
			if err != nil {
				return 0, 0, 0, fmt.Errorf("insert pos (leg %d): %w", leg, err)
			}
			if err := UpsertRTree(ctx, q, id, nextLon, nextLat); err != nil {
				return 0, 0, 0, fmt.Errorf("rtree upsert: %w", err)
			}

			points++
			curLat, curLon = nextLat, nextLon
			lastLat, lastLon = nextLat, nextLon
		}
	}

	if err := q.EndTrack(ctx, dbgen.EndTrackParams{
		EndedAt: sql.NullInt64{Int64: t, Valid: true},
		ID:      int64(tr.ID),
	}); err != nil {
		// Not fatal
	}

	return int64(tr.ID), points, t, nil
}
