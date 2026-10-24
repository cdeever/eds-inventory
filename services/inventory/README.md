# inventory

The inventory's store, its collectors, its enrichment and its HTTP contract.
So far one source, the owner's iTunes library, read from the XML iTunes
writes; and one enrichment, each artist's discography from MusicBrainz,
fetched in the background by inventoryd.

```bash
make check                                   # gofmt, vet, tests
make collect XML="iTunes Library.xml"        # into DB (default ./inventory.db)
make run                                     # inventoryd on 127.0.0.1:8740
go run ./cmd/inventory find Arrowsmith       # by hand, on $INV_DB
go run ./cmd/inventory enrich Genesis        # one artist from MusicBrainz, now
go run ./cmd/inventory nth Genesis 1         # their first album the owner has
make build-arm64                             # both binaries for a Pi
```

| Package | What |
|---|---|
| `internal/itunes` | reads the library's XML: albums, by album artist, one for all their spellings |
| `internal/store` | SQLite: artists, albums, and which source holds each; IDs of the inventory's own, kept across collections |
| `internal/catalog` | finds the nearest artist and album, the first of several titles held, an artist's albums by year |
| `internal/musicbrainz` | an artist and their aliases, their studio albums in order, an album's other names; one call each 1.5 s |
| `internal/enrich` | works through the artists in the background, the ones asked about first |
| `internal/fold` | how names are compared |
| `internal/api` | the HTTP contract, version 1, and `openapi.json` |

Why it is built this way:
[ADR-0001](https://cdeever.github.io/eds-inventory/docs/architecture/decisions/0001-go-services-and-a-sqlite-store/)
(Go and SQLite) and
[ADR-0002](https://cdeever.github.io/eds-inventory/docs/architecture/decisions/0002-the-itunes-library-is-collected-from-its-xml/)
(the iTunes library) and
[ADR-0004](https://cdeever.github.io/eds-inventory/docs/architecture/decisions/0004-discographies-are-enriched-once-from-musicbrainz/)
(discographies). The matching came from EdS's `internal/library`
(EdS ADR-0046), where its thresholds were measured.

| Setting | Default | |
|---|---|---|
| `INV_DB` | `/var/lib/eds-inventory/inventory.db` (`inventory.db` for the command) | the database; one that cannot be opened stops `inventoryd` |
| `INV_ADDR` | `127.0.0.1:8740` | where `inventoryd` listens |
| `INV_MUSICBRAINZ` | `https://musicbrainz.org` | where artists are enriched from; `off` enriches nothing |
| `INV_CONTACT` | the repository's address | who MusicBrainz is told is asking |

The owner's library, and any database made from it, are never committed.
