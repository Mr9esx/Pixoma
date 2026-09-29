import { useMemo, useRef, useState } from 'react'
import { createRoot } from 'react-dom/client'
import {
  ArrowDownUp,
  AudioLines,
  ChevronDown,
  Download,
  FileText,
  Folder,
  FolderPlus,
  Grid2X2,
  ImageIcon,
  Library,
  List,
  Maximize2,
  Menu,
  MessageCircle,
  Minimize2,
  Pencil,
  Search,
  Settings2,
  SlidersHorizontal,
  Star,
  Tag,
  Upload,
  Video,
  WandSparkles,
  X,
} from 'lucide-react'
import { Logo } from '@/assets/logo'
import { FilterSegment } from '@/components/filters/filter-segment'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { cn } from '@/lib/utils'
import '@/styles/index.css'

type AssetKind = 'image' | 'document' | 'audio' | 'video'
type KindFilter = AssetKind | 'all'
type SortOrder = 'recent' | 'name' | 'rating'
type Artwork = 'portrait' | 'city' | 'frames' | 'cover' | 'document' | 'audio' | 'video'
type TreeMode = 'asset' | 'conversation' | 'category' | 'format' | 'rating' | 'tag'

type DemoAsset = {
  id: string
  name: string
  originalName?: string
  project: string
  sourceSession?: string
  kind: AssetKind
  category: string
  tags: string[]
  rating: number
  sizeBytes: number
  format: string
  dimensions: string
  addedAt: string
  createdAt: string
  modifiedAt: string
  source: string
  version: number
  artwork: Artwork
  palette?: string[]
  paletteError?: string
  uploadedFile?: File
  duplicate?: boolean
  previewUrl?: string
}

const samplePalettes: Partial<Record<Artwork, string[]>> = {
  portrait: ['#374E46', '#5E7770', '#859485', '#B7AAA0', '#D3BDAB', '#6C6760', '#A8A69E', '#34332E'],
  city: ['#1D3134', '#34515B', '#496873', '#79909A', '#AAADB0', '#D5C6B3', '#7E776C', '#353A3A'],
  frames: ['#353F3C', '#61736E', '#8A9990', '#B6BBB1', '#D1C2B0', '#787167', '#555F5B', '#232E30'],
  cover: ['#282E30', '#3F5556', '#687E7D', '#A5AAA2', '#D8C8B6', '#B5937F', '#745D57', '#353F40'],
  video: ['#202B31', '#344750', '#5C6C6F', '#879395', '#C5B8A4', '#8D7869', '#494E4B', '#2A3335'],
}

const initialAssets: DemoAsset[] = [
  {
    id: 'character',
    name: '角色三视图.png',
    project: '雨夜侦探漫画',
    sourceSession: '角色设计',
    kind: 'image',
    category: '角色设定',
    tags: ['角色', '设定'],
    rating: 4,
    sizeBytes: 109281,
    format: 'PNG',
    dimensions: '2000 × 2000',
    addedAt: '2026/09/28 12:41',
    createdAt: '2026/04/28 13:08',
    modifiedAt: '2026/04/28 13:08',
    source: '模型生成',
    version: 1,
    artwork: 'portrait',
    palette: samplePalettes.portrait,
  },
  {
    id: 'street',
    name: '雨夜街景.png',
    project: '雨夜侦探漫画',
    sourceSession: '城市分镜',
    kind: 'image',
    category: '场景参考',
    tags: ['场景', '夜景'],
    rating: 5,
    sizeBytes: 2394931,
    format: 'PNG',
    dimensions: '2560 × 1440',
    addedAt: '2026/09/27 18:26',
    createdAt: '2026/09/27 18:24',
    modifiedAt: '2026/09/27 18:24',
    source: '工作流',
    version: 2,
    artwork: 'city',
    palette: samplePalettes.city,
  },
  {
    id: 'storyboard',
    name: '分镜草稿 03.png',
    project: '雨夜侦探漫画',
    sourceSession: '城市分镜',
    kind: 'image',
    category: '分镜',
    tags: ['分镜', '精选'],
    rating: 4,
    sizeBytes: 1428074,
    format: 'PNG',
    dimensions: '1920 × 1080',
    addedAt: '2026/09/26 15:19',
    createdAt: '2026/09/26 14:52',
    modifiedAt: '2026/09/26 15:03',
    source: '工作流',
    version: 3,
    artwork: 'frames',
    palette: samplePalettes.frames,
    duplicate: true,
  },
  {
    id: 'cover',
    name: '封面候选 A.png',
    project: '封面探索',
    sourceSession: '封面草图',
    kind: 'image',
    category: '场景参考',
    tags: ['封面', '场景'],
    rating: 3,
    sizeBytes: 1874321,
    format: 'PNG',
    dimensions: '1600 × 2400',
    addedAt: '2026/09/25 11:08',
    createdAt: '2026/09/25 11:07',
    modifiedAt: '2026/09/25 11:07',
    source: '模型生成',
    version: 1,
    artwork: 'cover',
    palette: samplePalettes.cover,
  },
  {
    id: 'story',
    name: '故事设定.md',
    project: '雨夜侦探漫画',
    sourceSession: '故事设定',
    kind: 'document',
    category: '文档',
    tags: ['故事', '设定'],
    rating: 5,
    sizeBytes: 18342,
    format: 'MD',
    dimensions: '—',
    addedAt: '2026/09/23 09:34',
    createdAt: '2026/09/21 10:16',
    modifiedAt: '2026/09/23 09:32',
    source: 'Studio 编辑',
    version: 4,
    artwork: 'document',
  },
  {
    id: 'sound',
    name: '雨声参考.mp3',
    project: '雨夜侦探漫画',
    sourceSession: '城市分镜',
    kind: 'audio',
    category: '声音',
    tags: ['环境音'],
    rating: 0,
    sizeBytes: 4227416,
    format: 'MP3',
    dimensions: '—',
    addedAt: '2026/09/22 16:40',
    createdAt: '—',
    modifiedAt: '2026/09/19 20:12',
    source: '本机上传',
    version: 1,
    artwork: 'audio',
  },
  {
    id: 'shot',
    name: '镜头试剪.mp4',
    project: '雨夜侦探漫画',
    sourceSession: '城市分镜',
    kind: 'video',
    category: '分镜',
    tags: ['分镜', '视频'],
    rating: 2,
    sizeBytes: 16935731,
    format: 'MP4',
    dimensions: '1920 × 1080',
    addedAt: '2026/09/21 14:05',
    createdAt: '2026/09/21 13:48',
    modifiedAt: '2026/09/21 13:55',
    source: '工作流',
    version: 1,
    artwork: 'video',
    palette: samplePalettes.video,
  },
  {
    id: 'storyboard-copy',
    name: '分镜草稿 03 副本.png',
    project: '未归属项目',
    kind: 'image',
    category: '未分类',
    tags: ['分镜'],
    rating: 0,
    sizeBytes: 1428074,
    format: 'PNG',
    dimensions: '1920 × 1080',
    addedAt: '2026/09/20 12:15',
    createdAt: '—',
    modifiedAt: '2026/09/20 12:10',
    source: '本机上传',
    version: 1,
    artwork: 'frames',
    palette: samplePalettes.frames,
    duplicate: true,
  },
]

