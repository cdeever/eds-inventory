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
	// Aliases are the other names MusicBrainz knows the artist by.
	Aliases []string
	// Discography is the artist's studio albums, or nil when not yet
	// enriched.
	Discography *Discography
}

// Discography is an artist's studio albums, oldest first, as MusicBrainz
// counts them (ADR-0004).
type Discography struct {
	MBID string
	// Name is the artist as MusicBrainz names them.
	Name string
	// Found is false when MusicBrainz knows no such artist, or no studio
	// album by them: asked, and nothing to count by.
	Found   bool
	Fetched time.Time
	Albums  []StudioAlbum
}

// StudioAlbum is one album of a discography.
type StudioAlbum struct {
	ReleaseGroup string
	Title        string
	Released     string // YYYY[-MM[-DD]], or ""
	// OtherTitles are the names its releases have gone by, most used first,
	// when they have been asked for.
	OtherTitles   []string
	TitlesFetched bool
}

// ArtistRef names an artist, for work to be done on them.
type ArtistRef struct {
	ID, Name string
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
	// 2: discographies, enriched from MusicBrainz (ADR-0004). Each belongs
	// to an artist and goes with it.
	`CREATE TABLE discographies (
		artist_id TEXT PRIMARY KEY REFERENCES artists(id) ON DELETE CASCADE,
		mbid      TEXT NOT NULL DEFAULT '',
		name      TEXT NOT NULL DEFAULT '',
		found     INTEGER NOT NULL,
		fetched   TEXT NOT NULL
	);
	CREATE TABLE studio_albums (
		artist_id      TEXT NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
		position       INTEGER NOT NULL,
		release_group  TEXT NOT NULL,
		title          TEXT NOT NULL,
		released       TEXT NOT NULL DEFAULT '',
		titles_fetched TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (artist_id, position)
	);
	CREATE INDEX studio_albums_by_group ON studio_albums(release_group);
	CREATE TABLE artist_aliases (
		artist_id TEXT NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
		alias     TEXT NOT NULL,
		folded    TEXT NOT NULL,
		PRIMARY KEY (artist_id, folded)
	);
	CREATE TABLE release_titles (
		release_group TEXT NOT NULL,
		title         TEXT NOT NULL,
		uses          INTEGER NOT NULL,
		PRIMARY KEY (release_group, title)
	);
	CREATE TABLE enrichments (
		n INTEGER NOT NULL
	);
	INSERT INTO enrichments (n) VALUES (0);`,
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
	if _, err := tx.ExecContext(ctx, `DELETE FROM release_titles WHERE release_group NOT IN (SELECT release_group FROM studio_albums)`); err != nil {
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
	if err := s.enrichment(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

// enrichment fills in each artist's aliases and discography.
func (s *Store) enrichment(ctx context.Context, artists []Artist) error {
	byID := map[string]*Artist{}
	for i := range artists {
		byID[artists[i].ID] = &artists[i]
	}
	rows, err := s.db.QueryContext(ctx, `SELECT artist_id, alias FROM artist_aliases ORDER BY artist_id, alias`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id, alias string
		if err := rows.Scan(&id, &alias); err != nil {
			rows.Close()
			return err
		}
		if a := byID[id]; a != nil {
			a.Aliases = append(a.Aliases, alias)
		}
	}
	rows.Close()

	rows, err = s.db.QueryContext(ctx, `SELECT artist_id, mbid, name, found, fetched FROM discographies`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id, fetched string
		var d Discography
		if err := rows.Scan(&id, &d.MBID, &d.Name, &d.Found, &fetched); err != nil {
			rows.Close()
			return err
		}
		d.Fetched, _ = time.Parse(time.RFC3339Nano, fetched)
		if a := byID[id]; a != nil {
			a.Discography = &d
		}
	}
	rows.Close()

	titles := map[string][]string{}
	rows, err = s.db.QueryContext(ctx, `SELECT release_group, title FROM release_titles ORDER BY release_group, uses DESC, title`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var g, t string
		if err := rows.Scan(&g, &t); err != nil {
			rows.Close()
			return err
		}
		titles[g] = append(titles[g], t)
	}
	rows.Close()

	rows, err = s.db.QueryContext(ctx, `SELECT artist_id, release_group, title, released, titles_fetched FROM studio_albums ORDER BY artist_id, position`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, fetched string
		var sa StudioAlbum
		if err := rows.Scan(&id, &sa.ReleaseGroup, &sa.Title, &sa.Released, &fetched); err != nil {
			return err
		}
		sa.TitlesFetched = fetched != ""
		sa.OtherTitles = titles[sa.ReleaseGroup]
		if a := byID[id]; a != nil && a.Discography != nil {
			a.Discography.Albums = append(a.Discography.Albums, sa)
		}
	}
	return rows.Err()
}

// ToEnrich are artists whose discography is to be asked for: never asked,
// or asked before staleBefore, never asked first. An artist with nothing but
// compilations, "Various Artists" among them, has no discography to ask for.
func (s *Store) ToEnrich(ctx context.Context, staleBefore time.Time, limit int) ([]ArtistRef, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT ar.id, ar.name FROM artists ar
		LEFT JOIN discographies d ON d.artist_id = ar.id
		WHERE EXISTS (SELECT 1 FROM albums al WHERE al.artist_id = ar.id AND al.compilation = 0)
		  AND (d.artist_id IS NULL OR d.fetched < ?)
		ORDER BY d.fetched IS NOT NULL, d.fetched, ar.folded
		LIMIT ?`, staleBefore.UTC().Format(time.RFC3339Nano), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ArtistRef
	for rows.Next() {
		var r ArtistRef
		if err := rows.Scan(&r.ID, &r.Name); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SaveDiscography replaces an artist's discography and aliases. Names
// already asked for of an album that is still in it are kept.
func (s *Store) SaveDiscography(ctx context.Context, artistID string, d Discography, aliases []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM artists WHERE id = ?`, artistID).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return nil // collected away while it was being asked for
	}
	fetchedTitles := map[string]string{}
	rows, err := tx.QueryContext(ctx, `SELECT release_group, titles_fetched FROM studio_albums WHERE artist_id = ?`, artistID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var g, f string
		rows.Scan(&g, &f)
		fetchedTitles[g] = f
	}
	rows.Close()
	stamp := d.Fetched.UTC().Format(time.RFC3339Nano)
	for _, q := range []string{`DELETE FROM studio_albums WHERE artist_id = ?`, `DELETE FROM artist_aliases WHERE artist_id = ?`, `DELETE FROM discographies WHERE artist_id = ?`} {
		if _, err := tx.ExecContext(ctx, q, artistID); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO discographies (artist_id, mbid, name, found, fetched) VALUES (?, ?, ?, ?, ?)`,
		artistID, d.MBID, d.Name, d.Found, stamp); err != nil {
		return err
	}
	for i, sa := range d.Albums {
		if _, err := tx.ExecContext(ctx, `INSERT INTO studio_albums (artist_id, position, release_group, title, released, titles_fetched) VALUES (?, ?, ?, ?, ?, ?)`,
			artistID, i+1, sa.ReleaseGroup, sa.Title, sa.Released, fetchedTitles[sa.ReleaseGroup]); err != nil {
			return err
		}
	}
	for _, al := range aliases {
		f := fold.Name(al)
		if f == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO artist_aliases (artist_id, alias, folded) VALUES (?, ?, ?)`, artistID, al, f); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM release_titles WHERE release_group NOT IN (SELECT release_group FROM studio_albums)`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE enrichments SET n = n + 1`); err != nil {
		return err
	}
	return tx.Commit()
}

// SaveTitles keeps the other names an album's releases have gone by, and
// that they have been asked for.
func (s *Store) SaveTitles(ctx context.Context, releaseGroup string, titles map[string]int, at time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM release_titles WHERE release_group = ?`, releaseGroup); err != nil {
		return err
	}
	for t, uses := range titles {
		if _, err := tx.ExecContext(ctx, `INSERT INTO release_titles (release_group, title, uses) VALUES (?, ?, ?)`, releaseGroup, t, uses); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE studio_albums SET titles_fetched = ? WHERE release_group = ?`, at.UTC().Format(time.RFC3339Nano), releaseGroup); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE enrichments SET n = n + 1`); err != nil {
		return err
	}
	return tx.Commit()
}

// Version says which collection and enrichment the inventory is at; it
// changes whenever either is made, so a reader knows to read it again.
func (s *Store) Version(ctx context.Context) (string, error) {
	var v string
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(collected), '') || '/' || COUNT(*) || '/' || (SELECT n FROM enrichments) FROM collections`).Scan(&v)
	return v, err
}
