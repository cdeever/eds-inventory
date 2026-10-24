# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with
code in this repository.

## What this is

eds-inventory is one inventory of the owner's music, in every format, with
one identity per album, enriched once and asked over HTTP. Its first client
is EdS (`../eds`, <https://github.com/cdeever/eds>). It runs in EdS's
Deevnet IoTaaS tenant, beside EdS's services, because a tenant has no way in
from another; it shares the tenant, not the codebase or the boundary.

A monorepo of independent projects, as EdS is: each has its own `Makefile`
and README, and nothing builds the whole.

| Path | What | Build entry |
|---|---|---|
| `services/inventory` | Go 1.26, SQLite (no cgo): store, iTunes collector, HTTP API v1 | `make check build`, `make collect XML=…`, `make run` |
| `deploy` | onto EdS's services workload (ADR-0003) | `make image load unit WHERE=builder`, `make library XML=…` |
| `docs` | Hugo | `make server` |

`internal/fold` and `internal/catalog`'s matching came from EdS's
`services/nowplaying/internal/library` (EdS ADR-0046); their thresholds were
measured there. Only `internal/store` writes SQL. The contract is
`internal/api`, and `internal/api/openapi.json` is held to its routes by a
test: change both together. Likewise the schema in `internal/store` is
drawn and described in `docs/content/docs/reference/schema/` (`schema.dot`
and a data dictionary); a store test fails if a column is missing from the
dictionary. **An owner's library, and any database made
from it, is never committed.**

`deploy/load.sh` is EdS's, copied; keep them alike (EdS INC-0002: never
stream images to the workload by hand). EdS's voiced asks inventoryd at
`127.0.0.1:8740` (EdS ADR-0047); a change to the contract is a change to
its client in `../eds/services/nowplaying/internal/voice/inventory.go`.

## The boundary

**The inventory knows music and nothing about its clients**: no MQTT, no
rooms, no lights, no turntable, no "now playing". That is the line EdS's
`palette` keeps (color, and nothing about music). Do not add a client's
concepts here; a client adapts to the inventory's contract.

## Decisions are recorded — read them first

The same three records as EdS, on this repository's docs site:

| Record | Where |
|---|---|
| **ADR-NNNN** | `docs/content/docs/architecture/decisions/` |
| **CHG-NNNN** | `docs/content/docs/changes/<year>/` |
| **INC-NNNN** | `docs/content/docs/incidents/<year>/` |

- Read the decision index before proposing a design.
- **A bare `ADR-0001` means an eds-inventory record.** An EdS record is
  always written `EdS ADR-0046`, and a Deevnet IoTaaS one `Deevnet ADR-0012`.
- Records move in the same commit as the work, each with a row in its index.
- Rules and templates: `docs/content/docs/develop/records/`.

## Docs

```bash
cd docs
make server        # live reload on :1313/eds-inventory/
make html          # build to public/
make diagrams      # draw content/**/*.dot to .svg (needs GraphViz); commit both
```

Deployed to GitHub Pages by `.github/workflows/docs.yml` on a push to `main`
under `docs/**`; a pull request builds it without deploying.

## Conventions

- **American English** everywhere a person reads, as in EdS.
- **Secrets never land in the repository.** Service tokens (Discogs, Spotify,
  and the rest) arrive as environment variables.
- Contracts are versioned JSON (`"v": 1`) with an OpenAPI document; a client
  that sees a version it does not know refuses it.
