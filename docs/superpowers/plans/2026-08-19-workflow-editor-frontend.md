# 工作流编辑器前端 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 Case 编辑页改成不熟悉 ComfyUI/JSON 的人也能配明白的步骤式表单：导入工作流 → 可视化配置输入/输出字段 → 自动生成 bindings 与 input_schema。

**Architecture:** 纯函数层（`workflow-parse.ts` / `node-catalog.ts` / `derive.ts`）与组件层分离；纯函数先 TDD，组件只做状态编排与渲染。`CaseForm` 持有 `workflow graph + 输入/输出 draft`，保存时用 `derive` 生成 `bindings` 与 `input_schema`。原始 JSON 只在「高级模式」（默认只读）暴露。

**Tech Stack:** React 19 + TypeScript + Vite + TanStack Router/Query + Tailwind v4 + radix-ui（Select/Dialog）+ vitest（node 环境，contract test 读源码断言）。

## Global Constraints

- 测试命令：`pnpm test`（vitest run，项目根为 `web/admin`）。
- 类型：`pnpm exec tsc -b` 必须零错误；格式：`pnpm exec prettier --write <files>`；lint：`pnpm exec eslint <files>`。
- 所有用户可见文案走 i18n，zh/en 必须成对；`web/admin/src/lib/i18n/locale.test.ts` 只校验 command-menu keys，成对性由本计划的 contract test 覆盖。
- 普通路径不出现原始 JSON 文本框；高级模式默认只读，进入编辑需二次确认。
- 代码标识（`case` / `InputBinding` / `bindings` 等）保持现状，只改 UI 文案。
- 现有 contract 测试模式（读源码字符串断言）保持一致；list panel 不得引入 `Select` 的限制仅限列表页，编辑器内可用 radix Select。

---

### Task 1.1: 工作流编辑器 i18n 文案（zh/en 成对）

**Files:**
- Create: `web/admin/src/features/cases/workflow-editor.contract.test.ts`
- Modify: `web/admin/src/lib/i18n/locales/zh.json`
- Modify: `web/admin/src/lib/i18n/locales/en.json`

**Interfaces:**
- Consumes: 无
- Produces: `cases.*` 新增 keys（`importHeading`、`inputsHeading`、`outputsHeading`、`advancedLabel`、`previewHeading`、`saveWorkflow` 等，见下方列表），后续组件任务全部通过 `t()` 引用。

- [ ] **Step 1: 写失败测试**

创建 `web/admin/src/features/cases/workflow-editor.contract.test.ts`：

```ts
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const ZH = join(here, '../../lib/i18n/locales/zh.json')
const EN = join(here, '../../lib/i18n/locales/en.json')

const REQUIRED_KEYS = [
  'importHeading',
  'importHint',
  'importDropHint',
  'importValid',
  'importNodesCount',
  'importReimport',
  'importFailed',
  'importReasonNotComfy',
  'importReasonOldExport',
  'importReasonBroken',
  'inputsHeading',
  'inputsHint',
  'outputsHeading',
  'outputsHint',
  'fieldFromNode',
  'fieldParam',
  'fieldOutput',
  'fieldEnumOptions',
  'addInput',
  'addOutput',
  'autoBoundHint',
  'singleOutputAuto',
  'nodeSearchPlaceholder',
  'advancedLabel',
  'advancedReadonlyHint',
  'advancedEnterEdit',
  'advancedOverwriteWarn',
  'previewHeading',
  'previewHint',
  'emptyWorkflowLock',
  'saveWorkflow',
  'errDuplicateKey',
  'errInputNotBound',
  'errNoOutput',
] as const

function read(path: string) {
  return JSON.parse(readFileSync(path, 'utf8')) as {
    cases: Record<string, string>
  }
}

describe('workflow editor i18n', () => {
  it('zh and en define every editor key', () => {
    const zh = read(ZH)
    const en = read(EN)
    for (const key of REQUIRED_KEYS) {
      expect(zh.cases[key], `zh missing cases.${key}`).toBeTruthy()
      expect(en.cases[key], `en missing cases.${key}`).toBeTruthy()
    }
  })
})
```

- [ ] **Step 2: 运行确认失败**

Run: `cd web/admin && pnpm test`
Expected: `workflow editor i18n` FAIL（`zh missing cases.importHeading`）。

- [ ] **Step 3: 添加 zh/en 文案**

在 `zh.json` 的 `cases` 段末尾（`menuPlacementsEmpty` 后）添加：

```json
    "importHeading": "导入工作流",
    "importHint": "从 ComfyUI 导出的 JSON 文件或文本",
    "importDropHint": "拖拽 JSON 文件到这里，或选择文件 / 粘贴 JSON",
    "importValid": "校验通过",
    "importNodesCount": "已识别 {{count}} 个节点 · {{inputs}} 个可调输入 · {{outputs}} 个输出",
    "importReimport": "重新导入",
    "importFailed": "校验失败",
    "importReasonNotComfy": "文件不是 JSON，或不是从 ComfyUI 导出的",
    "importReasonOldExport": "用的是旧版导出（缺少 API 格式字段）",
    "importReasonBroken": "JSON 被手动改坏（多了逗号/括号不匹配）",
    "inputsHeading": "用户需要提供什么",
    "inputsHint": "用户使用时按顺序被问到这些内容",
    "outputsHeading": "用户会得到什么",
    "outputsHint": "用户完成后收到这些结果",
    "fieldFromNode": "来自节点",
    "fieldParam": "参数",
    "fieldOutput": "输出",
    "fieldEnumOptions": "选项（逗号分隔）",
    "addInput": "添加一个输入",
    "addOutput": "添加一个输出",
    "autoBoundHint": "节点和参数是从导入的工作流自动读出来的，不需要手填任何 ID",
    "singleOutputAuto": "该节点只有 1 个输出，已自动选好",
    "nodeSearchPlaceholder": "搜索节点…",
    "advancedLabel": "高级",
    "advancedReadonlyHint": "只读；进入编辑需二次确认",
    "advancedEnterEdit": "进入编辑",
    "advancedOverwriteWarn": "手动修改后，表单重新保存会覆盖此处",
    "previewHeading": "预览（自动生成，只读）",
    "previewHint": "绑定关系与校验规则都由上面的表单自动生成；需要核对或复制时再展开这里。",
    "emptyWorkflowLock": "先导入工作流，这里才会开放",
    "saveWorkflow": "保存工作流",
    "errDuplicateKey": "字段名不能重复",
    "errInputNotBound": "必填输入需要选择节点和参数",
    "errNoOutput": "至少需要一个输出"
```

`en.json` 同位置对应英文：

