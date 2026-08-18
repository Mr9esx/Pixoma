export function edgeDeployCommand(input: {
  controlPlaneURL: string
  token: string
  edgeId: string
  blobDriver: string
  comfyURL?: string
  maskToken?: boolean
}): string {
  const token = input.maskToken ? '********' : input.token
  // Comfy runs on the same machine as the agent; the operator edits this line
  // only when Comfy is not on the default localhost:8188.
  const comfyURL = input.comfyURL ?? 'http://127.0.0.1:8188'
  return [
    `export CONTROL_PLANE_URL=${input.controlPlaneURL}`,
    `export AGENT_TOKEN=${token}`,
    `export EDGE_ID=${input.edgeId}`,
    `export BLOB_DRIVER=${input.blobDriver}`,
    `export COMFYUI_BASE_URL=${comfyURL}`,
    'pixoma-edge-agent',
  ].join('\n')
}
