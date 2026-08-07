Status: DONE
RED: go test ./internal/runtime/infrastructure/actuator/ -count=1 → undefined CaseSnapshot
GREEN: same command ok 0.255s
Commit: 8115c55d4f6776a93b3203c2eb08b95548ecc6b3
Files: snapshot.go, snapshot_test.go
Risk: cross-module, diff>200 lines
Notes: image type only checks binding; UploadImage deferred to task 2