```json
    "importHeading": "Import workflow",
    "importHint": "From a JSON file exported by ComfyUI",
    "importDropHint": "Drop a JSON file here, or choose file / paste JSON",
    "importValid": "Valid",
    "importNodesCount": "{{count}} nodes · {{inputs}} adjustable inputs · {{outputs}} outputs",
    "importReimport": "Re-import",
    "importFailed": "Validation failed",
    "importReasonNotComfy": "Not JSON, or not exported from ComfyUI",
    "importReasonOldExport": "Old export format (missing API fields)",
    "importReasonBroken": "JSON was edited by hand and is malformed",
    "inputsHeading": "What the user needs to provide",
    "inputsHint": "Users are asked these in order",
    "outputsHeading": "What the user gets back",
    "outputsHint": "Delivered when the run finishes",
    "fieldFromNode": "From node",
    "fieldParam": "Parameter",
    "fieldOutput": "Output",
    "fieldEnumOptions": "Options (comma-separated)",
    "addInput": "Add an input",
    "addOutput": "Add an output",
    "autoBoundHint": "Nodes and parameters are read from the imported workflow — no IDs needed",
    "singleOutputAuto": "This node has a single output; already selected",
    "nodeSearchPlaceholder": "Search nodes…",
    "advancedLabel": "Advanced",
    "advancedReadonlyHint": "Read-only; editing requires confirmation",
    "advancedEnterEdit": "Enter edit mode",
    "advancedOverwriteWarn": "Saving the form will overwrite manual edits here",
    "previewHeading": "Preview (auto-generated, read-only)",
    "previewHint": "Bindings and validation rules are generated from the form above.",
    "emptyWorkflowLock": "Import a workflow first to unlock this",
    "saveWorkflow": "Save workflow",
    "errDuplicateKey": "Field keys must be unique",
    "errInputNotBound": "Required inputs must pick a node and parameter",
    "errNoOutput": "At least one output is required"
```

- [ ] **Step 4: 运行确认通过**

Run: `cd web/admin && pnpm test`
Expected: 全部通过（30 文件 + 新增 contract 文件）。

- [ ] **Step 5: 格式化并提交**

```bash
cd web/admin
pnpm exec prettier --write src/features/cases/workflow-editor.contract.test.ts src/lib/i18n/locales/zh.json src/lib/i18n/locales/en.json
cd /Users/mr9esx/Documents/Pixoma
git add web/admin/src/features/cases/workflow-editor.contract.test.ts web/admin/src/lib/i18n/locales/zh.json web/admin/src/lib/i18n/locales/en.json
git commit -m "feat(admin): add workflow editor i18n copy"
```

---

### Task 1.2: 节点目录（中文别名 / 输入类型推断 / 输出数）

**Files:**
- Create: `web/admin/src/features/cases/lib/node-catalog.ts`
- Create: `web/admin/src/features/cases/lib/node-catalog.test.ts`

**Interfaces:**
- Consumes: 无
- Produces:
  - `export const NODE_LABELS: Record<string, string>`
  - `export function nodeLabel(classType: string): string`（未知节点返回原始 classType）
  - `export const NODE_OUTPUT_COUNTS: Record<string, number>`
  - `export function outputCountFor(classType: string): number`（未知默认 1）
  - `export const NODE_INPUT_KINDS: Record<string, Record<string, string>>`
  - `export function inputKindFor(classType: string, field: string): InputKind`（未知返回 `'unknown'`）

在 `node-catalog.ts` 顶部定义共享类型：

```ts
export type InputKind =
  | 'string'
  | 'number'
  | 'boolean'
  | 'enum'
  | 'image'
  | 'unknown'
```

- [ ] **Step 1: 写失败测试**

创建 `node-catalog.test.ts`：

```ts
import { describe, expect, it } from 'vitest'
import {
  inputKindFor,
  nodeLabel,
  outputCountFor,
} from './node-catalog'

describe('node catalog', () => {
  it('labels common nodes in Chinese and falls back to class type', () => {
    expect(nodeLabel('LoadImage')).toBe('加载图片')
    expect(nodeLabel('CLIPTextEncode')).toBe('写提示词')
    expect(nodeLabel('TotallyUnknownNode')).toBe('TotallyUnknownNode')
  })

  it('infers input kinds for known nodes and returns unknown otherwise', () => {
    expect(inputKindFor('LoadImage', 'image')).toBe('image')
    expect(inputKindFor('KSampler', 'seed')).toBe('number')
    expect(inputKindFor('KSampler', 'scheduler')).toBe('enum')
    expect(inputKindFor('CLIPTextEncode', 'text')).toBe('string')
    expect(inputKindFor('RandomNode', 'anything')).toBe('unknown')
  })

  it('defaults output count to 1 for unknown nodes', () => {
    expect(outputCountFor('SaveImage')).toBe(1)
    expect(outputCountFor('SomethingElse')).toBe(1)
  })
})
```

- [ ] **Step 2: 运行确认失败**

Run: `cd web/admin && pnpm test src/features/cases/lib/node-catalog.test.ts`
Expected: FAIL（`nodeLabel` not exported / import error）。

- [ ] **Step 3: 实现 node-catalog.ts**

```ts
export type InputKind =
  | 'string'
  | 'number'
  | 'boolean'
  | 'enum'
  | 'image'
  | 'unknown'

export const NODE_LABELS: Record<string, string> = {
  LoadImage: '加载图片',
  CLIPTextEncode: '写提示词',
  KSampler: '采样',
  SaveImage: '保存图片',
  EmptyLatentImage: '空白画布',
  VAELoader: '加载 VAE',
  UNETLoader: '加载模型',
  CLIPLoader: '加载 CLIP',
  CheckpointLoaderSimple: '加载模型',
  LoraLoader: '加载 LoRA',
  VAEDecode: '解码图像',
  VAEEncode: '编码图像',
  PreviewImage: '预览图像',
}

export function nodeLabel(classType: string): string {
  return NODE_LABELS[classType] ?? classType
}

export const NODE_OUTPUT_COUNTS: Record<string, number> = {
  LoadImage: 1,
  SaveImage: 1,
  KSampler: 1,
  VAEDecode: 1,
  VAEEncode: 1,
  EmptyLatentImage: 1,
  CLIPTextEncode: 1,
  PreviewImage: 1,
}

export function outputCountFor(classType: string): number {
  return NODE_OUTPUT_COUNTS[classType] ?? 1
}

export const NODE_INPUT_KINDS: Record<string, Record<string, string>> = {
  LoadImage: { image: 'image' },
  CLIPTextEncode: { text: 'string' },
  KSampler: {
    seed: 'number',
    steps: 'number',
    cfg: 'number',
    sampler_name: 'enum',
    scheduler: 'enum',
    denoise: 'number',
  },
  EmptyLatentImage: {
    width: 'number',
    height: 'number',
    batch_size: 'number',
  },
  SaveImage: { filename_prefix: 'string' },
}

export function inputKindFor(classType: string, field: string): InputKind {
  const kind = NODE_INPUT_KINDS[classType]?.[field]
  if (kind === undefined) return 'unknown'
  return kind as InputKind
}
```

- [ ] **Step 4: 运行确认通过**

Run: `cd web/admin && pnpm test src/features/cases/lib/node-catalog.test.ts`
Expected: PASS。

- [ ] **Step 5: 格式化并提交**

```bash
cd web/admin
pnpm exec prettier --write src/features/cases/lib/node-catalog.ts src/features/cases/lib/node-catalog.test.ts
cd /Users/mr9esx/Documents/Pixoma
git add web/admin/src/features/cases/lib/node-catalog.ts web/admin/src/features/cases/lib/node-catalog.test.ts
git commit -m "feat(admin): add comfy node catalog"
```

---

### Task 1.3: 工作流 JSON 解析器（API / UI 格式 + 校验）

**Files:**
- Create: `web/admin/src/features/cases/lib/workflow-parse.ts`
- Create: `web/admin/src/features/cases/lib/workflow-parse.test.ts`

