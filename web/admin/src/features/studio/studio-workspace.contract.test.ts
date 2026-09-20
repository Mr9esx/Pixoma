import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

const read = (path: string) =>
  readFileSync(new URL(path, import.meta.url), 'utf8')

describe('Studio production workspace contract', () => {
  it('uses the official assistant-ui AG-UI runtime and Pixoma primitives', () => {
    const source = read('./studio-chat.tsx')
    expect(source).toContain("from '@assistant-ui/react-ag-ui'")
    expect(source).toContain("from '@assistant-ui/react'")
    expect(source).toContain("from '@ag-ui/client'")
    expect(source).toContain('ComposerPrimitive.Input')
    expect(source).toContain('ThreadPrimitive.Messages')
    expect(source).toContain('@/components/ui/button')
  })

  it('renders an editable React Flow asset road and session assets', () => {
    const source = read('./studio-flow.tsx')
    expect(source).toContain("from '@xyflow/react'")
    expect(source).toContain('onNodesChange')
    expect(source).toContain('Background')
    expect(source).toContain('Controls')

    const workspace = read('./studio-workspace.tsx')
    expect(workspace).toContain('StudioFlow')
    expect(workspace).toContain('StudioAssets')
    expect(workspace).toContain('StudioLibrary')
  })

  it('provides the independent Studio shell and the single platform entry', () => {
    expect(read('../../routes/_app.tsx')).toContain('StudioLayout')
    expect(read('../../config/menu.ts')).toContain("id: 'studio'")
    expect(read('../../routes/_app/studio.tsx')).toContain('StudioWorkspace')
  })
})
