-- Demo seed v1: Sydney Harbour -> Huskisson (Jervis Bay)
-- Leg A (~first 15 NM): ~9.8 kts; Leg B: 8–9 kts. Stays offshore and inside bay water only; no land crossings.
-- Waypoints included exactly (lon,lat):
--  * (151.326870,-33.960709)
--  * (151.196408,-34.295209)
--  * (150.984921,-34.495782)
--  * (150.875058,-35.066512)
--  * (150.813650,-35.111917)
--  * (150.686620,-35.050388)
PRAGMA foreign_keys=ON;
BEGIN;

-- Idempotent cleanup (mirrors demo pattern)
-- Idempotent cleanup (only this seed)
DELETE FROM positions WHERE track_id IN (
  SELECT id FROM tracks WHERE name = 'Demo: Sydney Harbour → Huskisson'
);
DELETE FROM tracks WHERE name = 'Demo: Sydney Harbour → Huskisson';

-- Create track (duration based on final dt_min below)
INSERT INTO tracks (name, started_at, ended_at, notes)
VALUES (
  'Demo: Sydney Harbour → Huskisson',
  strftime('%s','now') - (315 * 60), -- start now - total minutes
  strftime('%s','now'),
  'Start ~9.8 kts for ~15 NM, then 8–9 kts. Includes required waypoints; offshore route avoiding land; finishes near Huskisson.'
);

-- Points schema: pt, lon, lat, dt_min (since start), sog_ms, cog_rad
-- cog_rad are approximate bearings (radians) for realism; adjust if you compute per-segment bearings.
WITH points(pt, lon, lat, dt_min, sog_ms, cog_rad) AS (VALUES
  -- START: Sydney Harbour (33°52\'06.0\"S, 151°17\'52.0\"E) => (151.297778,-33.868333)
  ( 1, 151.297778, -33.868333,   0, 5.040, 3.20),   -- ~9.8 kt
  ( 2, 151.305000, -33.900000,  10, 5.040, 3.20),
  -- Required waypoint 1 (offshore of South Head/Coogee sector)
  ( 3, 151.326870, -33.960709,  25, 5.040, 3.05),
  ( 4, 151.290000, -34.100000,  45, 5.040, 2.95),
  ( 5, 151.250000, -34.160000,  60, 5.040, 3.05),
  -- Required waypoint 2 (off Wollongong/Shellharbour offshore)
  ( 6, 151.196408, -34.295209,  90, 5.040, 3.10),  -- ≈ first 15 NM complete here

  -- CRUISE: 8–9 kts thereafter (4.12–4.63 m/s). Slight variation for realism.
  ( 7, 151.100000, -34.380000, 110, 4.500, 3.20),
  -- Required waypoint 3 (off Kiama/Gerringong offshore)
  ( 8, 150.984921, -34.495782, 135, 4.400, 3.30),
  ( 9, 150.930000, -34.650000, 165, 4.500, 3.40),
  (10, 150.900000, -34.800000, 195, 4.300, 3.55),
  -- Required waypoint 4 (approach towards Jervis Bay latitude)
  (11, 150.875058, -35.066512, 245, 4.500, 3.60),
  -- Required waypoint 5 (inside Jervis Bay water; stay clear of land)
  (12, 150.813650, -35.111917, 260, 4.400, 1.90),
  (13, 150.761468, -35.081793, 275, 4.300, 1.20),
  -- Required waypoint 6 (inside bay, east of Huskisson)
  (14, 150.686620, -35.050388, 305, 4.400, 1.30),
  -- FINAL: Huskisson vicinity (east side of township, in water)
  (15, 150.675637, -35.036775, 315, 0.800, 1.10)   -- slow on final approach
)
INSERT INTO positions (track_id, t, lon, lat, sog_ms, cog_rad, src, qual)
SELECT
  (SELECT last_insert_rowid()),
  (strftime('%s','now') - (315 * 60)) + (dt_min * 60),
  lon, lat,
  sog_ms,
  cog_rad,
  'seed_syd_huskisson_v1',
  1
FROM points;

-- Update track end time to max sample time (defensive if you tweak dt_min)
UPDATE tracks
SET ended_at = (
  SELECT MAX(t) FROM positions WHERE track_id = (SELECT last_insert_rowid())
)
WHERE id = (SELECT last_insert_rowid());

COMMIT;
