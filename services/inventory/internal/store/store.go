// Package store keeps the inventory: artists, albums, and which source holds
// each album. It is SQLite, through a driver with no cgo, so the binary
// builds for a Pi from anywhere (ADR-0001).
//
// The inventory gives each artist and album an ID of its own, and keeps it
// across collections: a source's album is found again by its holding, the
// key the source knows it by. Each source is the truth for what it holds,
// so a collection replaces that source's holdings, and what it no longer
// holds goes.
package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"time"

	_ "modernc.org/sqlite"

	"github.com/cdeever/eds-inventory/services/inventory/internal/fold"
)

// Store is the inventory on disk.
type Store struct {
	db *sql.DB
}

// Holding is one album as one source holds it.
type Holding struct {
	// Key is what the source knows the album by, the same on every
	// collection.
	Key         string
	Artist      string
	Title       string
	Year        int
	Tracks      int
	Compilation bool
}

// Artist is an artist in the inventory, with their albums by title.
type Artist struct {
	ID     string
	Name   string
	Albums []Album
}

// Album is an album in the inventory.
type Album struct {
	ID          string
	Title       string
	Year        int
	Compilation bool
	// Tracks is the most tracks any source holds of it.
	Tracks int
	// Sources are the sources that hold it, by name.
	Sources []string
}

// Summary says what a collection changed.
type Summary struct {
	Added, Kept, Removed int
}

// schema is the database at each version; a database is brought up to the
// last, one step at a time, and its version kept in user_version.
var schema = []string{
	`CREATE TABLE artists (
		id     TEXT PRIMARY KEY,
		name   TEXT NOT NULL,
		folded TEXT NOT NULL UNIQUE
	);
	CREATE TABLE albums (
		id          TEXT PRIMARY KEY,
		artist_id   TEXT NOT NULL REFERENCES artists(id),
		title       TEXT NOT NULL,
		year        INTEGER NOT NULL DEFAULT 0,
		compilation INTEGER NOT NULL DEFAULT 0
	);
	CREATE INDEX albums_by_artist ON albums(artist_id);
	CREATE TABLE holdings (
		source    TEXT NOT NULL,
		key       TEXT NOT NULL,
		album_id  TEXT NOT NULL REFERENCES albums(id),
		title     TEXT NOT NULL,
		tracks    INTEGER NOT NULL,
		collected TEXT NOT NULL,
		PRIMARY KEY (source, key)
	);
	CREATE INDEX holdings_by_album ON holdings(album_id);
	CREATE TABLE collections (
		source    TEXT PRIMARY KEY,
		collected TEXT NOT NULL,
		albums    INTEGER NOT NULL
	);`,
}

// Open opens the inventory at path, making it if there is none, and brings
// its schema up to date.
func Open(path string) (*Store, error) {
	q := url.Values{}
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "journal_mode(WAL)")
	db, err := sql.Open("sqlite", "file:"+path+"?"+q.Encode())
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: %s: %w", path, err)
	}
	return s, nil
}

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	var v int
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		return err
	}
	if v > len(schema) {
		return fmt.Errorf("the database is version %d, and this program knows %d", v, len(schema))
	}
	for ; v < len(schema); v++ {
		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(schema[v]); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, v+1)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// Collect replaces what source holds with holdings, in one transaction: a
