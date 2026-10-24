package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func open(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "inventory.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

var ctx = context.Background()

func albumIDs(t *testing.T, s *Store) map[string]string {
	t.Helper()
	artists, err := s.Artists(ctx)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, a := range artists {
		for _, al := range a.Albums {
			out[a.Name+" / "+al.Title] = al.ID
		}
	}
	return out
}

// An album collected again keeps its ID: the ID is the inventory's, and a
// client may hold on to it.
func TestAnAlbumKeepsItsIDAcrossCollections(t *testing.T) {
	s := open(t)
	h := []Holding{
		{Key: "aerosmith|aerosmith", Artist: "Aerosmith", Title: "Aerosmith", Year: 1973, Tracks: 9},
		{Key: "beatles|abbey road", Artist: "The Beatles", Title: "Abbey Road", Year: 1969, Tracks: 17},
	}
	sum, err := s.Collect(ctx, "itunes", h, time.Unix(1000, 0))
	if err != nil || sum.Added != 2 {
		t.Fatalf("%+v %v", sum, err)
	}
	before := albumIDs(t, s)
	h[1].Title = "Abbey Road (Remastered)" // the library renamed it
	sum, err = s.Collect(ctx, "itunes", h, time.Unix(2000, 0))
	if err != nil || sum.Kept != 2 || sum.Added != 0 {
		t.Fatalf("%+v %v", sum, err)
	}
	after := albumIDs(t, s)
	if after["Aerosmith / Aerosmith"] != before["Aerosmith / Aerosmith"] ||
		after["The Beatles / Abbey Road (Remastered)"] != before["The Beatles / Abbey Road"] {
		t.Errorf("IDs moved: %v then %v", before, after)
	}
}

// The source is the truth: what it no longer holds, the inventory no longer
// has, and an artist with nothing left goes too.
func TestWhatASourceNoLongerHoldsIsGone(t *testing.T) {
	s := open(t)
	if _, err := s.Collect(ctx, "itunes", []Holding{
		{Key: "a|x", Artist: "A", Title: "X", Tracks: 1},
		{Key: "b|y", Artist: "B", Title: "Y", Tracks: 1},
	}, time.Unix(1000, 0)); err != nil {
		t.Fatal(err)
	}
	sum, err := s.Collect(ctx, "itunes", []Holding{{Key: "a|x", Artist: "A", Title: "X", Tracks: 1}}, time.Unix(2000, 0))
	if err != nil || sum.Removed != 1 {
		t.Fatalf("%+v %v", sum, err)
	}
	artists, _ := s.Artists(ctx)
	if len(artists) != 1 || artists[0].Name != "A" {
		t.Errorf("%+v", artists)
	}
}

// One source's collection leaves another's alone.
func TestACollectionIsOneSources(t *testing.T) {
	s := open(t)
	s.Collect(ctx, "itunes", []Holding{{Key: "a|x", Artist: "A", Title: "X", Tracks: 1}}, time.Unix(1000, 0))
	s.Collect(ctx, "musicbee", []Holding{{Key: "b|y", Artist: "B", Title: "Y", Tracks: 2}}, time.Unix(1000, 0))
	s.Collect(ctx, "itunes", nil, time.Unix(2000, 0))
	artists, _ := s.Artists(ctx)
	if len(artists) != 1 || artists[0].Albums[0].Sources[0] != "musicbee" {
		t.Errorf("%+v", artists)
	}
}

func TestArtistsSpeltAlikeAreOne(t *testing.T) {
	s := open(t)
	s.Collect(ctx, "itunes", []Holding{
		{Key: "1", Artist: "The Beatles", Title: "Abbey Road", Tracks: 17},
		{Key: "2", Artist: "Beatles", Title: "Let It Be", Tracks: 12},
	}, time.Unix(1000, 0))
	artists, _ := s.Artists(ctx)
	if len(artists) != 1 || len(artists[0].Albums) != 2 || artists[0].Name != "The Beatles" {
		t.Errorf("%+v", artists)
	}
}

func TestTheVersionMovesWithEachCollection(t *testing.T) {
	s := open(t)
	v0, _ := s.Version(ctx)
	s.Collect(ctx, "itunes", nil, time.Unix(1000, 0))
	v1, _ := s.Version(ctx)
	s.Collect(ctx, "itunes", nil, time.Unix(2000, 0))
	v2, _ := s.Version(ctx)
	if v0 == v1 || v1 == v2 {
		t.Errorf("%q %q %q", v0, v1, v2)
	}
}

func TestADatabaseIsOpenedAgainAsItWasLeft(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	s.Collect(ctx, "itunes", []Holding{{Key: "a|x", Artist: "A", Title: "X", Tracks: 1}}, time.Unix(1000, 0))
	s.Close()
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	artists, _ := s.Artists(ctx)
	if len(artists) != 1 {
		t.Errorf("%+v", artists)
	}
}
