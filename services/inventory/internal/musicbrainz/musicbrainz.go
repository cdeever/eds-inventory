// Package musicbrainz asks MusicBrainz what the inventory enriches albums
// with: an artist and the names they go by, their studio albums in the order
// they were first released, and the names each album's releases have had.
//
// Carried over from EdS's internal/resolve (EdS ADR-0041, EdS ADR-0045),
// with its rules: a studio album is an album with no secondary type, with an
// official release, credited to the artist alone. MusicBrainz asks for one
// call a second and a User-Agent that says who is asking; both are kept.
package musicbrainz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cdeever/eds-inventory/services/inventory/internal/fold"
)

// ErrNotFound means MusicBrainz answered, and has no such thing.
var ErrNotFound = errors.New("musicbrainz: not found")

// Client asks MusicBrainz.
type Client struct {
	Base      string // https://musicbrainz.org
	HTTP      *http.Client
	UserAgent string // "eds-inventory/VERSION ( contact )"
	// Gap is the time between calls; zero is 1.5 s, inside MusicBrainz's
	// one a second.
	Gap time.Duration

	mu    sync.Mutex
	last  time.Time
	calls int
}

// Artist is an artist as MusicBrainz knows them.
type Artist struct {
	ID      string
	Name    string
	Aliases []string
}

// StudioAlbum is one album of a discography.
type StudioAlbum struct {
	// ReleaseGroup is MusicBrainz's album, whatever its releases are called.
	ReleaseGroup string
	Title        string
	Released     string // its first release, YYYY[-MM[-DD]]
}

// Title is a name an album's releases have gone by, and how many of them.
type Title struct {
	Title string
	Uses  int
}

// maxGroupPages bounds how much of a discography is read: a hundred release
// groups a page, and the busiest band has a few hundred, most of them live
// albums and compilations.
const maxGroupPages = 5

// Artist finds the artist whose name, or one of whose aliases, is the name
// asked for, folded alike: "Leonard Skinner" is an alias of Lynyrd Skynyrd
// (EdS, 2026-10-24). ErrNotFound when none is.
func (c *Client) Artist(ctx context.Context, name string) (Artist, error) {
	var found struct {
		Artists []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Aliases []struct {
				Name string `json:"name"`
			} `json:"aliases"`
		} `json:"artists"`
	}
	term := fmt.Sprintf(`artist:"%s" OR alias:"%s"`, quoted(name), quoted(name))
	if err := c.get(ctx, "/ws/2/artist/?fmt=json&limit=10&query="+url.QueryEscape(term), &found); err != nil {
		return Artist{}, err
	}
	want := same(name)
	for _, a := range found.Artists {
		if a.ID == "" {
			continue
		}
		named := same(a.Name) == want
		var aliases []string
		for _, al := range a.Aliases {
			named = named || same(al.Name) == want
			if n := typographic.Replace(strings.TrimSpace(al.Name)); n != "" {
				aliases = append(aliases, n)
			}
		}
		if named {
			return Artist{ID: a.ID, Name: typographic.Replace(a.Name), Aliases: aliases}, nil
		}
	}
	return Artist{}, fmt.Errorf("%w: no artist %q", ErrNotFound, name)
}

// StudioAlbums are an artist's studio albums, oldest first; an album with no
// date comes last. ErrNotFound when there are none.
func (c *Client) StudioAlbums(ctx context.Context, a Artist) ([]StudioAlbum, error) {
	var out []StudioAlbum
	for page, offset := 0, 0; page < maxGroupPages; page++ {
		var found struct {
			Count  int `json:"release-group-count"`
			Groups []struct {
				ID        string   `json:"id"`
				Title     string   `json:"title"`
				Primary   string   `json:"primary-type"`
				Secondary []string `json:"secondary-types"`
				First     string   `json:"first-release-date"`
				Credit    []struct {
					Name string `json:"name"`
				} `json:"artist-credit"`
			} `json:"release-groups"`
		}
		// website-default is MusicBrainz's own filter for release groups with
		// an official release: without it, a band's later years are bootlegs.
		path := fmt.Sprintf("/ws/2/release-group?artist=%s&type=album&release-group-status=website-default&inc=artist-credits&fmt=json&limit=100&offset=%d",
			url.QueryEscape(a.ID), offset)
		if err := c.get(ctx, path, &found); err != nil {
			return nil, err
		}
		for _, g := range found.Groups {
			names := make([]string, 0, len(g.Credit))
			for _, cr := range g.Credit {
				names = append(names, cr.Name)
			}
			// The artist alone: a collaboration is someone else's album too.
			if g.Primary != "Album" || len(g.Secondary) > 0 || same(strings.Join(names, " ")) != same(a.Name) {
				continue
			}
			out = append(out, StudioAlbum{ReleaseGroup: g.ID, Title: LibraryName(g.Title), Released: g.First})
		}
		offset += len(found.Groups)
		if len(found.Groups) == 0 || offset >= found.Count {
			break
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		x, y := out[i].Released, out[j].Released
		if x == "" || y == "" {
			return x != "" && y == ""
		}
		return x < y
	})
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: no studio album by %s", ErrNotFound, a.Name)
	}
	return out, nil
}

