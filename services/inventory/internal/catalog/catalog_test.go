package catalog

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cdeever/eds-inventory/services/inventory/internal/itunes"
	"github.com/cdeever/eds-inventory/services/inventory/internal/store"
)

var ctx = context.Background()

// collect puts albums in a fresh store, as the iTunes collector does.
func collect(t *testing.T, xml string) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "inventory.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	albums, err := itunes.Read(strings.NewReader(xml))
	if err != nil {
		t.Fatal(err)
	}
	var h []store.Holding
	for _, a := range albums {
		h = append(h, store.Holding{Key: a.Key, Artist: a.Artist, Title: a.Title, Year: a.Year, Tracks: a.Tracks, Compilation: a.Compilation})
	}
	if _, err := s.Collect(ctx, itunes.Source, h, time.Unix(1000, 0)); err != nil {
		t.Fatal(err)
	}
	return s
}

func fixture(t *testing.T) *Catalog {
	t.Helper()
	c, err := (&Live{Store: collect(t, itunes.FixtureXML())}).Catalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func artist(t *testing.T, c *Catalog, name string) *store.Artist {
	t.Helper()
	a, _ := c.FindArtist(name)
	if a == nil {
		t.Fatalf("no %q", name)
	}
	return a
}

// --- artists -------------------------------------------------------------------

func TestArtistsAreFoundAsTheyAreMisheard(t *testing.T) {
	c := fixture(t)
	for heard, want := range map[string]string{
		"Arrowsmith":     "Aerosmith", // as EdS's scribe wrote it, 2026-10-24
		"van halens":     "Van Halen",
		"the beatles":    "The Beatles",
		"Beatles":        "The Beatles",
		"Led Zepplin":    "Led Zeppelin",
		"LYNYRD SKYNYRD": "Lynyrd Skynyrd",
	} {
		a, s := c.FindArtist(heard)
		if a == nil || a.Name != want {
			t.Errorf("%q: %+v (%.2f), want %s", heard, a, s, want)
		}
	}
	for _, heard := range []string{"Ruth", "Leonard Skinner", "Nobody", ""} {
		if a, s := c.FindArtist(heard); a != nil {
			t.Errorf("%q: found %s (%.2f)", heard, a.Name, s)
		}
	}
}

func TestAGuestIsNotAnArtist(t *testing.T) {
	c := fixture(t)
	if a, _ := c.FindArtist("Santana"); a == nil || len(a.Albums) != 1 {
		t.Errorf("Santana: %+v", a)
	}
	if a, _ := c.FindArtist("Santana feat. Rob Thomas"); a != nil && a.Name != "Santana" {
		t.Errorf("a guest's name is not an artist: %+v", a)
	}
}

// --- albums --------------------------------------------------------------------

func TestAlbumsAreFoundByTheirNamesAsCatalogsWriteThem(t *testing.T) {
	c := fixture(t)
	for _, tc := range []struct{ artist, asked, want string }{
		{"Genesis", "And the Word Was…", "And The Word Was"},
		{"Lynyrd Skynyrd", "(pronounced 'lĕh-'nérd 'skin-'nérd)", "Pronounced leh-nerd skin-nerd"},
		{"The Beatles", "Abby Road", "Abbey Road"}, // misheard
		{"Led Zeppelin", "Presence", "Presence (Remastered)"},
		{"Van Halen", "Van Halen", "Van Halen"},
	} {
		got, s := FindAlbum(artist(t, c, tc.artist), tc.asked)
		if got == nil || got.Title != tc.want {
			t.Errorf("%s / %q: %+v (%.2f), want %q", tc.artist, tc.asked, got, s, tc.want)
		}
	}
}

// Another number is another album, however like it: never given for it.
func TestANumberMakesAnotherAlbum(t *testing.T) {
	c := fixture(t)
	if got, _ := FindAlbum(artist(t, c, "Led Zeppelin"), "Led Zeppelin"); got != nil {
		t.Errorf("Led Zeppelin was taken for %q", got.Title)
	}
	if got, _ := FindAlbum(artist(t, c, "Van Halen"), "Van Halen III"); got != nil {
		t.Errorf("Van Halen III was taken for %q", got.Title)
	}
}

// The first of a catalog's names the inventory has; a compilation never.
func TestTheFirstOwnedNameIsTheAlbum(t *testing.T) {
	c := fixture(t)
	got, by := Owned(artist(t, c, "Genesis"), "From Genesis to Revelation", "And the Word Was…", "In the Beginning")
	if got == nil || got.Title != "And The Word Was" || by != "And the Word Was…" {
		t.Errorf("Genesis: %+v by %q", got, by)
	}
	if got, _ := Owned(artist(t, c, "Various Artists"), "Now That's What I Call Music"); got != nil {
		t.Errorf("a compilation was an album by its place: %+v", got)
	}
}

func TestByYearLeavesOutWhatIsNotAnAlbum(t *testing.T) {
	a := ByYear(artist(t, fixture(t), "Aerosmith"))
	if len(a) != 1 || a[0].Title != "Aerosmith" {
		t.Errorf("%+v", a)
	}
	g := ByYear(artist(t, fixture(t), "Genesis"))
	if len(g) != 3 || g[0].Title != "Trespass" || g[2].Title != "Abacab" {
		t.Errorf("%+v", g)
	}
}

// --- live ----------------------------------------------------------------------

func TestTheCatalogIsReadAgainAfterACollection(t *testing.T) {
	s := collect(t, itunes.Library([]string{itunes.TrackXML(1, map[string]string{"Artist": "Rush", "Album": "2112"})}))
	l := &Live{Store: s}
	c, _ := l.Catalog(ctx)
	if a, _ := c.FindArtist("Rush"); a == nil {
		t.Fatal("no Rush")
	}
	s.Collect(ctx, itunes.Source, []store.Holding{{Key: "genesis|abacab", Artist: "Genesis", Title: "Abacab", Tracks: 8}}, time.Unix(2000, 0))
	c, _ = l.Catalog(ctx)
	if a, _ := c.FindArtist("Rush"); a != nil {
		t.Error("Rush is still there after a collection without it")
	}
	if a, _ := c.FindArtist("Genesis"); a == nil {
		t.Error("Genesis is not there after a collection with it")
	}
}