**Interfaces:**
- Consumes: `node-catalog.ts` 的 `InputKind` / `inputKindFor` / `outputCountFor`
- Produces:
  - `export type WorkflowNodeInput = { name: string; kind: InputKind }`
  - `export type WorkflowNode = { id: string; class_type: string; inputs: WorkflowNodeInput[]; outputCount: number }`
  - `export type WorkflowGraph = { nodes: WorkflowNode[]; api: Record<string, unknown> }`
  - `export type WorkflowParseResult = { ok: true; graph: WorkflowGraph } | { ok: false; error: string }`
  - `export function parseWorkflow(raw: string): WorkflowParseResult`

- [ ] **Step 1: 写失败测试**

创建 `workflow-parse.test.ts`：

```ts
import { describe, expect, it } from 'vitest'
import { parseWorkflow } from './workflow-parse'

const API_JSON = JSON.stringify({
  '1': { class_type: 'LoadImage', inputs: { image: 'ref.png' } },
  '2': { class_type: 'CLIPTextEncode', inputs: { text: 'hi' } },
  '3': { class_type: 'SaveImage', inputs: { filename_prefix: 'out' } },
})

const UI_JSON = JSON.stringify({
  nodes: [
    { id: 1, type: 'LoadImage', inputs: [{ name: 'image', link: null }], widgets_values: ['ref.png'] },
    { id: 2, type: 'CLIPTextEncode', inputs: [{ name: 'text', link: null }], widgets_values: ['hi'] },
    { id: 3, type: 'SaveImage', inputs: [{ name: 'filename_prefix', link: null }], widgets_values: ['out'] },
  ],
  links: [],
})

describe('parseWorkflow', () => {
  it('parses ComfyUI API format', () => {
    const result = parseWorkflow(API_JSON)
    expect(result.ok).toBe(true)
    if (!result.ok) return
    expect(result.graph.nodes).toHaveLength(3)
    expect(result.graph.nodes[0]).toMatchObject({
      id: '1',
      class_type: 'LoadImage',
      outputCount: 1,
    })
    expect(result.graph.nodes[0].inputs[0]).toEqual({
      name: 'image',
      kind: 'image',
    })
  })

  it('converts ComfyUI UI format to API format', () => {
    const result = parseWorkflow(UI_JSON)
    expect(result.ok).toBe(true)
    if (!result.ok) return
    expect(result.graph.api['1']).toEqual({
      class_type: 'LoadImage',
      inputs: { image: 'ref.png' },
    })
  })

  it('keeps connected inputs as node links in API format', () => {
    const raw = JSON.stringify({
      nodes: [
        { id: 1, type: 'LoadImage', inputs: [{ name: 'image', link: 10 }], widgets_values: [] },
        { id: 2, type: 'SaveImage', inputs: [{ name: 'images', link: 11 }], widgets_values: [] },
      ],
      links: [
        [10, 1, 0, 2, 0, 'IMAGE'],
        [11, 1, 0, 2, 0, 'IMAGE'],
      ],
    })
    const result = parseWorkflow(raw)
    expect(result.ok).toBe(true)
    if (!result.ok) return
    expect(result.graph.api['2']).toEqual({
      class_type: 'SaveImage',
      inputs: { images: ['1', 0] },
    })
  })

  it('returns readable errors for invalid input', () => {
    expect(parseWorkflow('not json').ok).toBe(false)
    expect(parseWorkflow(JSON.stringify([1, 2])).ok).toBe(false)
    expect(
      parseWorkflow(JSON.stringify({ nodes: [{ id: 1 }] })).ok,
    ).toBe(false)
    expect(parseWorkflow(JSON.stringify({})).ok).toBe(false)
  })
})
```

- [ ] **Step 2: 运行确认失败**

Run: `cd web/admin && pnpm test src/features/cases/lib/workflow-parse.test.ts`
Expected: FAIL（模块不存在）。

- [ ] **Step 3: 实现 workflow-parse.ts**

```ts
import { inputKindFor, outputCountFor, type InputKind } from './node-catalog'

export type WorkflowNodeInput = { name: string; kind: InputKind }
export type WorkflowNode = {
  id: string
  class_type: string
  inputs: WorkflowNodeInput[]
  outputCount: number
}
export type WorkflowGraph = { nodes: WorkflowNode[]; api: Record<string, unknown> }
export type WorkflowParseResult =
  | { ok: true; graph: WorkflowGraph }
  | { ok: false; error: string }

type UiLink = [number, number, number, number, number, string]

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v)
}

function isApiNode(v: unknown): v is Record<string, unknown> {
  return (
    isRecord(v) &&
    typeof v['class_type'] === 'string' &&
    (v['inputs'] === undefined || isRecord(v['inputs']))
  )
}

function isApiGraph(value: unknown): value is Record<string, unknown> {
  if (!isRecord(value)) return false
  const entries = Object.values(value)
  return entries.length > 0 && entries.every(isApiNode)
}

function apiInputsToNames(inputs: Record<string, unknown>): WorkflowNodeInput[] {
  return Object.entries(inputs).map(([name, value]) => ({
    name,
    kind: typeof value === 'string' ? 'string' : 'unknown',
  }))
}

function convertUiToApi(
  value: Record<string, unknown>,
): { ok: true; api: Record<string, unknown> } | { ok: false; error: string } {
  const nodes = value['nodes']
  if (!Array.isArray(nodes) || nodes.length === 0) {
    return { ok: false, error: 'nodes array is missing or empty' }
  }
  const linksRaw = Array.isArray(value['links']) ? value['links'] : []
  const links = new Map<number, UiLink>()
  for (const link of linksRaw) {
    if (Array.isArray(link) && typeof link[0] === 'number') {
      links.set(link[0], link as UiLink)
    }
  }
  const api: Record<string, unknown> = {}
  for (const [index, node] of nodes.entries()) {
    if (!isRecord(node)) return { ok: false, error: `第 ${index + 1} 个节点格式不正确` }
    const id = String(node['id'])
    const classType = node['type']
    if (!id || typeof classType !== 'string') {
      return { ok: false, error: `第 ${index + 1} 个节点缺少类型信息，无法识别` }
    }
    const rawInputs = Array.isArray(node['inputs']) ? node['inputs'] : []
    const widgets = Array.isArray(node['widgets_values']) ? node['widgets_values'] : []
    let widgetIndex = 0
    const inputs: Record<string, unknown> = {}
    for (const input of rawInputs) {
      if (!isRecord(input) || typeof input['name'] !== 'string') continue
      const name = input['name']
      const linkId = input['link']
      if (typeof linkId === 'number' && links.has(linkId)) {
        const link = links.get(linkId)!
        inputs[name] = [String(link[1]), link[2]]
      } else if (linkId === null && widgetIndex < widgets.length) {
        inputs[name] = widgets[widgetIndex]
        widgetIndex += 1
      }
    }
    for (; widgetIndex < widgets.length; widgetIndex += 1) {
      inputs[`widget_${widgetIndex}`] = widgets[widgetIndex]
    }
    api[id] = { class_type: classType, inputs }
  }
  return { ok: true, api }
}

export function parseWorkflow(raw: string): WorkflowParseResult {
  let parsed: unknown
  try {
    parsed = JSON.parse(raw)
  } catch {
    return { ok: false, error: '不是有效的 JSON' }
  }
  if (!isRecord(parsed)) return { ok: false, error: 'JSON 顶层必须是对象' }

  let api: Record<string, unknown>
  if (isApiGraph(parsed)) {
    api = parsed
  } else if (Array.isArray(parsed['nodes'])) {
    const converted = convertUiToApi(parsed)
    if (!converted.ok) return { ok: false, error: converted.error }
    api = converted.api
  } else {
    return { ok: false, error: '不是 ComfyUI 导出的工作流' }
  }

  const nodes: WorkflowNode[] = Object.entries(api).map(([id, rawNode]) => {
    const node = rawNode as Record<string, unknown>
    const classType = String(node['class_type'])
    const inputs = isRecord(node['inputs'])
      ? apiInputsToNames(node['inputs']).map((input) => ({
          ...input,
          kind: inputKindFor(classType, input.name),
        }))
      : []
    return {
      id,
      class_type: classType,
      inputs,
      outputCount: outputCountFor(classType),
    }
  })
  return { ok: true, graph: { nodes, api } }
}
```

