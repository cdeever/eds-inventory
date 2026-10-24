---
title: "Where It Runs"
weight: 2
---

# Where It Runs

eds-inventory runs in **EdS's Deevnet IoTaaS tenant**, beside EdS's own
services. It is its own project, in its own repository, with its own
releases; it shares a tenant with its first client, not a codebase.

## Why the same tenant

- **A tenant has no way in.** Deevnet IoTaaS's tenant fabric has no inbound
  path by design (Deevnet ADR-0001, Deevnet ADR-0003); it is why EdS's
  broker lives outside EdS's tenant. A service in another tenant could not be
  asked over HTTP by EdS's services without a path Deevnet IoTaaS does not
  offer. In the same tenant, EdS asks the inventory as `lightd` asks
  `palette`: one service to another, on the same network.
- **Nothing is invented for it.** No new path between tenants, and no
  Deevnet IoTaaS decision to wait on.

Sharing a tenant does not change the boundary. The inventory still knows
nothing about EdS, and is still reached only through its contract.

## Open

- **Which services run where.** EdS's back end can run on one Pi
  (EdS ADR-0033, `Proposed`). With the inventory beside it, one Pi may
  become several, and something will have to say which services are moved
  to which. Not decided now.
- **Where it is declared.** EdS's tenant is declared in EdS's repository
  (`infra/deevnet-tenant-eds`), and deployed from it. How the inventory's
  services, its database and its secrets join that tenant, and from which
  repository, is open.
- **How it reaches its sources.** iTunes and MusicBee keep their libraries
  on the media server, on the home network; Discogs, Spotify and Tidal are
  on the internet. EdS reaches the home network through a relay Pi
  (EdS ADR-0010); whether the inventory uses the same, or reads exports, is
  open.
