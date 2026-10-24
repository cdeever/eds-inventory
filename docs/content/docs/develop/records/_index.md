---
title: "Keeping Records"
weight: 90
bookCollapseSection: true
---

# Keeping Records

eds-inventory writes three kinds of record, as EdS does, each answering a different question. They
are what stops a settled design question being argued again from memory, and
what lets a change made in a hurry be understood a month later.

| Record | Question it answers | Where |
|---|---|---|
| **ADR** | Why is it built this way, and what did we turn down? | [Decisions](/docs/architecture/decisions/) |
| **CHG** | What was changed on something running, and what happened? | [Change Records](/docs/changes/) |
| **INC** | What broke, why, and what was done about it? | [Incident Records](/docs/incidents/) |

## The working rule

1. **Before reopening a design question, read the
   [decision index](/docs/architecture/decisions/).** If a record answers it,
   that answer stands until a new record changes it.
2. **A design fork gets an ADR before the code.** If two approaches were
   seriously considered, write down both and why one won. If the choice rests
   on something unproven, the record is `Proposed` until the evidence is in.
3. **A change to the running tenant or a workload gets a CHG**, written
   before it runs and completed after.
4. **A failure someone would notice gets an INC**, started while it is fresh.

Records move in the same commit as the work they describe, like the rest of
this site.

## Language

Records, like the rest of this site, are written in American English:
"color", "catalog", "recognizer", "behavior". A name that already exists
keeps its spelling, whatever it is: a setting, a file, a link, or words
quoted from a program.

## Numbering

Each kind has its own series — `ADR-0001`, `CHG-0001`, `INC-0001` — numbered
in the order records are opened and never renumbered or reused. Change and
incident records are grouped in a folder per year; the number runs across
years.

EdS and Deevnet IoTaaS use the same three names. A bare number in this
repository is always eds-inventory's own. Anyone else's is always written
with its owner — **EdS ADR-0046**, **Deevnet ADR-0012** — and linked to
their site.

## Templates

- An ADR's format is described on the
  [decision index](/docs/architecture/decisions/#format); copy the most recent
  record.
- [Change record template](change-record/)
- [Incident record template](incident-record/)
