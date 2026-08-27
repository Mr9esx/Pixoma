# Source Project Context

This design-system workspace was created from an existing OpenDesign project. Treat the copied project files as the primary source evidence for the generated design system.

## Source project

- Source project id: 9d4ece8d-9cf2-411b-8589-7e499a46e3fe
- Source project name: 整理这个项目的设计系统、设计语言、以
- New design-system project id: 978ed576-2033-47d3-974e-3115b16a4595
- New design-system id: user:design-system
- Source skill id: (none)
- Source design system id: (none)

## Source metadata

```json
{
  "kind": "prototype",
  "nameSource": "prompt",
  "linkedDirs": [
    "/Users/mr9esx/Documents/Pixoma"
  ],
  "strategyBinding": {
    "schemaVersion": 1,
    "provenance": "automatic_default",
    "taskProfile": "prototype",
    "boundAt": 1787757523588
  },
  "scenarioBinding": {
    "schemaVersion": 1,
    "provenance": "automatic_default",
    "pluginId": "example-web-prototype",
    "snapshotId": "e723fa6d-b486-4477-981b-992e1ada0425",
    "taskProfile": "prototype",
    "boundAt": 1787757974450
  }
}
```

## Copied files

- pixoma-design-system.html
- pixoma-design-tokens.css
- brand-spec.md

## Skipped files

- (none)

## Generation contract

- Read this file before editing design-system outputs.
- Read the copied files directly from the project workspace; they are source evidence, not generated design-system output.
- Preserve high-signal assets, source examples, UI surfaces, copy, tokens, typography, and interaction patterns from the copied project.
- Generate a reusable OpenDesign design-system package in this same project: DESIGN.md, README.md, SKILL.md, colors_and_type.css, context/provenance, focused preview cards, preserved assets/build/fonts when available, and ui_kits/app/.
- Before final response, run `"$OD_NODE_BIN" "$OD_BIN" tools connectors design-system-package-audit --path . --fail-on-warnings` and fix every actionable issue.