// holding seen before keeps its album and that album's ID, a new one makes
// an album, and one not seen this time is gone, with any album or artist
// nothing holds any more.
func (s *Store) Collect(ctx context.Context, source string, holdings []Holding, at time.Time) (Summary, error) {
	var sum Summary
	stamp := at.UTC().Format(time.RFC3339Nano)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return sum, err
	}
	defer tx.Rollback()

	for _, h := range holdings {
		var albumID string
		err := tx.QueryRowContext(ctx, `SELECT album_id FROM holdings WHERE source = ? AND key = ?`, source, h.Key).Scan(&albumID)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			artistID, err := artist(ctx, tx, h.Artist)
			if err != nil {
				return sum, err
			}
			// Each new holding is a new album for now. Knowing that one
			// source's album is another's is matching, which is not built.
			albumID = newID("al")
			if _, err := tx.ExecContext(ctx, `INSERT INTO albums (id, artist_id, title, year, compilation) VALUES (?, ?, ?, ?, ?)`,
				albumID, artistID, h.Title, h.Year, h.Compilation); err != nil {
				return sum, err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO holdings (source, key, album_id, title, tracks, collected) VALUES (?, ?, ?, ?, ?, ?)`,
				source, h.Key, albumID, h.Title, h.Tracks, stamp); err != nil {
				return sum, err
			}
			sum.Added++
		case err != nil:
			return sum, err
		default:
			// The source is the truth: what it says of the album now is
			// what the inventory says.
			if _, err := tx.ExecContext(ctx, `UPDATE holdings SET title = ?, tracks = ?, collected = ? WHERE source = ? AND key = ?`,
				h.Title, h.Tracks, stamp, source, h.Key); err != nil {
				return sum, err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE albums SET title = ?, year = ?, compilation = ? WHERE id = ?`,
				h.Title, h.Year, h.Compilation, albumID); err != nil {
				return sum, err
			}
			sum.Kept++
		}
	}

	res, err := tx.ExecContext(ctx, `DELETE FROM holdings WHERE source = ? AND collected <> ?`, source, stamp)
	if err != nil {
		return sum, err
	}
	n, _ := res.RowsAffected()
	sum.Removed = int(n)
	if _, err := tx.ExecContext(ctx, `DELETE FROM albums WHERE id NOT IN (SELECT album_id FROM holdings)`); err != nil {
		return sum, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM artists WHERE id NOT IN (SELECT artist_id FROM albums)`); err != nil {
		return sum, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO collections (source, collected, albums) VALUES (?, ?, ?)
		ON CONFLICT(source) DO UPDATE SET collected = excluded.collected, albums = excluded.albums`,
		source, stamp, len(holdings)); err != nil {
		return sum, err
	}
	return sum, tx.Commit()
}

// artist is the ID of the artist named, made if there is none. Two
// spellings of one name, folded alike, are one artist; the name kept is the
// first collected.
func artist(ctx context.Context, tx *sql.Tx, name string) (string, error) {
	folded := fold.Name(name)
	var id string
	err := tx.QueryRowContext(ctx, `SELECT id FROM artists WHERE folded = ?`, folded).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		id = newID("ar")
		_, err = tx.ExecContext(ctx, `INSERT INTO artists (id, name, folded) VALUES (?, ?, ?)`, id, name, folded)
	}
	return id, err
}

// newID is a new ID of the inventory's own: a kind and twelve hex digits.
func newID(kind string) string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand does not fail on any supported system
	}
	return kind + "_" + hex.EncodeToString(b)
}

// Artists is the whole inventory: every artist, by name, with their albums,
// by title.
func (s *Store) Artists(ctx context.Context) ([]Artist, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT ar.id, ar.name, al.id, al.title, al.year, al.compilation, h.source, h.tracks
		FROM artists ar
		JOIN albums al ON al.artist_id = ar.id
		JOIN holdings h ON h.album_id = al.id
		ORDER BY ar.folded, ar.id, al.title, al.id, h.source`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Artist
	for rows.Next() {
		var arID, arName, alID, title, source string
		var year, tracks int
		var compilation bool
		if err := rows.Scan(&arID, &arName, &alID, &title, &year, &compilation, &source, &tracks); err != nil {
			return nil, err
		}
		if len(out) == 0 || out[len(out)-1].ID != arID {
			out = append(out, Artist{ID: arID, Name: arName})
		}
		ar := &out[len(out)-1]
		if len(ar.Albums) == 0 || ar.Albums[len(ar.Albums)-1].ID != alID {
			ar.Albums = append(ar.Albums, Album{ID: alID, Title: title, Year: year, Compilation: compilation})
		}
		al := &ar.Albums[len(ar.Albums)-1]
		al.Tracks = max(al.Tracks, tracks)
		al.Sources = append(al.Sources, source)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		sort.SliceStable(out[i].Albums, func(a, b int) bool { return out[i].Albums[a].Title < out[i].Albums[b].Title })
	}
	return out, nil
}

// Version says which collection the inventory is at; it changes whenever a
// collection is made, so a reader knows to read the inventory again.
func (s *Store) Version(ctx context.Context) (string, error) {
	var v string
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(collected), '') || '/' || COUNT(*) FROM collections`).Scan(&v)
	return v, err
}
