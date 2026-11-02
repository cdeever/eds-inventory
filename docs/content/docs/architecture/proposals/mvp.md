---
title: "The MVP: the Turntable First"
weight: 1
---

# Proposal: The MVP, the Turntable First

| | |
|---|---|
| **Status** | Draft, under review: not decided and not built |
| **Date** | 2026-11-02 |
| **Where** | eds-inventory's `inventoryd`, on EdS's services workload; and EdS's `nowplayd`, which asks it |
| **Would extend** | [ADR-0002](/docs/architecture/decisions/0002-the-itunes-library-is-collected-from-its-xml/), [ADR-0004](/docs/architecture/decisions/0004-discographies-are-enriched-once-from-musicbrainz/), the [Proposed Schema](/docs/reference/schema-proposed/); in EdS, [EdS ADR-0022](https://cdeever.github.io/eds/docs/architecture/decisions/0022-the-owners-collection-names-the-record/) and [EdS ADR-0038](https://cdeever.github.io/eds/docs/architecture/decisions/0038-the-turntable-names-the-original-album-by-what-is-on-it/) |

The inventory is under way, but nothing says where its first useful version
stops. This proposal sets that line. The MVP is the inventory rebuilt on
its new schema, holding the owner's vinyl from Discogs beside the iTunes
library, and answering one new question for EdS: what a record heard on the
turntable is, and which sleeve goes with it. Everything else waits for v2
or later.

## Why

What runs today ([CHG-0001](/docs/changes/2026/0001-inventoryd-on-eds-workload/))
is one source, iTunes, in a flat schema, answering voiced's questions about
artists and their albums. It works, and it is not yet what the inventory is
for: one album, in every format, known once.

The [requirements](/docs/requirements/) have grown to cover every source,
fingerprints, a MusicBrainz of its own and captured vinyl, and the list of
open questions is longer than the list of settled ones. Without a line, the
next thing built is whichever is most interesting.

The turntable is where EdS is wrong most visibly today. The
[cases it must get right](/docs/requirements/#cases-it-must-get-right) are
both records on the platter: Journey's *Infinity*, named as a live album
because Deezer never lists "Wheel in the Sky" on it, and Kansas'
*Point of Know Return*, named correctly and shown with a live album's
cover. In both, the owner holds the record, and Discogs knows exactly which
pressing. The inventory can answer from that; a catalog asked by name
cannot.

## What it is

- **The schema is rebuilt first.** The [Proposed Schema](/docs/reference/schema-proposed/)
  is finished (its gray tables decided), built as one clean version 1, the
  workload's database wiped, and the library collected again. Album,
  edition and holding are what make a vinyl pressing and an iTunes rip the
  same album; everything after builds on them.
- **Discogs is collected beside iTunes.** The owner's Discogs collection
  comes in as holdings of editions, each edition a Discogs release with its
  medium, year and track list. A Discogs release is already an edition, so
  the vinyl arrives identified. A release's facts are CC0 and kept; the
  collection itself is Discogs' Restricted Data, so it is collected again
  at least every six hours, as Discogs' terms ask
  ([Requirements](/docs/requirements/)).
- **Editions are matched to albums.** A Discogs edition and an iTunes copy
  of the same work are found to be one album, through MusicBrainz, as
  enrichment already places owned albums among an artist's studio albums.
- **The inventory does the fuzzy matching.** Given what the recognizer
  heard (an artist, a track, and the album it thinks the track is on, each
  spelled however it was spelled), the inventory finds the owned edition.
  Matching names against what the owner holds is knowledge of music, and it
  is already the inventory's (`internal/fold`, `catalog.Assign`, from
  EdS ADR-0046). Deciding what is playing, and hearing the clip, stay EdS's.
- **Any owned edition can answer.** Vinyl is looked at first, since it is
  what plays on the turntable, but a record not in Discogs is still named
  from an iTunes copy of the same album.
- **The answer says how sure it is.** One of three: **named** (the edition,
  its album with the original release date, and the sleeve); **not sure**
  (the candidates, none chosen); or **not owned**. Never a best guess
  presented as an answer.
- **The sleeve is the album's original art.** Whatever pressing is on the
  platter, the cover shown is the one the album is known by: its original
  release's, from the Cover Art Archive, and kept. The 2016 remaster of
  *No Jacket Required* shows the 1985 sleeve, not older Phil. Discogs'
  pictures may not be kept, so a Discogs image is only a fallback: the
  answer carries its URL, and the client fetches and credits it when it
  shows it, as EdS does today.
- **A Discogs fact carries its credit.** Every answer drawn from Discogs
  carries the discogs.com page it came from, so a client can show "Data
  provided by Discogs." beside it, as Discogs requires.
- **EdS asks it first, and falls back.** `nowplayd` asks the inventory
  before its own chase. On *not sure*, *not owned*, or no answer, it does
  what it does today.

## What it is not

- **Not MusicBee, Spotify, Tidal or CDs.** One new source, Discogs.
- **Not "best copy".** Which of an album's copies to play is v2's, with
  MusicBee, when there is more than one digital copy to choose between.
- **Not the owner's own pressing's sleeve.** A reissue's or a colored
  variant's own cover is later, if at all.
- **Not covers for digital plays.** A track played from the library still
  gets its cover by EdS's chase in the MVP.
- **Not the end of EdS's chase.** Deezer, iTunes Search and the Cover Art
  Archive stay in `nowplayd` as the fallback until v2 retires them.
- **Not deciding what is playing.** The inventory says what an album is,
  never what is current, and hears nothing.
- **Not a new upstream.** MusicBrainz is asked through its API, paced, as
  [ADR-0004](/docs/architecture/decisions/0004-discographies-are-enriched-once-from-musicbrainz/)
  has it; a self-hosted one is later.

## The MVP

The smallest thing that is useful end to end: the owner puts a record on,
EdS hears it, asks the inventory, and shows the right album with its
original art.

It is done when:

- the workload runs the version 1 schema, with the iTunes library and the
  Discogs collection collected into it;
- the inventory answers the turntable question, *named*, *not sure* or
  *not owned*, in its contract and OpenAPI document;
- `nowplayd` asks it first and falls back to today's chase;
- Journey's *Infinity* is named *Infinity*, 1978, with its original sleeve, and
  Kansas' *Point of Know Return* is shown with the 1977 sleeve, both as
  tests in the inventory and both on the real turntable;
- voiced's questions are still answered as they are today, from the new
  schema.

On the first day, the owner plays records they have played before and sees
each one named as the record on the shelf, not a reissue, a live album or a
best-of.

## Versions

- **v1, the MVP:** the schema rebuilt; Discogs collected beside iTunes; the
  turntable named and sleeved by the inventory, with EdS falling back.
- **v2, digital plays:** MusicBee collected; the best copy of an album;
  covers for what plays from the library; voiced and nowplayd asking only
  the inventory, and EdS's chase and `library.json` retired.
- **Later:** Spotify and Tidal's saved albums; CDs and other formats
  entered by hand; captured vinyl and what is still missing from an
  edition; a MusicBrainz of its own; identifying files by their sound
  (AcoustID); naming the turntable locally from a landmark index; the
  owner correcting the inventory directly; a preference of medium ahead of
  resolution in the best copy; the sleeve of the owner's own pressing, if
  ever wanted.

## Open questions

- **Which picture is the original art.** Discogs' terms were read on
  2026-11-02 (settled in [Requirements](/docs/requirements/)): no Discogs
  picture is kept, so the art is the Cover Art Archive's. Whether it is the
  front chosen for the album as a whole (MusicBrainz' release group) or the
  first release's own, and how often the archive has neither, is measured
  on the owner's collection.
- **How Discogs is collected.** Its API, with the owner's token from the
  environment, at least every six hours; whether more often, or on a
  signal, and how a release's facts are refreshed.
- **The schema's gray tables.** The enrichment tables and `collection_run`
  are still to review; the rebuild waits on them.
- **How a Discogs release is matched to an album.** By its MusicBrainz
  release, where MusicBrainz links it, and otherwise by artist and title
  against the artist's albums. How often the second is wrong is measured on
  the owner's collection.
- **The turntable question's contract.** Its route, what it is given, and
  how *not sure* carries its candidates, and the matching change to EdS's
  client.
- **The thresholds.** The fuzzy matching's thresholds were measured for
  voiced's requests (EdS ADR-0046); whether they hold for what a recognizer
  hears is measured, not assumed.
- **ADR-0004.** It is still `Proposed`; the rebuild is when its evidence is
  gathered and its status settled.
- **On EdS's side**, nowplayd asking the inventory first extends EdS
  ADR-0022 and EdS ADR-0038.

No decision is recorded from this proposal until it is reviewed and
accepted; each question above becomes an ADR then, if it needs one.
