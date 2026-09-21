# Studio Shell and Capability Controls Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the Studio session sidebar visually consistent with the platform and let administrators create, enable, select, and authorize the Agent capabilities that its chat actually uses.

**Architecture:** Keep the Studio shell and its dedicated session sidebar, but compose that sidebar from the existing platform Sidebar primitives and semantic tokens. Add account-scoped Skill, MCP connector, and workflow-capability records behind the Studio repository; a capability registry resolves an immutable Run snapshot from selected Skills and enabled configuration. The frontend uses those APIs in four AI settings tabs and sends selected Skill IDs with each AG-UI run.

**Tech Stack:** Go, GORM/SQLite, Chi, React, TypeScript, TanStack Query, assistant-ui AG-UI adapter, shadcn/ui, Vitest.

## Global Constraints

- Keep Studio's dedicated session sidebar; reuse Sidebar primitives and styles, not `AppSidebar` menu data or navigation structure.
- Use existing shadcn/ui components. Use Pixoma semantic tokens and `StatusDot`; do not introduce raw hex colors, `dark:` classes, gradients, or surface shadows.
- Use space, typography, and Card hierarchy first. Retain borders only for panel boundaries, floating containers, tables, and required form controls.
- All config reads and writes are account-scoped. API responses never contain plaintext credentials.
- A chat Run uses a Skill only when enabled and explicitly selected. It uses a connector/workflow only when enabled and permitted by the current Run policy.
- Start every behavior change with a failing test, observe the expected failure, then write the smallest implementation that passes it.
- Every commit uses `李卓洲 <1138099359@qq.com>` as both author and committer.

---

## File Structure

| File | Responsibility |
| --- | --- |
| `web/admin/src/routes/_app.tsx` | Keeps the Studio shell separate from the platform app shell. |
| `web/admin/src/features/studio/studio-workspace.tsx` | Studio three-column content layout, session and content mode. |
| `web/admin/src/features/studio/studio-sidebar.tsx` | Studio session sidebar composed from platform Sidebar primitives. |
| `web/admin/src/features/studio/studio-settings.tsx` | Models, Skills, connectors, workflow settings views and dialogs. |
| `web/admin/src/features/studio/studio-chat.tsx` | Composer Skill state and AG-UI forwarded properties. |
| `web/admin/src/lib/api/studio.ts` | Typed capability API client. |
| `internal/studio/domain/capability.go` | Skill, connector, workflow access, policy, and Run snapshot types. |
| `internal/studio/application/capability_config.go` | Capability CRUD, validation, and secret protection. |
| `internal/studio/application/capability_registry.go` | Per-Run immutable capability resolver. |
| `internal/studio/infrastructure/persistence/capability_config.go` | GORM records and Studio repository implementation. |
| `internal/httpapi/studio/handler.go` | Capability REST endpoints. |
| `internal/httpapi/studio/agui.go` | Decodes selected Skill IDs from AG-UI properties. |

### Task 1: Restyle the Studio session sidebar with platform primitives

**Files:**
- Modify: `web/admin/src/features/studio/studio-workspace.tsx`
- Modify: `web/admin/src/features/studio/studio-sidebar.tsx`
- Modify: `web/admin/src/features/studio/studio-workspace.contract.test.ts`
- Test: `web/admin/src/components/layout/shell-layout.contract.test.ts`

**Interfaces:**
- Consumes: `StudioWorkspace()`, `StudioSidebar()`, and installed Sidebar primitives.
- Produces: a dedicated Studio sidebar that retains sessions while using `SidebarHeader`, `SidebarContent`, `SidebarGroup`, `SidebarMenu`, and `SidebarFooter`.

- [ ] **Step 1: Write the failing shell contract**

```ts
it('keeps the Studio session sidebar while reusing platform sidebar styles', () => {
  expect(read('./studio-workspace.tsx')).toContain('StudioSidebar')
  const sidebar = read('./studio-sidebar.tsx')
  expect(sidebar).toContain("from '@/components/ui/sidebar'")
  expect(sidebar).toContain('<SidebarContent')
  expect(sidebar).toContain('最近对话')
})
```

- [ ] **Step 2: Run it red**

Run: `pnpm --dir web/admin vitest run src/features/studio/studio-workspace.contract.test.ts`

Expected: FAIL because the session sidebar does not yet use Sidebar primitives.

- [ ] **Step 3: Implement the minimum shared shell**

```tsx
// StudioSidebar: use SidebarHeader, SidebarContent, SidebarGroup,
// SidebarMenu and SidebarFooter around existing session interactions.
```

