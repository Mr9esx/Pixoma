# link-health-single-source 验证报告

- Change: link-health-single-source
- Date: 2026-09-13
- 说明：沿用当前稳定代码重跑验证，未修改业务代码。

## 验证命令

- `go test ./internal/packaging/linkhealth/ ./internal/httpapi/linkhealth/ ./internal/channels/application/ ./internal/httpapi/channels/ -count=1`
- `pnpm --dir web/admin exec vitest run src/features/link-health/link-health.contract.test.ts src/features/channels/channel-layout.contract.test.ts src/lib/api/link-health.test.ts src/lib/api/channels.test.ts`

## 结果

- Go：全部通过
- Vitest：4 files / 29 tests passed
