# inventory

The inventory's store, its collectors and its HTTP contract. So far one
source: the owner's iTunes library, read from the XML iTunes writes.

```bash
make check                                   # gofmt, vet, tests
make collect XML="iTunes Library.xml"        # into DB (default ./inventory.db)
make run                                     # inventoryd on 127.0.0.1:8740
go run ./cmd/inventory find Arrowsmith       # by hand, on $INV_DB
make build-arm64                             # both binaries for a Pi
```

| Package | What |
|---|---|
| `internal/itunes` | reads the library's XML: albums, by album artist, one for all their spellings |
| `internal/store` | SQLite: artists, albums, and which source holds each; IDs of the inventory's own, kept across collections |
| `internal/catalog` | finds the nearest artist and album, the first of several titles held, an artist's albums by year |
| `internal/fold` | how names are compared |
| `internal/api` | the HTTP contract, version 1, and `openapi.json` |

Why it is built this way:
[ADR-0001](https://cdeever.github.io/eds-inventory/docs/architecture/decisions/0001-go-services-and-a-sqlite-store/)
(Go and SQLite) and
[ADR-0002](https://cdeever.github.io/eds-inventory/docs/architecture/decisions/0002-the-itunes-library-is-collected-from-its-xml/)
(the iTunes library). The matching came from EdS's `internal/library`
(EdS ADR-0046), where its thresholds were measured.

| Setting | Default | |
|---|---|---|
| `INV_DB` | `/var/lib/eds-inventory/inventory.db` (`inventory.db` for the command) | the database; one that cannot be opened stops `inventoryd` |
| `INV_ADDR` | `127.0.0.1:8740` | where `inventoryd` listens |

The owner's library, and any database made from it, are never committed.
