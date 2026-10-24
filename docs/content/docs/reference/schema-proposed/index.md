---
title: "Proposed Schema"
weight: 3
---

# Proposed Schema

**Not built.** This is the database as it is being reworked, table by table,
in conversation with the owner (started 2026-10-24). What is agreed is in
color; what is still to review, or proposed but not agreed, is gray. When the
review is done, the schema is rebuilt as one clean version 1, the workload's
database is wiped, and the library collected again. Until then,
[Database Schema](../schema/) is what runs.

[![The proposed tables: artist, album, edition, track, recording, holding, holding_track and source, with reference tables for outside IDs; collection_run and the enrichment tables still gray.](schema-proposed.svg)](schema-proposed.svg)

## Principles

- **Our keys are ours.** Every table's primary key is the inventory's own
  ID (`ar_`, `al_`, `ed_`, `tr_`, `rc_`, `ho_` and twelve hex digits). An ID
  from a service the inventory does not control (MusicBrainz, Discogs,
  iTunes, Spotify) is kept as a reference, in a `…_ref` table, and is never
  a primary key.
- **Normalized, mostly.** A fact is kept once, on the row it belongs to.
- **Names are singular**: `artist`, `album`, `edition`.
- **Conventions**: a flag is named `is_…`; what is not known is `NULL`,
  never `0` or `''`; a time is RFC 3339 in UTC, in a column named `…_at`.

## The shape, by example

Phil Collins' *No Jacket Required*, as the owner holds it: the 1985 LP, the
2016 remastered LP (older Phil on the cover), the same remaster on orange
vinyl, the 1985 CD with an extra track, and two rips of that CD, MP3 in
iTunes and FLAC in MusicBee.

```
artist     Phil Collins (person)
└─ album   No Jacket Required, 1985, studio
   ├─ edition  1985 LP, vinyl, 10 tracks ─────────── holding  the LP (discogs)
   ├─ edition  2016 LP, vinyl, 10 tracks ─────────── holding  the LP (discogs)
   ├─ edition  2016 LP, vinyl, orange, 10 tracks ─── holding  the LP (discogs)
   └─ edition  1985 CD, 11 tracks ─┬──────────────── holding  the CD (discogs)
                                   ├──────────────── holding  iTunes, mp3 256
                                   └──────────────── holding  MusicBee, flac 16/44.1
```

- **An album is the work**: what a person names, what a discography counts,
  what the original release date belongs to. *Are You Experienced* and *Sgt.
  Pepper's* are 1967, without exception. An album exists whether or not it
  is owned, so "Genesis' third album" is counted against the inventory's own
  rows. An album is owned when it has a holding.
- **An edition is one set of tracks, as issued**: its medium, year, variant
  and track list. The 10-track LP and the 11-track CD are two editions; so
  are the black and the orange remaster, as Discogs counts them.
- **A holding is one copy**: from one source, with that source's key and
  title for it. A rip is a holding **of the edition it was ripped from**, and
  says what it is: `encoding`, `bitrate_kbps`, `sample_rate_hz`,
  `bit_depth` (all `NULL` for a physical copy).
- **A track is a place on an edition** (disc and number); **a recording is
  the song as recorded**, with its own artist, so "Sussudio" is one
  recording on four editions. The CD's extra track is a recording on the CD
  alone. A remaster is the same recording, as MusicBrainz counts it.
- **`holding_track`** says which of its edition's tracks a copy has (a rip
  of 9 of 11 has 9 rows) and the source's key for each file.
- **An album's artist is optional and a recording's is required**: a K-tel
  record is an album with no artist and a track list of many.

## Decisions so far

1. **An artist has a kind**: `person`, `group`, `orchestra`, `choir`,
   `character` or `other`, from MusicBrainz, which answers band or solo.
2. **Titles are kept once for each thing they name.** `album.title` is the
   album's name (MusicBrainz's once matched, the first source's until then);
   `holding.source_title` is what a source calls its copy, which is what a
   player must be asked for ("Toys in the Attic - 2013 Remaster").
3. **A copy not yet identified gets an edition of its own**, with
   `is_identified` 0 and the source's track list, as every album is today.
   Identifying it points the holding at the real edition and drops the
   stand-in; eleven tracks ending with "We Said Hello Goodbye" is the 1985
   CD.
4. **"Play the best version"** orders an album's holdings by quality:
   lossless before lossy, then resolution and bitrate. The inventory says
   which is best; EdS says which is reachable now, as it knows the house and
   the inventory does not.
5. **The cover is the original's when owned**: the earliest owned edition,
   vinyl first on a tie, since the sleeve is what the art was made for. With
   no edition owned outright, the cover comes from outside as today. How
   covers are kept is a decision still to make.
6. **A source is a table**, `source` (`id`, `last_collected_at`), and a
   holding's source is a foreign key to it. A collection's album count is
   gone; it is a count of holdings.
7. **An iTunes album renamed is a new copy**: iTunes gives albums no ID, so
   the key is the folded artist and title. Accepted; it costs a match.

## Views

**`album_by_date`**: the owner's albums, by artist, then original release;
one not matched yet comes last for its artist, never placed by a copy's
year.

**`discography_in_order`**: each artist's studio albums (`kind = 'studio'`),
numbered by release date, with whether each is owned. "The Nth album" starts
at `n = N`, plays the first owned from there, and names the next two for
`instead`.

```sql
SELECT al.artist_id, al.title, al.released,
       ROW_NUMBER() OVER (PARTITION BY al.artist_id
                          ORDER BY al.released, al.title) AS n,
       EXISTS (SELECT 1 FROM edition ed JOIN holding ho ON ho.edition_id = ed.id
               WHERE ed.album_id = al.id) AS is_owned
FROM album al
WHERE al.kind = 'studio';
```

A date known only to the year (`1967`) sorts before one known to the day
(`1967-05-12`); a tie falls to the title.

## Still open

- **The enrichment tables**: `artist_alias`; `album_alias` (was
  `release_titles`, the names an album's releases go by, to an album what an
  alias is to an artist); `discography`, which becomes enrichment status
  once its MusicBrainz ID moves to `artist_ref`; and the `enrichment`
  counter, which may be replaced by the latest times in the tables.
- **How enrichment fills the new shape**: an artist's studio albums made as
  album rows, an owned album matched to one by reference, two rows for one
  album merged; editions identified; tracks gathered into recordings.
- **`collection_run`**, a history of each collection (when, added, kept,
  removed): proposed, not agreed.
- **Two artists who fold alike are one**: a known limit for now.
- **A featured artist is kept as written** ("Santana feat. Rob Thomas")
  until credits are split.
- **Compilations by no one artist** are not in `album_by_date`; whether they
  need an order of their own.
- **What leaves the Mac grows**: track titles, numbers, lengths and file
  formats beside the albums, still with no paths or play counts.
