-- Only for sqlc analysis; NOT applied at runtime.
CREATE TABLE IF NOT EXISTS positions_rtree (
  id   INTEGER PRIMARY KEY,
  minX REAL NOT NULL,
  maxX REAL NOT NULL,
  minY REAL NOT NULL,
  maxY REAL NOT NULL
);
