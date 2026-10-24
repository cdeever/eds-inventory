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
- **One identity per album, its own.** The inventory gives each album an ID
  of its own, and keeps the IDs other services use for it (MusicBrainz,
  Discogs, iTunes, Spotify, Tidal) beside it as cross-references.
- **It enriches, once.** The inventory itself asks the services that know
  about music for covers, originals and discographies, the first time it
  meets an album, and keeps what it is allowed to keep. A client asks only
  the inventory.
- **It is asked over HTTP**, with a versioned JSON contract (`"v": 1`) and an
  OpenAPI document.
- **It knows nothing about its clients.**
- **It runs in EdS's Deevnet IoTaaS tenant**, beside EdS's services: a
  tenant has no way in from another, so its clients must share it.
- **It is a monorepo** of independent services, a database, and the
  workflows that collect and enrich.

## What its first client would ask

From [EdS's proposal](https://cdeever.github.io/eds/docs/architecture/proposals/music-inventory/):

- Which owned album a track is on, and in which format.
- The cover for an album.
- The original album, among reissues, compilations and box sets.
- An artist's albums, in order, so that "their first album" can be counted.
- Whether the owner has an album, by the nearest artist and title.

## Open

- **The database.** What kind; SQLite may be enough for one owner's
  collection.
- **Collecting.** How each source is read (an API, an export, a file on the
  media server), and how often, or on what signal.
- **Matching.** How a record in Discogs, a folder in MusicBee and a saved
  album in Spotify are known to be the same album, and what happens when the
  inventory is not sure.
- **Enrichment.** Which services are asked for what, in what order, and how
  often an album is looked at again.
- **What may be kept.** Discogs' terms ask that nothing of theirs be kept
  longer than needed, and EdS today holds one Discogs cover, in memory. A
  cache of Discogs' data and pictures may not be allowed, so covers may have
  to come from elsewhere, the Cover Art Archive first.
- **Covers.** Stored by the inventory, or pointed to; at what sizes.
- **The contract.** Which questions are asked, and what an answer that is not
  sure looks like.
- **Which services run where**, when EdS's one-Pi back end becomes
  several, and where the inventory's part of the tenant is declared.
- **Editing.** Whether the owner corrects the inventory directly, or only at
  the sources.
- **Formats with no service behind them**, such as CDs: how they are entered.
