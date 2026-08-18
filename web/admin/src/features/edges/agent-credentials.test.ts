import { describe, expect, it } from 'vitest'
import { edgeDeployCommand } from './deploy-command'

const input = {
  controlPlaneURL: 'https://cp.example',
  token: 'secret-token',
  edgeId: 'node-abc',
  blobDriver: 'localfs',
}

describe('edgeDeployCommand', () => {
  it('includes the real token when copying', () => {
    const command = edgeDeployCommand(input)
    expect(command).toContain('export AGENT_TOKEN=secret-token')
    expect(command).toContain('export EDGE_ID=node-abc')
    expect(command).not.toContain('INSTANCE_ID')
  })

  it('masks AGENT_TOKEN instead of hiding the command', () => {
    const command = edgeDeployCommand({ ...input, maskToken: true })
    expect(command).toContain('export AGENT_TOKEN=********')
    expect(command).not.toContain('secret-token')
    expect(command).toContain('export CONTROL_PLANE_URL=https://cp.example')
    expect(command).toContain('pixoma-edge-agent')
  })
})
