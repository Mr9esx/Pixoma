# Local process entrypoints. Ctrl-C stops all children started by run-admin / run-all.
SHELL := /bin/bash

# Bot Comfy mode: 1/true = in-process mock (default for local); 0/false = real ComfyUI HTTP
COMFY_MOCK ?= 1

.PHONY: build test run-bot run-admin run-admin-api run-admin-web run-all

build:
	go build -o bin/comfyui-bot ./apps/bot/cmd/comfyui-bot
	go build -o bin/admin-api ./apps/admin-api/cmd/admin-api

test:
	go test ./...

run-bot: build
	COMFY_MOCK=$(COMFY_MOCK) go run ./apps/bot/cmd/comfyui-bot

run-admin-api:
	@test -f configs/admin-api.yaml || cp configs/admin-api.example.yaml configs/admin-api.yaml
	go run ./apps/admin-api/cmd/admin-api

run-admin-web:
	cd web/admin && { [ -d node_modules ] || pnpm install; } && pnpm dev

# Admin API + web console (both). Ctrl-C stops both.
run-admin:
	@trap 'kill 0' INT TERM EXIT; \
	$(MAKE) --no-print-directory run-admin-api & \
	$(MAKE) --no-print-directory run-admin-web & \
	wait

# Bot + admin API + admin web. Ctrl-C stops all.
run-all: build
	@trap 'kill 0' INT TERM EXIT; \
	COMFY_MOCK=$(COMFY_MOCK) go run ./apps/bot/cmd/comfyui-bot & \
	$(MAKE) --no-print-directory run-admin-api & \
	$(MAKE) --no-print-directory run-admin-web & \
	wait
