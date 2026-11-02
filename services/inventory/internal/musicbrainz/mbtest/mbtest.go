// Package mbtest is a stand-in for MusicBrainz, answering the three calls
// the inventory makes from a fixture: the artists, studio albums and release
// names that went wrong for EdS on 2026-10-24, and a live album and a
// compilation that are not studio albums.
package mbtest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/cdeever/eds-inventory/services/inventory/internal/fold"
)

type group struct {
	id, title, primary, first string
	secondary                 []string
	releases                  []string
}

type artist struct {
	id, name string
	aliases  []string
	groups   []group
}

var fixture = []artist{
	{id: "mb-genesis", name: "Genesis", groups: []group{
		{id: "g1", title: "From Genesis to Revelation", primary: "Album", first: "1969-03-07",
			releases: []string{"From Genesis to Revelation", "From Genesis to Revelation", "And the Word Was…", "In the Beginning", "Genesis"}},
		{id: "g2", title: "Trespass", primary: "Album", first: "1970-10-23"},
		{id: "g3", title: "Nursery Cryme", primary: "Album", first: "1971-11-12"},
		{id: "g4", title: "Abacab", primary: "Album", first: "1981-09-18"},
		{id: "g7", title: "Genesis", primary: "Album", first: "1983-10-03"},
		{id: "g5", title: "Seconds Out", primary: "Album", first: "1977-10-14", secondary: []string{"Live"}},
		{id: "g6", title: "Turn It On Again: The Hits", primary: "Album", first: "1999-10-25", secondary: []string{"Compilation"}},
	}},
	{id: "mb-skynyrd", name: "Lynyrd Skynyrd", aliases: []string{"Leonard Skinner", "Skynyrd"}, groups: []group{
		{id: "s1", title: "(pronounced ’lĕh-’nérd ’skin-’nérd)", primary: "Album", first: "1973-08-13"},
		{id: "s2", title: "Second Helping", primary: "Album", first: "1974-04-15"},
	}},
	{id: "mb-zeppelin", name: "Led Zeppelin", groups: []group{
		{id: "z1", title: "Led Zeppelin", primary: "Album", first: "1969-01-12"},
		{id: "z2", title: "Led Zeppelin II", primary: "Album", first: "1969-10-22"},
		{id: "z3", title: "Led Zeppelin III", primary: "Album", first: "1970-10-05"},
		{id: "z4", title: "Presence", primary: "Album", first: "1976-03-31"},
	}},
	{id: "mb-aerosmith", name: "Aerosmith", groups: []group{
		{id: "a1", title: "Aerosmith", primary: "Album", first: "1973-01-05"},
		{id: "a2", title: "Get Your Wings", primary: "Album", first: "1974-03-01"},
		{id: "a3", title: "Toys in the Attic", primary: "Album", first: "1975-04-08"},
		{id: "a4", title: "Greatest Hits", primary: "Album", first: "1980-10-11", secondary: []string{"Compilation"}},
	}},
	{id: "mb-vanhalen", name: "Van Halen", groups: []group{
		{id: "v1", title: "Van Halen", primary: "Album", first: "1978-02-10"},
		{id: "v2", title: "Van Halen II", primary: "Album", first: "1979-03-23"},
		{id: "v3", title: "Women and Children First", primary: "Album", first: "1980-03-26"},
	}},
	{id: "mb-beatles", name: "The Beatles", groups: []group{
		{id: "b1", title: "Abbey Road", primary: "Album", first: "1969-09-26"},
	}},
}

// Server is the stand-in, with what was asked of it.
type Server struct {
	*httptest.Server
	mu    sync.Mutex
	paths []string
	// Down makes every call answer 503.
	Down bool
}

// Asked is every path asked, in order.
func (s *Server) Asked() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.paths...)
}

// New starts the stand-in for the test.
func New(t *testing.T) *Server {
	t.Helper()
	s := &Server{}
	s.Server = httptest.NewServer(http.HandlerFunc(s.serve))
	t.Cleanup(s.Close)
	return s
}

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.paths = append(s.paths, r.URL.RequestURI())
	down := s.Down
	s.mu.Unlock()
	if down {
		http.Error(w, "slow down", http.StatusServiceUnavailable)
		return
	}
	q := r.URL.Query()
	var out any
	switch r.URL.Path {
	case "/ws/2/artist/":
		var found []map[string]any
		query := fold.Name(q.Get("query"))
		for _, a := range fixture {
			hit := strings.Contains(query, fold.Name(a.name))
			var aliases []map[string]string
			for _, al := range a.aliases {
				hit = hit || strings.Contains(query, fold.Name(al))
				aliases = append(aliases, map[string]string{"name": al})
			}
			if hit {
				found = append(found, map[string]any{"id": a.id, "name": a.name, "aliases": aliases})
			}
		}
		out = map[string]any{"artists": found}
	case "/ws/2/release-group":
		var groups []map[string]any
		for _, a := range fixture {
			if a.id != q.Get("artist") {
				continue
			}
			for _, g := range a.groups {
				groups = append(groups, map[string]any{"id": g.id, "title": g.title, "primary-type": g.primary,
					"secondary-types": g.secondary, "first-release-date": g.first,
					"artist-credit": []map[string]string{{"name": a.name}}})
			}
		}
		out = map[string]any{"release-group-count": len(groups), "release-groups": groups}
	case "/ws/2/release":
		var releases []map[string]string
		for _, a := range fixture {
			for _, g := range a.groups {
				if g.id == q.Get("release-group") {
					titles := g.releases
					if len(titles) == 0 {
						titles = []string{g.title}
					}
					for _, t := range titles {
						releases = append(releases, map[string]string{"title": t})
					}
				}
			}
		}
		out = map[string]any{"releases": releases}
	default:
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}
