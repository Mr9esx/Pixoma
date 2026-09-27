import { memo, useEffect, useMemo, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { unifiedMergeView } from '@codemirror/merge'
import { matter } from 'gray-matter-es'
import {
  Copy,
  Download,
  FileCode2,
  FilePlus2,
  FolderPlus,
  PanelRightClose,
  PanelRightOpen,
  Pencil,
  Trash2,
  Upload,
  X,
} from 'lucide-react'
import ReactMarkdown, { defaultUrlTransform } from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { toast } from 'sonner'
import { ApiError } from '@/lib/api/client'
import {
  createStudioSkill,
  getStudioSkill,
  getStudioSkillVersion,
  listStudioSkillVersions,
  updateStudioSkill,
  updateStudioSkillEnabled,
  type StudioSkill,
  type StudioSkillFile,
} from '@/lib/api/studio'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ButtonGroup } from '@/components/ui/button-group'
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from '@/components/ui/context-menu'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { IconButtonTooltip } from '@/components/ui/icon-button-tooltip'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import {
  FileTree,
  FileTreeFile,
  FileTreeFolder,
} from '@/components/ai-elements/file-tree'
import { CodeEditor } from '@/components/code-editor'
import {
  addFile,
  addFolder,
  applySavedSkill,
  copyPath,
  deletePath,
  initDraft,
  readMetadata,
  renamePath,
  serializeDraft,
  updateFileContent,
  updateMetadata,
  validateDraft,
  type SkillDraft,
} from './skill-document'
import {
  listSkillFileChanges,
  nextSkillVersion,
  skillVersionError,
  type SkillFileChange,
} from './skill-version'

type ImportedSkill = Pick<StudioSkill, 'name' | 'description' | 'files'>
type EditorMode = 'source' | 'preview' | 'split'
const modeButtonClass =
  'h-7 min-w-12 px-2.5 text-xs font-normal text-muted-foreground aria-pressed:bg-background aria-pressed:font-medium aria-pressed:text-foreground aria-pressed:hover:bg-background'
function focusDialogOnOpen(event: Event) {
  event.preventDefault()
  const dialog = event.currentTarget as HTMLElement
  dialog.focus()
}
type PathAction = {
  kind: 'file' | 'folder' | 'rename' | 'copy'
  source?: string
  parent?: string
}
type SkillSaveReview = {
  submitted: SkillDraft
  data: ReturnType<typeof serializeDraft>
}

