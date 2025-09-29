PRAGMA foreign_keys=ON;

CREATE TABLE IF NOT EXISTS tracks (
  id         INTEGER PRIMARY KEY,
  name       TEXT NOT NULL,
  started_at INTEGER NOT NULL,     -- epoch seconds
  ended_at   INTEGER,
  distance_m REAL,
  notes      TEXT
);

CREATE TABLE IF NOT EXISTS positions (
  id       INTEGER PRIMARY KEY,
  track_id INTEGER NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
  t        INTEGER NOT NULL,       -- epoch seconds
  lon      REAL NOT NULL,
  lat      REAL NOT NULL,
  sog_ms   REAL,
  cog_rad  REAL,
  src      TEXT,
  qual     INTEGER
);

CREATE VIRTUAL TABLE IF NOT EXISTS positions_rtree USING rtree(
  id, minX, maxX, minY, maxY
);

CREATE TRIGGER IF NOT EXISTS positions_rtree_ins
AFTER INSERT ON positions BEGIN
  INSERT OR REPLACE INTO positions_rtree(id,minX,maxX,minY,maxY)
  VALUES (new.id, new.lon, new.lon, new.lat, new.lat);
END;

CREATE TRIGGER IF NOT EXISTS positions_rtree_upd
AFTER UPDATE OF lon,lat ON positions BEGIN
  INSERT OR REPLACE INTO positions_rtree(id,minX,maxX,minY,maxY)
  VALUES (new.id, new.lon, new.lon, new.lat, new.lat);
END;

CREATE TRIGGER IF NOT EXISTS positions_rtree_del
AFTER DELETE ON positions BEGIN
  DELETE FROM positions_rtree WHERE id = old.id;
END;

CREATE INDEX IF NOT EXISTS idx_positions_track_time ON positions(track_id, t);
CREATE INDEX IF NOT EXISTS idx_tracks_started_at ON tracks(started_at DESC);