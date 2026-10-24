---
title: "CHG-0001: inventoryd on EdS's Workload"
weight: 1
---

# CHG-0001: inventoryd on EdS's Services Workload, with the Owner's iTunes Library

| | |
|---|---|
| **Date** | 2026-10-24 |
| **Change type** | Deployment |
| **Status** | {{< status-badge "planned" "Planned" >}} |
| **Systems** | EdS's services workload (`services.eds.mobile.deevnet.net`): the image `localhost/eds-inventory`, `eds-inventory.service`, `/var/lib/eds-inventory` |
| **How** | `make -C deploy image load WHERE=builder`, `make -C deploy unit`, `make -C deploy library XML=…` |
| **Risk** | Low. Nothing running asks it until EdS's voiced is pointed at it (EdS CHG-0041). Most likely to go wrong: the Builder's first build of the image |
| **Decisions** | [ADR-0001](/docs/architecture/decisions/0001-go-services-and-a-sqlite-store/), [ADR-0002](/docs/architecture/decisions/0002-the-itunes-library-is-collected-from-its-xml/), [ADR-0003](/docs/architecture/decisions/0003-it-deploys-itself-to-eds-workload/) |
| **Related** | EdS CHG-0041, which points voiced at it |

---

## Summary

The first deployment of eds-inventory: inventoryd onto EdS's services
workload, answering on `127.0.0.1:8740`, with the owner's iTunes library
collected into it.

## Goal

- `systemctl is-active eds-inventory` on the workload is `active`.
- On the workload, `curl -s '127.0.0.1:8740/v1/artists/find?name=Arrowsmith'`
  answers Aerosmith.
- The collection says 3,665 albums, as on the Mac on 2026-10-24.

## Procedure

1. `make -C deploy image load WHERE=builder`: built on the Builder, sent at
   5 MB/s.
2. `make -C deploy unit`: the unit, the data directory; started.
3. `make -C deploy library XML="…/iTunes Library.xml"`: the albums, about
   600 KB, collected and removed.

## Undo

`sudo systemctl disable --now eds-inventory` on the workload, and remove
`/etc/systemd/system/eds-inventory.service`, `/var/lib/eds-inventory` and
the image. Nothing else uses them.

## Outcome

*Completed after the change has run.*

| When | Step | What happened |
|---|---|---|
| | | |