- [ ] **Step 4: 运行确认通过**

Run: `cd web/admin && pnpm test src/features/cases/lib/workflow-parse.test.ts`
Expected: PASS。

- [ ] **Step 5: 格式化并提交**

```bash
cd web/admin
pnpm exec prettier --write src/features/cases/lib/workflow-parse.ts src/features/cases/lib/workflow-parse.test.ts
cd /Users/mr9esx/Documents/Pixoma
git add web/admin/src/features/cases/lib/workflow-parse.ts web/admin/src/features/cases/lib/workflow-parse.test.ts
git commit -m "feat(admin): parse and validate comfy workflows"
```

---

### Task 1.4: bindings / input_schema 自动生成 + 保存校验

**Files:**
- Create: `web/admin/src/features/cases/lib/derive.ts`
- Create: `web/admin/src/features/cases/lib/derive.test.ts`

**Interfaces:**
- Consumes: `workflow-parse.ts` 类型（仅 `WorkflowNode` 用于节点下拉校验，可选）
- Produces:
  - `export type InputFieldDraft = { key: string; type: string; required: boolean; node_id: string; field_path: string; description?: string; enum_values?: string[] }`
  - `export type OutputFieldDraft = { key: string; type: string; node_id: string; index?: number; description?: string }`
  - `export function deriveInputSchema(inputs: InputFieldDraft[]): Record<string, unknown>`
  - `export function deriveBindings(inputs: InputFieldDraft[], outputs: OutputFieldDraft[]): { inputs: InputBinding[]; outputs: OutputBinding[] }`
  - `export function validateEditor(inputs: InputFieldDraft[], outputs: OutputFieldDraft[]): { duplicateKey?: string; unboundRequired?: string; noOutput?: boolean }`
  - 返回的 `InputBinding` / `OutputBinding` 与 `@/lib/api/types` 同名类型兼容（`{ key; node_id; field_path }` / `{ key; node_id; index? }`）。

- [ ] **Step 1: 写失败测试**

创建 `derive.test.ts`：

```ts
import { describe, expect, it } from 'vitest'
import {
  deriveBindings,
  deriveInputSchema,
  validateEditor,
  type InputFieldDraft,
  type OutputFieldDraft,
} from './derive'

const inputs: InputFieldDraft[] = [
  { key: 'reference', type: 'image', required: true, node_id: '1', field_path: 'image' },
  { key: 'prompt', type: 'string', required: true, node_id: '2', field_path: 'text' },
  { key: 'style', type: 'enum', required: false, node_id: '2', field_path: 'style', enum_values: ['anime', 'photo'] },
  { key: 'seed', type: 'number', required: false, node_id: '3', field_path: 'seed' },
]

const outputs: OutputFieldDraft[] = [
  { key: 'image', type: 'image', node_id: '9', index: 0 },
]

describe('derive', () => {
  it('derives bindings from field drafts', () => {
    expect(deriveBindings(inputs, outputs)).toEqual({
      inputs: [
        { key: 'reference', node_id: '1', field_path: 'image' },
        { key: 'prompt', node_id: '2', field_path: 'text' },
        { key: 'style', node_id: '2', field_path: 'style' },
        { key: 'seed', node_id: '3', field_path: 'seed' },
      ],
      outputs: [{ key: 'image', node_id: '9', index: 0 }],
    })
  })

  it('derives input_schema with types, enums and required', () => {
    expect(deriveInputSchema(inputs)).toEqual({
      type: 'object',
      additionalProperties: false,
      required: ['reference', 'prompt'],
      properties: {
        reference: { type: 'string' },
        prompt: { type: 'string' },
        style: { type: 'string', enum: ['anime', 'photo'] },
        seed: { type: 'number' },
      },
    })
  })

  it('drops empty keys from schema and bindings', () => {
    const withEmpty = [{ ...inputs[0], key: '' }]
    expect(deriveInputSchema(withEmpty)).toEqual({
      type: 'object',
      additionalProperties: false,
      required: [],
      properties: {},
    })
    expect(deriveBindings(withEmpty, []).inputs).toEqual([])
  })

  it('validates duplicates, unbound required inputs and missing outputs', () => {
    const bad: InputFieldDraft[] = [
      { key: 'x', type: 'string', required: true, node_id: '', field_path: '' },
      { key: 'x', type: 'string', required: false, node_id: '1', field_path: 'a' },
    ]
    expect(validateEditor(bad, [])).toEqual({
      duplicateKey: 'x',
      unboundRequired: 'x',
      noOutput: true,
    })
    expect(validateEditor([inputs[0]], outputs)).toEqual({})
  })
})
```

- [ ] **Step 2: 运行确认失败**

Run: `cd web/admin && pnpm test src/features/cases/lib/derive.test.ts`
Expected: FAIL（模块不存在）。

- [ ] **Step 3: 实现 derive.ts**

```ts
import type { InputBinding, OutputBinding } from '@/lib/api/types'

export type InputFieldDraft = {
  key: string
  type: string
  required: boolean
  node_id: string
  field_path: string
  description?: string
  enum_values?: string[]
}

export type OutputFieldDraft = {
  key: string
  type: string
  node_id: string
  index?: number
  description?: string
}

function schemaTypeFor(field: InputFieldDraft): unknown {
  switch (field.type) {
    case 'number':
      return { type: 'number' }
    case 'boolean':
      return { type: 'boolean' }
    case 'enum':
      return {
        type: 'string',
        ...(field.enum_values?.length ? { enum: field.enum_values } : {}),
      }
    case 'image':
    case 'video':
      return { type: 'string' }
    default:
      return { type: 'string' }
  }
}

export function deriveInputSchema(
  inputs: InputFieldDraft[],
): Record<string, unknown> {
  const properties: Record<string, unknown> = {}
  const required: string[] = []
  for (const field of inputs) {
    const key = field.key.trim()
    if (!key) continue
    properties[key] = schemaTypeFor(field)
    if (field.required) required.push(key)
  }
  return {
    type: 'object',
    additionalProperties: false,
    required,
    properties,
  }
}

export function deriveBindings(
  inputs: InputFieldDraft[],
  outputs: OutputFieldDraft[],
): { inputs: InputBinding[]; outputs: OutputBinding[] } {
  return {
    inputs: inputs
      .filter((field) => field.key.trim() && field.node_id && field.field_path)
      .map((field) => ({
        key: field.key.trim(),
        node_id: field.node_id,
        field_path: field.field_path,
      })),
    outputs: outputs
      .filter((field) => field.key.trim() && field.node_id)
      .map((field) => ({
        key: field.key.trim(),
        node_id: field.node_id,
        index: field.index ?? 0,
      })),
  }
}

export function validateEditor(
  inputs: InputFieldDraft[],
  outputs: OutputFieldDraft[],
): { duplicateKey?: string; unboundRequired?: string; noOutput?: boolean } {
  const seen = new Set<string>()
  for (const field of inputs) {
    const key = field.key.trim()
    if (!key) continue
    if (seen.has(key)) return { duplicateKey: key }
    seen.add(key)
    if (field.required && (!field.node_id || !field.field_path)) {
      return { unboundRequired: key }
    }
  }
  if (outputs.filter((field) => field.key.trim() && field.node_id).length === 0) {
    return { noOutput: true }
  }
  return {}
}
```

