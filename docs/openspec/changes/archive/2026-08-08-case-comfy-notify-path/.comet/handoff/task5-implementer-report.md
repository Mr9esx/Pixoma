Status: DONE
Commit: 38b0f916a4e877ef24f8824b123e2a862bab9d35
RED: photo not as Blob; text on image accepted; number/bool as Text; SendPhoto name had /
GREEN: go test ./internal/channel/tg/... PASS (10 tests)
Files: adapter.go, bot.go, adapter_test.go, bot_test.go
Risk: diff>200, external input (Telegram GetFile + HTTP)