Keep `StudioLayout`, `StudioSidebar`, and its mobile Sheet. Replace hand-built header, navigation, session list and footer wrappers with existing Sidebar primitives. Remove dashed empty-state and structural divider borders; retain the session list and existing callbacks.

- [ ] **Step 4: Run focused contracts green**

Run: `pnpm --dir web/admin vitest run src/features/studio/studio-workspace.contract.test.ts src/components/layout/shell-layout.contract.test.ts`

Expected: PASS with the session list intact and no duplicated menu data.

- [ ] **Step 5: Commit**

```bash
git add web/admin/src/features/studio/studio-sidebar.tsx web/admin/src/features/studio/studio-workspace.contract.test.ts
GIT_AUTHOR_NAME='李卓洲' GIT_AUTHOR_EMAIL='1138099359@qq.com' GIT_COMMITTER_NAME='李卓洲' GIT_COMMITTER_EMAIL='1138099359@qq.com' git commit -m 'refactor(studio): align session sidebar with platform styles'
```

### Task 2: Persist Skills and MCP connectors safely

**Files:**
- Create: `internal/studio/domain/capability.go`
- Modify: `internal/studio/domain/repository.go`
- Create: `internal/studio/application/capability_config.go`
- Create: `internal/studio/application/capability_config_test.go`
- Create: `internal/studio/infrastructure/persistence/capability_config.go`
- Modify: `internal/studio/infrastructure/persistence/gorm_repository.go`
- Create: `internal/studio/infrastructure/persistence/capability_config_test.go`

**Interfaces:**
- Consumes: `platformcrypto.Encrypt`, `platformcrypto.Decrypt`, account-scoped Studio repository conventions.
- Produces: `CapabilityConfigService.CreateSkill`, `UpdateSkill`, `ListSkills`, `CreateConnector`, `UpdateConnector`, `ListConnectors`.

- [ ] **Step 1: Write failing lifecycle and masking tests**

```go
func TestCapabilityConfigCreatesEnabledSkill(t *testing.T) {
  skill, err := service.CreateSkill(ctx, studioapp.CreateSkillInput{
    AccountID: "account-a", Name: "漫画分镜", Description: "把故事拆成镜头", Prompt: "先输出镜头表", Enabled: true,
  })
  if err != nil || !skill.Enabled || skill.Prompt != "先输出镜头表" { t.Fatalf("skill = %#v, err = %v", skill, err) }
}

func TestCapabilityConfigMasksConnectorCredential(t *testing.T) {
  c, err := service.CreateConnector(ctx, studioapp.CreateConnectorInput{
    AccountID: "account-a", Name: "资料库", URL: "https://mcp.example.test/mcp", Credential: "secret", Policy: domain.ConnectorPolicyApproval,
  })
  if err != nil || c.CredentialMasked == "secret" || c.CredentialMasked == "" { t.Fatalf("connector = %#v, err = %v", c, err) }
}
```

- [ ] **Step 2: Run tests red**

Run: `go test ./internal/studio/application -run 'TestCapabilityConfig' -count=1`

Expected: FAIL because capability types and service methods do not exist.

- [ ] **Step 3: Implement domain records, GORM rows, and CRUD**

```go
type Skill struct { ID, AccountID, Name, Description, Prompt string; Enabled bool; CreatedAt, UpdatedAt time.Time }
type ConnectorPolicy string
const (
  ConnectorPolicyAuto ConnectorPolicy = "auto"
  ConnectorPolicyApproval ConnectorPolicy = "approval"
  ConnectorPolicyForbidden ConnectorPolicy = "forbidden"
)
type MCPConnector struct { ID, AccountID, Name, URL, CredentialCipher string; Enabled bool; Policy ConnectorPolicy; DiscoveredTools []MCPTool; CreatedAt, UpdatedAt time.Time }
```

Add account-scoped create/get/list/update methods for Skills and connectors to `domain.Repository`. Validate nonempty Skill name/prompt, HTTP(S) URLs, and policy. Add `SkillRow` and `MCPConnectorRow` to `Models()`; JSON-encode discovered tools and encrypt the credential before persistence.

- [ ] **Step 4: Run tests green**

Run: `go test ./internal/studio/application ./internal/studio/infrastructure/persistence -run 'TestCapabilityConfig|TestGorm.*Capability' -count=1`

Expected: PASS; plaintext credentials occur in neither row values nor public views.

- [ ] **Step 5: Commit**

