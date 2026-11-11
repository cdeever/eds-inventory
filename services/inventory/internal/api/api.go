// Package api is the inventory's HTTP contract, version 1: what a client may
// ask, and how it is answered. Every answer is JSON and carries "v": 1. An
// answer the inventory is not sure of is null, with the score it reached,
// never the nearest thing it has: a client can act on a gap, but not on a
// wrong answer it cannot tell from a right one.
//
// openapi.json, beside this file, describes every route; a test holds the
// two together.
package api

import (
	"context"
	_ "embed"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/cdeever/eds-inventory/services/inventory/internal/catalog"
	"github.com/cdeever/eds-inventory/services/inventory/internal/store"
)

// Version is the contract's version, in every answer.
const Version = 1

//go:embed openapi.json
var openAPI []byte

// Catalogs gives the inventory as it is now.
type Catalogs interface {
	Catalog(ctx context.Context) (*catalog.Catalog, error)
}

// Asker puts an artist who is not enriched yet first in line; nil when
// nothing enriches.
type Asker interface {
	Ask(artistID string)
}

// MaxInstead bounds the albums after the Nth an answer carries.
const MaxInstead = 20

// Routes are the contract's routes, for the server and for the test that
// holds them to openapi.json.
var Routes = []string{
	"GET /healthz",
	"GET /openapi.json",
	"GET /v1/artists/find",
	"GET /v1/artists/{id}",
	"GET /v1/artists/{id}/albums/find",
	"GET /v1/artists/{id}/albums/by-year",
	"GET /v1/artists/{id}/albums/nth",
	"GET /v1/artists/{id}/discography",
	"GET /v1/albums/find",
}

// Handler serves the contract. asker may be nil.
func Handler(c Catalogs, asker Asker, log *slog.Logger) http.Handler {
	h := &handler{c: c, asker: asker, log: log}
	mux := http.NewServeMux()
	handlers := map[string]http.HandlerFunc{
		"GET /healthz":                        func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok\n")) },
		"GET /openapi.json":                   h.openAPI,
		"GET /v1/artists/find":                h.findArtist,
		"GET /v1/artists/{id}":                h.artist,
		"GET /v1/artists/{id}/albums/find":    h.findAlbum,
		"GET /v1/artists/{id}/albums/by-year": h.byYear,
		"GET /v1/artists/{id}/albums/nth":     h.nth,
		"GET /v1/artists/{id}/discography":    h.discography,
		"GET /v1/albums/find":                 h.findAlbums,
	}
	for _, r := range Routes {
		mux.HandleFunc(r, handlers[r])
	}
	return mux
}

type handler struct {
	c     Catalogs
	asker Asker
	log   *slog.Logger
}

// Artist is an artist as the contract gives one.
type Artist struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Albums []Album `json:"albums"`
}

// Album is an album as the contract gives one: Title as the source spells
// it, so a player asked for it by name finds it.
type Album struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Year        int      `json:"year,omitempty"`
	Tracks      int      `json:"tracks"`
	Compilation bool     `json:"compilation,omitempty"`
	Sources     []string `json:"sources"`
}

func artistOf(a *store.Artist) *Artist {
	if a == nil {
		return nil
	}
	out := &Artist{ID: a.ID, Name: a.Name, Albums: []Album{}}
	for i := range a.Albums {
		out.Albums = append(out.Albums, *albumOf(&a.Albums[i]))
	}
	return out
}

func albumOf(a *store.Album) *Album {
	if a == nil {
		return nil
	}
	return &Album{ID: a.ID, Title: a.Title, Year: a.Year, Tracks: a.Tracks, Compilation: a.Compilation, Sources: a.Sources}
}

func (h *handler) openAPI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write(openAPI)
}

// catalog is the inventory, or false when it cannot be read, which has been
// answered.
func (h *handler) catalog(w http.ResponseWriter, r *http.Request) (*catalog.Catalog, bool) {
	c, err := h.c.Catalog(r.Context())
	if err != nil {
		h.log.Error("the inventory could not be read", "error", err)
		fail(w, http.StatusServiceUnavailable, "the inventory could not be read")
		return nil, false
	}
	return c, true
}

// GET /v1/artists/find?name=NAME
func (h *handler) findArtist(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		fail(w, http.StatusBadRequest, "name is required")
		return
	}
	c, ok := h.catalog(w, r)
	if !ok {
		return
	}
	a, score := c.FindArtist(name)
	answer(w, map[string]any{"v": Version, "name": name, "score": score, "artist": artistOf(a)})
}

// GET /v1/artists/{id}
func (h *handler) artist(w http.ResponseWriter, r *http.Request) {
	c, ok := h.catalog(w, r)
	if !ok {
		return
	}
	a := c.Artist(r.PathValue("id"))
	if a == nil {
		fail(w, http.StatusNotFound, "no artist with that id")
		return
	}
	answer(w, map[string]any{"v": Version, "artist": artistOf(a)})
}

// GET /v1/artists/{id}/albums/find?title=T[&title=T2...][&studio=true]
//
// One title is the nearest album to it, with its score. Several titles, or
// studio=true, are an album's names as a catalog gives them, main one first:
// the answer is the first the inventory has, never a compilation, and says
// which title found it.
func (h *handler) findAlbum(w http.ResponseWriter, r *http.Request) {
	titles := r.URL.Query()["title"]
	if len(titles) == 0 {
		fail(w, http.StatusBadRequest, "title is required")
		return
	}
	c, ok := h.catalog(w, r)
	if !ok {
		return
	}
	a := c.Artist(r.PathValue("id"))
	if a == nil {
		fail(w, http.StatusNotFound, "no artist with that id")
		return
	}
	if len(titles) == 1 && r.URL.Query().Get("studio") != "true" {
		album, score := catalog.FindAlbum(a, titles[0])
		answer(w, map[string]any{"v": Version, "title": titles[0], "score": score, "album": albumOf(album)})
		return
	}
	album, by := catalog.Owned(a, titles...)
	var found any
	if by != "" {
		found = by
	}
	answer(w, map[string]any{"v": Version, "title": found, "album": albumOf(album)})
}

