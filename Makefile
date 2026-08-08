.PHONY: build test run run-mock run-admin-api

build:
	go build -o bin/comfyui-bot ./apps/bot/cmd/comfyui-bot
	go build -o bin/admin-api ./apps/admin-api/cmd/admin-api

test:
	go test ./...

run: build
	COMFY_MOCK=0 go run ./apps/bot/cmd/comfyui-bot

run-mock: build
	COMFY_MOCK=1 go run ./apps/bot/cmd/comfyui-bot

run-admin-api:
	go run ./apps/admin-api/cmd/admin-api