```bash
git add internal/studio/domain/capability.go internal/studio/domain/repository.go internal/studio/application/capability_config.go internal/studio/application/capability_config_test.go internal/studio/infrastructure/persistence/capability_config.go internal/studio/infrastructure/persistence/capability_config_test.go internal/studio/infrastructure/persistence/gorm_repository.go
GIT_AUTHOR_NAME='李卓洲' GIT_AUTHOR_EMAIL='1138099359@qq.com' GIT_COMMITTER_NAME='李卓洲' GIT_COMMITTER_EMAIL='1138099359@qq.com' git commit -m 'feat(studio): persist skills and MCP connectors'
```

### Task 3: Register workflow and runtime capabilities per Run

**Files:**
- Modify: `internal/studio/domain/capability.go`
- Modify: `internal/studio/domain/repository.go`
- Modify: `internal/studio/application/capability_config.go`
- Create: `internal/studio/application/capability_registry.go`
- Create: `internal/studio/application/capability_registry_test.go`
- Modify: `internal/studio/application/executor.go`
- Modify: `internal/studio/application/runner.go`
- Modify: `internal/studio/infrastructure/persistence/capability_config.go`

**Interfaces:**
- Consumes: enabled Studio model config, `Skill`, `MCPConnector`, existing Case metadata, and `domain.PermissionMode`.
- Produces: `CapabilityRegistry.Resolve(ctx, CapabilityRequest) (CapabilitySnapshot, error)` stored on the Run before Agent startup.

- [ ] **Step 1: Write failing capability filtering tests**

```go
func TestCapabilityRegistryUsesOnlyExplicitlySelectedEnabledSkills(t *testing.T) {
  snapshot, err := registry.Resolve(ctx, studioapp.CapabilityRequest{
    AccountID: "account-a", SkillIDs: []string{"enabled", "disabled"}, PermissionMode: domain.PermissionApproval,
  })
  if err != nil { t.Fatal(err) }
  if got := snapshot.SkillIDs; !reflect.DeepEqual(got, []string{"enabled"}) { t.Fatalf("SkillIDs = %#v", got) }
}
func TestCapabilityRegistryExcludesUnhealthyOrAgentDisabledWorkflows(t *testing.T) {
  snapshot, err := registry.Resolve(ctx, studioapp.CapabilityRequest{AccountID: "account-a", PermissionMode: domain.PermissionApproval})
  if err != nil { t.Fatal(err) }
  if slices.Contains(snapshot.WorkflowIDs, uint64(42)) { t.Fatalf("workflows = %#v", snapshot.WorkflowIDs) }
}
```

- [ ] **Step 2: Run tests red**

Run: `go test ./internal/studio/application -run 'TestCapabilityRegistry' -count=1`

Expected: FAIL because the registry and snapshot do not exist.

- [ ] **Step 3: Implement opt-in workflow records and snapshots**

```go
type WorkflowCapability struct { AccountID string; WorkflowID uint64; AgentEnabled bool; UpdatedAt time.Time }
type CapabilityRequest struct { AccountID, ModelConfigID string; SkillIDs []string; PermissionMode domain.PermissionMode }
type CapabilitySnapshot struct { ModelConfigID string; SkillIDs, ConnectorIDs []string; WorkflowIDs []uint64; CreatedAt time.Time }
type CapabilityRegistry interface { Resolve(context.Context, CapabilityRequest) (CapabilitySnapshot, error) }
```

Persist workflow opt-in by account and workflow ID. Resolve only an enabled/Agent-enabled model, explicitly selected enabled Skills, connectors whose policy accepts the permission mode, and existing Case workflows that are enabled, have a definition, and opted in. Serialize the snapshot on the Run before it enters the queue; executor input must receive the snapshot rather than rereading settings.

- [ ] **Step 4: Run application tests green**

Run: `go test ./internal/studio/application -run 'TestCapabilityRegistry|Test.*Run' -count=1`

Expected: PASS; later config changes do not change a queued/running Run.

- [ ] **Step 5: Commit**

```bash
git add internal/studio/domain/capability.go internal/studio/domain/repository.go internal/studio/application/capability_config.go internal/studio/application/capability_registry.go internal/studio/application/capability_registry_test.go internal/studio/application/executor.go internal/studio/application/runner.go internal/studio/infrastructure/persistence/capability_config.go
GIT_AUTHOR_NAME='李卓洲' GIT_AUTHOR_EMAIL='1138099359@qq.com' GIT_COMMITTER_NAME='李卓洲' GIT_COMMITTER_EMAIL='1138099359@qq.com' git commit -m 'feat(studio): resolve Agent capability snapshots'
```

### Task 4: Expose capability configuration and selected Skills via API

