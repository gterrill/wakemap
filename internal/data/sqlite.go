package data

import (
	"context"
	"database/sql"
	"errors"
	_ "github.com/mattn/go-sqlite3"
	"time"
	"wakemap/internal/db" // your sqlc models.go package
)

type Store struct {
	DB *sql.DB
	Q  *db.Queries
}

func Open(path string) (*Store, error) {
	d, err := sql.Open("sqlite3", "file:"+path+"?_busy_timeout=5000&_journal_mode=WAL&_fk=1")
	if err != nil {
		return nil, err
	}
	if err := d.Ping(); err != nil {
		_ = d.Close()
		return nil, err
	}
	if err := ensureSchema(d); err != nil {
		_ = d.Close()
		return nil, err
	}
	return &Store{DB: d, Q: db.New(d)}, nil
}

func (s *Store) Close() error { return s.DB.Close() }

// List the last N tracks (by started_at desc)
func (s *Store) ListTracks(ctx context.Context, limit int) ([]db.Track, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.Q.ListTracks(ctx, int64(limit))
}

func (s *Store) TrackBBox(ctx context.Context, trackID int64) (db.TrackBBoxRow, error) {
	return s.Q.TrackBBox(ctx, trackID)
}

func (s *Store) TrackPositions(ctx context.Context, trackID int64) ([]db.Position, error) {
	return s.Q.TrackPositions(ctx, trackID)
}

// Helpers
func UnixToTime(ts int64) time.Time {
	if ts <= 0 {
		return time.Time{}
	}
	return time.Unix(ts, 0).UTC()
}

var ErrNotFound = errors.New("not found")
