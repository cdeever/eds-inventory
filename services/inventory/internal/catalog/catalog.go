// Package catalog finds what is asked for in the inventory: the nearest
// artist to a name, the nearest of their albums to a title, the first of
// several titles the inventory has, an artist's own albums by year.
//
// The matching is carried over from EdS's internal/library (EdS ADR-0046),
// where its thresholds were measured against the owner's library and
// requests written down from speech. It searches a copy of the inventory
// held in memory, read again whenever a collection changes it: one owner's
// music is small, and a search is then no more than a loop.
package catalog

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/cdeever/eds-inventory/services/inventory/internal/fold"
	"github.com/cdeever/eds-inventory/services/inventory/internal/store"
)

// Close enough: an artist or an album this like the one asked for is taken
// for it. Measured on EdS's requests (2026-10-24): "Arrowsmith" is
// Aerosmith, "Abby Road" is Abbey Road; "Ruth" is not Rush.
const (
	ArtistLike = 0.90
	AlbumLike  = 0.90
)

// Catalog is the inventory, as it was when read.
type Catalog struct {
	Artists []store.Artist
}

// Artist is the artist with the ID, or nil.
func (c *Catalog) Artist(id string) *store.Artist {
	for i := range c.Artists {
		if c.Artists[i].ID == id {
			return &c.Artists[i]
		}
	}
	return nil
}

// FindArtist is the artist nearest the name, and how near, or nil when none
// is near enough. A name equal once folded is 1; otherwise spelling
// (Jaro-Winkler) and sound (a Soundex-like key) are weighed, which is what
// speech written down gets wrong.
func (c *Catalog) FindArtist(name string) (*store.Artist, float64) {
	want := fold.Name(name)
	if want == "" {
		return nil, 0
	}
	var best *store.Artist
	score := 0.0
	for i := range c.Artists {
		a := &c.Artists[i]
		s := artistScore(want, fold.Name(a.Name))
		// The same score: the artist with more albums is likelier the one.
		if s > score || (s == score && best != nil && len(a.Albums) > len(best.Albums)) {
			best, score = a, s
		}
	}
	if score < ArtistLike {
		return nil, score
	}
	return best, score
}

func artistScore(want, name string) float64 {
	if want == name {
		return 1
	}
	w, n := strings.ReplaceAll(want, " ", ""), strings.ReplaceAll(name, " ", "")
	if w == n {
		return 0.99 // "AC DC" and "ACDC"
	}
	s := JaroWinkler(w, n)
	// Sound counts only for a name long enough to have one, and spelled near
	// enough already: "Arrowsmith" is Aerosmith, but "Ruth" and Ratt are both
	// R3, and no more alike than that.
	if len(w) >= 6 && s >= 0.8 && soundKey(w) == soundKey(n) {
		s = max(s, 0.92)
	}
	return s
}

// FindAlbum is the artist's album nearest the title, and how near, or nil
// when none is near enough.
func FindAlbum(a *store.Artist, title string) (*store.Album, float64) {
	return find(a, title, false)
}

// Owned is the first of titles the artist has an album for, in order, and
// the title it was found by: an album's names as a catalog gives them, its
// main one first and its reissues' after. A compilation is not an album by
// its place, so it is never the answer.
func Owned(a *store.Artist, titles ...string) (*store.Album, string) {
	for _, t := range titles {
		if album, _ := find(a, t, true); album != nil {
			return album, t
		}
	}
	return nil, ""
}

func find(a *store.Artist, title string, studio bool) (*store.Album, float64) {
	want := fold.Title(title)
	if want == "" {
		return nil, 0
	}
	var best *store.Album
	score := 0.0
	for i := range a.Albums {
		al := &a.Albums[i]
		if studio && al.Compilation {
			continue
		}
		s := albumScore(want, fold.Title(al.Title))
		if s > score || (s == score && best != nil && al.Tracks > best.Tracks) {
			best, score = al, s
		}
	}
	if score < AlbumLike {
		return nil, score
	}
	return best, score
}

func albumScore(want, title string) float64 {
	if want == title {
		return 1
	}
	// A different number is a different album, however alike the rest:
	// "Led Zeppelin" is not "Led Zeppelin II".
	if numbers(want) != numbers(title) {
		return 0
	}
	return JaroWinkler(want, title)
}

