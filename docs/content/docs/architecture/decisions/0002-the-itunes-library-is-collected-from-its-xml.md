---
title: "ADR-0002: The iTunes Library Is Collected from Its XML"
weight: 2
---

# ADR-0002: The iTunes Library Is Collected from Its XML, and Asked for by Nearest Name

| | |
|---|---|
| **Status** | {{< adr-status "Accepted" >}} 2026-10-24 |
| **Date** | 2026-10-24 |
| **Scope** | The first source, the owner's iTunes library: how it is read, how its albums are kept, and the first version of the contract that answers for it. Not how the XML reaches the inventory, which waits on where the inventory runs. |
| **Takes over** | EdS ADR-0046, whose index of the library this replaces |

---

## Context

EdS's voiced matches spoken requests against an index of the owner's iTunes
library: `library.json`, made by EdS from the XML iTunes writes and copied
to voiced by hand (EdS ADR-0046). It asks four things of it: the artist
nearest a name, the album nearest a title, the first of several titles the
library has, and an artist's own albums by year. That index is an inventory
concern living in EdS.

iTunes for Windows will not list its library over the remote protocol EdS
uses (EdS ADR-0040). The XML it writes is the one complete list.

## Options considered

**The XML, read by a collector.** The same file EdS reads today, in the
same way: proven against the real library.

**Asking iTunes**, over DACP or anything else. It refuses its library to a
remote.

**MusicBee instead**, which also reads the files. A later source; it does
not replace iTunes, which is what the owner plays from.

## Decision

**The iTunes library is collected from its XML by `inventory collect
itunes FILE`, and asked for over HTTP, contract version 1.**

- **What is read**: each album's artist (the album artist where there is
  one), title, year, track count and whether it is a compilation. No paths,
  plays or ratings. Video, podcasts and the rest are not music.
- **One album, however it is spelled**: spellings that fold alike are one
  album, titled as most of its tracks spell it.
- **Its key** is the folded artist and title, since iTunes gives albums no ID.
  A collection finds each album it collected before by that key, so the
  inventory's own ID for it is kept.
- **The source is the truth**: a collection replaces what iTunes holds, and
  an album no longer in the library is gone.
- **The contract** (`/v1`): the nearest artist to a name; an artist and
  their albums; the nearest album to a title, or the first of several
  titles the inventory has; an artist's own albums by year. Not near enough
  is `null`, with the score reached. The thresholds and the matching are
  EdS's, carried over unchanged.

## Consequences

- **The XML is copied in by hand** until the inventory's place is settled:
  the same step EdS takes today, done once for both.
- **EdS's index can go** once voiced asks the inventory: `cmd/library`,
  `internal/library`'s index file, and the copying.
- **Only iTunes, so far.** An album is one source's album until matching
  across sources is built.

## What would settle it

- voiced asks the inventory for a spoken request, and plays what it answers.

**Settled, 2026-10-24** (EdS CHG-0041): EdS's voiced asked the inventory
for "Led Zeppelin's first album", and iTunes played the album it answered.
The same 1,912 artists and 3,665 albums as EdS's own index of the same XML.
