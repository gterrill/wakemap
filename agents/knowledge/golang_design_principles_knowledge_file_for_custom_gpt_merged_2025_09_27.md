# Golang Design Principles – Knowledge File

_Last updated: 2025-09-27_

This file is the **source of truth** for how an agent answers questions about Go design, project layout, testing, HTTP APIs, errors, observability, and the specific **wakemap** system we’re building.

If guidance here conflicts with general knowledge, **follow this file**.

---

## Mission (wakemap)
A self-hosted, offline-first mapping system that:
- Renders self-hosted **OpenFreeMap** tiles with **OpenSeaMap** seamarks overlay.
- Logs tracks from **Signal K**, exports static PNG maps, and generates **Immich** sidecar metadata.
- Surfaces nearby POIs using open data, optionally re-ranked with a compliant **GooglePlacesHotnessProxyScore**.

**Stack**: Go server (net/http) + MapLibre GL JS; tiles self-hosted; SQLite + R*Tree; optional PMTiles; chromedp for PNG exports.

---

## Operating Principles
- **Offline-first**: Prefer local assets (self-hosted tiles, PMTiles, cached POI text). Degrade gracefully.
- **Self-hosted**: All services run on the NAS or LAN VM/container (Tailscale if remote). No public exposure required.
- **Clarity & Actionability**: Provide concise steps, file trees, commands, and minimal code that runs.
- **Compliance**: Proper attribution for OpenSeaMap; Google Places data used **only** for server-side ranking (no display on non-Google maps; strict caching rules).
- **Safety & Reliability**: Rate-limits, caching headers, health checks, backups, and privacy features (redaction zones).

---

## Go Design Principles
- **Main is thin**: Wiring only. Business logic in `internal/domain` / `internal/data`; HTTP in `internal/server`.
- **Small, consumer-led interfaces**; avoid god interfaces. Prefer concrete types; use interfaces at boundaries.
- **Zero-value-safe types** and constructors when necessary.
- **Context-aware handlers**; pass `context.Context` to all IO-bound ops.
- **Structured logging**; include request IDs and key fields.
- **Errors are JSON** for HTTP: `{"error": {"code": string, "message": string, "details": object}}` with accurate status codes.
- **Observability**: basic request logging, latency counters, error counts; health endpoints.
- **Testing**: unit tests around data and handlers; golden files for JSON; avoid hidden globals.

---

## Project Layout (wakemap)
```
cmd/
  wakemap/
    main.go              # thin: env, DB open, router, listen
internal/
  data/
    store.go             # DB open, ensureSchema, Store struct
    migrate.go           # embedded migrations + ensureSchema()
    tracks_stats.go      # ComputeTrackStats(ctx, id)
    migrations/
      001_init.sql       # runtime-embedded DDL
  db/
    migrations/          # sqlc’s authoritative schema
      001_init.sql
    queries/             # sqlc queries (ListTracks, etc.)
      tracks.sql
    queries.sql.go       # generated (by sqlc)
  server/
    api.go               # API struct, writeError helper
    router.go            # NewMux(api *API) *http.ServeMux
    handlers_tracks.go   # ListTracks, TrackGeoJSONByID
web/
  ... (Vite + MapLibre)
internal/db/seed/
  001_demo_broughton_to_newcastle.sql  # demo track (v3 path)
```

---

## Environment & Config
- Use `github.com/joho/godotenv` in `cmd/wakemap/main.go`.
- Expand vars with `os.ExpandEnv`.
- Prefer absolute paths for the DB in `.env` or `${PWD}/...`.
- Log a **single** startup line: `wakemap dev server on http://localhost:${PORT}  db=${dbPath}`.

**Snippet**
```go
// cmd/wakemap/main.go (essentials)
godotenv.Load(".env")
port := getenv("PORT", "8080")
dbPath := os.ExpandEnv(getenv("WAKEMAP_DB", "./devdata/wakemap.db"))
store, err := data.Open(dbPath) // calls ensureSchema()
api := &server.API{Store: store}
mux := server.NewMux(api)
log.Printf("wakemap dev server on http://localhost:%s  db=%s", port, dbPath)
log.Fatal(http.ListenAndServe(":"+port, mux))
```

---

## SQLite & Migrations (Runtime Embedded)
- Place runtime DDL at `internal/data/migrations/001_init.sql` and embed it.
- Run `ensureSchema` on boot to create tables if missing.

**Snippet**
```go
// internal/data/migrate.go
//go:build !js
package data

import (
  "database/sql"
  "embed"
)

//go:embed migrations/001_init.sql
var migFS embed.FS

func ensureSchema(db *sql.DB) error {
  ddl, err := migFS.ReadFile("migrations/001_init.sql")
  if err != nil { return err }
  _, err = db.Exec(string(ddl))
  return err
}
```

---

## sqlc (Schema & Queries)
- Keep `sqlc.yaml` at repo root; point inputs to `internal/db/migrations/*.sql` and `internal/db/queries/*.sql`.
- Generated code to `internal/db`.

**Example query**
```sql
-- internal/db/queries/tracks.sql
-- name: ListTracks :many
SELECT id, name, started_at, ended_at, distance_m
FROM tracks
ORDER BY started_at DESC
LIMIT ?;
```

Run:
```bash
sqlc generate
```