**Files:**
- Modify: `internal/httpapi/studio/handler.go`
- Modify: `internal/httpapi/studio/agui.go`
- Modify: `internal/httpapi/studio/handler_test.go`
- Modify: Studio dependency bootstrap constructor found with `rg -n 'studioapi.Handler' internal`
- Modify: `web/admin/src/lib/api/studio.ts`
- Modify: `web/admin/src/lib/api/studio.test.ts`

**Interfaces:**
- Consumes: `CapabilityConfigService`, `CapabilityRegistry`, account context.
- Produces: capability REST endpoints and AG-UI `forwardedProps.runConfig.selectedSkillIds`.

- [ ] **Step 1: Write failing HTTP and client tests**

```go
func TestStudioSkillsAreAccountScoped(t *testing.T) {
  response := request(t, router, http.MethodGet, "/skills", nil, "account-b")
  if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "account-a-skill") { t.Fatalf("response = %s", response.Body.String()) }
}
func TestAGUIForwardsSelectedSkillsToRunConfig(t *testing.T) {
  response := request(t, router, http.MethodPost, "/agui", map[string]any{
    "forwardedProps": map[string]any{"runConfig": map[string]any{"selectedSkillIds": []string{"skill-1"}}},
  }, "account-a")
  if response.Code != http.StatusOK { t.Fatalf("status = %d", response.Code) }
}
```

```ts
it('sends selected Skill IDs with a REST turn', async () => {
  await sendStudioMessage({ sessionId: 's1', text: '写分镜', permissionMode: 'request_approval', skillIds: ['skill-1'] })
  expect(JSON.parse(String(fetchMock.mock.calls[0][1].body))).toMatchObject({ skill_ids: ['skill-1'] })
})
```

- [ ] **Step 2: Run tests red**

Run: `go test ./internal/httpapi/studio -run 'TestStudioSkillsAreAccountScoped|TestAGUIForwardsSelectedSkillsToRunConfig' -count=1 && pnpm --dir web/admin vitest run src/lib/api/studio.test.ts`

Expected: FAIL because the routes and selected Skill request fields do not exist.

- [ ] **Step 3: Implement routes, request parsing, and client functions**

```go
r.Get("/skills", h.listSkills); r.Post("/skills", h.createSkill); r.Patch("/skills/{skillID}", h.updateSkill)
r.Get("/connectors", h.listConnectors); r.Post("/connectors", h.createConnector); r.Patch("/connectors/{connectorID}", h.updateConnector); r.Post("/connectors/{connectorID}/probe", h.probeConnector)
r.Get("/workflows", h.listWorkflowCapabilities); r.Put("/workflows/{workflowID}", h.upsertWorkflowCapability)
```

Decode `selectedSkillIds []string` from AG-UI and `skill_ids []string` from REST message input. Feed both into the registry before enqueueing. Responses expose only masked connector credential state, enabled/policy/tool summaries, and workflow display metadata. Wire non-nil services in the production bootstrap.

- [ ] **Step 4: Run tests green**

Run: `go test ./internal/httpapi/studio -count=1 && pnpm --dir web/admin vitest run src/lib/api/studio.test.ts`

Expected: PASS; foreign account records are absent and selected IDs reach the Run configuration.

- [ ] **Step 5: Commit**

```bash
git add internal/httpapi/studio/handler.go internal/httpapi/studio/agui.go internal/httpapi/studio/handler_test.go internal/platform web/admin/src/lib/api/studio.ts web/admin/src/lib/api/studio.test.ts
GIT_AUTHOR_NAME='李卓洲' GIT_AUTHOR_EMAIL='1138099359@qq.com' GIT_COMMITTER_NAME='李卓洲' GIT_COMMITTER_EMAIL='1138099359@qq.com' git commit -m 'feat(studio): expose Agent capability settings API'
```

### Task 5: Build AI settings and composer Skill selection

**Files:**
- Modify: `web/admin/src/features/studio/studio-settings.tsx`
- Create: `web/admin/src/features/studio/studio-settings.contract.test.tsx`
- Modify: `web/admin/src/features/studio/studio-chat.tsx`
- Modify: `web/admin/src/features/studio/studio-workspace.tsx`
- Modify: `web/admin/src/features/studio/studio-workspace.contract.test.ts`

**Interfaces:**
- Consumes: typed Skill, connector and workflow API functions from `studio.ts`.
- Produces: four functional AI settings tabs and `StudioChat.selectedSkillIds: string[]`, forwarded as `runConfig.selectedSkillIds`.

- [ ] **Step 1: Write failing UI contracts**

