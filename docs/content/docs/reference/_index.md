---
title: "Reference"
weight: 40
---

# Reference

The contract clients depend on. Everything here is versioned explicitly: a
payload carries a `v`, and a client that sees a version it does not know
refuses it rather than guessing.

## The HTTP API, version 1

Served by `inventoryd`; `GET /openapi.json` is the full description, and a
test holds it to the routes. Every answer is JSON with `"v": 1`. When nothing
is near enough, the answer is `null` with the score reached, never the
nearest thing the inventory has.

| Route | Answers |
|---|---|
| `GET /v1/artists/find?name=NAME` | the artist nearest a name, weighing spelling and sound, with its score; `null` below 0.90 |
| `GET /v1/artists/{id}` | an artist and their albums |
| `GET /v1/artists/{id}/albums/find?title=T` | the artist's album nearest a title, with its score; `null` below 0.90 |
| `GET /v1/artists/{id}/albums/find?title=T1&title=T2…` | the first of several titles the inventory has, never a compilation, and which title found it (also `studio=true` with one title) |
| `GET /v1/artists/{id}/albums/by-year` | the artist's own albums, earliest first, without compilations or what is named as one |
| `GET /healthz` | `ok` |

An album is `{id, title, year, tracks, compilation, sources}`: `title` as the
source spells it, so a player asked for it by name finds it, and `id` the
inventory's own (`al_` and twelve hex digits), kept across collections.
