package itunes

import (
	"strings"
	"testing"
)

func read(t *testing.T) map[string]Album {
	t.Helper()
	albums, err := Read(strings.NewReader(FixtureXML()))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]Album{}
	for _, a := range albums {
		out[a.Artist+" / "+a.Title] = a
	}
	return out
}

func TestTheLibrarysMusicIsItsAlbums(t *testing.T) {
	got := read(t)
	if len(got) != 14 {
		t.Errorf("%d albums: %v", len(got), got)
	}
	for _, not := range []string{"Genesis / Live at Wembley", "Genesis / A Podcast"} {
		if _, ok := got[not]; ok {
			t.Errorf("%s is not music", not)
		}
	}
}

// Two spellings of one album are one album, the most tracks' spelling.
func TestAnAlbumSpeltTwoWaysIsOne(t *testing.T) {
	a, ok := read(t)["Lynyrd Skynyrd / Pronounced leh-nerd skin-nerd"]
	if !ok || a.Tracks != 13 || a.Year != 1973 {
		t.Errorf("%+v", a)
	}
}

func TestAnAlbumIsItsAlbumArtists(t *testing.T) {
	got := read(t)
	if _, ok := got["Santana / Supernatural"]; !ok {
		t.Errorf("Supernatural is Santana's: %v", got)
	}
	if a := got["Various Artists / Now That's What I Call Music"]; !a.Compilation {
		t.Errorf("a compilation: %+v", a)
	}
}

// The same library always gives the same keys: they are how a collection
// finds the album it collected before.
func TestKeysAreTheFoldedArtistAndTitle(t *testing.T) {
	if k := Key("The Beatles", "Abbey Road"); k != "beatles|abbey road" {
		t.Errorf("%q", k)
	}
	if Key("Lynyrd Skynyrd", "Pronounced Leh-Nerd Skin-Nerd") != Key("Lynyrd Skynyrd", "Pronounced leh-nerd skin-nerd") {
		t.Error("two spellings, two keys")
	}
}

func TestABrokenFileIsAnError(t *testing.T) {
	if _, err := Read(strings.NewReader("<plist><dict><key>Tracks</key><dict><key>1</key><dict>")); err == nil {
		t.Error("no error for a file cut short")
	}
}