```ts
it('uses connected capability tabs instead of empty settings', () => {
  const source = read('./studio-settings.tsx')
  expect(source).toContain('createStudioSkill')
  expect(source).toContain('createStudioConnector')
  expect(source).toContain('setStudioWorkflowCapability')
  expect(source).not.toContain("title='Skills' description=")
})
it('forwards a multi-selected Skill set from the composer', () => {
  const source = read('./studio-chat.tsx')
  expect(source).toContain('selectedSkillIds')
  expect(source).toContain('SkillPicker')
  expect(source).toContain('runConfig')
})
```

- [ ] **Step 2: Run tests red**

Run: `pnpm --dir web/admin vitest run src/features/studio/studio-settings.contract.test.tsx src/features/studio/studio-workspace.contract.test.ts`

Expected: FAIL because three settings tabs are empty placeholders and the composer has no Skill picker.

- [ ] **Step 3: Implement controls and visual integration**

```tsx
type SkillPickerProps = { skills: StudioSkill[]; value: string[]; onChange(ids: string[]): void }
function SkillPicker({ skills, value, onChange }: SkillPickerProps) {
  // Popover + CommandItem checkmarks; only enabled skills are selectable.
}
```

Use one primary action per configurable tab: 「添加模型」、「添加 Skill」、「添加连接器」. The workflow tab only toggles existing workflows. Add loading, empty, pending, error, retry states. Use Cards and spacing, remove enclosing `border-b` and dashed empty-state borders, and retain only required control/floating-surface borders. Use `StatusDot` for connector/workflow health; unavailable or unchecked health is yellow, never green.

- [ ] **Step 4: Run tests green**

Run: `pnpm --dir web/admin vitest run src/features/studio/studio-settings.contract.test.tsx src/features/studio/studio-workspace.contract.test.ts src/lib/api/studio.test.ts`

Expected: PASS; each settings tab reaches a live API function and the composer has a multi-select Skill control.

- [ ] **Step 5: Commit**

```bash
git add web/admin/src/features/studio/studio-settings.tsx web/admin/src/features/studio/studio-settings.contract.test.tsx web/admin/src/features/studio/studio-chat.tsx web/admin/src/features/studio/studio-workspace.tsx web/admin/src/features/studio/studio-workspace.contract.test.ts
GIT_AUTHOR_NAME='李卓洲' GIT_AUTHOR_EMAIL='1138099359@qq.com' GIT_COMMITTER_NAME='李卓洲' GIT_COMMITTER_EMAIL='1138099359@qq.com' git commit -m 'feat(studio): configure and select Agent capabilities'
```

### Task 6: Run full regression and design verification

**Files:**
- Modify: only files from Tasks 1–5 when fresh verification identifies a failing requirement.
- Modify: `docs/superpowers/specs/2026-09-20-ai-studio-ux-design.md` only for a verified implementation deviation.

**Interfaces:**
- Consumes: platform shell, capability API, settings UI, and composer selection.
- Produces: fresh test, lint, format and build evidence.

- [ ] **Step 1: Add a failing regression only if an uncovered requirement remains**

```ts
it('does not render a second Studio navigation surface', () => {
  expect(read('./studio-workspace.tsx')).not.toContain('Sidebar')
  expect(read('./studio-workspace.tsx')).not.toContain('border-r bg-sidebar')
})
```

Do not duplicate this test when Task 1 already proves the same behavior.

- [ ] **Step 2: Verify red then green if a test was added**

Run: `pnpm --dir web/admin vitest run src/features/studio/studio-workspace.contract.test.ts`

Expected: the new test fails only against the old duplicated-navigation code, then passes with the implementation.

- [ ] **Step 3: Run the full verification suite**

Run: `go test ./...`

Run: `pnpm --dir web/admin test`

Run: `pnpm --dir web/admin lint`

Run: `pnpm --dir web/admin format:check`

Run: `pnpm --dir web/admin build`

Expected: every command exits 0.

- [ ] **Step 4: Check the design acceptance list**

Verify: Studio retains its session sidebar and mobile Sheet without copying platform menu data; all four settings tabs call live APIs; disabled Skills cannot be selected; Run snapshots record explicit Skill IDs; connector credentials remain masked; only opted-in healthy workflows enter the registry; no raw colors or decorative line grids were added.

- [ ] **Step 5: Commit verification corrections**

```bash
git add web/admin/src internal/studio internal/httpapi/studio docs/superpowers/specs/2026-09-20-ai-studio-ux-design.md
git diff --cached --check
GIT_AUTHOR_NAME='李卓洲' GIT_AUTHOR_EMAIL='1138099359@qq.com' GIT_COMMITTER_NAME='李卓洲' GIT_COMMITTER_EMAIL='1138099359@qq.com' git commit -m 'test(studio): verify capability controls'
```
