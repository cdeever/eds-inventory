package musicbrainz

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cdeever/eds-inventory/services/inventory/internal/musicbrainz/mbtest"
)

func client(t *testing.T) (*Client, *mbtest.Server) {
	mb := mbtest.New(t)
	return &Client{Base: mb.URL, UserAgent: "eds-inventory-test", Gap: time.Millisecond}, mb
}

var ctx = context.Background()

func TestAnArtistIsFoundByAnAlias(t *testing.T) {
	c, _ := client(t)
	a, err := c.Artist(ctx, "Leonard Skinner")
	if err != nil || a.Name != "Lynyrd Skynyrd" || len(a.Aliases) != 2 {
		t.Errorf("%+v %v", a, err)
	}
	if _, err := c.Artist(ctx, "Nobody At All"); !errors.Is(err, ErrNotFound) {
		t.Errorf("%v", err)
	}
}

// Studio albums only, in the order they came out, as a library spells them.
func TestStudioAlbumsAreTheArtistsOwnInOrder(t *testing.T) {
	c, _ := client(t)
	a, _ := c.Artist(ctx, "Genesis")
	albums, err := c.StudioAlbums(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, al := range albums {
		titles = append(titles, al.Title)
	}
	if strings.Join(titles, ", ") != "From Genesis to Revelation, Trespass, Nursery Cryme, Abacab, Genesis" {
		t.Errorf("%v", titles)
	}
	s, _ := c.Artist(ctx, "Lynyrd Skynyrd")
	albums, _ = c.StudioAlbums(ctx, s)
	if albums[0].Title != "(pronounced 'lĕh-'nérd 'skin-'nérd)" {
		t.Errorf("straight quotes: %q", albums[0].Title)
	}
}

func TestTitlesAreAnAlbumsOtherNamesMostUsedFirst(t *testing.T) {
	c, _ := client(t)
	titles, err := c.Titles(ctx, StudioAlbum{ReleaseGroup: "g1", Title: "From Genesis to Revelation"})
	if err != nil || len(titles) != 3 || titles[0].Title != "And the Word Was…" {
		t.Errorf("%+v %v", titles, err)
	}
}

func TestBusyIsAskedOnceMoreThenAnError(t *testing.T) {
	c, mb := client(t)
	mb.Down = true
	if _, err := c.Artist(ctx, "Genesis"); err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("%v", err)
	}
	if n := len(mb.Asked()); n != 2 {
		t.Errorf("asked %d times", n)
	}
}

func TestLibraryNames(t *testing.T) {
	if LibraryName("[Led Zeppelin IV]") != "Led Zeppelin IV" || LibraryName("Guns N’ Roses") != "Guns N' Roses" {
		t.Error(LibraryName("[Led Zeppelin IV]"), LibraryName("Guns N’ Roses"))
	}
}
