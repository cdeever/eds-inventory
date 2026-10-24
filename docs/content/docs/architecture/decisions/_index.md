---
title: "Decisions"
weight: 90
bookCollapseSection: true
---

# Architecture Decision Records

The design forks: where more than one option was viable, the reasoning that
picked one, and what was accepted in doing so. Each record is the durable
answer to "why is it built this way, and what did we turn down?"

**Read this index before reopening a question.** If a record already answers
it, the answer stands until a new record changes it. That is the point of the
section: a decision argued once should not have to be argued again from
memory.

Records are **point-in-time**. They say what was decided and why on the day it
was decided, and they are not rewritten afterward. The
[architecture](/docs/architecture/) and [reference](/docs/reference/) pages are
kept current; the decision log is kept honest.

## When a decision changes

Two different things happen to an old record, and they are not the same:

- **Extended.** The record was right but incomplete: building the thing
  exposed a question it never asked. A new record answers that question, the
  old one stays `Accepted`, and each links to the other.
- **Superseded.** Acting on the record would now be a mistake. A new record
  replaces it, the old one is marked `Superseded` and links forward.

The test is whether the earlier record is still safe to act on. If yes, extend
it. If not, supersede it.

## Format

| Field | Purpose |
|---|---|
| **Status** | `Proposed`, `Accepted`, `Superseded` or `Deprecated` |
| **Date** | The day the decision was made, not the day it was written up |
| **Context** | The situation and the goals that forced a choice |
| **Options considered** | The viable alternatives, each with what it would cost |
| **Decision** | What was chosen |
| **Consequences** | What it commits us to, good and bad |

- Records are numbered in the order they are **opened** and never renumbered.
  The number is claimed when the question is written down, not when it is
  answered.
- A record sits at `Proposed` while it rests on something not yet proven: a
  source's API not yet tried, a match not yet measured against the real
  collection.
  The evidence is added to the record when it arrives, and the status moves
  with it.
## Whose record is it

eds-inventory, EdS and Deevnet IoTaaS keep records under the same names. To
keep them apart:

- A bare **ADR-0001**, **CHG-0001** or **INC-0001** anywhere in this
  repository means an **eds-inventory** record, on this site.
- An EdS record is always written **EdS ADR-0046**, and links to
  [EdS's docs](https://cdeever.github.io/eds/docs/architecture/decisions/).
- A Deevnet IoTaaS record is always written **Deevnet ADR-0012**, and links
  to [deevnet-docs](https://deevnet.github.io/deevnet-docs/docs/architecture/decisions/).

## Records

| ADR | Date | Decision | Status |
|---|---|---|---|
| [0001](0001-go-services-and-a-sqlite-store/) | 2026-10-24 | The services are Go, and the inventory is kept in SQLite, with no cgo | {{< adr-status "Proposed" >}} |
| [0002](0002-the-itunes-library-is-collected-from-its-xml/) | 2026-10-24 | The iTunes library is collected from its XML, and asked for by nearest name | {{< adr-status "Proposed" >}} |
| [0003](0003-it-deploys-itself-to-eds-workload/) | 2026-10-24 | eds-inventory deploys itself onto EdS's services workload, from its own repository | {{< adr-status "Proposed" >}} |
