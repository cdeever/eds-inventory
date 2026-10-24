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

Requirements are being gathered. Nothing is built yet.

## Layout

A monorepo of independent projects, each with its own `Makefile` and README,
as EdS is. Nothing builds the repository as a whole.

| Path | What | Build entry |
|---|---|---|
| `docs` | Hugo, the hugo-book theme | `make server` |

Services, the database and the collection and enrichment workflows will be
added beside it as they are decided.

After a fresh clone:

```bash
git submodule update --init --recursive
```
