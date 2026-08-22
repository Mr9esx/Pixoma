# Admin 左栏列表筛选：核心设计原则

适用：`web/admin` 所有 Master–Detail 资源左栏（Cases / Tasks / Users / Sessions，以及后续同类页）。实例页若无筛选，不硬加。

目标：**把高度留给列表**，筛选只占「搜索 + 一行枚举」的视觉预算。

## 原则（必须遵守）

1. **列表优先**  
   左栏筛选默认高度应尽量低。禁止「每个字段 Label + 全宽控件」三层以上堆叠。

2. **能并进搜索的字段，并进一个搜索框**  
   文本/标识类条件（id、名称、tg_user_id、user_id 等）合并为单一 `q`（或等价统一搜索）。  
   Placeholder / `aria-label` 写清可搜范围；**不要**再为每个字段单独放 Input。  
   后端 `q` MUST 覆盖 UI 承诺的字段（模糊或等价匹配）；不要只改文案不改查询。

3. **枚举类筛选用横向可滑 Segment**  
   启用状态、任务/会话状态等封闭枚举：用 `FilterSegment`（内容溢出时在 **segment 边框内** 叠圆形左右箭头点击滚动，**不展示滚动条**），**不要**用占一整行的 Select + Label。  
   **少于 5 个选项**时均分铺满整行，不显示箭头；**≥5 个**才启用横向滚动与尽头箭头淡出。不要用动态左右 padding 给箭头让位。选中项变化时才滚入可见区，避免与箭头抢滚动。

4. **去掉字段 Label**  
   搜索用 placeholder + `aria-label`；Segment 用 `aria-label` / `role="group"`。可见 Label 浪费垂直空间。

5. **高级/低频条件默认收起**  
   仅当条件无法并进搜索、且非常低频时，才考虑弹出层 + chip（方案 B）。默认路径不要用。

6. **共享组件**  
   新页枚举筛选 MUST 使用 `@/components/filters/filter-segment`，不要复制一份按钮条。

## 推荐布局（自上而下）

```
[ 标题行 · 可选「新建」]
[ 搜索 Input —— 无 Label ]
[ FilterSegment —— 横向可滑 ]
[ 列表 … ]
```

间距：筛选区块用 `space-y-2`、`px-4 py-3` 即可，避免再套多层 `space-y-3` + 每字段 `space-y-1`。

## 反例

- Label「搜索」+ Input，再 Label「启用」+ Select  
- 状态 7 项做成 `grid-cols-7` 挤在一行（字不可读）  
- UI 写「可搜 tg_user_id」但后端 `q` 不查该列  

## 参考实现

- 组件：`web/admin/src/components/filters/filter-segment.tsx`  
- 范例：`web/admin/src/features/cases/list-panel.tsx`  
- OpenSpec：`docs/openspec/specs/admin-resource-pages/spec.md`（左栏紧凑筛选需求）
