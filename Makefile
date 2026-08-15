.PHONY: build test run run-mock run-admin-api embed-admin

build:
	go build -o bin/pixoma ./apps/pixoma/cmd/pixoma
	go build -o bin/pixoma-edge-agent ./apps/edge-agent/cmd/edge-agent
	go build -o bin/admin-api ./apps/admin-api/cmd/admin-api

test:
	go test ./...

run: build
	COMFY_MOCK=0 go run ./apps/pixoma/cmd/pixoma

run-mock: build
	COMFY_MOCK=1 go run ./apps/pixoma/cmd/pixoma

run-admin-api:
	go run ./apps/admin-api/cmd/admin-api

embed-admin:
	cd web/admin && pnpm build
	rm -rf apps/pixoma/internal/webembed/dist
	cp -R web/admin/dist apps/pixoma/internal/webembed/dist
