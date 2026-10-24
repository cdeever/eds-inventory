# Deploy inventoryd to EdS's services workload, where EdS's voiced asks it
# (ADR-0003). The same pattern as EdS's deploy: build the image, copy it
# over SSH if its sources changed, and run it under systemd. No registry.
#
# Plain recipes, one shell per line: macOS ships GNU make 3.81.
SHELL := /bin/bash

HOST  ?= tenant@services.eds.mobile.deevnet.net
REPO  := $(CURDIR)/..
SVC   := $(REPO)/services

# Where there is no container tool - a Mac with no podman - WHERE=builder
# builds and loads from the Builder, which has one and can reach the
# workload. Its build directory is under the Builder login's home.
CONTAINER ?= podman
WHERE     ?= local
BUILDER   ?= cdeever@builder
BUILD_DIR ?= eds-inventory-build
VERSION   ?= $(shell git -C $(REPO) describe --always --dirty 2>/dev/null || echo dev)
XML       ?=

SERVICE := eds-inventory

.PHONY: help image load ship unit library status logs restart

help:
	@echo "deploy - inventoryd onto $(HOST), beside EdS's services"
	@echo
	@echo "  ship     build the image, send it if its sources changed (rate-capped), restart"
	@echo "  unit     install the unit and its data directory; start"
	@echo "  library  collect XML=\"…/iTunes Library.xml\": only its albums are sent"
	@echo "  status   systemd status"
	@echo "  logs     follow its log"
	@echo
	@echo "No container tool here? Add WHERE=builder."

ifeq ($(WHERE),builder)

image:
	ssh $(BUILDER) 'mkdir -p $(BUILD_DIR)/deploy'
	rsync -a --delete --exclude bin/ --exclude '*.db*' --exclude .DS_Store $(SVC)/ $(BUILDER):$(BUILD_DIR)/services/
	rsync -a Makefile load.sh $(BUILDER):$(BUILD_DIR)/deploy/
	ssh $(BUILDER) 'make -C $(BUILD_DIR)/deploy image VERSION=$(VERSION)'

load:
	rsync -a load.sh $(BUILDER):$(BUILD_DIR)/deploy/
	ssh $(BUILDER) 'make -C $(BUILD_DIR)/deploy load HOST=$(HOST)'

else

# The same labels as EdS's images, so load.sh treats it as one of theirs:
# eds.source is a hash of its sources, and an image the workload already
# has with the same hash is not sent.
source = $(shell cd $(SVC)/inventory && find . -type f -not -path './bin/*' -not -name '*.db*' | LC_ALL=C sort | xargs sha256sum | sha256sum | cut -c1-16)

image:
	@src=$(source); \
	if $(CONTAINER) image inspect --format '{{index .Labels "eds.source"}}' localhost/eds-inventory:latest 2>/dev/null | grep -qx "$$src"; then \
		echo "inventory: sources unchanged ($$src), not built"; \
	else \
		$(CONTAINER) build --platform linux/amd64 --layers=false --label eds.deploy=1 --label eds.source=$$src \
			--build-arg VERSION=$(VERSION) -t localhost/eds-inventory:latest $(SVC)/inventory; \
	fi
	$(CONTAINER) image prune -f --filter label=eds.deploy=1 >/dev/null

load:
	CONTAINER=$(CONTAINER) ./load.sh $(HOST) inventory

endif

ship: image
	$(MAKE) load
	ssh $(HOST) 'sudo systemctl restart $(SERVICE) && sudo podman image prune -f >/dev/null'

unit:
	scp $(SERVICE).service $(HOST):$(SERVICE).service
	ssh $(HOST) 'sudo install -d -m 0700 -o 65532 -g 65532 /var/lib/eds-inventory && \
		sudo install -m 0644 $(SERVICE).service /etc/systemd/system/ && rm -f $(SERVICE).service && \
		sudo systemctl daemon-reload && sudo systemctl enable $(SERVICE) && sudo systemctl restart $(SERVICE)'

# Only the albums leave this machine: inventory albums writes each one's
# artist, title, year, track count and whether it is a compilation, and
# nothing else of the XML. The file takes the data directory's SELinux
# label, which podman's :Z gave the container: installed, it would be
# var_lib_t and unreadable there (EdS INC-0003).
library:
	@if [[ ! -s "$(XML)" ]]; then echo 'usage: make library XML="/path/to/iTunes Library.xml"' >&2; exit 2; fi
	@umask 077; cd $(SVC)/inventory && go run ./cmd/inventory albums itunes "$(XML)" > $(CURDIR)/albums.json
	scp -l 40000 albums.json $(HOST):albums.json
	ssh $(HOST) 'sudo install -m 0400 -o 65532 -g 65532 albums.json /var/lib/eds-inventory/ && \
		sudo chcon --reference=/var/lib/eds-inventory /var/lib/eds-inventory/albums.json && rm -f albums.json && \
		sudo podman exec eds-inventory /inventory collect itunes /data/albums.json; \
		sudo rm -f /var/lib/eds-inventory/albums.json'
	@rm -f albums.json

restart:
	ssh $(HOST) sudo systemctl restart $(SERVICE)

status:
	ssh $(HOST) systemctl --no-pager status $(SERVICE)

logs:
	ssh $(HOST) sudo journalctl -f -u $(SERVICE)