const projects = ['雨夜侦探漫画', '封面探索', '未归属项目']
const initialCategoriesByProject: Record<string, string[]> = {
  雨夜侦探漫画: ['角色设定', '场景参考', '分镜', '文档', '声音'],
  封面探索: ['场景参考'],
  未归属项目: [],
}
const kindOptions: { value: KindFilter; label: string }[] = [
  { value: 'all', label: '全部' },
  { value: 'image', label: '图片' },
  { value: 'video', label: '视频' },
  { value: 'audio', label: '音频' },
  { value: 'document', label: '文档' },
]
const treeModeOptions: { value: TreeMode; label: string }[] = [
  { value: 'asset', label: '项目 → 资产' },
  { value: 'conversation', label: '项目 → 对话' },
  { value: 'category', label: '项目 → 分类' },
  { value: 'format', label: '项目 → 格式' },
  { value: 'rating', label: '项目 → 评分' },
  { value: 'tag', label: '项目 → 标签' },
]
const tagOptions = ['角色', '设定', '场景', '分镜', '精选', '夜景']

function formatSize(bytes: number) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(2) + ' MB'
}

function paletteFromFrames(frames: Uint8ClampedArray[]) {
  const counts = new Map<number, number>()
  for (const frame of frames) {
    for (let index = 0; index < frame.length; index += 16) {
      if (frame[index + 3] < 128) continue
      const red = Math.min(255, Math.round(frame[index] / 32) * 32)
      const green = Math.min(255, Math.round(frame[index + 1] / 32) * 32)
      const blue = Math.min(255, Math.round(frame[index + 2] / 32) * 32)
      const color = red * 65536 + green * 256 + blue
      counts.set(color, (counts.get(color) ?? 0) + 1)
    }
  }
  const selected: number[] = []
  for (const [color] of [...counts].sort((a, b) => b[1] - a[1])) {
    const red = color >> 16
    const green = (color >> 8) & 255
    const blue = color & 255
    if (selected.every((item) => {
      const distance = (red - (item >> 16)) ** 2 + (green - ((item >> 8) & 255)) ** 2 + (blue - (item & 255)) ** 2
      return distance >= 1600
    })) selected.push(color)
    if (selected.length === 10) break
  }
  return selected.map((color) => '#' + color.toString(16).padStart(6, '0').toUpperCase())
}

async function analyzePalette(file: File, previewUrl: string) {
  const canvas = document.createElement('canvas')
  canvas.width = 96
  canvas.height = 96
  const context = canvas.getContext('2d', { willReadFrequently: true })
  if (!context) throw new Error('当前浏览器无法分析调色盘')
  const frames: Uint8ClampedArray[] = []
  const sample = (source: CanvasImageSource) => {
    context.clearRect(0, 0, 96, 96)
    context.drawImage(source, 0, 0, 96, 96)
    frames.push(context.getImageData(0, 0, 96, 96).data)
  }
  let dimensions: string
  if (file.type.startsWith('image/')) {
    const image = await createImageBitmap(file)
    dimensions = image.width + ' × ' + image.height
    sample(image)
    image.close()
  } else {
    const video = document.createElement('video')
    video.muted = true
    video.preload = 'auto'
    await new Promise<void>((resolve, reject) => {
      video.onloadeddata = () => resolve()
      video.onerror = () => reject(new Error('当前浏览器无法读取该视频'))
      video.src = previewUrl
      video.load()
    })
    dimensions = video.videoWidth + ' × ' + video.videoHeight
    const duration = Number.isFinite(video.duration) ? video.duration : 0
    for (const fraction of [0.1, 0.5, 0.9]) {
      const time = duration * fraction
      if (Math.abs(video.currentTime - time) > 0.05) {
        await new Promise<void>((resolve, reject) => {
          video.onseeked = () => resolve()
          video.onerror = () => reject(new Error('无法读取视频画面'))
          video.currentTime = time
        })
      }
      sample(video)
    }
    video.removeAttribute('src')
    video.load()
  }
  const palette = paletteFromFrames(frames)
  if (!palette.length) throw new Error('文件中没有可分析的颜色')
  return { palette, dimensions }
}

export function ArtworkPreview({ asset, large = false }: { asset: DemoAsset; large?: boolean }) {
  if (asset.previewUrl && asset.kind === 'image') {
    return <img src={asset.previewUrl} alt='' className='size-full object-contain' />
  }
  if (asset.artwork === 'portrait') {
    return (
      <div className='flex size-full items-end justify-center gap-3 overflow-hidden bg-secondary px-4 pt-5 text-foreground/30'>
        {[0, 1, 2].map((item) => (
          <div key={item} className='flex h-[85%] w-[27%] flex-col items-center justify-end rounded-t-full border border-current bg-card/50'>
            <div className='mb-2 size-9 rounded-full bg-current opacity-60' />
            <div className='h-[42%] w-[70%] rounded-t-full bg-current opacity-60' />
          </div>
        ))}
      </div>
    )
  }
  if (asset.artwork === 'city') {
    return (
      <div className='relative size-full overflow-hidden bg-foreground text-background'>
        <div className='absolute top-[20%] right-[22%] size-9 rounded-full bg-current opacity-70' />
        <div className='absolute inset-x-0 bottom-0 flex h-[65%] items-end justify-between gap-1 px-4 text-background/55'>
          {[42, 71, 56, 85, 63, 49, 77].map((height, index) => (
            <div key={index} className='w-full bg-current' style={{ height: height + '%' }} />
          ))}
        </div>
        <div className='absolute inset-x-0 bottom-[22%] border-t border-background/60' />
      </div>
    )
  }
  if (asset.artwork === 'frames') {
    return (
      <div className='grid size-full grid-cols-2 gap-2 bg-muted p-4 text-foreground/35'>
        {[0, 1, 2, 3].map((item) => (
          <div key={item} className='relative overflow-hidden rounded-sm border border-current bg-card'>
            <div className='absolute top-[22%] left-[38%] size-5 rounded-full bg-current opacity-65' />
            <div className='absolute right-[14%] bottom-0 left-[14%] h-[48%] rounded-t-full bg-current opacity-65' />
          </div>
        ))}
      </div>
    )
  }
  if (asset.artwork === 'cover') {
    return (
      <div className='flex size-full flex-col items-center justify-center border-[12px] border-secondary bg-foreground text-background'>
        <span className={cn('tracking-[0.25em]', large ? 'text-3xl' : 'text-xl')}>雨夜侦探</span>
        <span className='mt-2 border-t border-current pt-2 text-[10px] tracking-[0.4em]'>封面探索</span>
      </div>
    )
  }
  if (asset.artwork === 'document') {
    return (
      <div className='flex size-full items-center justify-center bg-muted p-5'>
        <div className='flex h-full w-[70%] flex-col gap-2 rounded-sm border bg-card p-4'>
          <span className='text-xs font-semibold'>雨夜侦探</span>
          {[85, 66, 94, 73, 57].map((width, index) => (
            <div key={index} className='h-1 rounded-full bg-muted-foreground/25' style={{ width: width + '%' }} />
          ))}
        </div>
      </div>
    )
  }
  if (asset.artwork === 'audio') {
    return (
      <div className='flex size-full items-center justify-center gap-1 bg-secondary text-foreground/45'>
        {[25, 48, 80, 39, 66, 95, 53, 72, 34, 60, 87, 45, 28].map((height, index) => (
          <div key={index} className='w-1.5 rounded-full bg-current' style={{ height: height + '%' }} />
        ))}
      </div>
    )
  }
  return (
    <div className='flex size-full items-center justify-center bg-foreground text-background'>
      <Video className={large ? 'size-14' : 'size-10'} strokeWidth={1.25} />
    </div>
  )
}