- [ ] **Step 4: 运行确认通过**

Run: `cd web/admin && pnpm test src/features/cases/lib/derive.test.ts`
Expected: PASS。

- [ ] **Step 5: 格式化并提交**

```bash
cd web/admin
pnpm exec prettier --write src/features/cases/lib/derive.ts src/features/cases/lib/derive.test.ts
cd /Users/mr9esx/Documents/Pixoma
git add web/admin/src/features/cases/lib/derive.ts web/admin/src/features/cases/lib/derive.test.ts
git commit -m "feat(admin): derive bindings and input schema from editor"
```

---

### Task 1.5: 导入区组件 WorkflowImportSection

**Files:**
- Create: `web/admin/src/features/cases/sections/workflow-import.tsx`
- Modify: `web/admin/src/features/cases/workflow-editor.contract.test.ts`（追加断言）

**Interfaces:**
- Consumes: `parseWorkflow` / `WorkflowGraph` / `WorkflowParseResult`；i18n keys（Task 1.1）
- Produces:
  - `export function WorkflowImportSection(props: { value: string; graph?: WorkflowGraph; error?: string; onChange: (next: string) => void; disabled?: boolean }): React.JSX.Element`
  - `data-testid='case-section-workflow-import'`；校验成功横幅 `data-testid='workflow-import-valid'`；失败横幅 `data-testid='workflow-import-error'`

- [ ] **Step 1: 追加失败测试（contract）**

在 `workflow-editor.contract.test.ts` 追加：

```ts
const IMPORT_SECTION = join(here, 'sections/workflow-import.tsx')

describe('workflow import section', () => {
  it('renders import UI and uses parseWorkflow', () => {
    const source = readFileSync(IMPORT_SECTION, 'utf8')
    expect(source).toContain("data-testid='case-section-workflow-import'")
    expect(source).toContain("parseWorkflow(")
    expect(source).toContain("cases.importNodesCount")
    expect(source).toContain("cases.importValid")
    expect(source).toContain("cases.importFailed")
  })
})
```

同时把 `read` 改为可读源码字符串（现有 `read` 是 JSON.parse，不能用于 tsx）：直接改用 `readFileSync(IMPORT_SECTION, 'utf8')`。

- [ ] **Step 2: 运行确认失败**

Run: `cd web/admin && pnpm test src/features/cases/workflow-editor.contract.test.ts`
Expected: FAIL（文件不存在）。

- [ ] **Step 3: 实现 workflow-import.tsx**

```tsx
import { useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { parseWorkflow } from '../lib/workflow-parse'
import type { WorkflowGraph } from '../lib/workflow-parse'

type Props = {
  value: string
  graph?: WorkflowGraph
  error?: string
  onChange: (next: string) => void
  disabled?: boolean
}

export function WorkflowImportSection({
  value,
  graph,
  error,
  onChange,
  disabled,
}: Props) {
  const { t } = useTranslation()
  const fileRef = useRef<HTMLInputElement>(null)

  async function onFile(file: File | undefined) {
    if (!file) return
    onChange(await file.text())
  }

  return (
    <section className='space-y-3' data-testid='case-section-workflow-import'>
      <div>
        <h3 className='text-sm font-semibold'>{t('cases.importHeading')}</h3>
        <p className='text-muted-foreground text-xs'>{t('cases.importHint')}</p>
      </div>

      <div
        className='rounded-md border border-dashed p-4 text-center text-sm text-muted-foreground'
        onDragOver={(e) => e.preventDefault()}
        onDrop={(e) => {
          e.preventDefault()
          void onFile(e.dataTransfer.files[0])
        }}
      >
        <p>{t('cases.importDropHint')}</p>
        <input
          ref={fileRef}
          type='file'
          accept='.json,application/json'
          className='hidden'
          onChange={(e) => void onFile(e.target.files?.[0])}
        />
      </div>

      {graph ? (
        <div
          data-testid='workflow-import-valid'
          className='rounded-md border border-emerald-600/30 bg-emerald-50 px-3 py-2 text-sm text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-400'
        >
          {t('cases.importValid')} ·{' '}
          {t('cases.importNodesCount', {
            count: graph.nodes.length,
            inputs: graph.nodes.reduce(
              (sum, node) => sum + node.inputs.length,
              0,
            ),
            outputs: graph.nodes.reduce(
              (sum, node) => sum + node.outputCount,
              0,
            ),
          })}
        </div>
      ) : null}

      {error ? (
        <div
          data-testid='workflow-import-error'
          className='rounded-md border border-red-600/30 bg-red-50 px-3 py-2 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400'
        >
          <p className='font-medium'>{t('cases.importFailed')}</p>
          <p>{error}</p>
        </div>
      ) : null}

      <Label htmlFor='case-workflow-json' className='sr-only'>
        {t('cases.importHeading')}
      </Label>
      <Textarea
        id='case-workflow-json'
        value={value}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
        rows={6}
        className='font-mono text-xs'
        placeholder={t('cases.importDropHint')}
      />
    </section>
  )
}

export function parseRawWorkflow(raw: string) {
  return parseWorkflow(raw)
}
```

> 说明：`parseRawWorkflow` 是为 contract test 的 `parseWorkflow(` 断言保留的再导出；组件内部也可直接使用。

- [ ] **Step 4: 运行确认通过**

Run: `cd web/admin && pnpm test src/features/cases/workflow-editor.contract.test.ts && pnpm exec tsc -b`
Expected: PASS；tsc 零错误。

- [ ] **Step 5: 格式化并提交**

```bash
cd web/admin
pnpm exec prettier --write src/features/cases/sections/workflow-import.tsx src/features/cases/workflow-editor.contract.test.ts
cd /Users/mr9esx/Documents/Pixoma
git add web/admin/src/features/cases/sections/workflow-import.tsx web/admin/src/features/cases/workflow-editor.contract.test.ts
git commit -m "feat(admin): workflow import section"
```

---

### Task 1.6: 输入/输出字段卡片组件（节点下拉 + 参数选择）

**Files:**
- Create: `web/admin/src/features/cases/sections/field-cards.tsx`
- Modify: `web/admin/src/features/cases/workflow-editor.contract.test.ts`（追加断言）

**Interfaces:**
- Consumes: `WorkflowNode`（`workflow-parse.ts`）、`InputFieldDraft` / `OutputFieldDraft`（`derive.ts`）、`nodeLabel` / `inputKindFor`（`node-catalog.ts`）、i18n keys
- Produces:
  - `export function InputFieldCard(props: { nodes: WorkflowNode[]; value: InputFieldDraft; onChange: (next: InputFieldDraft) => void; onRemove: () => void; disabled?: boolean }): React.JSX.Element`
  - `export function OutputFieldCard(props: { nodes: WorkflowNode[]; value: OutputFieldDraft; onChange: (next: OutputFieldDraft) => void; onRemove: () => void; disabled?: boolean }): React.JSX.Element`
  - `data-testid='input-field-card'` / `data-testid='output-field-card'`

- [ ] **Step 1: 追加失败测试（contract）**

