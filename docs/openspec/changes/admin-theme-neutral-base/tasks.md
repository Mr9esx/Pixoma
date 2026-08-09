## 1. 基色迁移

- [ ] 1.1 将 `web/admin/components.json` 的 `tailwind.baseColor` 从 `slate` 改为 `neutral`
- [ ] 1.2 用 shadcn `migrate base-color`（若可用）或对照官方 neutral 预设，更新 `web/admin/src/styles/theme.css` 的 `:root` 与 `.dark` 语义 token；保留项目约定的 `--radius`、字体与 sidebar 变量映射策略

## 2. 硬编码色审计

- [ ] 2.1 检索 `web/admin` 中影响表面观感的 `slate-*`（及明显冲突的硬编码背景色），改为语义 token 类或可接受的中性写法；保留状态色（如 emerald）与无关装饰色

## 3. 验证

- [ ] 3.1 本地切换 light / dark / system，确认主背景与卡片不呈蓝灰主调，主题切换与偏好持久化仍正常
- [ ] 3.2 运行相关前端测试（至少 theme / shell 相关合同测试），必要时补充 `baseColor=neutral` 或 token 抽样断言