var roman = map[string]bool{"ii": true, "iii": true, "iv": true, "vi": true, "vii": true, "viii": true, "ix": true}

// numbers are a folded title's numbers, in order: "led zeppelin ii" is "ii",
// "1984" is "1984".
func numbers(title string) string {
	var out []string
	for _, w := range strings.Fields(title) {
		if roman[w] || strings.Trim(w, "0123456789") == "" {
			out = append(out, w)
		}
	}
	return strings.Join(out, " ")
}

// notAlbum names what is not an album by its place, when there is no
// catalog's count to go by.
var notAlbum = regexp.MustCompile(`(?i)\b(greatest hits|best of|live|collection|anthology|essentials?|hits|singles)\b`)

// ByYear are the artist's own albums by the year the inventory gives them:
// without compilations and what is named as one, and those with no year
// last. A library's year is often a reissue's, so an outside catalog's
// order is the better one where it is known.
func ByYear(a *store.Artist) []store.Album {
	var out []store.Album
	for _, al := range a.Albums {
		if !al.Compilation && !notAlbum.MatchString(al.Title) {
			out = append(out, al)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		yi, yj := out[i].Year, out[j].Year
		if (yi == 0) != (yj == 0) {
			return yj == 0
		}
		return yi < yj
	})
	return out
}

// JaroWinkler is how alike two strings are, from 0 to 1, favoring a shared
// beginning: the measure spelling mistakes in names are usually taken by.
func JaroWinkler(a, b string) float64 {
	ra, rb := []rune(a), []rune(b)
	if len(ra) == 0 || len(rb) == 0 {
		return 0
	}
	window := max(len(ra), len(rb))/2 - 1
	window = max(window, 0)
	ma, mb := make([]bool, len(ra)), make([]bool, len(rb))
	matches := 0
	for i := range ra {
		lo, hi := max(0, i-window), min(len(rb)-1, i+window)
		for j := lo; j <= hi; j++ {
			if !mb[j] && ra[i] == rb[j] {
				ma[i], mb[j] = true, true
				matches++
				break
			}
		}
	}
	if matches == 0 {
		return 0
	}
	transpositions, k := 0, 0
	for i := range ra {
		if !ma[i] {
			continue
		}
		for !mb[k] {
			k++
		}
		if ra[i] != rb[k] {
			transpositions++
		}
		k++
	}
	m := float64(matches)
	jaro := (m/float64(len(ra)) + m/float64(len(rb)) + (m-float64(transpositions)/2)/m) / 3
	prefix := 0
	for prefix < min(4, len(ra), len(rb)) && ra[prefix] == rb[prefix] {
		prefix++
	}
	return jaro + float64(prefix)*0.1*(1-jaro)
}

// soundKey is a name as it sounds, roughly: its first letter, then its
// consonants by the groups Soundex puts them in, repeats run together.
// "arrowsmith" and "aerosmith" are both A6253.
func soundKey(s string) string {
	groups := map[rune]byte{
		'b': '1', 'f': '1', 'p': '1', 'v': '1',
		'c': '2', 'g': '2', 'j': '2', 'k': '2', 'q': '2', 's': '2', 'x': '2', 'z': '2',
		'd': '3', 't': '3', 'l': '4', 'm': '5', 'n': '5', 'r': '6',
	}
	var out []byte
	var last byte
	for i, r := range s {
		if i == 0 {
			out = append(out, byte(r))
			last = groups[r]
			continue
		}
		g, ok := groups[r]
		if !ok {
			if r != 'h' && r != 'w' {
				last = 0 // a vowel separates repeats
			}
			continue
		}
		if g != last {
			out = append(out, g)
		}
		last = g
	}
	return string(out)
}

// Live is the catalog of a store, read again when a collection changes it.
type Live struct {
	Store *store.Store

	mu      sync.Mutex
	version string
	cat     *Catalog
}

// Catalog is the store's catalog as it is now.
func (l *Live) Catalog(ctx context.Context) (*Catalog, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	v, err := l.Store.Version(ctx)
	if err != nil {
		return nil, err
	}
	if l.cat != nil && v == l.version {
		return l.cat, nil
	}
	artists, err := l.Store.Artists(ctx)
	if err != nil {
		return nil, err
	}
	l.cat, l.version = &Catalog{Artists: artists}, v
	return l.cat, nil
}
