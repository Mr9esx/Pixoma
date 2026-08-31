DATA_DIR ?= data

.PHONY: build test run run-livedemo embed-admin clean dev

build:
	go build -o bin/pixoma ./apps/pixoma/cmd/pixoma
	go build -o bin/pixoma-edge-agent ./apps/edge-agent/cmd/edge-agent

test:
	go test ./...

run: build
	go run ./apps/pixoma/cmd/pixoma

run-livedemo: embed-admin
	go run ./apps/pixoma/cmd/pixoma -livedemo

dev:
	bash scripts/dev.sh

embed-admin:
	cd web/admin && pnpm build
	rm -rf apps/pixoma/internal/webembed/dist
	cp -R web/admin/dist apps/pixoma/internal/webembed/dist

# Stop pixoma first. DATA_DIR=... if not using data/.
clean:
	@if [ -z "$(DATA_DIR)" ] || [ "$(DATA_DIR)" = "." ] || [ "$(DATA_DIR)" = "/" ]; then \
		echo "refusing to clean DATA_DIR=$(DATA_DIR)"; exit 1; \
	fi
	rm -rf "$(DATA_DIR)"
