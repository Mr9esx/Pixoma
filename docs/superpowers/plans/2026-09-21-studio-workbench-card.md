# Studio Workbench Card Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Place Studio chat and asset canvas in one padded, rounded workbench card and remove structural divider lines.

**Architecture:** Keep `StudioSidebar` outside the workbench. Restructure only the chat branch in `StudioWorkspace`: a shell provides page padding and a `StudioWorkbench` card contains the chat pane plus the optional canvas pane. Existing data queries and child components remain unchanged; the visual boundary is expressed through semantic background, rounded corners, and a small collapse control rather than `border-l`/`border-b` separators.

**Tech Stack:** React, TypeScript, Tailwind semantic tokens, shadcn/ui, Vitest.

## Global Constraints

- Use installed shadcn/ui primitives and semantic Tailwind tokens only; no bare hex colors.
- Preserve Studio's independent session sidebar and its `SidebarProvider`.
- Keep mobile on the existing Sheet navigation and hide the canvas pane below `xl`.
- Keep visible borders only on directly manipulable controls (composer, cards, React Flow controls); do not use structural divider borders.

---

### Task 1: Define the workbench layout contract

**Files:**
- Modify: `web/admin/src/features/studio/studio-workspace.contract.test.ts`
- Modify: `web/admin/src/features/studio/studio-workspace.tsx`

**Interfaces:**
- Consumes: existing `StudioWorkspace` chat/session queries and `rightOpen` state.
- Produces: a `StudioWorkbench` element that wraps both chat and canvas panes.

- [ ] **Step 1: Write the failing test**

```ts
it('groups chat and canvas in one padded rounded workbench without structural borders', () => {
  const source = read('./studio-workspace.tsx')
  expect(source).toContain('data-slot=\'studio-workbench\'')
  expect(source).toContain('rounded-2xl')
  expect(source).not.toContain('max-w-2xl min-w-80 shrink-0 border-l')
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm vitest run src/features/studio/studio-workspace.contract.test.ts`

Expected: FAIL because `data-slot='studio-workbench'` does not exist.

- [ ] **Step 3: Write the minimal layout implementation**

```tsx
<div className='min-h-0 flex-1 bg-muted/30 p-3 sm:p-4'>
  <section data-slot='studio-workbench' className='flex h-full min-h-0 overflow-hidden rounded-2xl bg-card shadow-sm'>
    <main className='flex min-w-0 flex-1 flex-col'>...</main>
    <aside className='hidden w-[42%] max-w-2xl min-w-80 shrink-0 bg-muted/20 xl:flex xl:flex-col'>...</aside>
  </section>
</div>
```

Remove `border-b` from the chat header and canvas tab bar, remove `border-l` from the canvas pane, and retain the existing collapse button as the only desktop separation affordance.

- [ ] **Step 4: Run test to verify it passes**

Run: `pnpm vitest run src/features/studio/studio-workspace.contract.test.ts`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/admin/src/features/studio/studio-workspace.tsx web/admin/src/features/studio/studio-workspace.contract.test.ts
git commit -m "refactor(studio): group chat and canvas in workbench"
```

### Task 2: Soften chat chrome inside the workbench

**Files:**
- Modify: `web/admin/src/features/studio/studio-chat.tsx`
- Modify: `web/admin/src/features/studio/studio-workspace.contract.test.ts`

**Interfaces:**
- Consumes: `StudioChat`'s existing assistant-ui root and composer primitives.
- Produces: a composer that preserves its own control boundary but does not add a full-width structural background strip.

- [ ] **Step 1: Write the failing test**

```ts
it('keeps the composer as the chat boundary without a full-width chrome strip', () => {
  const source = read('./studio-chat.tsx')
  expect(source).toContain("className='shrink-0 px-4 pt-2 pb-5'")
  expect(source).not.toContain("className='shrink-0 bg-background")
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pnpm vitest run src/features/studio/studio-workspace.contract.test.ts`

Expected: FAIL because the composer wrapper has `bg-background`.

- [ ] **Step 3: Write the minimal visual implementation**

```tsx
<div className='shrink-0 px-4 pt-2 pb-5'>
  <ComposerPrimitive.Root className='... rounded-2xl border bg-card p-2 ...'>
```

Keep the composer border because it is an interactive control; remove only the page-wide background strip.

- [ ] **Step 4: Run test to verify it passes**

Run: `pnpm vitest run src/features/studio/studio-workspace.contract.test.ts`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/admin/src/features/studio/studio-chat.tsx web/admin/src/features/studio/studio-workspace.contract.test.ts
git commit -m "style(studio): reduce chat structural chrome"
```

### Task 3: Verify the workspace build

**Files:**
- Verify only.

- [ ] **Step 1: Run focused tests**

Run: `pnpm vitest run src/features/studio/studio-workspace.contract.test.ts src/lib/api/studio.test.ts`

Expected: PASS.

- [ ] **Step 2: Run production build**

Run: `pnpm build`

Expected: exit code 0.

- [ ] **Step 3: Check the diff**

Run: `git diff --check && git status --short`

Expected: no whitespace errors and only expected Studio files.
