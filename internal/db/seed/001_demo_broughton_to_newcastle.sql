-- Demo seed v3: Broughton (5 NM E) -> Newcastle
-- Leg A: due south to abreast of Port Stephens entrance (stay offshore)
-- Leg B: vector SW toward Newcastle; remain >= 2 NM offshore until final fix
PRAGMA foreign_keys=ON;
BEGIN;

-- Idempotent cleanup
DELETE FROM positions WHERE track_id IN (
  SELECT id FROM tracks WHERE name = 'Demo: Broughton → Newcastle (SSW)'
);
DELETE FROM tracks WHERE name = 'Demo: Broughton → Newcastle (SSW)';

-- Create track (~90 min total)
INSERT INTO tracks (name, started_at, ended_at, notes)
VALUES (
  'Demo: Broughton → Newcastle (SSW)',
  strftime('%s','now') - 5400,
  strftime('%s','now'),
  'Due south to abreast of Port Stephens, then SW to Newcastle; >=2 NM offshore until the final point'
);

-- Notes:
--  * Coast near Port Stephens entrance ~152.20E; we stay at 152.30E on the southbound leg (~9 km offshore).
--  * On the SW leg we keep longitudes >= ~0.04–0.06° east of coastline (>= 2 NM) until the last fix.
--  * Final: ~100 m east of the breakwater ≈ lon +0.00107 from ~151.7900E at -32.9200S => 151.79107E.
WITH points(pt, lon, lat, dt_min) AS (VALUES
  -- LEG A: due south (constant lon ≈ 152.30E)
  ( 1, 152.3000, -32.6000,   0),  -- ~5 NM E of Broughton Islands
  ( 2, 152.3000, -32.6300,   7),
  ( 3, 152.3000, -32.6600,  14),
  ( 4, 152.3000, -32.6900,  21),
  ( 5, 152.3000, -32.7050,  27),  -- abreast of Port Stephens entrance (~-32.705)

  -- LEG B: turn SW toward Newcastle; keep >=2 NM offshore until final
  ( 6, 152.2500, -32.7400,  35),
  ( 7, 152.1800, -32.7800,  45),
  ( 8, 152.0800, -32.8300,  55),
  ( 9, 151.9600, -32.8800,  65),
  (10, 151.8600, -32.9100,  75),  -- still offshore approaching Newcastle
  -- Final: ~100 m east of breakwater
  (11, 151.79107, -32.9200, 90)
)
INSERT INTO positions (track_id, t, lon, lat, sog_ms, cog_rad, src, qual)
SELECT
  (SELECT last_insert_rowid()),
  (strftime('%s','now') - 5400) + (dt_min * 60),
  lon, lat,
  8.0,       -- ~15.5 kn
  3.75,      -- ~215° (SSW-ish overall)
  'seed_v3',
  1
FROM points;

UPDATE tracks
SET ended_at = (SELECT MAX(t) FROM positions WHERE track_id = (SELECT last_insert_rowid()))
WHERE id = (SELECT last_insert_rowid());

COMMIT;
