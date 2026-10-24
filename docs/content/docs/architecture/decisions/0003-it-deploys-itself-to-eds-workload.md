---
title: "ADR-0003: It Deploys Itself to EdS's Workload"
weight: 3
---

# ADR-0003: eds-inventory Deploys Itself onto EdS's Services Workload, from Its Own Repository

| | |
|---|---|
| **Status** | {{< adr-status "Proposed" >}} — until EdS's voiced has asked it there and played what it answered |
| **Date** | 2026-10-24 |
| **Scope** | How inventoryd reaches the machine it runs on, and how a library reaches it. Not which machine that is in the long run: EdS's back end may become several Pis, and which services go where is open. |
| **Extends** | [Where It Runs](/docs/architecture/tenant/): EdS's tenant, since a tenant has no way in from another |

---

## Context

The inventory runs in EdS's Deevnet IoTaaS tenant, and its first client,
EdS's voiced, runs on EdS's services workload. EdS deploys its own services
there from its repository (`make -C deploy`), as container images under
systemd, built on the Builder and sent at a capped rate after its deploys
hung the core router (EdS INC-0002).

## Options considered

**A `deploy/` of its own**, the same pattern as EdS's, aimed at EdS's
workload. eds-inventory ships when it changes, and EdS does not need to know
how it is built.

**EdS's deploy builds it**, from a checkout of this repository beside EdS's.
One command for everything, and EdS's deploy learning another project's
build.

**A host of its own.** More machines for one owner, and an HTTP path between
them that the tenant has no need for yet.

## Decision

**eds-inventory has its own `deploy/`, which builds `localhost/eds-inventory`
on the Builder and runs it on EdS's services workload as
`eds-inventory.service`, answering on `127.0.0.1:8740`.**

- **The same shipping as EdS's**: `load.sh` is EdS's, copied, and an image
  is sent only when the `eds.source` hash of its sources changed,
  compressed, at 5 MB/s.
- **Only what is collected leaves the owner's machine**: `make library`
  runs `inventory albums` there and sends the albums (names, years, counts),
  not the library's XML; the collector in the container reads them, and they
  are removed.
- **The database is `/var/lib/eds-inventory/inventory.db`** on the workload,
  owned by the container's nonroot user.

## Consequences

- **A replaced workload comes back without it**, as it does without EdS's
  services: `make -C deploy unit ship library` puts it back, with the
  library collected again.
- **Two repositories deploy to one workload.** Each restarts only its own
  services.
- **Moving it is moving `HOST`**, when EdS's back end becomes several
  machines.

## What would settle it

- inventoryd running on the workload with the owner's library collected,
  and EdS's voiced asking it for a spoken request.
