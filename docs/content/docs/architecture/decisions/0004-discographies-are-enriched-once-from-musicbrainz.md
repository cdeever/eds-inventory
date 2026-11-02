---
title: "ADR-0004: Discographies Are Enriched Once from MusicBrainz"
weight: 4
---

# ADR-0004: Discographies Are Enriched Once from MusicBrainz, in the Background, and Kept

| | |
|---|---|
| **Status** | {{< adr-status "Proposed" >}} — until EdS's voiced asks for an album by its place and no request waits on MusicBrainz |
| **Date** | 2026-10-24 |
| **Scope** | How an artist's studio albums, the names they go by, and their albums' other names come into the inventory, and how "their Nth album" is answered from it. Not covers, which are enrichment too and a later decision. |
| **Takes over** | EdS ADR-0041 and EdS ADR-0045, whose counting and fallback this carries over |

---

## Context

"Play Genesis' first album" needs the order an artist's albums came out
in, and a library cannot give it: its years are often a reissue's, and its
first album may be under a reissue's name (*And The Word Was*, for *From
Genesis to Revelation*). EdS's voiced asks MusicBrainz while the request
waits (EdS ADR-0041), and on 2026-10-24 that took 13 and 17 seconds, with
half the calls failing (EdS CHG-0041).

## Options considered

**Enrich in the background, once, and keep it.** The inventory asks
MusicBrainz for each artist the owner has, at MusicBrainz's pace, and
answers from what it kept.

**Ask when asked**, as voiced does: always current, and always as slow and
as unreliable as MusicBrainz is at that moment.

**Order by the library's years.** No outside service, and wrong wherever a
reissue's year stands in for the album's.

## Decision

**inventoryd enriches the owner's artists from MusicBrainz in the
background, keeps what it finds, and answers "their Nth album" from the
inventory alone.**

- **What is kept**, for each artist with an album that is not a
  compilation: the artist as MusicBrainz names them and the aliases it keeps
  ("Leonard Skinner" is Lynyrd Skynyrd); their studio albums, oldest first,
  by EdS's rules (an album with no secondary type, an official release,
  credited to the artist alone); and, only while one of the owner's albums
  is not found among them under its own name, the names their releases have
  gone by, at most eight albums an artist.
- **One owned album is one studio album.** Each studio album's own title is
  looked for first; another name is used only where its own is not found,
  and never a name that is another studio album's own: a release of Genesis'
  first is titled "Genesis", which is their 1983 album.
- **"No such artist" and "no studio albums" are answers**, kept, and the
  owner's albums are then counted by year. A failure to ask is not an
  answer: nothing is kept, and the artist is asked for again after a pause.
- **Asked first**: an artist a client asks about who is not enriched yet.
  Then the rest, never-asked first, one at a time, a call each 1.5 s, and
  each asked again after thirty days.
- **The contract** gains `/v1/artists/{id}/albums/nth` (the Nth owned, the
  album counted, and the owner's albums after it) and
  `/v1/artists/{id}/discography`. An artist is found by their aliases too.
- **`INV_MUSICBRAINZ=off`** enriches nothing.

## Consequences

- **The first pass over a library takes hours**: some 1,900 artists at two
  or three calls each, a call each 1.5 s. Until an artist is enriched, their
  Nth album is counted by the library's years, and the answer says so
  (`"order": "library"`).
- **A discography is up to a month old.** A new album is found when the
  artist is asked for again.
- **MusicBrainz is asked from the workload**, with a User-Agent naming
  eds-inventory, and only from there.
- **EdS's voiced can stop asking MusicBrainz**: its discographies, its
  alias lookup and its kept file go once it asks the inventory.

## What would settle it

- EdS's voiced asks the inventory for "their first album" and plays the
  answer in under a second.