// Titles are the names an album's releases have gone by, other than its own,
// most used first: Genesis' first, From Genesis to Revelation, is in one
// owner's library as its reissue, And the Word Was (EdS ADR-0045).
func (c *Client) Titles(ctx context.Context, album StudioAlbum) ([]Title, error) {
	var found struct {
		Releases []struct {
			Title string `json:"title"`
		} `json:"releases"`
	}
	if err := c.get(ctx, "/ws/2/release?release-group="+url.QueryEscape(album.ReleaseGroup)+"&fmt=json&limit=100", &found); err != nil {
		return nil, err
	}
	uses := map[string]int{}
	var order []string
	for _, r := range found.Releases {
		t := LibraryName(r.Title)
		if t == "" || fold.Title(t) == fold.Title(album.Title) {
			continue
		}
		if uses[t] == 0 {
			order = append(order, t)
		}
		uses[t]++
	}
	sort.SliceStable(order, func(i, j int) bool { return uses[order[i]] > uses[order[j]] })
	out := make([]Title, 0, len(order))
	for _, t := range order {
		out = append(out, Title{Title: t, Uses: uses[t]})
	}
	return out, nil
}

// Calls is how many calls have been made since the last time it was asked.
func (c *Client) Calls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := c.calls
	c.calls = 0
	return n
}

// get is one call, kept to its pace, and once more after a pause when
// MusicBrainz answers 503, which is how it says slow down.
func (c *Client) get(ctx context.Context, path string, into any) error {
	for attempt := 0; ; attempt++ {
		if err := c.wait(ctx); err != nil {
			return err
		}
		status, err := c.fetch(ctx, path, into)
		if err == nil || attempt > 0 || status != http.StatusServiceUnavailable {
			return err
		}
		c.mu.Lock()
		c.last = time.Now().Add(time.Second)
		c.mu.Unlock()
	}
}

func (c *Client) fetch(ctx context.Context, path string, into any) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.Base, "/")+path, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "application/json")
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	c.mu.Lock()
	c.calls++
	c.mu.Unlock()
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("musicbrainz: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return resp.StatusCode, fmt.Errorf("musicbrainz: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, fmt.Errorf("musicbrainz: %s", resp.Status)
	}
	if err := json.Unmarshal(body, into); err != nil {
		return resp.StatusCode, fmt.Errorf("musicbrainz: unreadable answer: %w", err)
	}
	return resp.StatusCode, nil
}

func (c *Client) wait(ctx context.Context) error {
	gap := c.Gap
	if gap == 0 {
		gap = 1500 * time.Millisecond
	}
	c.mu.Lock()
	wait := time.Until(c.last.Add(gap))
	c.mu.Unlock()
	if wait > 0 {
		t := time.NewTimer(wait)
		defer t.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
		}
	}
	c.mu.Lock()
	c.last = time.Now()
	c.mu.Unlock()
	return nil
}

// same is a name as MusicBrainz's and a library's are compared: folded, and
// without spaces, so "AC/DC" and "acdc" are one.
func same(s string) string { return strings.ReplaceAll(fold.Name(s), " ", "") }

func quoted(s string) string { return strings.TrimSpace(strings.ReplaceAll(s, `"`, " ")) }

// typographic are the quotation marks MusicBrainz writes and a library
// mostly does not: "Guns N’ Roses" is "Guns N' Roses" in iTunes.
var typographic = strings.NewReplacer("’", "'", "‘", "'", "“", `"`, "”", `"`)

// LibraryName is a MusicBrainz title as a library is likelier to have it:
// straight quotes, and an untitled album's name without the brackets
// MusicBrainz puts around it ("[Led Zeppelin IV]").
func LibraryName(title string) string {
	title = typographic.Replace(strings.TrimSpace(title))
	if len(title) > 2 && strings.HasPrefix(title, "[") && strings.HasSuffix(title, "]") && strings.Count(title, "[") == 1 {
		title = strings.TrimSpace(title[1 : len(title)-1])
	}
	return title
}
