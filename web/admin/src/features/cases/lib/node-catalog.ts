import {
  Image,
  FileText,
  SlidersHorizontal,
  Video,
  Music,
  Wand2,
  Package,
  Cpu,
  type Box,
} from 'lucide-react'

export type InputKind =
  | 'string'
  | 'number'
  | 'boolean'
  | 'enum'
  | 'image'
  | 'audio'
  | 'video'
  | 'ref'
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

/** 节点输出类型（供输出字段绑定自动带出）。 */
const NODE_OUTPUT_KINDS: Record<string, 'image' | 'text' | 'file'> = {
  LoadImage: 'image',
  SaveImage: 'image',
  KSampler: 'image',
  VAEDecode: 'image',
  VAEEncode: 'image',
  EmptyLatentImage: 'image',
  CLIPTextEncode: 'text',
  PreviewImage: 'image',
}

export function outputKindFor(classType: string): 'image' | 'text' | 'file' {
  return NODE_OUTPUT_KINDS[classType] ?? 'image'
}

/** 节点类型 → 图标 + 分类色（详情页流程图与绑定弹层共用）。 */
export function nodeVisualFor(classType: string): {
  Icon: typeof Box
  className: string
} {
  const c = classType.toLowerCase()
  if (c.includes('video') || c.includes('createvideo'))
    return { Icon: Video, className: 'bg-muted text-muted-foreground' }
  if (c.includes('audio') || c.includes('music'))
    return { Icon: Music, className: 'bg-muted text-muted-foreground' }
  if (c.includes('sample') || c.includes('noise') || c.includes('scheduler'))
    return {
      Icon: SlidersHorizontal,
      className: 'bg-muted text-muted-foreground',
    }
  if (c.includes('loader') || c.includes('lora') || c.includes('patch'))
    return { Icon: Package, className: 'bg-muted text-muted-foreground' }
  if (c.includes('text') || c.includes('clip') || c.includes('encode'))
    return { Icon: FileText, className: 'bg-muted text-muted-foreground' }
  if (c.includes('image') || c.includes('vae') || c.includes('decode'))
    return { Icon: Image, className: 'bg-muted text-muted-foreground' }
  if (c.includes('math') || c.includes('primitive') || c.includes('size'))
    return { Icon: Cpu, className: 'bg-muted text-muted-foreground' }
  return { Icon: Wand2, className: 'bg-muted text-muted-foreground' }
}
