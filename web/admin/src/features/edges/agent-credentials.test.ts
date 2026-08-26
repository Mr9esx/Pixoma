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

  it('adds TOS credentials to the deploy command', () => {
    const command = edgeDeployCommand({
      ...input,
      blobDriver: 'tos',
      blobEndpoint: 'https://tos-cn-beijing.volces.com',
      blobRegion: 'cn-beijing',
      blobBucket: 'pixoma',
      blobAccessKey: 'tos-ak',
      blobSecretKey: 'tos-sk',
    })
    expect(command).toContain(
      'export TOS_ENDPOINT=https://tos-cn-beijing.volces.com'
    )
    expect(command).toContain('export TOS_REGION=cn-beijing')
    expect(command).toContain('export TOS_BUCKET=pixoma')
    expect(command).toContain('export TOS_ACCESS_KEY=tos-ak')
    expect(command).toContain('export TOS_SECRET_KEY=tos-sk')
  })

  it('adds S3 credentials to the deploy command', () => {
    const command = edgeDeployCommand({
      ...input,
      blobDriver: 's3',
      blobEndpoint: 'http://minio.local:9000',
      blobRegion: 'us-east-1',
      blobBucket: 'pixoma',
      blobAccessKey: 's3-ak',
      blobSecretKey: 's3-sk',
    })
    expect(command).toContain('export S3_ENDPOINT=http://minio.local:9000')
    expect(command).toContain('export S3_REGION=us-east-1')
    expect(command).toContain('export S3_BUCKET=pixoma')
    expect(command).toContain('export S3_ACCESS_KEY=s3-ak')
    expect(command).toContain('export S3_SECRET_KEY=s3-sk')
  })

  it('masks blob secrets in the displayed command', () => {
    const command = edgeDeployCommand({
      ...input,
      blobDriver: 'tos',
      blobAccessKey: 'tos-ak',
      blobSecretKey: 'tos-sk',
      maskToken: true,
    })
    expect(command).toContain('export TOS_ACCESS_KEY=********')
    expect(command).toContain('export TOS_SECRET_KEY=********')
    expect(command).not.toContain('tos-ak')
    expect(command).not.toContain('tos-sk')
  })

  it('adds BLOB_LOCAL_ROOT for local directory drivers', () => {
    const command = edgeDeployCommand({
      ...input,
      blobDriver: 'sharedfs',
      blobRoot: '/mnt/pixoma-shared',
    })
    expect(command).toContain('export BLOB_LOCAL_ROOT=/mnt/pixoma-shared')
  })

  it('adds EDGE_SUBSCRIBE_TOPICS when topics are selected', () => {
    const command = edgeDeployCommand({
      ...input,
      subscribeTopics: ['default', 'fast-gpu'],
    })
    expect(command).toContain('export EDGE_SUBSCRIBE_TOPICS=default,fast-gpu')
  })

  it('omits EDGE_SUBSCRIBE_TOPICS when no topics are selected', () => {
    expect(edgeDeployCommand(input)).not.toContain('EDGE_SUBSCRIBE_TOPICS')
    expect(edgeDeployCommand({ ...input, subscribeTopics: [] })).not.toContain(
      'EDGE_SUBSCRIBE_TOPICS'
    )
  })
})
