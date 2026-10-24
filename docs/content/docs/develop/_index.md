---
title: "Developer Guide"
weight: 30
---

# Developer Guide

Building and changing eds-inventory. Each project has its own `Makefile` and
README; nothing builds the monorepo as a whole, and you work inside one
project directory at a time.

| Path | Language | Entry point |
|---|---|---|
| `docs` | Hugo | `make server` |

Services, the database and the collection and enrichment workflows will be
added here as they are decided.

After a fresh clone, fetch the docs theme:

```bash
git submodule update --init --recursive
```
