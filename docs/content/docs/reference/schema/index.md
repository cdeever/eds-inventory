---
title: "Database Schema"
weight: 2
---

# Database Schema

The inventory's database, as `internal/store` creates it: SQLite, schema
version **1** (kept in `PRAGMA user_version`). Nothing but `internal/store`
reads or writes it
([ADR-0001](/docs/architecture/decisions/0001-go-services-and-a-sqlite-store/)).

This schema is being reworked; the changes agreed so far are on
[Proposed Schema](../schema-proposed/), and this page stays what runs until
they are built.

[![The inventory's tables: artists, albums, holdings and collections, and how they refer to one another.](schema.svg)](schema.svg)

**An album is the inventory's; a holding is a source's.** Each source that
holds an album has a holding of it, found again on every collection by the
key the source knows it by, so the album keeps its ID. A collection replaces
its source's holdings; an album no holding refers to is removed, and then
an artist with no albums.

The picture is drawn by GraphViz from `schema.dot`, beside this page. Change
it, this dictionary and `internal/store/store.go` together; a test fails if
a column is missing here.

## Data dictionary

### `artists`

Each artist with albums in the inventory. Two spellings that fold alike
("The Beatles", "Beatles") are one artist.

| Column | Type | Null | Meaning |
|---|---|---|---|
| `id` | TEXT | no | **Primary key.** The inventory's own ID: `ar_` and twelve lowercase hex digits, random, never reused. What clients hold on to |
| `name` | TEXT | no | The name as first collected, as a source spells it |
| `folded` | TEXT | no | **Unique.** The name as names are compared (`fold.Name`): lowercase, without accents, a leading "the" or punctuation, "&" as "and". How a second spelling finds the same artist |

### `albums`

Each album in the inventory, whatever holds it.

| Column | Type | Null | Meaning |
|---|---|---|---|
| `id` | TEXT | no | **Primary key.** The inventory's own ID: `al_` and twelve lowercase hex digits. Kept across collections |
| `artist_id` | TEXT | no | **Foreign key** to `artists.id`: the album's artist, the album artist where a source names one |
| `title` | TEXT | no | The title as the source spells it, so a player asked for it by name finds it |
| `year` | INTEGER | no | The year most of its tracks give; 0 when none does. Often a reissue's |
| `compilation` | INTEGER | no | 1 when the source marks it a compilation, else 0. A compilation is never an album asked for by its place |

### `holdings`

One source's holding of one album: that the source has it, and what the
source says of it.

| Column | Type | Null | Meaning |
|---|---|---|---|
| `source` | TEXT | no | **Primary key, with `key`.** The source, by name: `itunes` |
| `key` | TEXT | no | **Primary key, with `source`.** What the source knows the album by, the same on every collection. For iTunes, which gives albums no ID, the folded artist and title: `beatles\|abbey road` |
| `album_id` | TEXT | no | **Foreign key** to `albums.id`: the album held |
| `title` | TEXT | no | The title as this source spells it |
| `tracks` | INTEGER | no | How many of the album's tracks the source has |
| `collected` | TEXT | no | When the collection that last saw it ran, RFC 3339 in UTC. A holding whose time is not its source's latest collection is removed by it |

### `collections`

The last collection of each source.

| Column | Type | Null | Meaning |
|---|---|---|---|
| `source` | TEXT | no | **Primary key.** The source, by name |
| `collected` | TEXT | no | When its last collection ran, RFC 3339 in UTC |
| `albums` | INTEGER | no | How many albums that collection held |

### Indexes

| Index | On | For |
|---|---|---|
| `albums_by_artist` | `albums(artist_id)` | an artist's albums |
| `holdings_by_album` | `holdings(album_id)` | an album's holdings, and finding albums nothing holds |

### What is not kept

No paths, play counts, ratings or anything else a source knows of how its
music is listened to. Covers, originals and discographies are not kept yet;
they come with enrichment.
