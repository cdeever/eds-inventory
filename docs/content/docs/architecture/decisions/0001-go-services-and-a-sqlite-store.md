---
title: "ADR-0001: Go Services and a SQLite Store"
weight: 1
---

# ADR-0001: The Services Are Go, and the Inventory Is Kept in SQLite, with No cgo

| | |
|---|---|
| **Status** | {{< adr-status "Proposed" >}} — until a real library has been collected and asked, on the machine it will run on |
| **Date** | 2026-10-24 |
| **Scope** | The language of eds-inventory's services, and the database the inventory is kept in. Not how sources are matched to one another, which is later. |

---

## Context

The inventory needs a store from its first slice, the owner's iTunes
library. It holds one owner's music: a few thousand albums, not millions.
It runs in EdS's tenant, where EdS's back end may run on one Pi or several
(EdS ADR-0033), so whatever it is made of must run there.

The matching the first slice needs already exists, in Go, in EdS
(`internal/library`, EdS ADR-0046), measured against the owner's library and
spoken requests.

## Options considered

**Go, with SQLite through a driver with no cgo** (`modernc.org/sqlite`). One
static binary for each service, built for a Pi from the Mac as EdS's are.
The database is one file, backed up by copying it. EdS's matching comes
across unchanged.

**Go, with Postgres.** A server to run, upgrade and back up beside the
service, for a collection one file holds easily. Better when several
writers work at once, which nothing here does yet.

**Python**, as EdS's `palette` and `scribe` are. Good for work that leans on
its libraries, as theirs does; nothing here does, and EdS's matching would
have to be ported.

## Decision

**The services are Go, one module for each, and the inventory is kept in
SQLite, through `modernc.org/sqlite`, with no cgo anywhere.**

- The schema is SQL in the store package, applied in steps and numbered in
  SQLite's `user_version`, so a database is brought up to date when it is
  opened.
- Everything that touches the database is in `internal/store`. Nothing else
  writes SQL, so moving to Postgres, if it is ever needed, changes one
  package.
- Searching is done on a copy of the inventory held in memory, read again
  when a collection changes it.

## Consequences

- **The module needs Go 1.26**, which the driver requires; EdS's are on
  1.25.
- **One writer at a time.** SQLite allows many readers and one writer; a
  collection is one transaction, and the server only reads. Several
  collectors writing at once would wait on one another.
- **The database is a file on one machine.** Where that file lives, and how
  it is backed up, is a question for where the inventory runs.
- **Postgres is the step if SQLite is outgrown**: several writers, or the
  inventory needed from more than one machine.

## What would settle it

- The owner's real iTunes library collected, and the inventory asked, on the
  machine it will run on, with times that are no concern.