```ts
const FIELD_CARDS = join(here, 'sections/field-cards.tsx')

describe('field cards', () => {
  it('input card picks node then parameter from workflow nodes', () => {
    const source = readFileSync(FIELD_CARDS, 'utf8')
    expect(source).toContain("data-testid='input-field-card'")
    expect(source).toContain("cases.fieldFromNode")
    expect(source).toContain("cases.fieldParam")
    expect(source).toContain('nodeLabel(')
    expect(source).toContain('nodes.find(')
  })

  it('output card hides index when node has a single output', () => {
    const source = readFileSync(FIELD_CARDS, 'utf8')
    expect(source).toContain("data-testid='output-field-card'")
    expect(source).toContain('value.outputCount')
    expect(source).toContain('cases.fieldOutput')
  })
})
```

- [ ] **Step 2: 运行确认失败**

Run: `cd web/admin && pnpm test src/features/cases/workflow-editor.contract.test.ts`
Expected: FAIL。

- [ ] **Step 3: 实现 field-cards.tsx**

```tsx
import { useTranslation } from 'react-i18next'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { WorkflowNode } from '../lib/workflow-parse'
import { inputKindFor, nodeLabel } from '../lib/node-catalog'
import type { InputFieldDraft, OutputFieldDraft } from '../lib/derive'

const INPUT_TYPES = ['string', 'image', 'video', 'number', 'boolean', 'enum']
const OUTPUT_TYPES = ['image', 'text', 'file']

type InputCardProps = {
  nodes: WorkflowNode[]
  value: InputFieldDraft
  onChange: (next: InputFieldDraft) => void
  onRemove: () => void
  disabled?: boolean
}

export function InputFieldCard({
  nodes,
  value,
  onChange,
  onRemove,
  disabled,
}: InputCardProps) {
  const { t } = useTranslation()
  const node = nodes.find((n) => n.id === value.node_id)

  return (
    <li data-testid='input-field-card' className='space-y-2 rounded-md border p-3'>
      <div className='flex flex-wrap items-center gap-2'>
        <span className='text-xs text-muted-foreground'>{t('cases.fieldKey')}</span>
        <Input
          className='h-8 w-36'
          value={value.key}
          onChange={(e) => onChange({ ...value, key: e.target.value })}
          disabled={disabled}
          autoComplete='off'
        />
        <span className='text-xs text-muted-foreground'>{t('cases.fieldType')}</span>
        <Select
          value={value.type}
          onValueChange={(type) =>
            onChange({ ...value, type: type as InputFieldDraft['type'] })
          }
          disabled={disabled}
        >
          <SelectTrigger className='h-8 w-28'>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {INPUT_TYPES.map((type) => (
              <SelectItem key={type} value={type}>
                {type}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <label className='ml-auto flex items-center gap-1.5 text-xs'>
          <Checkbox
            checked={value.required}
            onCheckedChange={(v) => onChange({ ...value, required: v === true })}
            disabled={disabled}
          />
          {t('cases.fieldRequired')}
        </label>
      </div>

      <div className='flex flex-wrap items-center gap-2'>
        <span className='text-xs text-muted-foreground'>{t('cases.fieldFromNode')}</span>
        <Select
          value={value.node_id || undefined}
          onValueChange={(nodeId) =>
            onChange({
              ...value,
              node_id: nodeId,
              field_path: '',
            })
          }
          disabled={disabled}
        >
          <SelectTrigger className='h-8 w-56'>
            <SelectValue placeholder={t('cases.nodeSearchPlaceholder')} />
          </SelectTrigger>
          <SelectContent>
            {nodes.map((n) => (
              <SelectItem key={n.id} value={n.id}>
                {nodeLabel(n.class_type)}（{n.id}）
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <span className='text-xs text-muted-foreground'>{t('cases.fieldParam')}</span>
        <Select
          value={value.field_path || undefined}
          onValueChange={(fieldPath) =>
            onChange({
              ...value,
              field_path: fieldPath,
              type:
                value.type === 'string' || value.type === 'image'
                  ? value.type
                  : inputKindFor(node?.class_type ?? '', fieldPath),
            })
          }
          disabled={disabled || !node}
        >
          <SelectTrigger className='h-8 w-48'>
            <SelectValue
              placeholder={node ? t('cases.fieldParam') : t('cases.emptyWorkflowLock')}
            />
          </SelectTrigger>
          <SelectContent>
            {node?.inputs.map((input) => (
              <SelectItem key={input.name} value={input.name}>
                {input.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {value.type === 'enum' ? (
        <div className='flex items-center gap-2'>
          <span className='text-xs text-muted-foreground'>{t('cases.fieldEnumOptions')}</span>
          <Input
            className='h-8 flex-1'
            value={(value.enum_values ?? []).join(', ')}
            onChange={(e) =>
              onChange({
                ...value,
                enum_values: e.target.value
                  .split(',')
                  .map((s) => s.trim())
                  .filter(Boolean),
              })
            }
            disabled={disabled}
            autoComplete='off'
          />
        </div>
      ) : null}

      <div className='flex items-center gap-2'>
        <span className='text-xs text-muted-foreground'>{t('cases.fieldDescription')}</span>
        <Input
          className='h-8 flex-1'
          value={value.description ?? ''}
          onChange={(e) => onChange({ ...value, description: e.target.value })}
          disabled={disabled}
          autoComplete='off'
        />
        <Button
          type='button'
          size='sm'
          variant='ghost'
          disabled={disabled}
          onClick={onRemove}
        >
          {t('cases.removeRow')}
        </Button>
      </div>
      <p className='text-xs text-emerald-600 dark:text-emerald-400'>
        {t('cases.autoBoundHint')}
      </p>
    </li>
  )
}

type OutputCardProps = {
  nodes: WorkflowNode[]
  value: OutputFieldDraft
  onChange: (next: OutputFieldDraft) => void
  onRemove: () => void
  disabled?: boolean
}

export function OutputFieldCard({
  nodes,
  value,
  onChange,
  onRemove,
  disabled,
}: OutputCardProps) {
  const { t } = useTranslation()
  const node = nodes.find((n) => n.id === value.node_id)

  return (
    <li data-testid='output-field-card' className='space-y-2 rounded-md border p-3'>
      <div className='flex flex-wrap items-center gap-2'>
        <span className='text-xs text-muted-foreground'>{t('cases.fieldKey')}</span>
        <Input
          className='h-8 w-36'
          value={value.key}
          onChange={(e) => onChange({ ...value, key: e.target.value })}
          disabled={disabled}
          autoComplete='off'
        />
        <span className='text-xs text-muted-foreground'>{t('cases.fieldType')}</span>
        <Select
          value={value.type}
          onValueChange={(type) =>
            onChange({ ...value, type: type as OutputFieldDraft['type'] })
          }
          disabled={disabled}
        >
          <SelectTrigger className='h-8 w-28'>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {OUTPUT_TYPES.map((type) => (
              <SelectItem key={type} value={type}>
                {type}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <span className='text-xs text-muted-foreground'>{t('cases.fieldFromNode')}</span>
        <Select
          value={value.node_id || undefined}
          onValueChange={(nodeId) =>
            onChange({
              ...value,
              node_id: nodeId,
              index:
                nodes.find((n) => n.id === nodeId)?.outputCount === 1
                  ? 0
                  : value.index,
            })
          }
          disabled={disabled}
        >
          <SelectTrigger className='h-8 w-56'>
            <SelectValue placeholder={t('cases.nodeSearchPlaceholder')} />
          </SelectTrigger>
          <SelectContent>
            {nodes.map((n) => (
              <SelectItem key={n.id} value={n.id}>
                {nodeLabel(n.class_type)}（{n.id}）
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {node && node.outputCount > 1 ? (
          <>
            <span className='text-xs text-muted-foreground'>{t('cases.fieldOutput')}</span>
            <Input
              className='h-8 w-20'
              type='number'
              min={0}
              value={value.index ?? 0}
              onChange={(e) =>
                onChange({
                  ...value,
                  index: Number.parseInt(e.target.value, 10) || 0,
                })
              }
              disabled={disabled}
            />
          </>
        ) : null}
        <Button
          type='button'
          size='sm'
          variant='ghost'
          className='ml-auto'
          disabled={disabled}
          onClick={onRemove}
        >
          {t('cases.removeRow')}
        </Button>
      </div>
      {node && node.outputCount === 1 ? (
        <p className='text-xs text-emerald-600 dark:text-emerald-400'>
          {t('cases.singleOutputAuto')}
        </p>
      ) : null}
    </li>
  )
}
```

