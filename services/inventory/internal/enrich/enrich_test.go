package enrich

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cdeever/eds-inventory/services/inventory/internal/catalog"
	"github.com/cdeever/eds-inventory/services/inventory/internal/itunes"
	"github.com/cdeever/eds-inventory/services/inventory/internal/musicbrainz"
	"github.com/cdeever/eds-inventory/services/inventory/internal/musicbrainz/mbtest"
	"github.com/cdeever/eds-inventory/services/inventory/internal/store"
)

var ctx = context.Background()

// enricher is the made-up library collected, and an Enricher asking the
// stand-in MusicBrainz.
func enricher(t *testing.T) (*Enricher, *mbtest.Server) {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "inventory.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	albums, _ := itunes.Read(strings.NewReader(itunes.FixtureXML()))
	var h []store.Holding
	for _, a := range albums {
		h = append(h, store.Holding{Key: a.Key, Artist: a.Artist, Title: a.Title, Year: a.Year, Tracks: a.Tracks, Compilation: a.Compilation})
	}
	if _, err := s.Collect(ctx, itunes.Source, h, time.Unix(1000, 0)); err != nil {
		t.Fatal(err)
	}
	mb := mbtest.New(t)
	return &Enricher{
		Store:  s,
		MB:     &musicbrainz.Client{Base: mb.URL, UserAgent: "test", Gap: time.Millisecond},
		Live:   &catalog.Live{Store: s},
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}, mb
}

// enrichAll works through everyone, as Run does, without waiting.
func enrichAll(t *testing.T, e *Enricher) {
	t.Helper()
	for {
		refs, err := e.Store.ToEnrich(ctx, time.Now().Add(-time.Hour), 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(refs) == 0 {
			return
		}
		if err := e.Artist(ctx, refs[0]); err != nil {
			t.Fatal(err)
		}
	}
}

func find(t *testing.T, e *Enricher, name string) *store.Artist {
	t.Helper()
	c, err := e.Live.Catalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := c.FindArtist(name)
	if a == nil {
		t.Fatalf("no %q", name)
	}
	return a
}

func titles(albums []store.Album) []string {
	var out []string
	for _, a := range albums {
		out = append(out, a.Title)
	}
	return out
}

// The evening's requests (EdS, 2026-10-24), answered from the inventory
// alone once it is enriched.
func TestTheNthAlbumOwnedIsCountedAsMusicBrainzCounts(t *testing.T) {
	e, _ := enricher(t)
	enrichAll(t, e)
	for _, c := range []struct {
		artist  string
		n       int
		album   string
		instead string
	}{
		// Owned under its reissue's name, found among its releases' names.
		// Never their 1983 album, Genesis, though a release of their first
		// has that name too: it is the album of its own name.
		{"Genesis", 1, "And The Word Was", "Trespass, Abacab, Genesis"},
		{"Genesis", 2, "Trespass", "Abacab, Genesis"},
		{"Genesis", -1, "Genesis", "Abacab, Trespass, And The Word Was"},
		// Heard as an alias MusicBrainz keeps.
		{"Leonard Skinner", 1, "Pronounced leh-nerd skin-nerd", "Second Helping"},
		// Not owned: the next that is.
		{"Led Zeppelin", 1, "Led Zeppelin II", "Presence (Remastered)"},
		// Never the greatest hits.
		{"Arrowsmith", 1, "Aerosmith", ""},
		{"Van Halen", 1, "Van Halen", "Van Halen II"},
	} {
		got := catalog.NthOwned(find(t, e, c.artist), c.n, 6)
		if got.Order != "musicbrainz" || got.Album == nil || got.Album.Title != c.album || strings.Join(titles(got.Instead), ", ") != c.instead {
			t.Errorf("%s %d: %+v; want %q then %q", c.artist, c.n, got, c.album, c.instead)
		}
	}
	if got := catalog.NthOwned(find(t, e, "Genesis"), 6, 6); got.Album != nil {
		t.Errorf("Genesis has five studio albums: %+v", got)
	}
}

// Release names are asked for only while an owned album is not found, and
// only of albums not owned under their own names.
func TestReleaseNamesAreAskedForOnlyWhenNeeded(t *testing.T) {
	e, mb := enricher(t)
	enrichAll(t, e)
	var releases []string
	for _, p := range mb.Asked() {
		if strings.HasPrefix(p, "/ws/2/release?") {
			releases = append(releases, p)
		}
	}
	// Genesis' first (owned as And the Word Was), Skynyrd's first (its
	// curly quotes fold to the library's), and Led Zeppelin's Presence is
	// found as Presence (Remastered) without asking.
	if len(releases) != 1 || !strings.Contains(releases[0], "release-group=g1") {
		t.Errorf("asked for %v", releases)
	}
}

// What MusicBrainz does not know is an answer, kept: the owner's albums by
// year are counted, and it is not asked again until Every.
func TestAnArtistMusicBrainzDoesNotKnowIsCountedByTheLibrary(t *testing.T) {
	e, _ := enricher(t)
	enrichAll(t, e)
	a := find(t, e, "Santana")
	if a.Discography == nil || a.Discography.Found {
		t.Fatalf("%+v", a.Discography)
	}
	got := catalog.NthOwned(a, 1, 6)
	if got.Order != "library" || got.Album == nil || got.Album.Title != "Supernatural" {
		t.Errorf("%+v", got)
	}
	if refs, _ := e.Store.ToEnrich(ctx, time.Now().Add(-time.Hour), 10); len(refs) != 0 {
		t.Errorf("asked again: %v", refs)
	}
}

// MusicBrainz failing keeps nothing: the artist is asked for again.
func TestAFailureKeepsNothing(t *testing.T) {
	e, mb := enricher(t)
	mb.Down = true
	refs, _ := e.Store.ToEnrich(ctx, time.Now(), 1)
	if err := e.Artist(ctx, refs[0]); err == nil {
		t.Fatal("no error with MusicBrainz down")
	}
	if again, _ := e.Store.ToEnrich(ctx, time.Now(), 1); len(again) != 1 || again[0].ID != refs[0].ID {
		t.Errorf("%v", again)
	}
}

// Various Artists, with nothing but a compilation, is never asked for.
func TestCompilationsAloneAreNotEnriched(t *testing.T) {
	e, _ := enricher(t)
	refs, _ := e.Store.ToEnrich(ctx, time.Now(), 100)
	for _, r := range refs {
		if r.Name == "Various Artists" {
			t.Error("Various Artists would be asked for")
		}
	}
}

// An artist asked about is put first, and Run enriches them.
func TestAnArtistAskedAboutIsEnrichedFirst(t *testing.T) {
	e, _ := enricher(t)
	e.Idle = time.Hour
	run, stop := context.WithCancel(ctx)
	defer stop()
	go e.Run(run)
	id := find(t, e, "Van Halen").ID
	e.Ask(id)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		c, _ := e.Live.Catalog(ctx)
		if a := c.Artist(id); a != nil && a.Discography != nil {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Error("Van Halen was not enriched")
}
