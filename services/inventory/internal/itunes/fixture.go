package itunes

import (
	"strconv"
	"strings"
)

// FixtureXML is a made-up library in the plist iTunes writes, for tests here
// and in the packages that store and search what it holds. It is the
// evening's albums as the owner's iTunes has them (2026-10-24): two
// spellings of one album, reissue names and years, a guest on an album
// artist's album, and a video, a podcast and a compilation that are not
// anyone's album. Real libraries are not committed.
func FixtureXML() string {
	var tracks []string
	id := 1
	add := func(n int, f map[string]string) {
		for i := 0; i < n; i++ {
			tracks = append(tracks, trackXML(id, f))
			id++
		}
	}
	add(9, map[string]string{"Artist": "Genesis", "Album": "And The Word Was", "Year": "1976", "Kind": "MPEG audio file"})
	add(6, map[string]string{"Artist": "Genesis", "Album": "Trespass", "Year": "1974", "Kind": "MPEG audio file"})
	add(8, map[string]string{"Artist": "Genesis", "Album": "Abacab", "Year": "1981", "Kind": "MPEG audio file"})
	add(8, map[string]string{"Artist": "Lynyrd Skynyrd", "Album": "Pronounced leh-nerd skin-nerd", "Year": "1973"})
	add(5, map[string]string{"Artist": "Lynyrd Skynyrd", "Album": "Pronounced Leh-Nerd Skin-Nerd", "Year": "1973"})
	add(8, map[string]string{"Artist": "Lynyrd Skynyrd", "Album": "Second Helping", "Year": "1974"})
	add(9, map[string]string{"Artist": "Aerosmith", "Album": "Aerosmith", "Year": "1973"})
	add(2, map[string]string{"Artist": "Aerosmith", "Album": "Aerosmith Greatest Hits", "Year": "1980"})
	add(9, map[string]string{"Artist": "Van Halen", "Album": "Van Halen II", "Year": "1979"})
	add(11, map[string]string{"Artist": "Van Halen", "Album": "Van Halen", "Year": "1978"})
	add(17, map[string]string{"Artist": "The Beatles", "Album": "Abbey Road", "Year": "1969"})
	add(9, map[string]string{"Artist": "Led Zeppelin", "Album": "Led Zeppelin II", "Year": "1969"})
	add(6, map[string]string{"Artist": "Led Zeppelin", "Album": "Presence (Remastered)", "Year": "1976"})
	add(3, map[string]string{"Artist": "Santana feat. Rob Thomas", "Album Artist": "Santana", "Album": "Supernatural", "Year": "1999"})
	add(12, map[string]string{"Artist": "Various", "Album Artist": "Various Artists", "Album": "Now That's What I Call Music", "Compilation": "true"})
	add(1, map[string]string{"Artist": "Genesis", "Album": "Live at Wembley", "Kind": "Purchased MPEG-4 video file"})
	add(1, map[string]string{"Artist": "Genesis", "Album": "A Podcast", "Podcast": "true"})
	return Library(tracks)
}

// Library wraps tracks, each from TrackXML, in the plist around them.
func Library(tracks []string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple Computer//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Major Version</key><integer>1</integer>
<key>Tracks</key><dict>` + strings.Join(tracks, "\n") + `</dict>
<key>Playlists</key><array><dict><key>Name</key><string>Library</string></dict></array>
</dict></plist>`
}

// TrackXML is one track of a made-up library.
func TrackXML(id int, fields map[string]string) string { return trackXML(id, fields) }

func trackXML(id int, fields map[string]string) string {
	var b strings.Builder
	n := strconv.Itoa(id)
	b.WriteString("<key>" + n + "</key><dict><key>Track ID</key><integer>" + n + "</integer>")
	for k, v := range fields {
		switch {
		case v == "true" || v == "false":
			b.WriteString("<key>" + k + "</key><" + v + "/>")
		case k == "Year":
			b.WriteString("<key>" + k + "</key><integer>" + v + "</integer>")
		default:
			b.WriteString("<key>" + k + "</key><string>" + v + "</string>")
		}
	}
	b.WriteString("</dict>")
	return b.String()
}
