---
title: "Architecture"
weight: 10
---

# Architecture

eds-inventory is shaped by a handful of principles. This page says what they
are and why each was chosen. The pages in this section show them at work,
and the [decisions](decisions/) record each place one was applied and what
was turned down.

## It knows music, and nothing about its clients

The inventory knows albums, artists, tracks, formats and covers. It does
not know MQTT, rooms, lights, turntables or what is playing. A client adapts
to the inventory's contract; the inventory never learns a client's words.

That is the line EdS's `palette` keeps, knowing color and nothing about
music, and it is what lets the inventory be built, tested and used without
EdS running at all, and by something other than EdS.

## Each source stays the truth for what it holds

The inventory does not take over from the places music is already kept.
Records are cataloged in Discogs, libraries are kept in iTunes and MusicBee,
albums are saved in Spotify and Tidal. The inventory keeps itself in step
with each, and when it and a source disagree, the source is right.

## One album, one identity

An album is the same album whether it is on vinyl, in a library or saved in
a streaming service. The inventory gives it one ID of its own, and keeps
every other service's ID for it beside that one. A client can ask by any
of them and get the same album.

## Look it up once

Enrichment happens when the inventory meets an album, not when a client
asks. Covers, originals and discographies are fetched from the services
that know them, once, and kept where their terms allow. A client's question
is answered from what the inventory already has, so asking costs nothing
outside it.

## Answer only what is known

An answer the inventory is not sure of says so. Two albums that might be the
same are not quietly merged; a track that might be on either of two albums
is not given one. A client can act on a gap, but not on a confident wrong
answer it cannot tell from a right one.

## One contract, versioned

Clients ask over HTTP, with JSON that carries `"v": 1` and an OpenAPI
document that describes it. A client that sees a version it does not know
refuses it rather than guessing. The contract is what clients depend on,
and it changes by a new version, never in place.

## Secrets never live in the repository

The tokens for Discogs, Spotify and the rest arrive as environment
variables, and are never committed or logged.

## Decisions are written down

Every place where more than one approach was viable has a record of what
was chosen, what was turned down, and why. A question settled once is looked
up, not argued again.

## In this section

- [Sources, Store and Clients](system/): how music comes in, where it is
  kept, and who asks for it.
- [Where It Runs](tenant/): its own Deevnet IoTaaS tenant.
- [Decisions](decisions/): every choice, argued.
