# 输入字段编辑交互重构设计

日期：2026-08-20
状态：已确认（经两轮 HTML 原型讨论）

## 背景

case 的"输入字段"编辑目前是两张表单卡片（`InputFieldCard` / `OutputFieldCard`）：
节点/参数各一个下拉，参数下拉在未选节点时禁用且无说明；绑定后类型被静默改写，
对不熟悉 ComfyUI 概念的用户不透明。目标是提升"可理解性"。

## 决策历程

1. 方案 A「卡片内增强」（搜索式节点选择 + 节点摘要条 + 类型来源标注）→ 反馈：信息仍偏技术化。
2. 方案 D「图上点选绑定」：绑定弹层复用详情页横向流程图（节点+箭头），点节点→点参数完成绑定。→ 选定。
3. 类型范围的两次澄清：
   - 曾尝试收敛为「文本/图片/音频/视频」四类 → 用户放弃该限制。
   - **最终：类型不做限制（string/image/video/audio/number/boolean/enum），仅排除"引用"（节点间连线）**。

## 最终设计

### 绑定交互（InputFieldCard 重构）

```
┌──────────────────────────────────────────────┐
│ 字段名 [prompt__________]     [✓] 必填        │
│ 绑定位置                                      │
│ ┌──────────────────────────────────────────┐ │
│ │ [📄] CLIPTextEncode #6 · text      ›     │ │  ← 大按钮（未绑定时显示占位）
│ └──────────────────────────────────────────┘ │
│ 类型 [文本 ▾] · 自动·来自 CLIPTextEncode 的 text  [恢复自动] │
│ 描述 [____________________________]   [删除]  │
└──────────────────────────────────────────────┘
```

- 绑定 = 点击大按钮 → 弹层：横向流程图（与详情页 process 图同款视觉，拓扑排序）+ 参数列表。
- 弹层内：点节点 → 节点高亮 + 下方列出该节点的**字面量参数**（参数名 + 类型徽标 + 当前值预览）；
  点参数 → 底部确认「将绑定：节点 · 参数」→ 完成。
- **参数过滤**：仅排除引用（连线）参数；text、seed、steps、ckpt_name、width 等字面量均可绑。
- 绑定结果按钮显示：`节点图标 + class_type + #id · 参数名`。
- 纯连线节点（无可绑参数）→ 提示"参数都是工作流内部连线，不能作为用户输入"。

### 类型

- 选项恢复完整集合：文本/图片/音频/视频/数字/布尔/枚举（value: string/image/audio/video/number/boolean/enum）。
- 绑定后自动带出（`inputKindFor` 推断），并标注来源「自动 · 来自 X 的 y」。
- 手动改类型 → 显示琥珀色「自定义」徽标 + 「恢复自动」链接；绑定变化时不覆盖自定义值。

### 详情不折叠

- 类型/必填/描述直接平铺在绑定按钮下方，不做折叠。

### 数据层（workflow-parse 增强）

- `WorkflowNodeInput` 增加 `ref: boolean`；引用参数 kind 置 `'ref'`（`InputKind` 增加该成员）。
- `WorkflowNode` 增加：
  - `literals: Array<[string, string]>` — 字面量参数名+值（字符串化），供弹层预览。
  - `links: Array<{ name, src, slot }>` — 引用参数，供弹层拓扑排序。
- 弹层拓扑排序在组件内完成（依赖 links），与详情页 process 分层算法一致。

## 涉及文件

- `web/admin/src/features/cases/lib/node-catalog.ts` — InputKind + 'ref'
- `web/admin/src/features/cases/lib/workflow-parse.ts` — ref/literals/links
- `web/admin/src/features/cases/sections/field-cards.tsx` — InputFieldCard 重构 + 绑定弹层组件
- `web/admin/src/features/cases/sections/workflow-graph-preview.tsx` — 适配（节点列表数据来源）
- `web/admin/src/lib/i18n/locales/{zh,en}.json` — 新文案
- contract 测试同步

## 测试策略

- 契约测试：InputFieldCard 含绑定按钮与弹层、参数过滤 ref、类型"自动/自定义/恢复"标记；
  双语 i18n key 齐全。
- 单测：workflow-parse 的 ref/literals/links 派生。
- 回归：tsc -b、vitest run、vite build 全绿。

## 明确不做（YAGNI）

- 不重构输出字段绑定（本次只改输入）。
- 不做节点搜索 Combobox（图上点选已够用，节点多时可后续加过滤框）。
- 不做字段级 API schema 实时预览。
