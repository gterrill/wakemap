-- name: CreateTrack :one
INSERT INTO tracks (name, started_at) VALUES (?, ?)
RETURNING id, name, started_at, ended_at, distance_m, notes;

-- name: EndTrack :exec
UPDATE tracks SET ended_at = ? WHERE id = ?;

-- name: ListTracks :many
SELECT id, name, started_at, ended_at, distance_m, notes
FROM tracks
ORDER BY started_at DESC
LIMIT ?;

-- name: InsertPosition :exec
INSERT INTO positions (track_id, t, lon, lat, sog_ms, cog_rad, src, qual)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpsertRTree :exec
INSERT OR REPLACE INTO positions_rtree (id, minX, maxX, minY, maxY)
VALUES (?, ?, ?, ?, ?);

-- name: BBoxForTrack :one
SELECT
  COALESCE(CAST(MIN(lon) AS REAL), 0) AS min_lon,
  COALESCE(CAST(MAX(lon) AS REAL), 0) AS max_lon,
  COALESCE(CAST(MIN(lat) AS REAL), 0) AS min_lat,
  COALESCE(CAST(MAX(lat) AS REAL), 0) AS max_lat
FROM positions
WHERE track_id = ?;


-- name: InsertPositionReturning :one
INSERT INTO positions (track_id, t, lon, lat, sog_ms, cog_rad, src, qual)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id;

-- name: PositionsForTrack :many
SELECT lon, lat
FROM positions
WHERE track_id = ?
ORDER BY t ASC;
