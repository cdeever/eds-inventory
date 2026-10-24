# eds-inventory

One inventory of the owner's music, in every format: the records on the
shelf, the digital libraries, the albums saved in streaming services, and
anything else owned. It collects from each source, gives every album one
identity, enriches it once from the services that know about music, and
answers lookups for the programs that need them.

Its first client is [EdS](https://github.com/cdeever/eds), whose Now Playing
subsystem would ask it first instead of chasing several services for each
question ([the proposal](https://cdeever.github.io/eds/docs/architecture/proposals/music-inventory/)).
It knows nothing about EdS.

**Documentation:** <https://cdeever.github.io/eds-inventory/>

## Status

Requirements are being gathered. The first slice, the owner's iTunes library, is built: collected from its XML into SQLite and asked over HTTP.

## Layout

A monorepo of independent projects, each with its own `Makefile` and README,
as EdS is. Nothing builds the repository as a whole.

| Path | What | Build entry |
|---|---|---|
| `services/inventory` | Go 1.26, SQLite: the store, the iTunes collector, the HTTP API | `make check build run` |
| `deploy` | onto EdS's services workload | `make image load unit library WHERE=builder` |
| `docs` | Hugo, the hugo-book theme | `make server` |

After a fresh clone:

```bash
git submodule update --init --recursive
```
