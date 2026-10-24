// Command inventory collects a source into the inventory, and finds things
// in it by hand.
//
//	inventory collect itunes "iTunes Library.xml"   (or albums.json)
//	inventory albums itunes "iTunes Library.xml" > albums.json
//	inventory find ARTIST [TITLE...]
//	inventory enrich ARTIST            (from MusicBrainz, now)
//	inventory nth ARTIST N
//
// albums writes only what is collected - each album's artist, title, year,
// track count and whether it is a compilation - so the library's XML, with
// its paths and play counts, need not leave the machine it is on. collect
// takes that file as it takes the XML.
//
// The database is INV_DB (default inventory.db, here).
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/cdeever/eds-inventory/services/inventory/internal/catalog"
	"github.com/cdeever/eds-inventory/services/inventory/internal/enrich"
	"github.com/cdeever/eds-inventory/services/inventory/internal/itunes"
	"github.com/cdeever/eds-inventory/services/inventory/internal/musicbrainz"
	"github.com/cdeever/eds-inventory/services/inventory/internal/store"
)

func main() {
	if len(os.Args) < 3 {
		usage()
	}
	if os.Args[1] == "albums" {
		if len(os.Args) != 4 || os.Args[2] != itunes.Source {
			usage()
		}
		albums, err := read(os.Args[3])
		check(err)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", " ")
		check(enc.Encode(albums))
		fmt.Fprintf(os.Stderr, "%s: %d albums\n", itunes.Source, len(albums))
		return
	}

	path := os.Getenv("INV_DB")
	if path == "" {
		path = "inventory.db"
	}
	s, err := store.Open(path)
	check(err)
	defer s.Close()
	ctx := context.Background()

	switch os.Args[1] {
	case "collect":
		if len(os.Args) != 4 || os.Args[2] != itunes.Source {
			usage()
		}
		albums, err := read(os.Args[3])
		check(err)
		h := make([]store.Holding, 0, len(albums))
		for _, a := range albums {
			h = append(h, store.Holding{Key: a.Key, Artist: a.Artist, Title: a.Title, Year: a.Year, Tracks: a.Tracks, Compilation: a.Compilation})
		}
		sum, err := s.Collect(ctx, itunes.Source, h, time.Now())
		check(err)
		fmt.Printf("%s: %d albums; %d new, %d kept, %d gone\n", itunes.Source, len(albums), sum.Added, sum.Kept, sum.Removed)

	case "find":
		c, err := (&catalog.Live{Store: s}).Catalog(ctx)
		check(err)
		a, score := c.FindArtist(os.Args[2])
		if a == nil {
			fmt.Printf("no artist like %q (%.2f)\n", os.Args[2], score)
			os.Exit(1)
		}
		fmt.Printf("%s  %q (%.2f), %d albums\n", a.ID, a.Name, score, len(a.Albums))
		titles := os.Args[3:]
		switch len(titles) {
		case 0:
			for _, al := range a.Albums {
				comp := ""
				if al.Compilation {
					comp = "  (compilation)"
				}
				fmt.Printf("  %s  %4d  %-50s %2d tracks  %v%s\n", al.ID, al.Year, al.Title, al.Tracks, al.Sources, comp)
			}
		case 1:
			al, s := catalog.FindAlbum(a, titles[0])
			if al == nil {
				fmt.Printf("no album like %q by %s (%.2f)\n", titles[0], a.Name, s)
				os.Exit(1)
			}
			fmt.Printf("  %s  %q (%.2f)\n", al.ID, al.Title, s)
		default:
			al, by := catalog.Owned(a, titles...)
			if al == nil {
				fmt.Printf("none of those by %s\n", a.Name)
				os.Exit(1)
			}
			fmt.Printf("  %s  %q, for %q\n", al.ID, al.Title, by)
		}

	case "enrich", "nth":
		live := &catalog.Live{Store: s}
		c, err := live.Catalog(ctx)
		check(err)
		a, score := c.FindArtist(os.Args[2])
		if a == nil {
			fmt.Printf("no artist like %q (%.2f)\n", os.Args[2], score)
			os.Exit(1)
		}
		if os.Args[1] == "enrich" {
			base := os.Getenv("INV_MUSICBRAINZ")
			if base == "" {
				base = "https://musicbrainz.org"
			}
			e := &enrich.Enricher{Store: s, Live: live,
				MB: &musicbrainz.Client{Base: base, UserAgent: "eds-inventory/dev ( https://github.com/cdeever/eds-inventory )"}}
			check(e.Artist(ctx, store.ArtistRef{ID: a.ID, Name: a.Name}))
			c, _ = live.Catalog(ctx)
			a = c.Artist(a.ID)
			if d := a.Discography; d != nil {
				fmt.Printf("%s: %d studio albums, %d aliases\n", d.Name, len(d.Albums), len(a.Aliases))
				assigned := catalog.Assign(a)
				for i, sa := range d.Albums {
					mark := ""
					if owned := assigned[sa.ReleaseGroup]; owned != nil {
						mark = "  owned as " + owned.Title
					}
					fmt.Printf("  %2d  %-10s %s%s\n", i+1, sa.Released, sa.Title, mark)
				}
			}
			return
		}
		if len(os.Args) != 4 {
			usage()
		}
		var n int
		_, err = fmt.Sscan(os.Args[3], &n)
		check(err)
		got := catalog.NthOwned(a, n, 6)
		fmt.Printf("%s, %d, by %s\n", a.Name, n, got.Order)
		if got.Counted != nil {
			fmt.Printf("  counted: %s (%s)\n", got.Counted.Title, got.Counted.Released)
		}
		if got.Album == nil {
			fmt.Println("  none owned")
			os.Exit(1)
		}
		fmt.Printf("  album:   %s\n", got.Album.Title)
		for _, al := range got.Instead {
			fmt.Printf("  instead: %s\n", al.Title)
		}

	default:
		usage()
	}
}

// read is a library's albums, from its XML or from what albums wrote.
func read(path string) ([]itunes.Album, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	for {
		b, err := r.Peek(1)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		switch b[0] {
		case ' ', '\t', '\r', '\n':
			r.ReadByte()
			continue
		case '[':
			var albums []itunes.Album
			if err := json.NewDecoder(r).Decode(&albums); err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			return albums, nil
		}
		return itunes.Read(r)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: inventory collect itunes "iTunes Library.xml" | albums.json
       inventory albums itunes "iTunes Library.xml" > albums.json
       inventory find ARTIST [TITLE...]
       inventory enrich ARTIST
       inventory nth ARTIST N
The database is $INV_DB, or inventory.db.`)
	os.Exit(2)
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "inventory:", err)
		os.Exit(1)
	}
}
