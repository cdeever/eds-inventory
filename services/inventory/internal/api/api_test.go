package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/cdeever/eds-inventory/services/inventory/internal/catalog"
	"github.com/cdeever/eds-inventory/services/inventory/internal/itunes"
	"github.com/cdeever/eds-inventory/services/inventory/internal/store"
)

func server(t *testing.T) *httptest.Server {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "inventory.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	albums, err := itunes.Read(strings.NewReader(itunes.FixtureXML()))
	if err != nil {
		t.Fatal(err)
	}
	var h []store.Holding
	for _, a := range albums {
		h = append(h, store.Holding{Key: a.Key, Artist: a.Artist, Title: a.Title, Year: a.Year, Tracks: a.Tracks, Compilation: a.Compilation})
	}
	if _, err := s.Collect(context.Background(), itunes.Source, h, time.Unix(1000, 0)); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(Handler(&catalog.Live{Store: s}, slog.New(slog.NewTextHandler(io.Discard, nil))))
	t.Cleanup(srv.Close)
	return srv
}

func get(t *testing.T, srv *httptest.Server, path string, want int) map[string]any {
	t.Helper()
	resp, err := http.Get(srv.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != want {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("%s: %d, want %d: %s", path, resp.StatusCode, want, body)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	if out["v"] != float64(1) {
		t.Errorf("%s: v = %v", path, out["v"])
	}
	return out
}

func artistID(t *testing.T, srv *httptest.Server, name string) string {
	t.Helper()
	a := get(t, srv, "/v1/artists/find?name="+url.QueryEscape(name), 200)["artist"].(map[string]any)
	return a["id"].(string)
}

func TestAMisheardArtistIsFound(t *testing.T) {
	srv := server(t)
	got := get(t, srv, "/v1/artists/find?name=Arrowsmith", 200)
	a, _ := got["artist"].(map[string]any)
	if a == nil || a["name"] != "Aerosmith" || !regexp.MustCompile(`^ar_[0-9a-f]{12}$`).MatchString(a["id"].(string)) {
		t.Errorf("%v", got)
	}
}

// Not near enough is null, with the score: never the nearest artist.
func TestNoArtistNearEnoughIsNull(t *testing.T) {
	got := get(t, server(t), "/v1/artists/find?name=Ruth", 200)
	if got["artist"] != nil || got["score"].(float64) >= catalog.ArtistLike {
		t.Errorf("%v", got)
	}
}

func TestAnAlbumIsFoundByTitle(t *testing.T) {
	srv := server(t)
	id := artistID(t, srv, "The Beatles")
	got := get(t, srv, "/v1/artists/"+id+"/albums/find?title=Abby+Road", 200)
	al, _ := got["album"].(map[string]any)
	if al == nil || al["title"] != "Abbey Road" || al["tracks"] != float64(17) || al["sources"].([]any)[0] != "itunes" {
		t.Errorf("%v", got)
	}
	got = get(t, srv, "/v1/artists/"+artistID(t, srv, "Led Zeppelin")+"/albums/find?title=Led+Zeppelin", 200)
	if got["album"] != nil {
		t.Errorf("Led Zeppelin II was given for Led Zeppelin: %v", got)
	}
}

// Several titles: the first the inventory has, and which title found it.
func TestTheFirstOwnedTitleIsTheAlbum(t *testing.T) {
	srv := server(t)
	id := artistID(t, srv, "Genesis")
	got := get(t, srv, "/v1/artists/"+id+"/albums/find?title=From+Genesis+to+Revelation&title=And+the+Word+Was", 200)
	al, _ := got["album"].(map[string]any)
	if al == nil || al["title"] != "And The Word Was" || got["title"] != "And the Word Was" {
		t.Errorf("%v", got)
	}
	id = artistID(t, srv, "Various Artists")
	got = get(t, srv, "/v1/artists/"+id+"/albums/find?studio=true&title=Now+That's+What+I+Call+Music", 200)
	if got["album"] != nil || got["title"] != nil {
		t.Errorf("a compilation was given: %v", got)
	}
}

func TestAnArtistsAlbumsByYear(t *testing.T) {
	srv := server(t)
	got := get(t, srv, "/v1/artists/"+artistID(t, srv, "Genesis")+"/albums/by-year", 200)
	albums := got["albums"].([]any)
	if len(albums) != 3 || albums[0].(map[string]any)["title"] != "Trespass" {
		t.Errorf("%v", got)
	}
}

func TestQuestionsThatCannotBeAnswered(t *testing.T) {
	srv := server(t)
	get(t, srv, "/v1/artists/find", 400)
	get(t, srv, "/v1/artists/ar_000000000000", 404)
	get(t, srv, "/v1/artists/ar_000000000000/albums/by-year", 404)
	get(t, srv, "/v1/artists/"+artistID(t, srv, "Genesis")+"/albums/find", 400)
}

// Every route is in openapi.json, and nothing is there that is not a route.
func TestTheContractDescribesEveryRoute(t *testing.T) {
	var doc struct {
		Paths map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(openAPI, &doc); err != nil {
		t.Fatal(err)
	}
	described := map[string]bool{}
	for path, ops := range doc.Paths {
		for method := range ops {
			described[strings.ToUpper(method)+" "+path] = true
		}
	}
	for _, r := range Routes {
		if !described[r] {
			t.Errorf("%s is not in openapi.json", r)
		}
		delete(described, r)
	}
	for r := range described {
		t.Errorf("openapi.json describes %s, which is not a route", r)
	}
	srv := server(t)
	resp, err := http.Get(srv.URL + "/openapi.json")
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("%v %v", resp, err)
	}
	resp.Body.Close()
}