export function SkillWorkspaceDialog({
  skillId,
  imported,
  open,
  onOpenChange,
}: {
  skillId?: string
  imported?: ImportedSkill
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  if (!open) return null
  return (
    <OpenSkillWorkspaceDialog
      key={skillId ?? imported?.name ?? 'new'}
      skillId={skillId}
      imported={imported}
      onOpenChange={onOpenChange}
    />
  )
}

function OpenSkillWorkspaceDialog({
  skillId,
  imported,
  onOpenChange,
}: {
  skillId?: string
  imported?: ImportedSkill
  onOpenChange: (open: boolean) => void
}) {
  const [detail, setDetail] = useState<{
    skill?: StudioSkill
    error?: unknown
  }>()
  const [retry, setRetry] = useState(0)
  useEffect(() => {
    if (!skillId) return
    let active = true
    void getStudioSkill(skillId).then(
      (skill) => {
        if (active) setDetail({ skill })
      },
      (error: unknown) => {
        if (active) setDetail({ error })
      }
    )
    return () => {
      active = false
    }
  }, [skillId, retry])
  if (skillId && !detail?.skill) {
    return (
      <Dialog open onOpenChange={onOpenChange}>
        <DialogContent
          tabIndex={-1}
          onOpenAutoFocus={focusDialogOnOpen}
          className='w-[calc(100vw-2rem)] max-w-[calc(100vw-2rem)] sm:max-w-lg'
        >
          <DialogHeader>
            <DialogTitle>编辑技能</DialogTitle>
            <DialogDescription className='sr-only'>
              读取技能文件
            </DialogDescription>
          </DialogHeader>
          <p
            role={detail?.error ? 'alert' : undefined}
            className='text-sm text-muted-foreground'
          >
            {detail?.error ? '读取技能失败' : '正在读取技能…'}
          </p>
          {detail?.error ? (
            <Button
              variant='outline'
              onClick={() => {
                setDetail(undefined)
                setRetry((value) => value + 1)
              }}
            >
              重试
            </Button>
          ) : null}
        </DialogContent>
      </Dialog>
    )
  }
  const importedSkill: StudioSkill | undefined = imported
    ? {
        id: '',
        name: imported.name,
        description: imported.description,
        files: imported.files,
        prompt:
          imported.files?.find((file) => file.path === 'SKILL.md')?.content ??
          '',
        version: '1.0.0',
        enabled: true,
        created_at: '',
        updated_at: '',
      }
    : undefined
  return (
    <SkillWorkspaceEditor
      key={skillId ?? imported?.name ?? 'new'}
      skill={detail?.skill ?? importedSkill}
      imported={Boolean(imported)}
      open
      onOpenChange={onOpenChange}
    />
  )
}

function SkillWorkspaceEditor({
  skill,
  imported,
  open,
  onOpenChange,
}: {
  skill?: StudioSkill
  imported: boolean
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const queryClient = useQueryClient()
  const [draft, setDraftState] = useState<SkillDraft>(() => initDraft(skill))
  const [baseline, setBaseline] = useState<SkillDraft>(() => initDraft(skill))
  const [baseSkill, setBaseSkill] = useState(skill)
  const [remoteSkill, setRemoteSkill] = useState<StudioSkill>()
  const [confirmOverwrite, setConfirmOverwrite] = useState(false)
  const draftRef = useRef(draft)
  const setDraft = (
    update: SkillDraft | ((current: SkillDraft) => SkillDraft)
  ) => {
    const next =
      typeof update === 'function' ? update(draftRef.current) : update
    draftRef.current = next
    setDraftState(next)
  }
  const [selectedPath, setSelectedPath] = useState('SKILL.md')
  const [openPaths, setOpenPaths] = useState<string[]>(['SKILL.md'])
  const [expandedFolders, setExpandedFolders] = useState<Set<string>>(() => {
    const paths = new Set<string>()
    for (const file of skill?.files ?? []) {
      const parts = file.path.split('/')
      for (let index = 1; index < parts.length; index += 1) {
        paths.add(parts.slice(0, index).join('/'))
      }
    }
    return paths
  })
  const [mode, setMode] = useState<EditorMode>('source')
  const [inspectorOpen, setInspectorOpen] = useState(true)
  const [pathAction, setPathAction] = useState<PathAction>()
  const [pathValue, setPathValue] = useState('')
  const [contextPath, setContextPath] = useState('')
  const [error, setError] = useState('')
  const [confirmClose, setConfirmClose] = useState(false)
  const [deletingPath, setDeletingPath] = useState('')
  const [review, setReview] = useState<SkillSaveReview>()
  const [reviewVersion, setReviewVersion] = useState('')
  const [reviewPath, setReviewPath] = useState('')
  const [reviewError, setReviewError] = useState('')
  const uploadRef = useRef<HTMLInputElement>(null)
  const uploadTargetRef = useRef('')
  const metadata = readMetadata(draft)
  const errors = validateDraft(draft)
  const selectedFile = draft.files.find(
    (file) => file.path === selectedPath && !file.directory
  )
  const markdown = Boolean(
    selectedFile && /\.md$/i.test(selectedFile.path) && !selectedFile.binary
  )
  const visibleMode = markdown ? mode : 'source'
  const previewContent =
    selectedFile?.path === 'SKILL.md' && !metadata.error
      ? matter(selectedFile.content).content
      : (selectedFile?.content ?? '')
  const resolveSkillFile = (url: string) => {
    if (url.startsWith('#') || url.startsWith('?')) return undefined
    try {
      const root = new URL('https://skill.local/')
      const resolved = new URL(url, new URL(selectedPath, root))
      if (resolved.origin !== root.origin) return undefined
      const path = decodeURIComponent(resolved.pathname.slice(1))
      return draft.files.find((file) => file.path === path && !file.directory)
    } catch {
      return undefined
    }
  }
  const dirty =
    JSON.stringify(draft) !== JSON.stringify(baseline) ||
    (imported && !baseSkill?.id)
  const fileCount = draft.files.filter((file) => !file.directory).length
  const reviewChanges = useMemo(
    () =>
      review
        ? listSkillFileChanges(
            baseSkill?.id ? baseline.files : [],
            review.data.files ?? [
              { path: 'SKILL.md', content: review.data.prompt },
            ]
          )
        : [],
    [review, baseSkill?.id, baseline.files]
  )
  const reviewChange =
    reviewChanges.find((change) => change.path === reviewPath) ??
    reviewChanges[0]
  const reviewMetadata = review?.data
  const previousMetadata = readMetadata(baseSkill?.id ? baseline : initDraft())
  const nameChanged =
    reviewMetadata && reviewMetadata.name !== previousMetadata.name
  const descriptionChanged =
    reviewMetadata &&
    reviewMetadata.description !== previousMetadata.description

  const openSaveReview = () => {
    if (!dirty || save.isPending || errors.length || remoteSkill) return
    const submitted = draftRef.current
    const data = serializeDraft(submitted)
    const changes = listSkillFileChanges(
      baseSkill?.id ? baseline.files : [],
      data.files ?? [{ path: 'SKILL.md', content: data.prompt }]
    )
    setReview({ submitted, data })
    setReviewVersion(
      nextSkillVersion(baseSkill?.id ? baseSkill.version : undefined)
    )
    setReviewPath(changes[0]?.path ?? '')
    setReviewError('')
    setError('')
  }

  const save = useMutation({
    mutationFn: ({
      review,
      version,
    }: {
      review: SkillSaveReview
      version: string
    }) => {
      const data = { ...review.data, version }
      return baseSkill?.id
        ? updateStudioSkill({ ...baseSkill, ...data })
        : createStudioSkill(data)
    },
    onSuccess: (result, { review }) => {
      const next = applySavedSkill(draftRef.current, review.submitted, result)
      setReview(undefined)
      setBaseSkill(next.skill)
      setBaseline(next.baseline)
      setDraft(next.draft)
      void queryClient.invalidateQueries({ queryKey: ['studio', 'skills'] })
      toast.success(baseSkill?.id ? '技能已保存' : '技能已创建')
      if (next.close) {
        onOpenChange(false)
      } else {
        setError('保存期间有新的修改，请再次保存。')
      }
    },
    onError: (cause) => {
      setReview(undefined)
      setError(
        cause instanceof ApiError
          ? (cause.detail ?? cause.message)
          : cause instanceof Error
            ? cause.message
            : '保存技能失败'
      )
    },
  })
  const toggle = useMutation({
    mutationFn: (enabled: boolean) =>
      updateStudioSkillEnabled(baseSkill!.id, enabled),
    onSuccess: async (result) => {
      setDraft((current) => ({ ...current, enabled: result.enabled }))
      setBaseline((current) => ({ ...current, enabled: result.enabled }))
      setBaseSkill((current) =>
        current ? { ...current, enabled: result.enabled } : current
      )
      await queryClient.invalidateQueries({ queryKey: ['studio', 'skills'] })
    },
    onError: (cause) =>
      setError(cause instanceof Error ? cause.message : '更新启用状态失败'),
  })

  const requestClose = () => {
    if (dirty) setConfirmClose(true)
    else onOpenChange(false)
  }
  const selectPath = (path: string) => {
    setSelectedPath(path)
    if (draft.files.some((file) => file.path === path && !file.directory)) {
      setOpenPaths((current) =>
        current.includes(path) ? current : [...current, path]
      )
    }
  }
  const startPathAction = (kind: PathAction['kind'], source = '') => {
    const parent =
      source &&
      (draft.files.find((file) => file.path === source)?.directory ||
        draft.files.some((file) => file.path.startsWith(`${source}/`)))
        ? source
        : source.includes('/')
          ? source.slice(0, source.lastIndexOf('/'))
          : ''
    setPathAction({ kind, source, parent })
    setPathValue(
      kind === 'rename' || kind === 'copy' ? source : parent ? `${parent}/` : ''
    )
    setError('')
  }
  const submitPathAction = () => {
    if (!pathAction) return
    try {
      const path = pathValue.trim()
      const next =
        pathAction.kind === 'folder'
          ? addFolder(draft, path)
          : pathAction.kind === 'rename'
            ? renamePath(draft, pathAction.source!, path)
            : pathAction.kind === 'copy'
              ? copyPath(draft, pathAction.source!, path)
              : addFile(draft, { path, content: '' })
      setDraft(next)
      setSelectedPath(path)
      setOpenPaths((current) =>
        current
          .map((item) => (item === pathAction.source ? path : item))
          .filter((item) =>
            next.files.some((file) => file.path === item && !file.directory)
          )
      )
      if (pathAction.kind === 'file' || pathAction.kind === 'copy')
        setOpenPaths((current) =>
          current.includes(path) ? current : [...current, path]
        )
      const expanded = new Set(expandedFolders)
      const parts = path.split('/')
      for (let index = 1; index < parts.length; index += 1)
        expanded.add(parts.slice(0, index).join('/'))
      setExpandedFolders(expanded)
      setPathAction(undefined)
      setPathValue('')
      setError('')
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : '文件操作失败')
    }
  }
  const removePath = (path: string) => {
    try {
      const next = deletePath(draft, path)
      setDraft(next)
      setOpenPaths((current) =>
        current.filter((item) =>
          next.files.some((file) => file.path === item && !file.directory)
        )
      )
      setSelectedPath('SKILL.md')
      setDeletingPath('')
      setError('')
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : '删除失败')
    }
  }
  const startUpload = (target: string) => {
    uploadTargetRef.current = target
    uploadRef.current?.click()
  }
  const uploadFiles = async (files: FileList | null, target: string) => {
    if (!files?.length) return
    try {
      const incoming: StudioSkillFile[] = []
      let lastPath = selectedPath
      const parent =
        target &&
        (draft.files.find((file) => file.path === target)?.directory ||
          draft.files.some((file) => file.path.startsWith(`${target}/`)))
          ? target
          : target.includes('/')
            ? target.slice(0, target.lastIndexOf('/'))
            : ''
      for (const file of Array.from(files)) {
        if (file.size > 256 * 1024) throw new Error(`${file.name} 超过 256 KB`)
        const bytes = new Uint8Array(await file.arrayBuffer())
        let content: string
        let binary = false
        try {
          content = new TextDecoder('utf-8', { fatal: true }).decode(bytes)
          if (content.includes('\0')) throw new Error('二进制文件')
        } catch {
          binary = true
          let encoded = ''
          for (let index = 0; index < bytes.length; index += 32768) {
            encoded += String.fromCharCode(
              ...bytes.subarray(index, index + 32768)
            )
          }
          content = btoa(encoded)
        }
        lastPath = parent ? `${parent}/${file.name}` : file.name
        incoming.push({ path: lastPath, content, binary })
      }
      let next = draftRef.current
      for (const file of incoming) next = addFile(next, file)
      setDraft(next)
      setSelectedPath(lastPath)
      setOpenPaths((current) =>
        current.includes(lastPath) ? current : [...current, lastPath]
      )
      setError('')
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : '上传文件失败')
    }
  }
  const downloadFile = (file: StudioSkillFile) => {
    const content = file.binary
      ? Uint8Array.from(atob(file.content), (character) =>
          character.charCodeAt(0)
        )
      : file.content
    const url = URL.createObjectURL(new Blob([content]))
    const link = document.createElement('a')
    link.href = url
    link.download = file.path.split('/').at(-1) ?? file.path
    link.click()
    URL.revokeObjectURL(url)
  }
  const refreshLatest = async () => {
    if (!baseSkill?.id) return
    try {
      const current = await getStudioSkill(baseSkill.id)
      setRemoteSkill(current)
      setError('已读取最新版本，请选择要保留的内容。')
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : '读取技能失败')
    }
  }
  const adoptLatest = () => {
    if (!remoteSkill) return
    const latest = initDraft(remoteSkill)
    setDraft(latest)
    setBaseline(latest)
    setBaseSkill(remoteSkill)
    setRemoteSkill(undefined)
    setSelectedPath('SKILL.md')
    setOpenPaths(['SKILL.md'])
    setMode('source')
    setError('')
    save.reset()
  }
  const overwriteLatest = () => {
    if (!remoteSkill) return
    setBaseSkill(remoteSkill)
    setBaseline(initDraft(remoteSkill))
    setRemoteSkill(undefined)
    setConfirmOverwrite(false)
    setError('已选择覆盖最新版本，请再次保存。')
    save.reset()
  }

  return (
    <>
      <Dialog
        open={open}
        onOpenChange={(next) => (next ? onOpenChange(true) : requestClose())}
      >
        <DialogContent
          showCloseButton={false}
          tabIndex={-1}
          onOpenAutoFocus={focusDialogOnOpen}
          className='flex h-[calc(100dvh-1rem)] w-[calc(100vw-1rem)] max-w-[calc(100vw-1rem)] flex-col gap-0 overflow-hidden rounded-lg p-0 shadow-md sm:max-w-[calc(100vw-1rem)]'
          onKeyDownCapture={(event) => {
            if (
              (event.metaKey || event.ctrlKey) &&
              event.key.toLowerCase() === 's'
            ) {
              event.preventDefault()
              openSaveReview()
            }
          }}
        >
          <DialogHeader className='flex h-11 shrink-0 flex-row items-center justify-between gap-3 border-b bg-muted/30 px-3 py-0 text-left'>
            <div className='flex min-w-0 items-center gap-2'>
              <FileCode2 className='size-4 shrink-0 text-muted-foreground' />
              <DialogTitle className='shrink-0 text-sm font-medium'>
                {baseSkill?.id
                  ? '编辑技能'
                  : imported
                    ? '导入技能'
                    : '新建 SKILL'}
              </DialogTitle>
              {dirty ? <Badge variant='secondary'>未保存</Badge> : null}
            </div>
            <DialogDescription className='sr-only'>
              技能文件编辑器
            </DialogDescription>
            <div className='flex shrink-0 items-center gap-1'>
              <IconButtonTooltip label='关闭'>
                <Button
                  type='button'
                  aria-label='关闭'
                  variant='ghost'
                  size='icon-sm'
                  onClick={requestClose}
                >
                  <X />
                </Button>
              </IconButtonTooltip>
            </div>
          </DialogHeader>
          <div className='flex min-h-0 flex-1 flex-col overflow-y-auto lg:flex-row lg:overflow-hidden'>
            <aside className='flex h-48 w-full shrink-0 flex-col bg-muted/20 lg:h-auto lg:w-60 lg:border-r'>
              <div className='flex h-10 shrink-0 items-center justify-between px-3'>
                <span className='text-sm font-medium text-muted-foreground'>
                  资源管理器
                </span>
                <div className='flex items-center gap-1'>
                  <IconButtonTooltip label='新建文件'>
                    <Button
                      type='button'
                      aria-label='新建文件'
                      variant='ghost'
                      size='icon-sm'
                      onClick={() => startPathAction('file', selectedPath)}
                    >
                      <FilePlus2 />
                    </Button>
                  </IconButtonTooltip>
                  <IconButtonTooltip label='新建文件夹'>
                    <Button
                      type='button'
                      aria-label='新建文件夹'
                      variant='ghost'
                      size='icon-sm'
                      onClick={() => startPathAction('folder', selectedPath)}
                    >
                      <FolderPlus />
                    </Button>
                  </IconButtonTooltip>
                  <IconButtonTooltip label='上传文件'>
                    <Button
                      type='button'
                      aria-label='上传文件'
                      variant='ghost'
                      size='icon-sm'
                      onClick={() => startUpload(selectedPath)}
                    >
                      <Upload />
                    </Button>
                  </IconButtonTooltip>
                </div>
              </div>
              <input
                ref={uploadRef}
                type='file'
                multiple
                className='sr-only'
                onChange={(event) => {
                  void uploadFiles(
                    event.currentTarget.files,
                    uploadTargetRef.current
                  )
                  event.currentTarget.value = ''
                }}
              />
              {pathAction ? (
                <form
                  className='flex flex-col gap-2 px-2 py-1'
                  onSubmit={(event) => {
                    event.preventDefault()
                    submitPathAction()
                  }}
                >
                  <Input
                    autoFocus
                    aria-label='文件路径'
                    value={pathValue}
                    onChange={(event) => setPathValue(event.target.value)}
                    className='h-8 text-xs'
                  />
                  <div className='flex gap-1'>
                    <Button type='submit' size='xs'>
                      确认
                    </Button>
                    <Button
                      type='button'
                      size='xs'
                      variant='ghost'
                      onClick={() => setPathAction(undefined)}
                    >
                      取消
                    </Button>
                  </div>
                </form>
              ) : null}
              <ContextMenu>
                <ContextMenuTrigger asChild>
                  <div
                    className='min-h-0 flex-1 overflow-y-auto px-2 pb-2'
                    tabIndex={0}
                    aria-label='技能文件树'
                    onKeyDownCapture={(event) => {
                      if (
                        (event.shiftKey && event.key === 'F10') ||
                        event.key === 'ContextMenu'
                      ) {
                        event.preventDefault()
                        const target = event.target as HTMLElement
                        const bounds = target.getBoundingClientRect()
                        queueMicrotask(() =>
                          target.dispatchEvent(
                            new MouseEvent('contextmenu', {
                              bubbles: true,
                              cancelable: true,
                              clientX: bounds.left,
                              clientY: bounds.bottom,
                            })
                          )
                        )
                      }
                    }}
                    onContextMenuCapture={(event) => {
                      const target = (
                        event.target as HTMLElement
                      ).closest<HTMLElement>('[data-skill-path]')
                      const path = target?.dataset.skillPath ?? ''
                      setContextPath(path)
                      if (path) selectPath(path)
                    }}
                  >
                    <FileTree
                      className='border-0 bg-transparent p-0 [&_[data-slot=collapsible-content]]:border-0 [&_[role=treeitem]]:rounded-sm'
                      expanded={expandedFolders}
                      onExpandedChange={setExpandedFolders}
                      selectedPath={selectedPath}
                      onSelect={selectPath}
                    >
                      <SkillFileTree
                        files={draft.files}
                        onFolderSelect={selectPath}
                      />
                    </FileTree>
                  </div>
                </ContextMenuTrigger>
                <ContextMenuContent>
                  <ContextMenuItem
                    onSelect={() => startPathAction('file', contextPath)}
                  >
                    <FilePlus2 />
                    新建文件
                  </ContextMenuItem>
                  <ContextMenuItem
                    onSelect={() => startPathAction('folder', contextPath)}
                  >
                    <FolderPlus />
                    新建文件夹
                  </ContextMenuItem>
                  <ContextMenuItem onSelect={() => startUpload(contextPath)}>
                    <Upload />
                    上传文件
                  </ContextMenuItem>
                  {contextPath && contextPath !== 'SKILL.md' ? (
                    <>
                      <ContextMenuSeparator />
                      <ContextMenuItem
                        onSelect={() => startPathAction('rename', contextPath)}
                      >
                        <Pencil />
                        重命名
                      </ContextMenuItem>
                      <ContextMenuItem
                        onSelect={() => startPathAction('copy', contextPath)}
                      >
                        <Copy />
                        复制
                      </ContextMenuItem>
                      <ContextMenuItem
                        variant='destructive'
                        onSelect={() => setDeletingPath(contextPath)}
                      >
                        <Trash2 />
                        删除
                      </ContextMenuItem>
                    </>
                  ) : null}
                  {contextPath &&
                  draft.files.find(
                    (file) => file.path === contextPath && !file.directory
                  ) ? (
                    <ContextMenuItem
                      onSelect={() =>
                        downloadFile(
                          draft.files.find((file) => file.path === contextPath)!
                        )
                      }
                    >
                      <Download />
                      下载文件
                    </ContextMenuItem>
                  ) : null}
                </ContextMenuContent>
              </ContextMenu>
            </aside>
            <section className='flex min-h-72 min-w-0 flex-1 flex-col bg-background lg:min-h-0'>
              <div className='flex h-10 shrink-0 items-stretch bg-muted/20'>
                <div className='flex min-w-0 flex-1 overflow-x-auto'>
                  {openPaths
                    .filter((path) =>
                      draft.files.some(
                        (file) => file.path === path && !file.directory
                      )
                    )
                    .map((path) => {
                      const changed =
                        draft.files.find((file) => file.path === path)
                          ?.content !==
                        baseline.files.find((file) => file.path === path)
                          ?.content
                      return (
                        <div
                          key={path}
                          className={`flex shrink-0 items-center ${selectedPath === path ? 'bg-background' : ''}`}
                        >
                          <Button
                            type='button'
                            variant='ghost'
                            size='xs'
                            className='h-10 max-w-36 rounded-none px-3 text-xs'
                            onClick={() => setSelectedPath(path)}
                          >
                            <span className='truncate'>
                              {path.split('/').at(-1)}
                            </span>
                            {changed ? (
                              <span aria-label='未保存'>•</span>
                            ) : null}
                          </Button>
                          {path !== 'SKILL.md' ? (
                            <IconButtonTooltip label={`关闭 ${path}`}>
                              <Button
                                type='button'
                                aria-label={`关闭 ${path}`}
                                variant='ghost'
                                size='icon-xs'
                                className='me-1'
                                onClick={() => {
                                  setOpenPaths((current) =>
                                    current.filter((item) => item !== path)
                                  )
                                  if (selectedPath === path)
                                    setSelectedPath('SKILL.md')
                                }}
                              >
                                ×
                              </Button>
                            </IconButtonTooltip>
                          ) : null}
                        </div>
                      )
                    })}
                </div>
                <div className='flex shrink-0 items-center gap-1 pe-3'>
                  <ButtonGroup
                    aria-label='编辑视图'
                    className='h-8 items-center rounded-md bg-muted p-0.5'
                  >
                    <Button
                      type='button'
                      variant='ghost'
                      size='xs'
                      aria-pressed={visibleMode === 'source'}
                      className={modeButtonClass}
                      onClick={() => setMode('source')}
                    >
                      编辑
                    </Button>
                    <Button
                      type='button'
                      variant='ghost'
                      size='xs'
                      disabled={!markdown}
                      aria-pressed={visibleMode === 'preview'}
                      className={modeButtonClass}
                      onClick={() => setMode('preview')}
                    >
                      预览
                    </Button>
                    <Button
                      type='button'
                      variant='ghost'
                      size='xs'
                      disabled={!markdown}
                      aria-pressed={visibleMode === 'split'}
                      className={modeButtonClass}
                      onClick={() => setMode('split')}
                    >
                      分栏
                    </Button>
                  </ButtonGroup>
                  {!inspectorOpen ? (
                    <IconButtonTooltip label='显示基础信息'>
                      <Button
                        type='button'
                        aria-label='显示基础信息'
                        aria-controls='skill-inspector'
                        aria-expanded={false}
                        variant='ghost'
                        size='icon-sm'
                        onClick={() => setInspectorOpen(true)}
                      >
                        <PanelRightOpen />
                      </Button>
                    </IconButtonTooltip>
                  ) : null}
                </div>
              </div>
              {selectedFile?.binary ? (
                <BinaryPreview
                  file={selectedFile}
                  onDownload={() => downloadFile(selectedFile)}
                />
              ) : selectedFile ? (
                <div className='flex min-h-0 flex-1 flex-col lg:flex-row'>
                  {visibleMode !== 'preview' ? (
                    <CodeEditor
                      key={`${selectedPath}:${visibleMode}`}
                      aria-label={`编辑 ${selectedPath}`}
                      language={markdown ? 'markdown' : 'text'}
                      value={selectedFile.content}
                      onChange={(content) =>
                        setDraft((current) =>
                          updateFileContent(current, selectedPath, content)
                        )
                      }
                      fill
                      className='min-h-0 min-w-0 flex-1 rounded-none border-0'
                      basicSetup={{ lineNumbers: true, foldGutter: true }}
                    />
                  ) : null}
                  {(visibleMode === 'preview' || visibleMode === 'split') &&
                  markdown ? (
                    <div
                      className='min-h-0 min-w-0 flex-1 overflow-y-auto bg-background p-5 text-sm leading-6 lg:border-l'
                      onClickCapture={(event) => {
                        const link = (event.target as HTMLElement).closest('a')
                        const href = link?.getAttribute('href')
                        if (!href?.startsWith('#skill-file/')) return
                        event.preventDefault()
                        selectPath(decodeURIComponent(href.slice(12)))
                      }}
                    >
                      <div className='space-y-2 break-words text-foreground [&_a]:text-primary [&_a]:underline [&_a]:underline-offset-4 [&_blockquote]:border-l-2 [&_blockquote]:border-border [&_blockquote]:pl-3 [&_h1]:text-xl [&_h1]:font-semibold [&_h2]:text-lg [&_h2]:font-semibold [&_h3]:font-semibold [&_li]:ms-5 [&_ol]:list-decimal [&_pre]:overflow-x-auto [&_pre]:rounded-md [&_pre]:border [&_pre]:bg-background [&_pre]:p-3 [&_ul]:list-disc'>
                        <ReactMarkdown
                          remarkPlugins={[remarkGfm]}
                          components={{
                            a: ({ href, title, children }) => (
                              <a
                                href={href}
                                title={title}
                                target={
                                  href?.startsWith('#') ? undefined : '_blank'
                                }
                                rel={
                                  href?.startsWith('#')
                                    ? undefined
                                    : 'noopener noreferrer'
                                }
                              >
                                {children}
                              </a>
                            ),
                          }}
                          urlTransform={(url, key) => {
                            const resource = resolveSkillFile(url)
                            if (resource && key === 'href') {
                              return `#skill-file/${encodeURIComponent(resource.path)}`
                            }
                            if (resource?.binary && key === 'src') {
                              const extension = resource.path.split('.').at(-1)
                              const mime =
                                extension === 'png'
                                  ? 'image/png'
                                  : extension === 'jpg' || extension === 'jpeg'
                                    ? 'image/jpeg'
                                    : extension === 'gif'
                                      ? 'image/gif'
                                      : extension === 'webp'
                                        ? 'image/webp'
                                        : extension === 'avif'
                                          ? 'image/avif'
                                          : ''
                              if (mime)
                                return `data:${mime};base64,${resource.content}`
                            }
                            return defaultUrlTransform(url)
                          }}
                        >
                          {previewContent}
                        </ReactMarkdown>
                      </div>
                    </div>
                  ) : null}
                </div>
              ) : (
                <div className='flex flex-1 items-center justify-center text-sm text-muted-foreground'>
                  选择文件
                </div>
              )}
            </section>
            <aside
              id='skill-inspector'
              className={`${inspectorOpen ? 'flex' : 'hidden'} h-56 min-h-0 w-full shrink-0 flex-col overflow-y-auto bg-muted/20 lg:h-auto lg:w-64 lg:border-l xl:w-72`}
            >
              <div className='flex h-10 shrink-0 items-center justify-between ps-4 pe-3 text-sm font-medium text-muted-foreground'>
                <span>基础信息</span>
                <IconButtonTooltip label='隐藏基础信息'>
                  <Button
                    type='button'
                    aria-label='隐藏基础信息'
                    aria-controls='skill-inspector'
                    aria-expanded={true}
                    variant='ghost'
                    size='icon-sm'
                    onClick={() => setInspectorOpen(false)}
                  >
                    <PanelRightClose />
                  </Button>
                </IconButtonTooltip>
              </div>
              <FieldGroup className='gap-4 px-4 pt-1 pb-4'>
                <Field data-invalid={Boolean(metadata.error)}>
                  <FieldLabel htmlFor='skill-name'>名称</FieldLabel>
                  <Input
                    id='skill-name'
                    value={metadata.name}
                    maxLength={64}
                    aria-invalid={Boolean(metadata.error)}
                    disabled={Boolean(metadata.error)}
                    onChange={(event) => {
                      try {
                        setDraft((current) =>
                          updateMetadata(current, { name: event.target.value })
                        )
                        setError('')
                      } catch (cause) {
                        setError(
                          cause instanceof Error
                            ? cause.message
                            : '更新名称失败'
                        )
                      }
                    }}
                  />
                </Field>
                <Field data-invalid={Boolean(metadata.error)}>
                  <FieldLabel htmlFor='skill-description'>说明</FieldLabel>
                  <Input
                    id='skill-description'
                    value={metadata.description}
                    maxLength={1024}
                    aria-invalid={Boolean(metadata.error)}
                    disabled={Boolean(metadata.error)}
                    onChange={(event) => {
                      try {
                        setDraft((current) =>
                          updateMetadata(current, {
                            description: event.target.value,
                          })
                        )
                        setError('')
                      } catch (cause) {
                        setError(
                          cause instanceof Error
                            ? cause.message
                            : '更新说明失败'
                        )
                      }
                    }}
                  />
                </Field>
                {metadata.error ? (
                  <FieldError>{metadata.error}</FieldError>
                ) : null}
                <Field orientation='horizontal'>
                  <FieldLabel htmlFor='skill-enabled'>启用技能</FieldLabel>
                  <Switch
                    id='skill-enabled'
                    checked={draft.enabled}
                    disabled={toggle.isPending}
                    onCheckedChange={(enabled) =>
                      baseSkill?.id
                        ? toggle.mutate(enabled)
                        : setDraft((current) => ({ ...current, enabled }))
                    }
                  />
                </Field>
              </FieldGroup>
              {baseSkill?.id ? (
                <SkillVersionHistory
                  skillId={baseSkill.id}
                  currentVersion={baseSkill.version}
                />
              ) : null}
              <div className='px-4 py-3'>
                <p className='mb-2 text-xs font-medium'>检查结果</p>
                {errors.length ? (
                  <ul className='flex list-disc flex-col gap-1 ps-4 text-xs text-destructive'>
                    {errors.map((item) => (
                      <li key={item}>{item}</li>
                    ))}
                  </ul>
                ) : (
                  <p className='text-xs text-muted-foreground'>文件包可保存</p>
                )}
                {draft.files.some((file) => file.binary) ? (
                  <p className='mt-2 text-xs text-muted-foreground'>
                    二进制资源可保存，Agent 运行时仅能读取文本文件
                  </p>
                ) : null}
                {draft.files.some((file) =>
                  file.path.startsWith('scripts/')
                ) ? (
                  <p className='mt-2 text-xs text-muted-foreground'>
                    脚本文件仅供阅读，不会执行
                  </p>
                ) : null}
              </div>
              {selectedFile ? (
                <div className='px-4 py-3 text-xs text-muted-foreground'>
                  <p className='mb-2 font-medium text-foreground'>文件信息</p>
                  <p className='break-all'>{selectedFile.path}</p>
                  <p className='mt-1'>
                    {selectedFile.binary ? '二进制文件' : '文本文件'}
                  </p>
                </div>
              ) : null}
            </aside>
          </div>
          <div className='flex min-h-8 shrink-0 flex-wrap items-center justify-between gap-x-3 gap-y-1 border-t bg-muted/30 px-3 py-1 text-xs text-muted-foreground'>
            <div className='flex min-w-0 items-center gap-3'>
              <span className='shrink-0'>{fileCount} 个文件</span>
              <span className='truncate' title={selectedPath}>
                {selectedPath}
              </span>
            </div>
            <div className='flex min-w-0 flex-wrap items-center justify-end gap-2'>
              {dirty ? <span>未保存</span> : null}
              {errors.length ? <span>{errors.length} 项检查问题</span> : null}
              {error ? (
                <span role='alert' className='text-destructive'>
                  {error}
                </span>
              ) : null}
              {baseSkill?.id &&
              save.error instanceof ApiError &&
              save.error.status === 409 &&
              !remoteSkill ? (
                <Button
                  type='button'
                  size='xs'
                  variant='outline'
                  onClick={() => void refreshLatest()}
                >
                  读取最新版本
                </Button>
              ) : null}
              {remoteSkill ? (
                <>
                  <Button
                    type='button'
                    size='xs'
                    variant='outline'
                    onClick={adoptLatest}
                  >
                    采用最新版本
                  </Button>
                  <Button
                    type='button'
                    size='xs'
                    variant='outline'
                    onClick={() => setConfirmOverwrite(true)}
                  >
                    覆盖最新版本
                  </Button>
                </>
              ) : null}
              <Button
                size='sm'
                disabled={
                  !dirty ||
                  save.isPending ||
                  errors.length > 0 ||
                  Boolean(remoteSkill)
                }
                onClick={openSaveReview}
              >
                {save.isPending ? '正在保存…' : '保存'}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
      <Dialog
        open={Boolean(review)}
        onOpenChange={(next) => {
          if (!next && !save.isPending) setReview(undefined)
        }}
      >
        <DialogContent
          showCloseButton={false}
          className='flex h-[min(82dvh,760px)] w-[calc(100vw-2rem)] max-w-[1000px] flex-col gap-0 overflow-hidden rounded-lg bg-popover p-0 shadow-md sm:max-w-[1000px]'
          onEscapeKeyDown={(event) => {
            if (save.isPending) event.preventDefault()
          }}
        >
          <DialogHeader className='gap-1 border-b px-5 py-4'>
            <DialogTitle>确认保存技能</DialogTitle>
            <DialogDescription className='sr-only'>
              确认版本号并查看文件变化
            </DialogDescription>
          </DialogHeader>
          <div className='flex min-h-0 flex-1 flex-col'>
            <div className='px-5 py-4'>
              <Field
                data-invalid={Boolean(reviewError)}
                className='max-w-56 gap-1.5'
              >
                <FieldLabel htmlFor='skill-save-version'>版本号</FieldLabel>
                <Input
                  id='skill-save-version'
                  value={reviewVersion}
                  aria-invalid={Boolean(reviewError)}
                  autoComplete='off'
                  onChange={(event) => {
                    setReviewVersion(event.target.value)
                    setReviewError('')
                  }}
                />
                {reviewError ? <FieldError>{reviewError}</FieldError> : null}
              </Field>
              {nameChanged || descriptionChanged ? (
                <div className='mt-3 grid max-h-28 grid-cols-[3rem_minmax(0,1fr)_minmax(0,1fr)] gap-x-3 gap-y-1.5 overflow-y-auto text-xs'>
                  <span />
                  <span className='text-muted-foreground'>保存前</span>
                  <span className='text-muted-foreground'>保存后</span>
                  {nameChanged ? (
                    <>
                      <span className='text-muted-foreground'>名称</span>
                      <span className='break-all'>
                        {previousMetadata.name || '—'}
                      </span>
                      <span className='break-all'>
                        {reviewMetadata?.name || '—'}
                      </span>
                    </>
                  ) : null}
                  {descriptionChanged ? (
                    <>
                      <span className='text-muted-foreground'>说明</span>
                      <span className='break-all'>
                        {previousMetadata.description || '—'}
                      </span>
                      <span className='break-all'>
                        {reviewMetadata?.description || '—'}
                      </span>
                    </>
                  ) : null}
                </div>
              ) : null}
            </div>
            <div className='flex min-h-0 flex-1 border-y bg-background'>
              <div className='w-52 shrink-0 overflow-y-auto border-r bg-muted/20 py-2'>
                <p className='px-3 pb-2 text-xs font-medium text-muted-foreground'>
                  文件变化 · {reviewChanges.length}
                </p>
                {reviewChanges.map((change) => (
                  <button
                    key={change.path}
                    type='button'
                    aria-current={
                      reviewChange?.path === change.path ? 'true' : undefined
                    }
                    className='flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs hover:bg-accent aria-[current=true]:bg-accent aria-[current=true]:text-accent-foreground'
                    onClick={() => setReviewPath(change.path)}
                  >
                    <span
                      className='min-w-0 flex-1 truncate'
                      title={change.path}
                    >
                      {change.path}
                    </span>
                    <span className='shrink-0 text-muted-foreground'>
                      {change.kind === 'added'
                        ? '新增'
                        : change.kind === 'deleted'
                          ? '删除'
                          : '修改'}
                    </span>
                  </button>
                ))}
              </div>
              <SkillFileDiff change={reviewChange} />
            </div>
          </div>
          <DialogFooter className='m-0 rounded-none border-0 bg-popover px-5 py-3'>
            <Button
              type='button'
              variant='outline'
              disabled={save.isPending}
              onClick={() => setReview(undefined)}
            >
              取消
            </Button>
            <Button
              type='button'
              disabled={save.isPending}
              onClick={() => {
                if (!review) return
                const message = skillVersionError(
                  reviewVersion,
                  baseSkill?.id ? baseSkill.version : undefined
                )
                if (message) {
                  setReviewError(message)
                  return
                }
                save.mutate({ review, version: reviewVersion })
              }}
            >
              {save.isPending ? '正在保存…' : '保存版本'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <AlertDialog open={confirmClose} onOpenChange={setConfirmClose}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>放弃未保存的修改？</AlertDialogTitle>
            <AlertDialogDescription>
              当前文件修改尚未保存。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>继续编辑</AlertDialogCancel>
            <AlertDialogAction onClick={() => onOpenChange(false)}>
              放弃修改
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      <AlertDialog open={confirmOverwrite} onOpenChange={setConfirmOverwrite}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>覆盖最新版本？</AlertDialogTitle>
            <AlertDialogDescription>
              确认后仍需再次保存。当前编辑内容将覆盖已读取的最新版本。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={overwriteLatest}>
              确认覆盖
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      <AlertDialog
        open={Boolean(deletingPath)}
        onOpenChange={(next) => {
          if (!next) setDeletingPath('')
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              删除
              {draft.files.find((file) => file.path === deletingPath)?.directory
                ? '文件夹'
                : '文件'}
              ？
            </AlertDialogTitle>
            <AlertDialogDescription>
              {deletingPath} 及其包含的内容将从技能中移除。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={() => removePath(deletingPath)}>
              删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}

const SkillFileDiff = memo(function SkillFileDiff({
  change,
}: {
  change?: SkillFileChange
}) {
  if (!change) {
    return (
      <div className='flex min-w-0 flex-1 items-center justify-center text-sm text-muted-foreground'>
        文件内容无变化
      </div>
    )
  }
  if (
    change.before?.binary ||
    change.after?.binary ||
    change.before?.directory ||
    change.after?.directory
  ) {
    return (
      <div className='flex min-w-0 flex-1 items-center justify-center text-sm text-muted-foreground'>
        {change.before?.directory || change.after?.directory
          ? '目录变化'
          : '二进制文件无法显示文本差异'}
      </div>
    )
  }
  return (
    <div className='flex min-w-0 flex-1 flex-col overflow-hidden'>
      <div className='truncate border-b bg-muted/20 px-4 py-2 font-mono text-xs text-muted-foreground'>
        {change.path}
      </div>
      <CodeEditor
        key={change.path}
        aria-label={`${change.path} 差异`}
        language={/\.md$/i.test(change.path) ? 'markdown' : 'text'}
        value={change.after?.content ?? ''}
        extensions={[
          unifiedMergeView({
            original: change.before?.content ?? '',
            mergeControls: false,
            collapseUnchanged: { margin: 3 },
          }),
        ]}
        readOnly
        editable={false}
        fill
        className='min-h-0 min-w-0 flex-1 rounded-none border-0'
        basicSetup={{ lineNumbers: true, foldGutter: false }}
      />
    </div>
  )
})

function SkillVersionHistory({
  skillId,
  currentVersion,
}: {
  skillId: string
  currentVersion: string
}) {
  const [selectedVersion, setSelectedVersion] = useState('')
  const [selectedPath, setSelectedPath] = useState('SKILL.md')
  const versions = useQuery({
    queryKey: ['studio', 'skills', skillId, 'versions'],
    queryFn: () => listStudioSkillVersions(skillId),
  })
  const snapshot = useQuery({
    queryKey: ['studio', 'skills', skillId, 'versions', selectedVersion],
    queryFn: () => getStudioSkillVersion(skillId, selectedVersion),
    enabled: Boolean(selectedVersion),
  })
  const files = snapshot.data?.files?.length
    ? snapshot.data.files
    : snapshot.data
      ? [{ path: 'SKILL.md', content: snapshot.data.prompt }]
      : []
  const selectedFile = files.find(
    (file) => file.path === selectedPath && !file.directory
  )

  return (
    <div className='px-4 py-3'>
      <p className='mb-2 text-xs font-medium'>版本历史</p>
      {versions.isPending ? (
        <p className='text-xs text-muted-foreground'>正在读取版本…</p>
      ) : versions.isError ? (
        <p role='alert' className='text-xs text-destructive'>
          读取版本失败
        </p>
      ) : (
        <div className='flex flex-col gap-0.5'>
          {versions.data.map((item) => (
            <button
              key={item.version}
              type='button'
              className='flex w-full items-center justify-between gap-2 rounded-md px-2 py-1.5 text-left text-xs hover:bg-accent hover:text-accent-foreground'
              onClick={() => {
                setSelectedVersion(item.version)
                setSelectedPath('SKILL.md')
              }}
            >
              <span className='flex min-w-0 flex-col gap-0.5'>
                <span className='font-mono'>v{item.version}</span>
                <span className='text-muted-foreground'>
                  {new Date(item.created_at).toLocaleString('zh-CN')}
                </span>
              </span>
              {item.version === currentVersion ? (
                <span className='text-muted-foreground'>当前版本</span>
              ) : null}
            </button>
          ))}
        </div>
      )}
      <Dialog
        open={Boolean(selectedVersion)}
        onOpenChange={(next) => {
          if (!next) setSelectedVersion('')
        }}
      >
        <DialogContent className='flex h-[min(78dvh,720px)] w-[calc(100vw-2rem)] max-w-[960px] flex-col gap-0 overflow-hidden rounded-lg bg-popover p-0 shadow-md sm:max-w-[960px]'>
          <DialogHeader className='gap-1 border-b px-5 py-4 pe-12'>
            <DialogTitle>技能版本 v{selectedVersion}</DialogTitle>
            <DialogDescription>
              {snapshot.data
                ? `${snapshot.data.name} · ${snapshot.data.description}`
                : '正在读取版本内容…'}
            </DialogDescription>
          </DialogHeader>
          {snapshot.isError ? (
            <p role='alert' className='p-5 text-sm text-destructive'>
              读取版本内容失败
            </p>
          ) : (
            <div className='flex min-h-0 flex-1 bg-background'>
              <div className='w-52 shrink-0 overflow-y-auto border-r bg-muted/20 py-2'>
                {files
                  .filter((file) => !file.directory)
                  .map((file) => (
                    <button
                      key={file.path}
                      type='button'
                      aria-current={
                        selectedPath === file.path ? 'true' : undefined
                      }
                      className='block w-full truncate px-3 py-1.5 text-left text-xs hover:bg-accent aria-[current=true]:bg-accent aria-[current=true]:text-accent-foreground'
                      title={file.path}
                      onClick={() => setSelectedPath(file.path)}
                    >
                      {file.path}
                    </button>
                  ))}
              </div>
              {selectedFile?.binary ? (
                <div className='flex min-w-0 flex-1 items-center justify-center text-sm text-muted-foreground'>
                  二进制文件无法显示文本内容
                </div>
              ) : selectedFile ? (
                <CodeEditor
                  key={`${selectedVersion}:${selectedPath}`}
                  aria-label={`${selectedPath} 版本内容`}
                  language={/\.md$/i.test(selectedPath) ? 'markdown' : 'text'}
                  value={selectedFile.content}
                  readOnly
                  editable={false}
                  fill
                  className='min-h-0 min-w-0 flex-1 rounded-none border-0'
                  basicSetup={{ lineNumbers: true, foldGutter: false }}
                />
              ) : null}
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}

function SkillFileTree({
  files,
  parent = '',
  onFolderSelect,
}: {
  files: StudioSkillFile[]
  parent?: string
  onFolderSelect: (path: string) => void
}) {
  const folders = new Set<string>()
  const directFiles: StudioSkillFile[] = []
  for (const file of files) {
    const relative = parent ? file.path.slice(parent.length + 1) : file.path
    const slash = relative.indexOf('/')
    if (slash >= 0) folders.add(relative.slice(0, slash))
    else if (file.directory) folders.add(relative)
    else directFiles.push(file)
  }
  return (
    <>
      {[...folders].sort().map((name) => {
        const path = parent ? `${parent}/${name}` : name
        return (
          <FileTreeFolder
            key={path}
            name={name}
            path={path}
            data-skill-path={path}
            onClick={(event) => {
              if (
                (event.target as HTMLElement)
                  .closest('[data-skill-path]')
                  ?.getAttribute('data-skill-path') === path
              )
                onFolderSelect(path)
            }}
          >
            <SkillFileTree
              files={files.filter((file) => file.path.startsWith(`${path}/`))}
              parent={path}
              onFolderSelect={onFolderSelect}
            />
          </FileTreeFolder>
        )
      })}
      {directFiles
        .sort((a, b) => a.path.localeCompare(b.path))
        .map((file) => (
          <FileTreeFile
            key={file.path}
            name={file.path.slice(parent ? parent.length + 1 : 0)}
            path={file.path}
            data-skill-path={file.path}
            icon={
              file.binary ? (
                <FileCode2 className='size-3.5 shrink-0' />
              ) : undefined
            }
          />
        ))}
    </>
  )
}

function BinaryPreview({
  file,
  onDownload,
}: {
  file: StudioSkillFile
  onDownload: () => void
}) {
  const extension = file.path.split('.').at(-1)?.toLowerCase()
  const mime =
    extension === 'png'
      ? 'image/png'
      : extension === 'jpg' || extension === 'jpeg'
        ? 'image/jpeg'
        : extension === 'gif'
          ? 'image/gif'
          : extension === 'webp'
            ? 'image/webp'
            : extension === 'mp4'
              ? 'video/mp4'
              : extension === 'webm'
                ? 'video/webm'
                : extension === 'mp3'
                  ? 'audio/mpeg'
                  : extension === 'wav'
                    ? 'audio/wav'
                    : ''
  const url = mime ? `data:${mime};base64,${file.content}` : ''
  return (
    <div className='flex min-h-0 flex-1 flex-col items-center justify-center gap-4 overflow-auto bg-background p-6'>
      {mime.startsWith('image/') ? (
        <img
          alt={file.path}
          src={url}
          className='max-h-full max-w-full object-contain'
        />
      ) : null}
      {mime.startsWith('video/') ? (
        <video controls src={url} className='max-h-full max-w-full' />
      ) : null}
      {mime.startsWith('audio/') ? <audio controls src={url} /> : null}
      {!mime ? (
        <p className='text-sm text-muted-foreground'>无法预览此文件</p>
      ) : null}
      <Button size='sm' variant='outline' onClick={onDownload}>
        <Download />
        下载文件
      </Button>
    </div>
  )
}
