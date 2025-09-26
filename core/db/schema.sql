PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS tracks (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  started_at INTEGER NOT NULL,
  ended_at INTEGER,
  distance_m REAL DEFAULT 0,
  notes TEXT
);

CREATE TABLE IF NOT EXISTS positions (
  id INTEGER PRIMARY KEY,
  track_id INTEGER NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
  t INTEGER NOT NULL,
  lon REAL NOT NULL,
  lat REAL NOT NULL,
  sog_ms REAL,
  cog_rad REAL,
  src TEXT,
  qual INTEGER
);

-- R*Tree: fast bbox/nearby queries
CREATE VIRTUAL TABLE IF NOT EXISTS positions_rtree
USING rtree(id, minX, maxX, minY, maxY);
