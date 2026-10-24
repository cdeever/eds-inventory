// Package itunes reads the owner's iTunes library from the XML iTunes writes
// ("iTunes Library.xml"), and says which albums it holds. Carried over from
// EdS's internal/library (EdS ADR-0046), which read the same file.
//
// Only names, years and counts are read: no paths, plays or ratings.
package itunes

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/cdeever/eds-inventory/services/inventory/internal/fold"
)

// Source is the name the inventory knows this source by.
const Source = "itunes"

// Album is one album the library holds: Title exactly as iTunes spells it,
// so a player asked for it by name finds it.
type Album struct {
	// Key says which album this is in the library across collections: its
	// artist and title, folded. iTunes gives albums no ID of their own.
	Key         string
	Artist      string
	Title       string
	Year        int
	Tracks      int
	Compilation bool
}

// track is what is read of one track.
type track struct {
	artist, albumArtist, album string
	year                       int
	compilation, notMusic      bool
}

// Read reads the library's XML as it streams by, and returns its albums,
// sorted by key. The file is tens of megabytes; nothing but the tracks'
// names is held.
//
// An album's artist is its album artist where the library has one, else its
// track artist, so a guest ("Santana feat. Rob Thomas") does not become an
// artist of their own. Several spellings of one album are one album, spelled
// as the most tracks spell it. A video, a podcast or anything else that is
// not music is no one's album.
func Read(r io.Reader) ([]Album, error) {
	type found struct {
		artist      string
		years       map[int]int
		spellings   map[string]int
		tracks      int
		compilation bool
	}
	albums := map[string]*found{}

	err := eachTrack(r, func(t track) {
		if t.notMusic || strings.TrimSpace(t.album) == "" {
			return
		}
		artist := strings.TrimSpace(t.albumArtist)
		if artist == "" {
			artist = strings.TrimSpace(t.artist)
		}
		if artist == "" {
			return
		}
		k := Key(artist, t.album)
		f := albums[k]
		if f == nil {
			f = &found{artist: artist, years: map[int]int{}, spellings: map[string]int{}}
			albums[k] = f
		}
		f.tracks++
		f.compilation = f.compilation || t.compilation
		if t.year > 0 {
			f.years[t.year]++
		}
		f.spellings[t.album]++
	})
	if err != nil {
		return nil, err
	}

	out := make([]Album, 0, len(albums))
	for k, f := range albums {
		out = append(out, Album{
			Key:         k,
			Artist:      f.artist,
			Title:       most(f.spellings),
			Year:        mostYear(f.years),
			Tracks:      f.tracks,
			Compilation: f.compilation,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

// Key is an album's key: its artist and title, folded. Folded names hold
// letters, digits and single spaces only, so "|" cannot be in either.
func Key(artist, title string) string {
	return fold.Name(artist) + "|" + fold.Fold(title)
}

// most is the spelling the most tracks use; a tie goes to the first in
// order, so the same library always gives the same title.
func most(spellings map[string]int) string {
	title, n := "", -1
	for s, c := range spellings {
		if c > n || (c == n && s < title) {
			title, n = s, c
		}
	}
	return title
}

// mostYear is the year the most tracks give, the earliest on a tie, or 0.
func mostYear(years map[int]int) int {
	year, n := 0, 0
	for y, c := range years {
		if c > n || (c == n && y < year) {
			year, n = y, c
		}
	}
	return year
}

// eachTrack walks the plist's Tracks dictionary, handing each track over.
func eachTrack(r io.Reader, each func(track)) error {
	d := xml.NewDecoder(r)
	d.Strict = false
	depth := 0
	inTracks := false
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("itunes: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "dict":
				depth++
				// plist > dict(1) > key Tracks > dict(2) > key id > dict(3)
				if inTracks && depth == 3 {
					tr, err := readTrack(d)
					if err != nil {
						return err
					}
					depth-- // readTrack consumed the closing </dict>
					each(tr)
				}
			case "key":
				var k string
				if err := d.DecodeElement(&k, &t); err != nil {
					return fmt.Errorf("itunes: %w", err)
				}
				if depth == 1 {
					inTracks = k == "Tracks"
				}
			}
		case xml.EndElement:
			if t.Name.Local == "dict" {
				depth--
				if depth == 1 {
					inTracks = false // the Tracks dictionary has ended
				}
			}
		}
	}
}

// readTrack reads one track's dictionary, up to and including its </dict>.
func readTrack(d *xml.Decoder) (track, error) {
	var t track
	var key string
	for {
		tok, err := d.Token()
		if err != nil {
			return t, fmt.Errorf("itunes: %w", err)
		}
		switch e := tok.(type) {
		case xml.EndElement:
			if e.Name.Local == "dict" {
				return t, nil
			}
		case xml.StartElement:
			if e.Name.Local == "key" {
				if err := d.DecodeElement(&key, &e); err != nil {
					return t, fmt.Errorf("itunes: %w", err)
				}
				continue
			}
			var v string
			switch e.Name.Local {
			case "true", "false":
				if err := d.Skip(); err != nil {
					return t, err
				}
				v = e.Name.Local
			default:
				if err := d.DecodeElement(&v, &e); err != nil {
					return t, fmt.Errorf("itunes: %w", err)
				}
			}
			switch key {
			case "Artist":
				t.artist = v
			case "Album Artist":
				t.albumArtist = v
			case "Album":
				t.album = v
			case "Kind":
				t.notMusic = t.notMusic || strings.Contains(strings.ToLower(v), "video")
			case "Year":
				t.year, _ = strconv.Atoi(v)
			case "Compilation":
				t.compilation = v == "true"
			case "Podcast", "Movie", "TV Show", "Music Video", "Has Video", "iTunesU", "Audiobook":
				t.notMusic = t.notMusic || v == "true"
			}
		}
	}
}
