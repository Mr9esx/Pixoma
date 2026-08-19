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
