.PHONY: build test run run-mock

build:
	go build -o bin/comfyui-bot ./apps/bot/cmd/comfyui-bot

test:
	go test ./...

run: build
	COMFY_MOCK=0 go run ./apps/bot/cmd/comfyui-bot

run-mock: build
	COMFY_MOCK=1 go run ./apps/bot/cmd/comfyui-bot
