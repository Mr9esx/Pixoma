# fonts — 字体托管说明

Pixoma 后台的字体事实来源是 `web/admin` 的 Tailwind 配置：`Inter`（默认正文 / 标题同族）与 `Manrope`（可选展示面）。源仓库没有提交字体文件，应用通过 Google Fonts 加载；本目录预留自托管位。

## 已使用字族

| 字族 | 用途 | 栈 |
|---|---|---|
| Inter | 默认正文与标题（同族，靠字重与字号分层） | `'Inter', system-ui, sans-serif` |
| Manrope | 可选展示面（字体开关可切） | `'Manrope', 'Inter', system-ui, sans-serif` |
| 系统栈 | 兜底，不加载字体时回退 | `-apple-system, BlinkMacSystemFont, 'Segoe UI', system-ui, sans-serif` |
| Mono | 数字 / 代码 / eyebrow | `ui-monospace, 'SF Mono', Menlo, Consolas, monospace` |

## 自托管接入

需要离线自托管时，把可变字体文件放进本目录，并在 `colors_and_type.css` 前追加：

```css
@font-face {
  font-family: 'Inter';
  src: url('./inter.woff2') format('woff2');
  font-style: normal;
  font-weight: 100 900;
  font-display: swap;
}
@font-face {
  font-family: 'Manrope';
  src: url('./manrope.woff2') format('woff2');
  font-style: normal;
  font-weight: 200 800;
  font-display: swap;
}
```

保持 `--font-display` / `--font-body` / `--font-manrope` 的栈不变，页面无需改动。