> 说明：`cases.fieldType`（类型）、`cases.fieldRequired`（必填）、`cases.fieldDescription`（描述）、`cases.removeRow`（删除）已在现有 locale 的 `cases` 段定义（zh.json:168-177 / en.json 对应位置），无需新增。

- [ ] **Step 4: 运行确认通过**

Run: `cd web/admin && pnpm test src/features/cases/workflow-editor.contract.test.ts && pnpm exec tsc -b`
Expected: PASS；tsc 零错误。

- [ ] **Step 5: 格式化并提交**

```bash
cd web/admin
pnpm exec prettier --write src/features/cases/sections/field-cards.tsx src/features/cases/workflow-editor.contract.test.ts
cd /Users/mr9esx/Documents/Pixoma
git add web/admin/src/features/cases/sections/field-cards.tsx web/admin/src/features/cases/workflow-editor.contract.test.ts
git commit -m "feat(admin): visual input and output field cards"
```

---

### Task 1.7: 预览 + 高级模式 + CaseForm 装配（删除旧 raw sections）

**Files:**
- Create: `web/admin/src/features/cases/sections/preview.tsx`
- Create: `web/admin/src/features/cases/sections/advanced.tsx`
- Modify: `web/admin/src/features/cases/case-form.tsx`
- Delete: `web/admin/src/features/cases/sections/workflow-json.tsx`
- Delete: `web/admin/src/features/cases/sections/input-schema.tsx`
- Delete: `web/admin/src/features/cases/sections/bindings.tsx`
- Delete: `web/admin/src/features/cases/sections/io-fields.tsx`
- Modify: `web/admin/src/features/cases/workflow-editor.contract.test.ts`（追加断言）

**Interfaces:**
- Consumes: 全部前序任务
- Produces:
  - `export function PreviewSection(props: { bindings: { inputs: InputBinding[]; outputs: OutputBinding[] }; inputSchema: Record<string, unknown> }): React.JSX.Element`，`data-testid='case-section-preview'`
  - `export function AdvancedSection(props: { open: boolean; editMode: boolean; text: string; onOpen: () => void; onEnterEdit: () => void; onTextChange: (next: string) => void; disabled?: boolean }): React.JSX.Element`，`data-testid='case-section-advanced'`

- [ ] **Step 1: 追加失败测试（contract）**

```ts
const CASE_FORM = join(here, 'case-form.tsx')
const PREVIEW = join(here, 'sections/preview.tsx')
const ADVANCED = join(here, 'sections/advanced.tsx')

describe('editor form assembly', () => {
  it('form renders step sections and no raw workflow textarea by default', () => {
    const form = readFileSync(CASE_FORM, 'utf8')
    expect(form).toContain('WorkflowImportSection')
    expect(form).toContain('InputFieldCard')
    expect(form).toContain('OutputFieldCard')
    expect(form).toContain('PreviewSection')
    expect(form).toContain('AdvancedSection')
    expect(form).toContain('deriveBindings(')
    expect(form).toContain('deriveInputSchema(')
    expect(form).not.toContain('WorkflowJsonSection')
    expect(form).not.toContain('InputSchemaSection')
    expect(form).not.toContain('BindingsSection')
    expect(form).not.toContain('IoFieldsSection')
  })

  it('preview is read-only and advanced is gated behind open + confirm', () => {
    const preview = readFileSync(PREVIEW, 'utf8')
    expect(preview).toContain("data-testid='case-section-preview'")
    expect(preview).toContain('readOnly')
    const advanced = readFileSync(ADVANCED, 'utf8')
    expect(advanced).toContain("data-testid='case-section-advanced'")
    expect(advanced).toContain('editMode')
    expect(advanced).toContain('cases.advancedEnterEdit')
    expect(advanced).toContain('readOnly')
  })
})
```

- [ ] **Step 2: 运行确认失败**

Run: `cd web/admin && pnpm test src/features/cases/workflow-editor.contract.test.ts`
Expected: FAIL。

- [ ] **Step 3: 实现 preview.tsx / advanced.tsx**

`preview.tsx`：

```tsx
import { useTranslation } from 'react-i18next'
import type { InputBinding, OutputBinding } from '@/lib/api/types'

type Props = {
  bindings: { inputs: InputBinding[]; outputs: OutputBinding[] }
  inputSchema: Record<string, unknown>
}

export function PreviewSection({ bindings, inputSchema }: Props) {
  const { t } = useTranslation()
  const preview = {
    bindings,
    input_schema: inputSchema,
  }
  return (
    <section className='space-y-2' data-testid='case-section-preview'>
      <div>
        <h3 className='text-sm font-semibold'>{t('cases.previewHeading')}</h3>
        <p className='text-muted-foreground text-xs'>{t('cases.previewHint')}</p>
      </div>
      <pre
        readOnly
        className='max-h-48 overflow-auto rounded-md border bg-muted/30 p-3 font-mono text-xs'
      >
        {JSON.stringify(preview, null, 2)}
      </pre>
    </section>
  )
}
```

`advanced.tsx`：

```tsx
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

type Props = {
  open: boolean
  editMode: boolean
  text: string
  onOpen: () => void
  onEnterEdit: () => void
  onTextChange: (next: string) => void
  disabled?: boolean
}

export function AdvancedSection({
  open,
  editMode,
  text,
  onOpen,
  onEnterEdit,
  onTextChange,
  disabled,
}: Props) {
  const { t } = useTranslation()
  if (!open) {
    return (
      <section data-testid='case-section-advanced' className='flex justify-end'>
        <Button type='button' size='sm' variant='ghost' onClick={onOpen} disabled={disabled}>
          {t('cases.advancedLabel')}
        </Button>
      </section>
    )
  }
  return (
    <section className='space-y-2' data-testid='case-section-advanced'>
      <div className='flex items-center justify-between gap-2'>
        <div>
          <h3 className='text-sm font-semibold'>{t('cases.advancedLabel')}</h3>
          <p className='text-muted-foreground text-xs'>
            {editMode
              ? t('cases.advancedOverwriteWarn')
              : t('cases.advancedReadonlyHint')}
          </p>
        </div>
        {!editMode ? (
          <Button type='button' size='sm' variant='outline' onClick={onEnterEdit} disabled={disabled}>
            {t('cases.advancedEnterEdit')}
          </Button>
        ) : null}
      </div>
      <Label htmlFor='case-advanced-json' className='sr-only'>
        {t('cases.advancedLabel')}
      </Label>
      <Textarea
        id='case-advanced-json'
        value={text}
        onChange={(e) => onTextChange(e.target.value)}
        readOnly={!editMode}
        disabled={disabled}
        rows={12}
        className='font-mono text-xs'
      />
    </section>
  )
}
```

