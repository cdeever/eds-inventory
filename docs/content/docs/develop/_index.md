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
| `services/inventory` | Go 1.26, SQLite | `make check build`, `make collect XML=…`, `make run` |
| `docs` | Hugo | `make server` |

After a fresh clone, fetch the docs theme:

```bash
git submodule update --init --recursive
```
