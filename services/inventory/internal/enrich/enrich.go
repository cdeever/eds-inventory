// Package enrich fills in what the inventory knows of the owner's artists
// from MusicBrainz, once and in the background (ADR-0004): the names they go
// by, their studio albums in order, and, where an owned album is not found
// among those under its own name, the names its releases have gone by.
//
// Nothing a client asks waits on it. An artist asked about who has not been
// enriched is asked for next; the rest are worked through one at a time, at
// MusicBrainz's pace, and asked for again after Every.
package enrich

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/cdeever/eds-inventory/services/inventory/internal/catalog"
	"github.com/cdeever/eds-inventory/services/inventory/internal/musicbrainz"
	"github.com/cdeever/eds-inventory/services/inventory/internal/store"
)

// Enricher enriches the inventory's artists.
type Enricher struct {
	Store  *store.Store
	MB     *musicbrainz.Client
	Live   *catalog.Live
	Logger *slog.Logger

	// Every is how long a discography is believed: zero is thirty days. A
	// band's next album is rare, and a month late is no harm.
	Every time.Duration
	// Idle is how long to wait when there is nothing to do; zero is ten
	// minutes. Backoff is how long to wait after MusicBrainz fails; zero is
	// a minute.
	Idle, Backoff time.Duration
	// MaxTitles bounds how many albums' release names are asked for one
	// artist; zero is eight.
	MaxTitles int

	once  sync.Once
	asked chan string
}

// queue is the line of artists asked about, made once.
func (e *Enricher) queue() chan string {
	e.once.Do(func() { e.asked = make(chan string, 64) })
	return e.asked
}

// Ask puts an artist first in line, if they are not enriched yet. It never
// waits.
func (e *Enricher) Ask(artistID string) {
	select {
	case e.queue() <- artistID:
	default:
	}
}

// Run enriches until ctx ends.
func (e *Enricher) Run(ctx context.Context) {
	asked := e.queue()
	for ctx.Err() == nil {
		ref, ok, err := e.next(ctx)
		if err != nil {
			e.logger().Warn("could not find who to enrich", "error", err)
			e.sleep(ctx, e.backoff())
			continue
		}
		if !ok {
			select {
			case <-ctx.Done():
			case id := <-asked:
				e.askedFor(ctx, id)
			case <-time.After(e.idle()):
			}
			continue
		}
		if err := e.Artist(ctx, ref); err != nil && ctx.Err() == nil {
			e.logger().Warn("MusicBrainz could not be asked; trying again later", "artist", ref.Name, "error", err)
			e.sleep(ctx, e.backoff())
		}
	}
}

// next is who to enrich: one asked about first, then the longest waiting.
func (e *Enricher) next(ctx context.Context) (store.ArtistRef, bool, error) {
	select {
	case id := <-e.queue():
		if ref, ok := e.unenriched(ctx, id); ok {
			return ref, true, nil
		}
	default:
	}
	refs, err := e.Store.ToEnrich(ctx, time.Now().Add(-e.every()), 1)
	if err != nil || len(refs) == 0 {
		return store.ArtistRef{}, false, err
	}
	return refs[0], true, nil
}

func (e *Enricher) askedFor(ctx context.Context, id string) {
	if ref, ok := e.unenriched(ctx, id); ok {
		if err := e.Artist(ctx, ref); err != nil && ctx.Err() == nil {
			e.logger().Warn("MusicBrainz could not be asked; trying again later", "artist", ref.Name, "error", err)
		}
	}
}

// unenriched is the artist with the ID, if they have no discography yet.
func (e *Enricher) unenriched(ctx context.Context, id string) (store.ArtistRef, bool) {
	c, err := e.Live.Catalog(ctx)
	if err != nil {
		return store.ArtistRef{}, false
	}
	a := c.Artist(id)
	if a == nil || a.Discography != nil {
		return store.ArtistRef{}, false
	}
	return store.ArtistRef{ID: a.ID, Name: a.Name}, true
}

// Artist enriches one artist. An error is MusicBrainz not answering, and
// nothing is kept; "no such artist" and "no studio albums" are answers, and
// are kept as found=false, to be asked again after Every.
func (e *Enricher) Artist(ctx context.Context, ref store.ArtistRef) error {
	start := time.Now()
	mb, err := e.MB.Artist(ctx, ref.Name)
	if errors.Is(err, musicbrainz.ErrNotFound) {
		e.logger().Info("MusicBrainz knows no such artist", "artist", ref.Name)
		return e.Store.SaveDiscography(ctx, ref.ID, store.Discography{Found: false, Fetched: time.Now()}, nil)
	}
	if err != nil {
		return err
	}
	albums, err := e.MB.StudioAlbums(ctx, mb)
	if err != nil && !errors.Is(err, musicbrainz.ErrNotFound) {
		return err
	}
	d := store.Discography{MBID: mb.ID, Name: mb.Name, Found: len(albums) > 0, Fetched: time.Now()}
	for _, al := range albums {
		d.Albums = append(d.Albums, store.StudioAlbum{ReleaseGroup: al.ReleaseGroup, Title: al.Title, Released: al.Released})
	}
	if err := e.Store.SaveDiscography(ctx, ref.ID, d, mb.Aliases); err != nil {
		return err
	}
	titles, err := e.titles(ctx, ref.ID)
	e.logger().Info("artist enriched", "artist", ref.Name, "musicbrainz", mb.Name,
		"studio_albums", len(albums), "aliases", len(mb.Aliases), "titles_asked", titles,
		"calls", e.MB.Calls(), "ms", time.Since(start).Milliseconds())
	return err
}

// titles asks for the release names of the artist's studio albums, in
// order, while any album the owner has is not found among them, and no more
// than MaxTitles. It says how many were asked for.
func (e *Enricher) titles(ctx context.Context, artistID string) (int, error) {
	asked := 0
	for asked < e.maxTitles() {
		c, err := e.Live.Catalog(ctx)
		if err != nil {
			return asked, err
		}
		a := c.Artist(artistID)
		if a == nil || len(catalog.Unmatched(a)) == 0 {
			return asked, nil
		}
		var next *store.StudioAlbum
		for i := range a.Discography.Albums {
			sa := &a.Discography.Albums[i]
			if owned, _ := catalog.Owned(a, sa.Title); owned == nil && !sa.TitlesFetched {
				next = sa
				break
			}
		}
		if next == nil {
			return asked, nil
		}
		found, err := e.MB.Titles(ctx, musicbrainz.StudioAlbum{ReleaseGroup: next.ReleaseGroup, Title: next.Title})
		if err != nil {
			return asked, err
		}
		uses := map[string]int{}
		for _, t := range found {
			uses[t.Title] = t.Uses
		}
		if err := e.Store.SaveTitles(ctx, next.ReleaseGroup, uses, time.Now()); err != nil {
			return asked, err
		}
		asked++
	}
	return asked, nil
}

func (e *Enricher) sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

func (e *Enricher) every() time.Duration   { return or(e.Every, 30*24*time.Hour) }
func (e *Enricher) idle() time.Duration    { return or(e.Idle, 10*time.Minute) }
func (e *Enricher) backoff() time.Duration { return or(e.Backoff, time.Minute) }
func (e *Enricher) maxTitles() int {
	if e.MaxTitles > 0 {
		return e.MaxTitles
	}
	return 8
}

func or(d, def time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return def
}

func (e *Enricher) logger() *slog.Logger {
	if e.Logger != nil {
		return e.Logger
	}
	return slog.Default()
}
