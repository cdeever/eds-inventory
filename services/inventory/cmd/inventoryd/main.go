// Command inventoryd answers questions about the inventory over HTTP: the
// contract in internal/api, version 1.
//
//	INV_DB    the database (default /var/lib/eds-inventory/inventory.db)
//	INV_ADDR  where to listen (default 127.0.0.1:8740)
//
// A database that cannot be opened stops it at once: an inventory that
// answers nothing would look like an owner with no music.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cdeever/eds-inventory/services/inventory/internal/api"
	"github.com/cdeever/eds-inventory/services/inventory/internal/catalog"
	"github.com/cdeever/eds-inventory/services/inventory/internal/store"
)

var version = "dev"

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	path := env("INV_DB", "/var/lib/eds-inventory/inventory.db")
	addr := env("INV_ADDR", "127.0.0.1:8740")
	if _, _, err := net.SplitHostPort(addr); err != nil {
		log.Error("INV_ADDR is not an address", "addr", addr, "error", err)
		os.Exit(1)
	}

	s, err := store.Open(path)
	if err != nil {
		log.Error("the inventory could not be opened", "db", path, "error", err)
		os.Exit(1)
	}
	defer s.Close()
	live := &catalog.Live{Store: s}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	c, err := live.Catalog(ctx)
	if err != nil {
		log.Error("the inventory could not be read", "db", path, "error", err)
		os.Exit(1)
	}
	log.Info("inventoryd listening", "version", version, "addr", addr, "db", path, "artists", len(c.Artists))

	srv := &http.Server{Addr: addr, Handler: api.Handler(live, log), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shut, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shut)
	}()
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("inventoryd stopped", "error", err)
		os.Exit(1)
	}
}

func env(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}
