-- name: ListTracks :many
SELECT
  id,
  name,
  started_at,
  ended_at,
  distance_m,
  notes
FROM tracks
ORDER BY started_at DESC
LIMIT ?;

-- name: TrackBBox :one
SELECT
  MIN(lon) AS min_x,
  MIN(lat) AS min_y,
  MAX(lon) AS max_x,
  MAX(lat) AS max_y
FROM positions
WHERE track_id = ?;

-- name: TrackPositions :many
SELECT
  id,
  track_id,
  t,
  lon,
  lat,
  sog_ms,
  cog_rad,
  src,
  qual
FROM positions
WHERE track_id = ?
ORDER BY t ASC;