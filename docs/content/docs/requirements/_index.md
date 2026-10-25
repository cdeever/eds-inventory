---
title: "Requirements"
weight: 5
---

# Requirements

What the inventory must do. This page is being written now, in
conversation; what is settled is stated plainly, and what is still open is
listed as a question.

## Settled

- **It holds the owner's music in every format**: vinyl, the digital
  libraries (iTunes, MusicBee), albums saved in streaming services (Spotify,
  Tidal), and other formats, CDs among them.
- **Each source stays the truth for what it holds.** Discogs remains where
  the vinyl is cataloged; the inventory keeps itself in step with it, and
  with each of the others.
- **Its keys are its own.** Every artist, album, edition, recording and
  copy has an ID the inventory gave it. The IDs other services use
  (MusicBrainz, Discogs, iTunes, Spotify, Tidal) are kept beside them as
  references, and are never a key.
- **An album, its editions, and the owner's copies are told apart.** An
  album is the work, known whether or not it is owned; an edition is one set
  of tracks as issued, on one medium; a copy is one holding of an edition,
  from one source. A rip is a copy of the edition it was ripped from, and
  says how it was encoded. A copy not yet identified stands as an edition of
  its own until it is. Two of the inventory's albums can never claim one
  outside ID. ([Proposed Schema](../reference/schema-proposed/) has the
  worked example.)
- **An album's date is its original release**, without exception: *Are You
  Experienced* is 1967, whatever a copy says.
- **Tracks are kept**, and each recording has its own artist, so a
  compilation of many artists is held as it is.
- **It knows whether an artist is a person or a group.**
- **It enriches, once.** The inventory itself asks the services that know
  about music for covers, originals and discographies, the first time it
  meets an album, and keeps what it is allowed to keep. A client asks only
  the inventory.
- **It is asked over HTTP**, with a versioned JSON contract (`"v": 1`) and an
  OpenAPI document.
- **It says which copy is best**: an album's copies ordered by quality,
  lossless before lossy, then resolution and bitrate. Which copy can be
  played right now is the client's to know. The owner's preference of medium
  comes before resolution (see Open).
- **The cover is the original's when owned**: the earliest edition the owner
  has, vinyl first when two came out the same year. Covers belong to
  editions.
- **It knows nothing about its clients.**
- **It runs in EdS's Deevnet IoTaaS tenant**, beside EdS's services: a
  tenant has no way in from another, so its clients must share it.
- **It is a monorepo** of independent services, a database, and the
  workflows that collect and enrich.
- **Go services, and SQLite to start**, with no cgo; Postgres if it is
  outgrown ([ADR-0001](/docs/architecture/decisions/0001-go-services-and-a-sqlite-store/)).
- **The first source is the iTunes library**, collected from its XML
  ([ADR-0002](/docs/architecture/decisions/0002-the-itunes-library-is-collected-from-its-xml/)).

## What its first client would ask

From [EdS's proposal](https://cdeever.github.io/eds/docs/architecture/proposals/music-inventory/):

- Which owned album a track is on, and in which format: the edition's
  medium, and how a copy is encoded.
- The cover for an album.
- The original album, among reissues, compilations and box sets.
- An artist's albums, in order, so that "their first album" can be counted.
- Whether the owner has an album, by the nearest artist and title.

## Open

- **Collecting.** How each source is read (an API, an export, a file on the
  media server), and how often, or on what signal.
- **Matching.** What "the same album" means is settled above; how a record
  in Discogs, a folder in MusicBee and a saved album in Spotify are found to
  be it, and what happens when the inventory is not sure, is not.
- **Enrichment.** Which services are asked for what, in what order, and how
  often an album is looked at again.
- **A MusicBrainz of its own.** MusicBrainz can be self-hosted: a mirror of
  its database, kept current by hourly replication. As the enricher's
  upstream in place of the paced API, it would end the waits and the
  refusals, and make identifying editions and gathering recordings
  practical, which would otherwise be thousands of calls. Its core data is
  CC0, so keeping it is allowed; the Cover Art Archive's pictures are not
  part of it. Open: which host has the disk (on the order of a hundred
  gigabytes) and the memory, since EdS's workload has neither; whether the
  inventory reaches the mirror, or is sent an extract of only the owner's
  artists; and whether the API stays as a fallback.
- **Identifying files by their sound.** AcoustID, MusicBrainz's companion,
  maps a Chromaprint fingerprint (`fpcalc`, open source, run locally) to
  MusicBrainz recordings. Fingerprinting each track of a rip would name its
  recordings by what they sound like, not what they are called, and eleven
  tracks of one release would identify the edition: an unidentified copy
  would become the 1985 CD on evidence. Only fingerprints would leave, never
  sound; AcoustID's server and data can be self-hosted too. Open: the
  fingerprinting has to run where the files are, on the media server, not
  the workload; and whether AcoustID is asked or kept beside a MusicBrainz
  of its own.
- **Naming the turntable locally.** EdS names a record from a few seconds
  heard anywhere in a track, today by Shazam through an unpublished endpoint
  that may stop working any day (EdS ADR-0021). Chromaprint is not made for
  that; landmark fingerprinting is, and there are open-source kinds
  (audfprint, Panako, Olaf, Dejavu). Indexed from the owner's own rips, a
  clip would be matched against the owner's own collection, edition
  included. Open: a record held only on vinyl, with no rip, is not in the
  index, so something else is still needed for it; whether a match holds
  across masterings; and where the parts live. The index is knowledge of
  music made from the owner's files, so it could be the inventory's, while
  matching a clip stays EdS's recognizer, asking it: EdS's proposal said the
  inventory hears nothing, and this would not change that.
- **Captured vinyl.** EdS proposes capturing every record played to FLAC,
  by itself, track by track
  ([Capturing Vinyl](https://cdeever.github.io/eds/docs/architecture/proposals/capturing-vinyl/)).
  To the inventory a capture is one more copy of the vinyl edition, from a
  source of its own, and a record is fully captured when that copy has
  every track of the edition. It would be asked one new question: what is
  still missing from an edition. Captures are what would fill the landmark
  index for the records held only on vinyl. Open: whether the files reach
  it through MusicBee's library, tagged with the edition, or are reported
  by EdS directly.
- **Vinyl or CD first.** For an album held as both a vinyl capture and a CD
  rip, which is the better copy is the owner's preference, not resolution:
  a 24-bit vinyl capture would otherwise always outrank a 16-bit CD. A
  preference of medium, for all albums and perhaps for one, would come
  before resolution in "best copy". Which way it goes is not decided.
- **What may be kept.** Discogs' terms ask that nothing of theirs be kept
  longer than needed, and EdS today holds one Discogs cover, in memory. A
  cache of Discogs' data and pictures may not be allowed, so covers may have
  to come from elsewhere, the Cover Art Archive first.
- **Covers.** Which cover is settled above; whether the inventory stores it
  or points to it, and at what sizes, is not.
- **The contract.** Beyond the best copy, which questions are asked, and
  what an answer that is not sure looks like.
- **Which services run where**, when EdS's one-Pi back end becomes
  several, and where the inventory's part of the tenant is declared.
- **Editing.** Whether the owner corrects the inventory directly, or only at
  the sources.
- **Formats with no service behind them**, such as CDs: how they are entered.