- [ ] **Step 4: 重写 case-form.tsx 装配**

关键改动（保留现有 mutation / enable-disable 逻辑，只替换表单主体）：

```tsx
import { useState } from 'react'
// ... 保留现有 react-query / i18n / api imports ...
import { parseWorkflow } from './lib/workflow-parse'
import {
  deriveBindings,
  deriveInputSchema,
  validateEditor,
  type InputFieldDraft,
  type OutputFieldDraft,
} from './lib/derive'
import { WorkflowImportSection } from './sections/workflow-import'
import { InputFieldCard, OutputFieldCard } from './sections/field-cards'
import { PreviewSection } from './sections/preview'
import { AdvancedSection } from './sections/advanced'

function toInputDrafts(record: CaseRecord): InputFieldDraft[] {
  const byKey = new Map(record.bindings.inputs.map((b) => [b.key, b]))
  return record.inputs.map((field) => ({
    key: field.key,
    type: field.type,
    required: field.required,
    node_id: byKey.get(field.key)?.node_id ?? '',
    field_path: byKey.get(field.key)?.field_path ?? '',
    description: field.description,
  }))
}

function toOutputDrafts(record: CaseRecord): OutputFieldDraft[] {
  const byKey = new Map(record.bindings.outputs.map((b) => [b.key, b]))
  return record.outputs.map((field) => ({
    key: field.key,
    type: field.type,
    node_id: byKey.get(field.key)?.node_id ?? '',
    index: byKey.get(field.key)?.index ?? 0,
    description: field.description,
  }))
}
```

组件内新增状态（初始化：edit 模式用 `toInputDrafts/toOutputDrafts` + `JSON.stringify(record.bindings.workflow)` 走 `parseWorkflow`；create 模式为空）：

```tsx
const initial = props.mode === 'edit' ? props.initial : emptyCase()
const initialWorkflowText = JSON.stringify(initial.bindings.workflow, null, 2)
const [workflowText, setWorkflowText] = useState(initialWorkflowText)
const [graph, setGraph] = useState(() => {
  if (props.mode !== 'edit' || !Object.keys(initial.bindings.workflow).length) {
    return undefined
  }
  const result = parseWorkflow(initialWorkflowText)
  return result.ok ? result.graph : undefined
})
const [importError, setImportError] = useState<string | undefined>(() => {
  if (props.mode !== 'edit' || !Object.keys(initial.bindings.workflow).length) {
    return undefined
  }
  const result = parseWorkflow(initialWorkflowText)
  return result.ok ? undefined : result.error
})
const [inputDrafts, setInputDrafts] = useState<InputFieldDraft[]>(() =>
  props.mode === 'edit' ? toInputDrafts(props.initial) : [],
)
const [outputDrafts, setOutputDrafts] = useState<OutputFieldDraft[]>(() =>
  props.mode === 'edit' ? toOutputDrafts(props.initial) : [],
)
const [advancedOpen, setAdvancedOpen] = useState(false)
const [advancedEdit, setAdvancedEdit] = useState(false)
const [advancedText, setAdvancedText] = useState(initialWorkflowText)
```

`buildPayload` 替换为：

```tsx
function buildPayload(): CaseRecord | null {
  const validation = validateEditor(inputDrafts, outputDrafts)
  if (validation.duplicateKey || validation.unboundRequired || validation.noOutput) {
    setEditorError(
      validation.duplicateKey
        ? t('cases.errDuplicateKey')
        : validation.unboundRequired
          ? t('cases.errInputNotBound')
          : t('cases.errNoOutput'),
    )
    return null
  }
  const bindings = deriveBindings(inputDrafts, outputDrafts)
  const inputSchema = deriveInputSchema(inputDrafts)
  return {
    ...draft,
    id: draft.id.trim(),
    name: draft.name.trim(),
    description: draft.description?.trim() || undefined,
    preview: draft.preview?.trim() || undefined,
    menu_key: draft.menu_key?.trim() || undefined,
    tags: draft.tags?.length ? draft.tags : undefined,
    categories: draft.categories?.length ? draft.categories : undefined,
    inputs: inputDrafts.map((f) => ({
      key: f.key.trim(),
      type: f.type,
      required: f.required,
      description: f.description?.trim() || undefined,
    })),
    outputs: outputDrafts.map((f) => ({
      key: f.key.trim(),
      type: f.type,
      description: f.description?.trim() || undefined,
    })),
    bindings: {
      workflow: graph?.api ?? draft.bindings.workflow,
      ...bindings,
    },
    input_schema: inputSchema,
  }
}
```

工作流文本变更处理器（组件内）：

```tsx
function onWorkflowTextChange(next: string) {
  setWorkflowText(next)
  setImportError(undefined)
  const result = parseWorkflow(next)
  if (result.ok) {
    setGraph(result.graph)
  } else if (next.trim()) {
    setGraph(undefined)
    setImportError(result.error)
  }
}
```

渲染顺序（`<form>` 内、Basics 之后）：`WorkflowImportSection` → 步骤 2（`inputsHeading` + 输入卡片列表 + 添加按钮）→ 步骤 3（`outputsHeading` + 输出卡片列表 + 添加按钮）→ `PreviewSection`（bindings + inputSchema 实时推导）→ `AdvancedSection`（`advancedText` 默认 `workflowText`）。步骤 2/3 在 `!graph` 时整体渲染为置灰锁定提示（`cases.emptyWorkflowLock`）。删除对 `WorkflowJsonSection` / `InputSchemaSection` / `BindingsSection` / `IoFieldsSection` 的 import 与使用，并删除四个旧文件。

- [ ] **Step 5: 运行确认通过**

Run:
```bash
cd web/admin
pnpm test
pnpm exec tsc -b
pnpm exec eslint src/features/cases/case-form.tsx src/features/cases/sections
```
Expected: 全部通过（旧 contract 测试如 `menu-placements.contract.test.ts` 仍通过，因为它只检查 `menu-placements.tsx` 与 `detail-panel.tsx`）。

- [ ] **Step 6: 格式化并提交**

```bash
cd web/admin
pnpm exec prettier --write src/features/cases/case-form.tsx src/features/cases/sections src/features/cases/workflow-editor.contract.test.ts
cd /Users/mr9esx/Documents/Pixoma
git add web/admin/src/features/cases
git rm web/admin/src/features/cases/sections/workflow-json.tsx web/admin/src/features/cases/sections/input-schema.tsx web/admin/src/features/cases/sections/bindings.tsx web/admin/src/features/cases/sections/io-fields.tsx
git commit -m "feat(admin): step-by-step workflow editor form"
```

---

### Task 1.8: 全量验证

**Files:** 无新增。

- [ ] **Step 1: 全量检查**

```bash
cd /Users/mr9esx/Documents/Pixoma
cd web/admin
pnpm test
pnpm exec tsc -b
pnpm exec eslint .
pnpm exec prettier --check .
```
Expected: 全部通过。

- [ ] **Step 2: 提交遗留改动**

```bash
cd /Users/mr9esx/Documents/Pixoma
git add -A web/admin
git commit -m "chore(admin): finalize workflow editor frontend"
```