export function App() {
  const [assets, setAssets] = useState(initialAssets)
  const [categoriesByProject, setCategoriesByProject] = useState(initialCategoriesByProject)
  const [project, setProject] = useState('雨夜侦探漫画')
  const [expandedProjects, setExpandedProjects] = useState<string[]>(['雨夜侦探漫画'])
  const [treeMode, setTreeMode] = useState<TreeMode>('asset')
  const [treeSelection, setTreeSelection] = useState<{ mode: TreeMode; value: string } | null>(null)
  const [category, setCategory] = useState('全部资产')
  const [sourceSession, setSourceSession] = useState('全部对话')
  const [query, setQuery] = useState('')
  const [kind, setKind] = useState<KindFilter>('all')
  const [formatFilter, setFormatFilter] = useState('全部格式')
  const [minimumRating, setMinimumRating] = useState(0)
  const [tag, setTag] = useState('全部标签')
  const [duplicatesOnly, setDuplicatesOnly] = useState(false)
  const [sort, setSort] = useState<SortOrder>('recent')
  const [view, setView] = useState<'grid' | 'list'>('grid')
  const [selectedId, setSelectedId] = useState('')
  const [checkedIds, setCheckedIds] = useState<string[]>([])
  const [detailOpen, setDetailOpen] = useState(false)
  const [previewExpanded, setPreviewExpanded] = useState(false)
  const [treeOpen, setTreeOpen] = useState(false)
  const [editingNameId, setEditingNameId] = useState<string | null>(null)
  const [nameDraft, setNameDraft] = useState('')
  const [nameError, setNameError] = useState('')
  const [categoryToRename, setCategoryToRename] = useState<string | null>(null)
  const [renameCategoryDraft, setRenameCategoryDraft] = useState('')
  const [categoryRenameError, setCategoryRenameError] = useState('')
  const [moveOpen, setMoveOpen] = useState(false)
  const [moveIds, setMoveIds] = useState<string[]>([])
  const [moveTarget, setMoveTarget] = useState('未分类')
  const [exportOpen, setExportOpen] = useState(false)
  const [exportIds, setExportIds] = useState<string[]>([])
  const [createCategoryOpen, setCreateCategoryOpen] = useState(false)
  const [categoryName, setCategoryName] = useState('')
  const [message, setMessage] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)
  const categories = categoriesByProject[project] ?? []
  const projectAssets = assets.filter((asset) => asset.project === project)
  const sourceSessions = [...new Set(projectAssets.map((asset) => asset.sourceSession).filter((item): item is string => Boolean(item)))]
  const formats = [...new Set(projectAssets.map((asset) => asset.format))].sort((a, b) => a.localeCompare(b))

  const visibleAssets = useMemo(() => {
    const normalizedQuery = query.trim().toLocaleLowerCase()
    const result = assets.filter((asset) => {
      if (asset.project !== project) return false
      if (category !== '全部资产' && asset.category !== category) return false
      if (sourceSession !== '全部对话' && asset.sourceSession !== sourceSession) return false
      if (kind !== 'all' && asset.kind !== kind) return false
      if (formatFilter !== '全部格式' && asset.format !== formatFilter) return false
      if (asset.rating < minimumRating) return false
      if (tag !== '全部标签' && !asset.tags.includes(tag)) return false
      if (duplicatesOnly && !asset.duplicate) return false
      if (treeSelection) {
        const value = treeSelection.value
        if (treeSelection.mode === 'conversation' && (asset.sourceSession ?? '无来源对话') !== value) return false
        if (treeSelection.mode === 'category' && asset.category !== value) return false
        if (treeSelection.mode === 'format' && asset.format !== value) return false
        if (treeSelection.mode === 'rating' && (asset.rating ? asset.rating + ' 星' : '未评分') !== value) return false
        if (treeSelection.mode === 'tag' && !(value === '无标签' ? asset.tags.length === 0 : asset.tags.includes(value))) return false
      }
      if (normalizedQuery && !(asset.name + ' ' + (asset.originalName ?? '') + ' ' + asset.tags.join(' ')).toLocaleLowerCase().includes(normalizedQuery)) return false
      return true
    })
    if (sort === 'name') result.sort((a, b) => a.name.localeCompare(b.name, 'zh-CN'))
    if (sort === 'rating') result.sort((a, b) => b.rating - a.rating || b.addedAt.localeCompare(a.addedAt))
    if (sort === 'recent') result.sort((a, b) => b.addedAt.localeCompare(a.addedAt))
    return result
  }, [assets, category, duplicatesOnly, formatFilter, kind, minimumRating, project, query, sort, sourceSession, tag, treeSelection])

  const activeAsset = assets.find((asset) => asset.id === selectedId)
  const moveProject = assets.find((asset) => asset.id === moveIds[0])?.project ?? project
  const filtersActive = minimumRating > 0 || tag !== '全部标签' || duplicatesOnly
  const targetAssets = exportIds.map((id) => assets.find((asset) => asset.id === id)).filter((asset): asset is DemoAsset => Boolean(asset))

  function changeScope(action: () => void) {
    action()
    setCheckedIds([])
  }

  function openProject(nextProject: string) {
    setExpandedProjects((current) => current.includes(nextProject) ? current : [...current, nextProject])
    if (nextProject === project) return
    setProject(nextProject)
    setTreeSelection(null)
    setCategory('全部资产')
    setFormatFilter('全部格式')
    setSourceSession('全部对话')
    setQuery('')
    setCheckedIds([])
  }

  function openAsset(id: string) {
    setSelectedId(id)
    setPreviewExpanded(false)
    setEditingNameId(null)
    setNameError('')
    setDetailOpen(true)
  }

  function saveAssetName(asset: DemoAsset) {
    const nextName = nameDraft.trim()
    if (!nextName) {
      setNameError('输入资产名称')
      return
    }
    setAssets((current) => current.map((item) => item.id === asset.id ? { ...item, name: nextName, originalName: item.originalName ?? item.name } : item))
    setEditingNameId(null)
    setNameError('')
    setMessage('已重命名 ' + nextName)
  }

  function saveCategoryName() {
    const nextName = renameCategoryDraft.trim()
    if (!nextName) {
      setCategoryRenameError('输入分类名称')
      return
    }
    if (nextName !== categoryToRename && categories.includes(nextName)) {
      setCategoryRenameError('分类名称已存在')
      return
    }
    setCategoriesByProject((current) => ({ ...current, [project]: current[project].map((item) => item === categoryToRename ? nextName : item) }))
    setAssets((current) => current.map((asset) => asset.project === project && asset.category === categoryToRename ? { ...asset, category: nextName } : asset))
    if (category === categoryToRename) setCategory(nextName)
    if (moveTarget === categoryToRename) setMoveTarget(nextName)
    setCategoryToRename(null)
    setCategoryRenameError('')
    setMessage('已重命名分类 ' + nextName)
  }

  function toggleChecked(id: string, checked: boolean) {
    setCheckedIds((current) => checked ? [...current, id] : current.filter((item) => item !== id))
  }

  function setRating(id: string, rating: number) {
    setAssets((current) => current.map((asset) => asset.id === id ? { ...asset, rating: asset.rating === rating ? 0 : rating } : asset))
  }

  function addTag(id: string, nextTag: string) {
    setAssets((current) => current.map((asset) => asset.id === id && !asset.tags.includes(nextTag) ? { ...asset, tags: [...asset.tags, nextTag] } : asset))
  }

  function removeTag(id: string, nextTag: string) {
    setAssets((current) => current.map((asset) => asset.id === id ? { ...asset, tags: asset.tags.filter((item) => item !== nextTag) } : asset))
  }

  function openMove(ids: string[]) {
    setDetailOpen(false)
    setMoveIds(ids)
    const targetProject = assets.find((asset) => asset.id === ids[0])?.project ?? project
    setMoveTarget(project === targetProject && category !== '全部资产' ? category : '未分类')
    setMoveOpen(true)
  }

  function openExport(ids: string[]) {
    setDetailOpen(false)
    setExportIds(ids)
    setExportOpen(true)
  }

  function startPaletteAnalysis(asset: DemoAsset) {
    if (!asset.uploadedFile || !asset.previewUrl) return
    setAssets((current) => current.map((item) => item.id === asset.id ? { ...item, paletteError: undefined } : item))
    analyzePalette(asset.uploadedFile, asset.previewUrl)
      .then(({ palette, dimensions }) => setAssets((current) => current.map((item) => item.id === asset.id ? { ...item, palette, dimensions } : item)))
      .catch((error: Error) => setAssets((current) => current.map((item) => item.id === asset.id ? { ...item, paletteError: error.message } : item)))
  }

  function uploadFile(file: File) {
    const now = new Date().toLocaleString('zh-CN', { hour12: false }).replaceAll('-', '/')
    const id = crypto.randomUUID()
    const kind: AssetKind = file.type.startsWith('image/') ? 'image' : file.type.startsWith('audio/') ? 'audio' : file.type.startsWith('video/') ? 'video' : 'document'
    const format = ({ 'image/jpeg': 'JPEG', 'image/png': 'PNG', 'image/webp': 'WebP', 'image/gif': 'GIF', 'image/svg+xml': 'SVG', 'video/mp4': 'MP4', 'video/webm': 'WebM', 'audio/mpeg': 'MP3', 'text/markdown': 'MD', 'application/pdf': 'PDF' } as Record<string, string>)[file.type] ?? (file.type.split('/')[1]?.toUpperCase() || '未知格式')
    const previewUrl = URL.createObjectURL(file)
    const asset: DemoAsset = {
      id,
      name: file.name,
      project,
      kind,
      category: category === '全部资产' ? '未分类' : category,
      tags: [],
      rating: 0,
      sizeBytes: file.size,
      format,
      dimensions: '—',
      addedAt: now,
      createdAt: '—',
      modifiedAt: file.lastModified ? new Date(file.lastModified).toLocaleString('zh-CN', { hour12: false }).replaceAll('-', '/') : '—',
      source: '本机上传',
      version: 1,
      artwork: kind === 'image' ? 'cover' : kind === 'audio' ? 'audio' : kind === 'video' ? 'video' : 'document',
      previewUrl,
      uploadedFile: file,
    }
    setAssets((current) => [asset, ...current])
    setSelectedId(id)
    setPreviewExpanded(false)
    setDetailOpen(true)
    setCategory('全部资产')
    setSourceSession('全部对话')
    setQuery('')
    setKind('all')
    setFormatFilter('全部格式')
    setTreeSelection(null)
    setMinimumRating(0)
    setTag('全部标签')
    setDuplicatesOnly(false)
    setMessage('已添加 ' + file.name)
    if (kind === 'image' || kind === 'video') startPaletteAnalysis(asset)
  }

  function renderProjectTree() {
    return (
      <nav aria-label='项目资产树' className='flex flex-col gap-1 p-2'>
        {projects.map((item) => {
          const projectItems = assets.filter((asset) => asset.project === item).sort((a, b) => b.addedAt.localeCompare(a.addedAt))
          const expanded = expandedProjects.includes(item)
          const counts = new Map<string, number>()
          if (treeMode !== 'asset') {
            for (const asset of projectItems) {
              const labels = treeMode === 'conversation' ? [asset.sourceSession ?? '无来源对话']
                : treeMode === 'category' ? [asset.category]
                  : treeMode === 'format' ? [asset.format]
                    : treeMode === 'rating' ? [asset.rating ? asset.rating + ' 星' : '未评分']
                      : asset.tags.length ? asset.tags : ['无标签']
              for (const label of labels) counts.set(label, (counts.get(label) ?? 0) + 1)
            }
          }
          const groups = [...counts].sort(([a], [b]) => treeMode === 'rating' ? b.localeCompare(a, 'zh-CN') : a.localeCompare(b, 'zh-CN'))
          return <div key={item}>
            <div className='flex items-center gap-1'>
              <Button variant={project === item ? 'secondary' : 'ghost'} className='h-10 min-w-0 flex-1 justify-start gap-2 px-2 font-normal' onClick={() => { openProject(item); setTreeOpen(false) }}>
                <Folder className='size-4 shrink-0' />
                <span className='min-w-0 flex-1 truncate text-left'>{item}</span>
                <span className='text-xs text-muted-foreground'>{projectItems.length}</span>
              </Button>
              <Button variant='ghost' size='icon' className='size-9 shrink-0' aria-label={(expanded ? '收起' : '展开') + item} aria-expanded={expanded} onClick={() => setExpandedProjects((current) => expanded ? current.filter((name) => name !== item) : [...current, item])}>
                <ChevronDown className={cn('size-4 transition-transform', !expanded && '-rotate-90')} />
              </Button>
            </div>
            {expanded ? <div className='ml-4 flex flex-col border-l pl-2'>
              {treeMode === 'asset' ? projectItems.map((asset) => <button key={asset.id} type='button' className={cn('flex h-9 min-w-0 items-center gap-2 rounded-md px-2 text-left text-sm hover:bg-secondary focus-visible:outline-2 focus-visible:outline-ring', selectedId === asset.id && detailOpen && 'bg-secondary')} onClick={() => { openProject(item); openAsset(asset.id); setTreeOpen(false) }}>
                {asset.kind === 'image' ? <ImageIcon className='size-4 shrink-0 text-muted-foreground' /> : asset.kind === 'audio' ? <AudioLines className='size-4 shrink-0 text-muted-foreground' /> : asset.kind === 'video' ? <Video className='size-4 shrink-0 text-muted-foreground' /> : <FileText className='size-4 shrink-0 text-muted-foreground' />}
                <span className='truncate'>{asset.name}</span>
              </button>) : groups.map(([label, count]) => <button key={label} type='button' aria-pressed={project === item && treeSelection?.mode === treeMode && treeSelection.value === label} className={cn('flex h-9 min-w-0 items-center gap-2 rounded-md px-2 text-left text-sm hover:bg-secondary focus-visible:outline-2 focus-visible:outline-ring', project === item && treeSelection?.mode === treeMode && treeSelection.value === label && 'bg-secondary')} onClick={() => {
                openProject(item)
                setTreeSelection({ mode: treeMode, value: label })
                setCheckedIds([])
                if (treeMode === 'conversation') setSourceSession('全部对话')
                if (treeMode === 'category') setCategory('全部资产')
                if (treeMode === 'format') setFormatFilter('全部格式')
                if (treeMode === 'rating') setMinimumRating(0)
                if (treeMode === 'tag') setTag('全部标签')
                setTreeOpen(false)
              }}>
                {treeMode === 'conversation' ? <MessageCircle className='size-4 shrink-0 text-muted-foreground' /> : treeMode === 'category' ? <Folder className='size-4 shrink-0 text-muted-foreground' /> : treeMode === 'rating' ? <Star className='size-4 shrink-0 text-muted-foreground' /> : treeMode === 'tag' ? <Tag className='size-4 shrink-0 text-muted-foreground' /> : <FileText className='size-4 shrink-0 text-muted-foreground' />}
                <span className='min-w-0 flex-1 truncate'>{label}</span><span className='text-xs text-muted-foreground'>{count}</span>
              </button>)}
            </div> : null}
          </div>
        })}
      </nav>
    )
  }

  function renderTreeSettings() {
    return <DropdownMenu>
      <DropdownMenuTrigger asChild><Button variant='ghost' size='icon' aria-label='设置文件树'><Settings2 /></Button></DropdownMenuTrigger>
      <DropdownMenuContent align='end'>
        <DropdownMenuRadioGroup value={treeMode} onValueChange={(value) => { setTreeMode(value as TreeMode); setTreeSelection(null); setCheckedIds([]) }}>
          {treeModeOptions.map((item) => <DropdownMenuRadioItem key={item.value} value={item.value}>{item.label}</DropdownMenuRadioItem>)}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  }

  function renderAssetDetails(asset: DemoAsset) {
    return (
      <div className={cn('flex min-h-0 flex-1 flex-col overflow-hidden md:grid', previewExpanded ? 'md:grid-cols-1' : 'md:grid-cols-[minmax(0,2fr)_minmax(320px,1fr)]')}>
        <div className={cn('relative min-h-0 overflow-hidden bg-background', previewExpanded ? 'flex-1' : 'h-[42svh] shrink-0 border-b md:h-full md:border-r md:border-b-0')}>
          <ArtworkPreview asset={asset} large />
          <Button
            variant='secondary'
            size='sm'
            className='absolute top-4 left-4 z-10 border border-border'
            onClick={() => setPreviewExpanded((current) => !current)}
          >
            {previewExpanded ? <Minimize2 /> : <Maximize2 />}
            {previewExpanded ? '恢复布局' : '放大预览'}
          </Button>
        </div>
        {!previewExpanded ? (
          <div className='flex min-h-0 min-w-0 flex-1 flex-col'>
            <ScrollArea className='min-h-0 flex-1'>
              <div className='flex flex-col gap-5 p-6'>
              <div className='flex flex-col gap-2'>
                <div className='min-w-0'>
                  {editingNameId === asset.id ? (
                    <form className='flex flex-col gap-2' onSubmit={(event) => { event.preventDefault(); saveAssetName(asset) }}>
                      <Input
                        autoFocus
                        aria-label='资产名称'
                        value={nameDraft}
                        onChange={(event) => { setNameDraft(event.target.value); setNameError('') }}
                      />
                      {nameError ? <p role='alert' className='text-xs text-destructive'>{nameError}</p> : null}
                      <div className='flex gap-2'>
                        <Button type='submit' size='sm'>保存名称</Button>
                        <Button type='button' variant='ghost' size='sm' onClick={() => { setEditingNameId(null); setNameError('') }}>取消</Button>
                      </div>
                    </form>
                  ) : (
                    <div className='flex items-start gap-2'>
                      <h2 className='min-w-0 flex-1 break-all text-base font-semibold'>{asset.name}</h2>
                      <Button
                        variant='ghost'
                        size='icon'
                        aria-label='重命名资产'
                        onClick={() => { setEditingNameId(asset.id); setNameDraft(asset.name); setNameError('') }}
                      >
                        <Pencil />
                      </Button>
                    </div>
                  )}
                  <p className='mt-1 text-xs text-muted-foreground'>{asset.project} · 固定版本 v{asset.version}</p>
                </div>
                <div className='flex flex-wrap gap-1.5'>
                  {asset.tags.map((item) => (
                    <Badge key={item} variant='secondary' className='gap-1 font-normal'>
                      {item}
                      <button type='button' aria-label={'移除标签 ' + item} onClick={() => removeTag(asset.id, item)}><X className='size-3' /></button>
                    </Badge>
                  ))}
                  <Popover>
                    <PopoverTrigger asChild><Button variant='outline' size='sm'>添加标签</Button></PopoverTrigger>
                    <PopoverContent align='start' className='flex w-44 flex-col gap-1 p-2'>
                      {tagOptions.filter((item) => !asset.tags.includes(item)).map((item) => (
                        <Button key={item} variant='ghost' size='sm' className='justify-start' onClick={() => addTag(asset.id, item)}>{item}</Button>
                      ))}
                    </PopoverContent>
                  </Popover>
                </div>
              </div>
              <div className='border-t pt-4'>
              <h3 className='mb-3 text-sm font-semibold'>基本信息</h3>
              <div className='grid grid-cols-[5.5rem_minmax(0,1fr)] items-center gap-x-3 gap-y-3 text-sm'>
                <span className='text-muted-foreground'>评分</span>
                <div className='flex items-center gap-0.5'>
                  {[1, 2, 3, 4, 5].map((value) => (
                    <button
                      key={value}
                      type='button'
                      aria-label={'评分 ' + value + ' 星'}
                      aria-pressed={asset.rating === value}
                      className={cn('rounded-sm p-0.5 transition-colors focus-visible:outline-2 focus-visible:outline-ring', value <= asset.rating ? 'text-foreground' : 'text-muted-foreground/45 hover:text-foreground')}
                      onClick={() => setRating(asset.id, value)}
                    >
                      <Star className='size-5' fill={value <= asset.rating ? 'currentColor' : 'none'} />
                    </button>
                  ))}
                </div>
                <InfoRow label='尺寸' value={asset.dimensions} />
                <InfoRow label='文件大小' value={formatSize(asset.sizeBytes)} />
                <InfoRow label='格式' value={asset.format} />
                <InfoRow label='添加日期' value={asset.addedAt} />
                <InfoRow label='创建日期' value={asset.createdAt} />
                <InfoRow label='修改日期' value={asset.modifiedAt} />
              </div>
              </div>
              {(asset.kind === 'image' || asset.kind === 'video') ? <div className='border-t pt-4'>
                <h3 className='mb-3 text-sm font-semibold'>调色盘</h3>
                {asset.palette?.length ? <div role='group' aria-label='资产调色盘' className='flex w-fit max-w-full gap-2 overflow-x-auto rounded-full border bg-secondary/40 px-3 py-2'>
                  {asset.palette.map((color, index) => <span key={color + index} role='img' aria-label={'颜色 ' + color} title={color} className='size-7 shrink-0 rounded-full border border-border' style={{ backgroundColor: color }} />)}
                </div> : <div className='flex items-center gap-2'><p className='text-sm text-muted-foreground'>{asset.paletteError ?? '正在分析调色盘…'}</p>{asset.paletteError && asset.uploadedFile ? <Button variant='outline' size='sm' onClick={() => startPaletteAnalysis(asset)}>重新分析</Button> : null}</div>}
              </div> : null}
              <div className='border-t pt-4'>
              <h3 className='mb-3 text-sm font-semibold'>来源与版本</h3>
              <div className='grid grid-cols-[5.5rem_minmax(0,1fr)] gap-x-3 gap-y-3 text-sm'>
                <InfoRow label='所属项目' value={asset.project} />
                <InfoRow label='来源' value={asset.source} />
                {asset.sourceSession ? <InfoRow label='来源对话' value={asset.sourceSession} /> : null}
                <InfoRow label='固定版本' value={'v' + asset.version} />
                <InfoRow label='所在分类' value={asset.category} />
                {asset.originalName && asset.originalName !== asset.name ? <InfoRow label='原始名称' value={asset.originalName} /> : null}
              </div>
              </div>
              </div>
            </ScrollArea>
            <div className='flex gap-2 border-t p-4'>
              <Button variant='outline' className='flex-1' onClick={() => openMove([asset.id])}>移动到分类</Button>
              <Button className='flex-1' onClick={() => openExport([asset.id])}><Download />导出</Button>
            </div>
          </div>
        ) : null}
      </div>
    )
  }

  return (
    <div className='flex h-svh min-w-0 bg-background text-foreground'>
      <aside className='hidden w-52 shrink-0 flex-col border-r bg-muted/40 p-3 2xl:flex'>
        <div className='flex h-12 items-center gap-2 px-2 text-sm font-semibold'><Logo />Pixoma Studio</div>
        <div className='mt-5 flex flex-col gap-1'>
          <div className='flex h-9 items-center gap-2 rounded-md px-2 text-sm text-muted-foreground'><WandSparkles className='size-4' />新对话</div>
          <div className='flex h-9 items-center gap-2 rounded-md bg-card px-2 text-sm font-medium'><Library className='size-4' />资产库</div>
        </div>
        <div className='mt-auto border-t px-2 pt-3 text-xs text-muted-foreground'>创作 Studio</div>
      </aside>

      <main className='min-h-0 min-w-0 flex-1 p-3 sm:p-4'>
        <section className='flex h-full min-h-0 overflow-hidden rounded-xl border bg-card'>
          <aside className='hidden w-60 shrink-0 flex-col border-r bg-muted/40 md:flex'>
            <div className='flex h-16 items-center justify-between border-b px-4'>
              <h2 className='text-sm font-semibold'>项目资产</h2>
              {renderTreeSettings()}
            </div>
            <ScrollArea className='min-h-0 flex-1'>{renderProjectTree()}</ScrollArea>
          </aside>

          <div className='flex min-w-0 flex-1 flex-col'>
            <header className='flex min-h-16 items-center justify-between gap-3 border-b px-4 sm:px-5'>
              <div className='flex min-w-0 items-center gap-2'>
                <Button variant='ghost' size='icon' className='md:hidden' aria-label='打开项目资产树' onClick={() => setTreeOpen(true)}><Menu /></Button>
                <div className='min-w-0'>
                  <h1 className='truncate text-base font-semibold'>{project}</h1>
                  <p className='text-xs text-muted-foreground'>资产库 / 对话资产自动收录</p>
                </div>
                <Badge variant='outline' className='ml-2 hidden font-normal sm:inline-flex'>交互演示</Badge>
              </div>
              <div className='flex items-center gap-2'>
                <input
                  ref={fileInputRef}
                  type='file'
                  className='sr-only'
                  aria-label='选择上传资产'
                  onChange={(event) => {
                    const file = event.target.files?.[0]
                    if (file) uploadFile(file)
                    event.currentTarget.value = ''
                  }}
                />
                <Button variant='outline' size='sm' onClick={() => fileInputRef.current?.click()}><Upload />上传资产</Button>
              </div>
            </header>

            <div className='flex flex-col gap-3 border-b px-4 py-3 sm:px-5'>
              <div className='flex flex-wrap items-center gap-2'>
                <Select value={kind} onValueChange={(value) => changeScope(() => setKind(value as KindFilter))}>
                  <SelectTrigger aria-label='按资产类型筛选' className='w-28 sm:w-32'><SelectValue /></SelectTrigger>
                  <SelectContent><SelectGroup>{kindOptions.map((item) => <SelectItem key={item.value} value={item.value}>{item.label}</SelectItem>)}</SelectGroup></SelectContent>
                </Select>
                <Select value={formatFilter} onValueChange={(value) => changeScope(() => setFormatFilter(value))}>
                  <SelectTrigger aria-label='按格式筛选' className='w-28 sm:w-32'><SelectValue /></SelectTrigger>
                  <SelectContent><SelectGroup>{['全部格式', ...formats].map((item) => <SelectItem key={item} value={item}>{item}</SelectItem>)}</SelectGroup></SelectContent>
                </Select>
                <Select value={sourceSession} onValueChange={(value) => changeScope(() => setSourceSession(value))}>
                  <SelectTrigger aria-label='按来源对话筛选' className='w-36 sm:w-44'><SelectValue /></SelectTrigger>
                  <SelectContent><SelectGroup>{['全部对话', ...sourceSessions].map((item) => <SelectItem key={item} value={item}>{item}</SelectItem>)}</SelectGroup></SelectContent>
                </Select>
                <div className='relative min-w-44 flex-1'>
                  <Search className='absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground' />
                  <Input
                    value={query}
                    onChange={(event) => changeScope(() => setQuery(event.target.value))}
                    aria-label='搜索名称或标签'
                    placeholder='搜索名称或标签'
                    className='pl-9'
                  />
                </div>
                <Popover>
                  <PopoverTrigger asChild>
                    <Button variant='outline' size='sm' aria-label='筛选资产'><SlidersHorizontal />筛选{filtersActive ? ' · 已启用' : ''}</Button>
                  </PopoverTrigger>
                  <PopoverContent align='end' className='flex flex-col gap-4'>
                    <div className='flex flex-col gap-2'>
                      <span className='text-xs font-medium'>最低评分</span>
                      <div className='flex gap-1'>
                        {[0, 3, 4, 5].map((value) => (
                          <Button key={value} variant={minimumRating === value ? 'secondary' : 'ghost'} size='sm' onClick={() => changeScope(() => setMinimumRating(value))}>{value === 0 ? '全部' : value + ' 星+'}</Button>
                        ))}
                      </div>
                    </div>
                    <div className='flex flex-col gap-2'>
                      <span className='text-xs font-medium'>标签</span>
                      <Select value={tag} onValueChange={(value) => changeScope(() => setTag(value))}>
                        <SelectTrigger aria-label='按标签筛选' className='w-full'><SelectValue /></SelectTrigger>
                        <SelectContent><SelectGroup>{['全部标签', ...tagOptions].map((item) => <SelectItem key={item} value={item}>{item}</SelectItem>)}</SelectGroup></SelectContent>
                      </Select>
                    </div>
                    <Button variant={duplicatesOnly ? 'secondary' : 'outline'} size='sm' onClick={() => changeScope(() => setDuplicatesOnly(!duplicatesOnly))}>重复文件</Button>
                    {filtersActive ? <Button variant='ghost' size='sm' onClick={() => changeScope(() => { setMinimumRating(0); setTag('全部标签'); setDuplicatesOnly(false) })}>清除筛选</Button> : null}
                  </PopoverContent>
                </Popover>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild><Button variant='outline' size='sm' aria-label='排序资产'><ArrowDownUp />排序<ChevronDown /></Button></DropdownMenuTrigger>
                  <DropdownMenuContent align='end'>
                    <DropdownMenuRadioGroup value={sort} onValueChange={(value) => changeScope(() => setSort(value as SortOrder))}>
                      <DropdownMenuRadioItem value='recent'>最近添加</DropdownMenuRadioItem>
                      <DropdownMenuRadioItem value='name'>名称</DropdownMenuRadioItem>
                      <DropdownMenuRadioItem value='rating'>评分</DropdownMenuRadioItem>
                    </DropdownMenuRadioGroup>
                  </DropdownMenuContent>
                </DropdownMenu>
                <Button variant='outline' size='icon' aria-label={view === 'grid' ? '切换列表视图' : '切换网格视图'} onClick={() => setView(view === 'grid' ? 'list' : 'grid')}>{view === 'grid' ? <List /> : <Grid2X2 />}</Button>
              </div>
              <div className='flex min-w-0 items-center gap-2'>
                <FilterSegment value={category} options={['全部资产', '未分类', ...categories].map((item) => ({ value: item, label: item }))} onValueChange={(value) => changeScope(() => setCategory(value))} aria-label='按分类筛选' className='min-w-0 flex-1' />
                <Button variant='ghost' size='icon' className='shrink-0' aria-label='新建分类' onClick={() => setCreateCategoryOpen(true)}><FolderPlus /></Button>
                {categories.includes(category) ? <Button variant='ghost' size='icon' className='shrink-0' aria-label={'重命名分类 ' + category} onClick={() => { setCategoryToRename(category); setRenameCategoryDraft(category); setCategoryRenameError('') }}><Pencil /></Button> : null}
              </div>
            </div>

            {treeSelection ? <div className='flex items-center gap-2 border-b px-4 py-2 text-xs sm:px-5'>
              <span className='text-muted-foreground'>文件树筛选</span>
              <Badge variant='secondary' className='gap-1 font-normal'>{treeSelection.value}<button type='button' aria-label='清除文件树筛选' onClick={() => setTreeSelection(null)}><X className='size-3' /></button></Badge>
            </div> : null}

            {checkedIds.length > 0 ? (
              <div className='flex flex-wrap items-center gap-2 border-b bg-muted/40 px-4 py-2 sm:px-5'>
                <span className='mr-auto text-xs font-medium'>已选 {checkedIds.length} 项</span>
                <Button variant='ghost' size='sm' onClick={() => setCheckedIds([])}>取消选择</Button>
                <Button variant='outline' size='sm' onClick={() => openMove(checkedIds)}>移动到分类</Button>
                <Button size='sm' onClick={() => openExport(checkedIds)}><Download />导出</Button>
              </div>
            ) : null}

            <ScrollArea className='min-h-0 flex-1'>
              <div className='flex items-center justify-between px-4 py-3 text-xs text-muted-foreground sm:px-5'>
                <span>{project} · {category} · {visibleAssets.length} 项资产</span>
                <span>显示固定版本</span>
              </div>
              {visibleAssets.length === 0 ? (
                <div className='flex min-h-56 flex-col items-center justify-center gap-3 text-muted-foreground'><Library className='size-8' /><span>没有匹配的资产</span><Button variant='outline' size='sm' onClick={() => { setQuery(''); setKind('all'); setMinimumRating(0); setTag('全部标签'); setDuplicatesOnly(false); setSourceSession('全部对话'); setCategory('全部资产') }}>清除筛选</Button></div>
              ) : (
                <div className={cn('grid gap-3 px-4 pb-6 sm:px-5', view === 'grid' ? 'grid-cols-1 sm:grid-cols-2 2xl:grid-cols-3' : 'grid-cols-1')}>
                  {visibleAssets.map((asset) => (
                    <div
                      key={asset.id}
                      className={cn('group relative overflow-hidden rounded-lg border bg-card transition-colors hover:border-foreground/30', view === 'list' && 'flex min-h-24 items-center')}
                    >
                      <div className='absolute top-3 left-3 z-10 rounded-md bg-card/90 p-1'>
                        <Checkbox checked={checkedIds.includes(asset.id)} onCheckedChange={(checked) => toggleChecked(asset.id, Boolean(checked))} aria-label={'选择 ' + asset.name} />
                      </div>
                      <button type='button' className={cn('flex min-w-0 text-left focus-visible:outline-2 focus-visible:outline-ring', view === 'grid' ? 'w-full flex-col' : 'w-full items-center')} onClick={() => openAsset(asset.id)}>
                        <div className={cn('overflow-hidden border-b bg-background', view === 'grid' ? 'aspect-[4/3] w-full' : 'ml-12 size-20 shrink-0 rounded-md border')}>
                          <ArtworkPreview asset={asset} />
                        </div>
                        <div className='flex min-w-0 flex-1 flex-col gap-2 p-3'>
                          <div className='flex min-w-0 items-center gap-2'>
                            {asset.kind === 'image' ? <ImageIcon className='size-4 shrink-0 text-muted-foreground' /> : asset.kind === 'audio' ? <AudioLines className='size-4 shrink-0 text-muted-foreground' /> : asset.kind === 'video' ? <Video className='size-4 shrink-0 text-muted-foreground' /> : <FileText className='size-4 shrink-0 text-muted-foreground' />}
                            <span className='truncate text-sm font-medium'>{asset.name}</span>
                          </div>
                          <div className='flex items-center justify-between gap-2 text-xs text-muted-foreground'>
                            <span className='truncate'>{asset.category}</span>
                            <span className='shrink-0 font-mono tabular-nums'>{asset.format} · {formatSize(asset.sizeBytes)}</span>
                          </div>
                          <div className='flex items-center gap-1'>
                            {asset.sourceSession && asset.source !== '本机上传' ? <Badge variant='outline' className='font-normal'>对话产物</Badge> : null}
                            {asset.tags.slice(0, 2).map((item) => <Badge key={item} variant='secondary' className='font-normal'>{item}</Badge>)}
                            {asset.duplicate ? <Badge variant='outline' className='font-normal'>重复</Badge> : null}
                            {asset.rating > 0 ? <span className='ml-auto flex items-center gap-0.5 text-xs'><Star className='size-3' fill='currentColor' />{asset.rating}</span> : null}
                          </div>
                        </div>
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </ScrollArea>
            <div className='flex min-h-10 items-center justify-between border-t px-4 text-xs text-muted-foreground sm:px-5'><span>{message || '点击资产查看详情'}</span><span>演示内容</span></div>
          </div>
        </section>
      </main>

      <Dialog open={treeOpen} onOpenChange={setTreeOpen}>
        <DialogContent aria-describedby={undefined} className='flex max-h-[85svh] flex-col overflow-hidden sm:max-w-md'>
          <DialogHeader className='pr-10'><div className='flex items-center justify-between'><DialogTitle>项目资产</DialogTitle>{renderTreeSettings()}</div></DialogHeader>
          <ScrollArea className='min-h-0 flex-1'>{renderProjectTree()}</ScrollArea>
        </DialogContent>
      </Dialog>
      <Dialog open={detailOpen} onOpenChange={(open) => { setDetailOpen(open); if (!open) { setEditingNameId(null); setPreviewExpanded(false) } }}>
        <DialogContent className='flex h-[min(94svh,1100px)] w-[calc(100vw-2rem)] flex-col gap-0 overflow-hidden p-0 sm:max-w-[1500px]'>
          <DialogHeader className={cn('shrink-0 border-b px-5 py-4 pr-12', previewExpanded && 'sr-only')}>
            <DialogTitle>资产详情</DialogTitle>
            <DialogDescription>{activeAsset ? '固定版本 v' + activeAsset.version : '固定版本'}</DialogDescription>
          </DialogHeader>
          {activeAsset ? renderAssetDetails(activeAsset) : null}
        </DialogContent>
      </Dialog>
      <Dialog open={moveOpen} onOpenChange={setMoveOpen}>
        <DialogContent>
          <DialogHeader><DialogTitle>移动到分类</DialogTitle><DialogDescription>{moveProject} · 已选 {moveIds.length} 项资产</DialogDescription></DialogHeader>
          <Select value={moveTarget} onValueChange={setMoveTarget}>
            <SelectTrigger aria-label='目标分类' className='w-full'><SelectValue /></SelectTrigger>
            <SelectContent><SelectGroup>{['未分类', ...(categoriesByProject[moveProject] ?? [])].map((item) => <SelectItem key={item} value={item}>{item}</SelectItem>)}</SelectGroup></SelectContent>
          </Select>
          <DialogFooter><Button variant='outline' onClick={() => setMoveOpen(false)}>取消</Button><Button onClick={() => { setAssets((current) => current.map((asset) => moveIds.includes(asset.id) ? { ...asset, category: moveTarget } : asset)); setCheckedIds([]); setMoveOpen(false); setMessage('已移动 ' + moveIds.length + ' 项资产') }}>移动资产</Button></DialogFooter>
        </DialogContent>
      </Dialog>
      <Dialog open={exportOpen} onOpenChange={setExportOpen}>
        <DialogContent>
          <DialogHeader><DialogTitle>导出资产</DialogTitle><DialogDescription>已选 {targetAssets.length} 个固定版本</DialogDescription></DialogHeader>
          <div className='flex max-h-52 flex-col gap-2 overflow-y-auto rounded-md border bg-muted/40 p-3 text-sm'>{targetAssets.map((asset) => <div key={asset.id} className='flex justify-between gap-3'><span className='truncate'>{asset.name}</span><span className='shrink-0 font-mono text-muted-foreground'>v{asset.version}</span></div>)}</div>
          <DialogFooter><Button onClick={() => setExportOpen(false)}>返回资产库</Button></DialogFooter>
        </DialogContent>
      </Dialog>
      <Dialog open={createCategoryOpen} onOpenChange={setCreateCategoryOpen}>
        <DialogContent>
          <DialogHeader><DialogTitle>新建分类</DialogTitle><DialogDescription>{project}</DialogDescription></DialogHeader>
          <Input value={categoryName} onChange={(event) => setCategoryName(event.target.value)} aria-label='分类名称' placeholder='分类名称' />
          <DialogFooter><Button variant='outline' onClick={() => setCreateCategoryOpen(false)}>取消</Button><Button disabled={!categoryName.trim() || categories.includes(categoryName.trim())} onClick={() => { const name = categoryName.trim(); setCategoriesByProject((current) => ({ ...current, [project]: [...current[project], name] })); setCategory(name); setCategoryName(''); setCreateCategoryOpen(false) }}>创建分类</Button></DialogFooter>
        </DialogContent>
      </Dialog>
      <Dialog open={categoryToRename !== null} onOpenChange={(open) => { if (!open) setCategoryToRename(null) }}>
        <DialogContent aria-describedby={undefined}>
          <form className='flex flex-col gap-4' onSubmit={(event) => { event.preventDefault(); saveCategoryName() }}>
            <DialogHeader><DialogTitle>重命名分类</DialogTitle></DialogHeader>
            <Input
              autoFocus
              aria-label='分类名称'
              value={renameCategoryDraft}
              onChange={(event) => { setRenameCategoryDraft(event.target.value); setCategoryRenameError('') }}
            />
            {categoryRenameError ? <p role='alert' className='text-xs text-destructive'>{categoryRenameError}</p> : null}
            <DialogFooter>
              <Button type='button' variant='outline' onClick={() => setCategoryToRename(null)}>取消</Button>
              <Button type='submit'>保存名称</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}

export function InfoRow({ label, value }: { label: string; value: string }) {
  return <><span className='text-muted-foreground'>{label}</span><span className='min-w-0 break-all font-mono text-xs tabular-nums'>{value}</span></>
}

createRoot(document.getElementById('root')!).render(<App />)