---

## HTTP API (Tracks)
### List
`GET /api/tracks` →
```json
{"tracks":[{"id":1,"name":"Demo...","started_at":1695800000,"ended_at":1695805400}]}
```

### GeoJSON (server-computed stats)
`GET /api/tracks/:id.geojson` → returns a **Feature** with `bbox` and **stats** in `properties`:
```json
{
  "type": "Feature",
  "properties": {
    "id": 123,
    "name": "Demo: Broughton → Newcastle (SSW) v3",
    "started_at": 1695800000,
    "ended_at": 1695805400,
    "duration_s": 5400,
    "distance_m": 65500.0,
    "distance_nm": 35.4,
    "avg_knots": 14.1
  },
  "geometry": { "type": "LineString", "coordinates": [[lon,lat], ...] },
  "bbox": [minLon, minLat, maxLon, maxLat]
}
```

### Errors
Always JSON with content type:
```
Content-Type: application/json; charset=utf-8
```
Body:
```json
{"error":{"code":"db_error","message":"failed to list tracks","details":{"err":"..."}}}
```

---

## Server-Side Stats (Source of Truth)
Compute total distance (Haversine), duration, avg speed, and bbox on the server.

**Data layer**
```go
// internal/data/tracks_stats.go (essentials)
type TrackStats struct {
  Name string
  StartedAt, EndedAt int64
  DistanceM float64
  MinX, MinY, MaxX, MaxY float64
  Coords [][2]float64
}

func (s *Store) ComputeTrackStats(ctx context.Context, id int64) (*TrackStats, error)
```

**Handler** enriches GeoJSON properties with `distance_m`, `distance_nm`, `duration_s`, `avg_knots`, `started_at`, `ended_at`.

---

## Router
Provide a constructor and keep `main` clean.

```go
// internal/server/router.go
func NewMux(api *API) *http.ServeMux {
  mux := http.NewServeMux()
  mux.HandleFunc("/api/tracks", api.ListTracks)
  mux.HandleFunc("/api/tracks/", api.TrackGeoJSONByID)
  mux.Handle("/", http.FileServer(http.Dir(filepath.Join("web","dist"))))
  return mux
}
```

---

## CORS & Tile Proxy (Seamarks)
- Avoid duplicate `Access-Control-Allow-Origin`. Set **one** origin (e.g., Vite origin) and `Vary: Origin`.
- Add caching for raster tiles.

**Headers**
```
Access-Control-Allow-Origin: http://localhost:5173
Vary: Origin
Cache-Control: public, max-age=3600
```

---

## Seeding Demo Data
- Demo: **Broughton → Newcastle (v3)** keeps ≥ 2 NM offshore until the final point (~100 m E of the Newcastle breakwater) and goes due south before vectoring SW.

Apply:
```bash
sqlite3 "$WAKEMAP_DB" < internal/db/seed/001_demo_broughton_to_newcastle.sql
```

---

## SQLite Driver Choice
Pick one and stick to it:
- **CGO**: `_ "github.com/mattn/go-sqlite3"` with `sql.Open("sqlite3", path)`
- **Pure Go**: `_ "modernc.org/sqlite"` with `sql.Open("sqlite", path)`

---

## Frontend Contract (MapLibre)
- Load `/api/tracks` to populate the last 50 tracks.
- On selection, fetch `/api/tracks/:id.geojson`, set the `track` source data, then `fitBounds(bbox)`.
- Display stats from **server** `properties` (fallback to client computation if missing).
- Provide a small stats bar (distance nm, duration, avg kn).

---

## Ops & Privacy
- **Caching/limits**: ETag/Last-Modified and `Cache-Control` on seamark proxy; soft rate limits.
- **Backups**: rotate SQLite by year (e.g., `tracks_YYYY.db`), nightly copy; keep exported PNGs + sidecars in versioned folders.
- **Privacy**: redaction polygons; private mode suppresses implicit logging.
- **Monitoring**: health endpoints, disk space checks, cron for OpenFreeMap updates and nightly POI/ratings refresh.

---

## Google Places Compliance (If Used For Ranking)
- Only for server-side ranking of a pre-filtered shortlist; **do not display** Google-derived ratings/reviews/photos on the map UI.
- Cache `place_id` indefinitely; **do not** store Place Details besides a short-lived, non-reversible score (TTL ≤ 7 days). Coordinates ≤ 30 days.
- Request minimal fields (FieldMask) and compute a proxy score (e.g., Bayesian-adjusted rating + log review count + photo presence).
- Optional "View on Google Maps" link (new tab). Gate calls by intent; debounce; rate-limit; fallback to open-data ranking.

---

## Changelog
### 2025-09-27
- Embedded migrations and `ensureSchema()` on boot.
- `.env` loading with `${VAR}` expansion; clear startup log.
- `server.NewMux(api)` and thin `main` wiring.
- `GET /api/tracks/:id.geojson` returns server-computed `duration_s`, `distance_nm`, `avg_knots`, `bbox`.
- sqlc layout clarified; root `sqlc.yaml` consuming `internal/db/migrations` + `internal/db/queries`.
- Demo seed updated (v3): due south to abreast of Port Stephens, then SW to Newcastle; ≥ 2 NM offshore until final point.
- CORS guidance fixed (single A-C-A-O value).