// GET /v1/albums/find?title=T
//
// The albums near enough the title, whoever they are by: each artist's
// nearest, nearest first. For an album asked for without its artist.
func (h *handler) findAlbums(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Query().Get("title")
	if title == "" {
		fail(w, http.StatusBadRequest, "title is required")
		return
	}
	c, ok := h.catalog(w, r)
	if !ok {
		return
	}
	type found struct {
		Artist ArtistRef `json:"artist"`
		Album  *Album    `json:"album"`
		Score  float64   `json:"score"`
	}
	albums := []found{}
	for _, m := range c.FindAlbums(title) {
		albums = append(albums, found{Artist: ArtistRef{ID: m.Artist.ID, Name: m.Artist.Name}, Album: albumOf(m.Album), Score: m.Score})
	}
	answer(w, map[string]any{"v": Version, "title": title, "albums": albums})
}

// ArtistRef names an artist without its albums.
type ArtistRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GET /v1/artists/{id}/albums/by-year
func (h *handler) byYear(w http.ResponseWriter, r *http.Request) {
	c, ok := h.catalog(w, r)
	if !ok {
		return
	}
	a := c.Artist(r.PathValue("id"))
	if a == nil {
		fail(w, http.StatusNotFound, "no artist with that id")
		return
	}
	albums := []Album{}
	for _, al := range catalog.ByYear(a) {
		albums = append(albums, *albumOf(&al))
	}
	answer(w, map[string]any{"v": Version, "albums": albums})
}

// StudioAlbum is a studio album as the contract gives one.
type StudioAlbum struct {
	Title        string   `json:"title"`
	Released     string   `json:"released,omitempty"`
	ReleaseGroup string   `json:"release_group"`
	OtherTitles  []string `json:"other_titles,omitempty"`
}

func studioOf(s *store.StudioAlbum) *StudioAlbum {
	if s == nil {
		return nil
	}
	return &StudioAlbum{Title: s.Title, Released: s.Released, ReleaseGroup: s.ReleaseGroup, OtherTitles: s.OtherTitles}
}

// artistFor is the artist the path names, or false when there is none, which
// has been answered.
func (h *handler) artistFor(w http.ResponseWriter, r *http.Request) (*store.Artist, bool) {
	c, ok := h.catalog(w, r)
	if !ok {
		return nil, false
	}
	a := c.Artist(r.PathValue("id"))
	if a == nil {
		fail(w, http.StatusNotFound, "no artist with that id")
		return nil, false
	}
	return a, true
}

// GET /v1/artists/{id}/albums/nth?n=N[&instead=K]
//
// The Nth album of the artist's that the owner has (1 the first, -1 the
// latest), counted among their studio albums as MusicBrainz orders them, or
// by the library's years when they are not enriched yet; then the owner's
// albums after it, up to K (default 6). An artist not enriched yet is put
// first in line, and the next answer may count differently.
func (h *handler) nth(w http.ResponseWriter, r *http.Request) {
	n, err := strconv.Atoi(r.URL.Query().Get("n"))
	if err != nil || n == 0 {
		fail(w, http.StatusBadRequest, "n is a whole number, not 0: 1 for the first, -1 for the latest")
		return
	}
	instead := 6
	if v := r.URL.Query().Get("instead"); v != "" {
		if instead, err = strconv.Atoi(v); err != nil || instead < 0 || instead > MaxInstead {
			fail(w, http.StatusBadRequest, "instead is from 0 to 20")
			return
		}
	}
	a, ok := h.artistFor(w, r)
	if !ok {
		return
	}
	if a.Discography == nil && h.asker != nil {
		h.asker.Ask(a.ID)
	}
	got := catalog.NthOwned(a, n, instead)
	rest := []Album{}
	for i := range got.Instead {
		rest = append(rest, *albumOf(&got.Instead[i]))
	}
	answer(w, map[string]any{"v": Version, "n": n, "order": got.Order, "counted": studioOf(got.Counted), "album": albumOf(got.Album), "instead": rest})
}

// GET /v1/artists/{id}/discography
//
// The artist's studio albums as MusicBrainz counts them, each with the
// owner's album it is found as, or null. The discography is null when the
// artist is not enriched yet.
func (h *handler) discography(w http.ResponseWriter, r *http.Request) {
	a, ok := h.artistFor(w, r)
	if !ok {
		return
	}
	if a.Discography == nil {
		if h.asker != nil {
			h.asker.Ask(a.ID)
		}
		answer(w, map[string]any{"v": Version, "artist": map[string]string{"id": a.ID, "name": a.Name}, "discography": nil})
		return
	}
	d := a.Discography
	owned := catalog.Assign(a)
	albums := []map[string]any{}
	for i := range d.Albums {
		sa := &d.Albums[i]
		albums = append(albums, map[string]any{"studio_album": studioOf(sa), "owned": albumOf(owned[sa.ReleaseGroup])})
	}
	answer(w, map[string]any{"v": Version, "artist": map[string]string{"id": a.ID, "name": a.Name},
		"discography": map[string]any{"name": d.Name, "found": d.Found, "fetched": d.Fetched.UTC().Format(time.RFC3339), "aliases": nonNil(a.Aliases), "albums": albums}})
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func answer(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"v": Version, "error": msg})
}
