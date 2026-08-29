export function edgeDeployCommand(input: {
  controlPlaneURL: string
  token: string
  edgeId: string
  blobDriver: string
  comfyURL?: string
  maskToken?: boolean
  blobRoot?: string
  blobEndpoint?: string
  blobRegion?: string
  blobBucket?: string
  blobAccessKey?: string
  blobSecretKey?: string
}): string {
  const token = input.maskToken ? '********' : input.token
  // Comfy runs on the same machine as the agent; the operator edits this line
  // only when Comfy is not on the default localhost:8188.
  const comfyURL = input.comfyURL ?? 'http://127.0.0.1:8188'
  // Topic bindings live on the control plane and are PATCHed via the admin
  // node form. The agent no longer reads a topic env var at startup;
  // it just reports presence and claims jobs filtered by what the admin set.
  const lines = [
    `export CONTROL_PLANE_URL=${input.controlPlaneURL}`,
    `export AGENT_TOKEN=${token}`,
    `export EDGE_ID=${input.edgeId}`,
    `export BLOB_DRIVER=${input.blobDriver}`,
  ]
  if (input.blobDriver === 'localfs' || input.blobDriver === 'sharedfs') {
    if (input.blobRoot) {
      lines.push(`export BLOB_LOCAL_ROOT=${input.blobRoot}`)
    }
  }
  if (input.blobDriver === 'tos') {
    const accessKey = input.maskToken ? '********' : (input.blobAccessKey ?? '')
    const secretKey = input.maskToken ? '********' : (input.blobSecretKey ?? '')
    lines.push(
      `export TOS_ENDPOINT=${input.blobEndpoint ?? ''}`,
      `export TOS_REGION=${input.blobRegion ?? ''}`,
      `export TOS_BUCKET=${input.blobBucket ?? ''}`,
      `export TOS_ACCESS_KEY=${accessKey}`,
      `export TOS_SECRET_KEY=${secretKey}`
    )
  }
  if (input.blobDriver === 's3') {
    const accessKey = input.maskToken ? '********' : (input.blobAccessKey ?? '')
    const secretKey = input.maskToken ? '********' : (input.blobSecretKey ?? '')
    lines.push(
      `export S3_ENDPOINT=${input.blobEndpoint ?? ''}`,
      `export S3_REGION=${input.blobRegion ?? 'us-east-1'}`,
      `export S3_BUCKET=${input.blobBucket ?? ''}`,
      `export S3_ACCESS_KEY=${accessKey}`,
      `export S3_SECRET_KEY=${secretKey}`
    )
  }
  lines.push(`export COMFYUI_BASE_URL=${comfyURL}`, 'pixoma-edge-agent')
  return lines.join('\n')
}
