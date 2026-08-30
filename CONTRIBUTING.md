# 贡献指南

感谢你考虑为 Pixoma 做贡献。

## 开发环境

- Go 1.25+
- Node.js 22+
- pnpm 10+

后端和集成测试：

```bash
go build ./...
go test ./...
```

管理前端在 `web/admin` 中运行：

```bash
pnpm install
pnpm lint
pnpm test
pnpm build
pnpm audit --prod --registry=https://registry.npmjs.org
```

安全漏洞请勿通过公开 Issue 或 Pull Request 报告，参见 [`SECURITY.md`](SECURITY.md)。

## 提交要求

- 保持改动聚焦；不要把无关重构、格式化或依赖升级混入功能提交。
- 提交信息使用清晰祈使句，例如 `fix: prevent unclaimed task status mutation`。
- 不要提交真实环境配置、Token、密码、DSN、数据库文件、Blob 文件或构建产物。
- 为行为变化补充或更新测试；修复安全问题时优先给出回归测试。
- UI 优先复用 `web/admin/src/components/ui` 中的 shadcn/ui 组件与现有设计令牌。
- CI 必须通过：构建、测试、lint、依赖审计和秘密扫描。
