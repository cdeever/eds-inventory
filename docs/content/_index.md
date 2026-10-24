---
title: "eds-inventory"
type: docs
---

# eds-inventory

<p class="subtitle"><em>Every album the owner has, in one place, known once.</em></p>

The owner's music is spread across formats and services: records on the
shelf, cataloged in Discogs; digital libraries in iTunes and MusicBee;
albums saved in Spotify and Tidal; CDs and anything else owned. Each knows a
piece of it, and none knows all of it.

eds-inventory collects from every one of them into a single inventory, gives
each album one identity whatever format it is in, enriches it once from the
services that know about music, and answers lookups for the programs that
need them.

## Who asks it

Its first client is [EdS](https://cdeever.github.io/eds/), whose Now Playing
subsystem today chases a different service for each question: which record
is on the platter, which album is the original, where to find a cover, what
the library holds. With an inventory, EdS asks one place first
([the proposal](https://cdeever.github.io/eds/docs/architecture/proposals/music-inventory/)).

The inventory knows nothing about EdS. Anything that wants to know what the
owner has, and what each album is, can ask it the same way.

## Where it stands

Requirements are being gathered, and nothing is built yet. The
[requirements](docs/requirements/) page has what is settled and what is
still open.

<div class="section-cards">
  <a class="section-card" href="docs/requirements/">
    <h3>Requirements</h3>
    <p>What the inventory must do, what is settled, and what is open.</p>
  </a>
  <a class="section-card" href="docs/architecture/">
    <h3>Architecture</h3>
    <p>The principles it is built on, and how sources, store and clients fit.</p>
  </a>
  <a class="section-card" href="docs/develop/">
    <h3>Developer Guide</h3>
    <p>Building, testing and keeping records.</p>
  </a>
  <a class="section-card" href="docs/reference/">
    <h3>Reference</h3>
    <p>The contract clients depend on.</p>
  </a>
</div>
