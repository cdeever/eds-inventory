---
title: "Where It Runs"
weight: 2
---

# Where It Runs

eds-inventory runs in a **Deevnet IoTaaS tenant of its own**, separate from
EdS's. It is its own project with its own clients, and it is not a part of
EdS that happens to live elsewhere.

## Why its own tenant

- **It is not EdS's.** EdS is its first client, not its owner. Other clients
  can come, and none of them should have to reach into EdS's tenant to ask.
- **It can be changed, deployed and taken down without EdS's tenant
  noticing**, and the reverse.
- **Its secrets are its own**: the tokens for the sources and the services
  it asks are held in its tenant and nowhere else.

## Open

- **How EdS reaches it.** EdS's services run in EdS's tenant, and the
  inventory's API in this one. Deevnet IoTaaS's tenant fabric has no inbound
  path by design (Deevnet ADR-0001, Deevnet ADR-0003), which is why EdS's
  broker is not in its tenant. How one tenant's service asks another's over
  HTTP is a question for Deevnet IoTaaS, and is not answered yet.
- **How it reaches its sources.** iTunes and MusicBee keep their libraries
  on the media server, on the home network; Discogs, Spotify and Tidal are
  on the internet. EdS reaches the home network through a relay Pi
  (EdS ADR-0010); whether the inventory needs the same, or reads exports, is
  open.
- **What its tenant is made of**: a workload, its database, and the
  Terraform that declares them, as EdS's tenant is declared.
