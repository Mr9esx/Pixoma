import { useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import type { FileUIPart } from 'ai'
import {
  ArrowUp,
  Check,
  ChevronDown,
  Folder,
  Plus,
  X,
} from 'lucide-react'
import {
  listStudioProjects,
  listStudioAgentWorkflows,
  listStudioLibraryAssets,
  sendStudioMessage,
  uploadStudioAsset,
  type StudioModel,
  type StudioPermissionMode,
  type StudioSkillSummary,
} from '@/lib/api/studio'
import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'
import { i18n } from '@/lib/i18n'
import { Command, CommandInput } from '@/components/ui/command'
import { Shimmer } from '@/components/ai-elements/shimmer'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  PromptInput,
  PromptInputBody,
  PromptInputFooter,
  PromptInputHeader,
  PromptInputSubmit,
  PromptInputTools,
} from '@/components/ai-elements/prompt-input'
import {
  AssetPicker,
  SkillPicker,
  StudioImageAttachments,
  StudioImagePicker,
  WorkflowPicker,
} from './studio-chat'
import { ModelPicker, PermissionPicker } from './studio-chat-controls'
import {
  StudioComposer,
  type StudioComposerHandle,
  type StudioReference,
} from './studio-composer'
import type { StudioComposerValue } from './studio-composer-content'
import composerSurface from './studio-composer-surface.module.css'
import { StudioProjectDialog } from './studio-project-dialog'

export function studioNewSessionActivity(
  pending: boolean,
  uploadingImages: boolean
) {
  if (uploadingImages) return '正在上传图片…'
  if (pending) return '正在创建对话…'
}

export function StudioNewSession({
  models,
  skills,
  initialProjectId,
}: {
  models: StudioModel[]
  skills: StudioSkillSummary[]
  initialProjectId?: string
}) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const composerRef = useRef<StudioComposerHandle>(null)
  const formRef = useRef<HTMLDivElement>(null)
  const requestId = useRef<string>(undefined)
  const submittedFiles = useRef<FileUIPart[]>([])
  const [projectId, setProjectId] = useState(initialProjectId ?? '')
  const [pickerOpen, setPickerOpen] = useState(false)
  const [projectDialogOpen, setProjectDialogOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [modelId, setModelId] = useState('')
  const [permissionMode, setPermissionMode] =
    useState<StudioPermissionMode>('request_approval')
  const [composerValue, setComposerValue] = useState<StudioComposerValue>({
    text: '',
    parts: [],
    selectedSkillIds: [],
    selectedAssets: [],
  })
  const [sendingImages, setSendingImages] = useState(false)
  const [attachmentError, setAttachmentError] = useState('')
  const [activePicker, setActivePicker] = useState<
    'skill' | 'asset' | 'workflow' | null
  >(null)
  const [activeSuggestion, setActiveSuggestion] = useState<{ kind: StudioReference['kind']; query: string } | null>(null)
  const changePicker = (
    picker: 'skill' | 'asset' | 'workflow',
    open: boolean
  ) => {
    setActivePicker((current) =>
      open ? picker : current === picker ? null : current
    )
  }
  const projects = useQuery({
    queryKey: ['studio', 'projects'],
    queryFn: listStudioProjects,
  })
  const suggestedWorkflows = useQuery({ queryKey: ['studio', 'agent-workflows'], queryFn: listStudioAgentWorkflows, enabled: activeSuggestion?.kind === 'workflow' })
  const suggestedLibraryAssets = useQuery({
    queryKey: ['studio', 'composer', 'library-assets', projectId, activeSuggestion?.kind === 'asset' ? activeSuggestion.query : ''],
    queryFn: () => listStudioLibraryAssets({ projectId, search: activeSuggestion?.query ?? '', limit: 100 }),
    enabled: activeSuggestion?.kind === 'asset',
  })
  const availableModels = models.filter(
    (model) => model.enabled && model.agent_enabled && model.capabilities.tools
  )
  const selectedModel =
    availableModels.find((model) => model.id === modelId) ??
    availableModels.find((model) => model.default) ??
    availableModels[0]
  const selectedProject = projects.data?.find(
    (project) => project.id === projectId
  )
  const projectName = projectId
    ? (selectedProject?.name ??
      (projects.isLoading ? '读取项目…' : '项目不存在'))
    : '不在项目内'
  const chooseProject = (id: string) => {
    requestId.current = undefined
    setPickerOpen(false)
    setProjectId(id)
  }
  const referenceItems: StudioReference[] = [
    ...skills.filter((skill) => skill.enabled).map((skill) => ({ kind: 'skill' as const, id: skill.id, label: skill.name })),
    ...(suggestedLibraryAssets.data?.items ?? []).map((item) => ({ kind: 'asset' as const, id: item.asset_id, label: item.display_name, versionId: item.version.id })),
    ...(suggestedWorkflows.data ?? []).filter((workflow) => workflow.workflow_enabled && workflow.agent_enabled).map((workflow) => ({ kind: 'workflow' as const, id: workflow.id, label: workflow.name })),
  ]
  const send = useMutation({
    mutationFn: async (files: FileUIPart[]) => {
      const value = composerRef.current?.serialize()
      if (!value || !selectedModel) throw new Error('请输入消息并选择模型')
      if (projectId && !selectedProject) throw new Error('所选项目不存在')
      let text = value.text
      const parts = [...value.parts]
      if (!text.trim() && files.length > 0) {
        text = '请分析这张图片'
        parts.push({ type: 'text', text })
      }
      if (!text.trim()) throw new Error('请输入消息并选择模型')
      if (files.length > 0) {
        if (!selectedModel.capabilities.vision)
          throw new Error(
            '当前模型未开启图片输入，请切换模型或在 AI 设置中开启。'
          )
        setSendingImages(true)
        try {
          const images = await Promise.all(
            files.map(async (attachment) => {
              if (
                !attachment.url ||
                !attachment.filename ||
                !attachment.mediaType
              ) {
                throw new Error('图片内容不完整，请重新选择。')
              }
              const blob = await fetch(attachment.url).then((response) =>
                response.blob()
              )
              return new File([blob], attachment.filename, {
                type: attachment.mediaType,
              })
            })
          )
          if (
            images.reduce((total, image) => total + image.size, 0) >
            20 << 20
          ) {
            throw new Error('图片总大小不能超过 20 MB。')
          }
          for (const image of images) {
            const asset = await uploadStudioAsset(image)
            const version = asset.versions.find(
              (item) => item.version === asset.current_version
            )
            if (!version) throw new Error('上传的图片缺少版本信息。')
            parts.push({ type: 'text', text: '\n' })
            parts.push({
              type: 'asset_ref',
              asset_id: asset.id,
              asset_version_id: version.id,
              name: asset.name,
            })
            text += `\n「${asset.name}」资产`
          }
        } finally {
          setSendingImages(false)
        }
      }
      if (
        submittedFiles.current.length !== files.length ||
        files.some(
          (file, index) => file.url !== submittedFiles.current[index]?.url
        )
      )
        requestId.current = undefined
      submittedFiles.current = files
      requestId.current ??= crypto.randomUUID()
      return sendStudioMessage({
        text,
        locale: i18n.language.startsWith('en') ? 'en' : 'zh',
        parts,
        projectId,
        requestId: requestId.current,
        modelConfigId: selectedModel.id,
        permissionMode,
      })
    },
    onSuccess: ({ session }) => {
      void queryClient.invalidateQueries({ queryKey: ['studio', 'sessions'] })
      void navigate({
        to: '/studio/sessions/$sessionId',
        params: { sessionId: session.id },
        search: {},
      })
    },
  })
  const activity = studioNewSessionActivity(send.isPending, sendingImages)

  return (
    <main
      id='main-content'
      className='flex min-h-0 min-w-0 flex-1 py-3 pr-3 sm:py-4 sm:pr-4'
    >
      <div className='flex min-h-0 w-full items-center justify-center rounded-2xl border bg-card pt-16 pb-20'>
        <div className='w-full max-w-3xl px-5'>
          <h1 className='mb-7 text-center font-heading text-2xl font-medium'>
            新对话
          </h1>
          <div ref={formRef}>
            <PromptInput
              aria-busy={Boolean(activity)}
              accept='image/png,image/jpeg,image/webp,image/gif'
              multiple
              maxFiles={4}
              maxFileSize={8 << 20}
              inputGroupClassName={`h-auto overflow-visible ${composerSurface.surface}`}
              onError={(error) =>
                setAttachmentError(
                  error.code === 'max_file_size'
                    ? '单张图片不能超过 8 MB。'
                    : error.code === 'max_files'
                      ? '最多添加 4 张图片。'
                      : '请选择 PNG、JPEG、WebP 或 GIF 图片。'
                )
              }
              onSubmit={async ({ files }) => {
                setAttachmentError('')
                await send.mutateAsync(files)
              }}
            >
              <PromptInputHeader className='mx-2 mt-2 w-[calc(100%-1rem)] rounded-xl bg-muted/60 px-2 py-1'>
                <Popover open={pickerOpen} onOpenChange={setPickerOpen}>
                  <PopoverTrigger asChild>
                    <Button
                      type='button'
                      variant='ghost'
                      size='sm'
                      className='h-7 max-w-full gap-2 px-2 font-normal'
                      aria-label={`选择项目：${projectName}`}
                    >
                      <Folder className='size-4 shrink-0' />
                      <span className='truncate'>{projectName}</span>
                      <ChevronDown className='size-3.5 shrink-0' />
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent
                    align='start'
                    side='top'
                    sideOffset={8}
                    avoidCollisions={false}
                    className='w-64 p-1'
                  >
                    <Command className='h-auto [&_[data-slot=command-input-wrapper]]:border-0'>
                      <CommandInput
                        aria-label='搜索项目'
                        placeholder='搜索项目'
                        value={search}
                        onValueChange={setSearch}
                      />
                    </Command>
                    <div className='studio-scrollbar max-h-[clamp(2rem,calc(50dvh-11rem),14rem)] overflow-y-auto'>
                      {projects.isError ? (
                        <Button
                          type='button'
                          variant='ghost'
                          size='sm'
                          className='w-full justify-start'
                          onClick={() => void projects.refetch()}
                        >
                          重试读取项目
                        </Button>
                      ) : null}
                      {projects.data
                        ?.filter((project) =>
                          project.name
                            .toLocaleLowerCase()
                            .includes(search.toLocaleLowerCase())
                        )
                        .map((project) => (
                          <Button
                            key={project.id}
                            type='button'
                            variant='ghost'
                            size='sm'
                            className='w-full justify-start gap-2 font-normal has-[>svg]:px-3'
                            onClick={() => chooseProject(project.id)}
                          >
                            <Folder className='size-4 shrink-0' />
                            <span className='min-w-0 flex-1 truncate text-left'>
                              {project.name}
                            </span>
                            {project.id === projectId ? (
                              <Check className='size-4' />
                            ) : null}
                          </Button>
                        ))}
                    </div>
                    <div className='mt-1 space-y-0.5 pt-1'>
                      <Button
                        type='button'
                        variant='ghost'
                        size='sm'
                        className='w-full justify-start gap-2 font-normal has-[>svg]:px-3'
                        onClick={() => {
                          setPickerOpen(false)
                          setProjectDialogOpen(true)
                        }}
                      >
                        <Plus className='size-4' />
                        新建项目
                      </Button>
                      <Button
                        type='button'
                        variant='ghost'
                        size='sm'
                        className='w-full justify-start gap-2 font-normal has-[>svg]:px-3'
                        onClick={() => chooseProject('')}
                      >
                        <X className='size-4' />
                        不在项目内
                        {!projectId ? (
                          <Check className='ms-auto size-4' />
                        ) : null}
                      </Button>
                    </div>
                  </PopoverContent>
                </Popover>
              </PromptInputHeader>
              <PromptInputBody>
                <StudioImageAttachments disabled={send.isPending} />
                <StudioComposer
                  ref={composerRef}
                  placeholder={
                    selectedModel
                      ? '描述你想创作的内容…'
                      : '先在 AI 设置中添加并启用模型'
                  }
                  disabled={send.isPending || !selectedModel}
                  referenceItems={referenceItems}
                  onSuggestionChange={(kind, query) => setActiveSuggestion(kind ? { kind, query } : null)}
                  suggestionLoading={activeSuggestion?.kind === 'asset' ? suggestedLibraryAssets.isPending || suggestedLibraryAssets.isFetching : activeSuggestion?.kind === 'workflow' && suggestedWorkflows.isPending}
                  suggestionError={activeSuggestion?.kind === 'asset' ? suggestedLibraryAssets.isError : activeSuggestion?.kind === 'workflow' && suggestedWorkflows.isError}
                  onValueChange={(value) => {
                    requestId.current = undefined
                    setComposerValue(value)
                  }}
                  onSubmit={() =>
                    formRef.current?.querySelector('form')?.requestSubmit()
                  }
                />
              </PromptInputBody>
              <PromptInputFooter className='flex-wrap sm:flex-nowrap'>
                <PromptInputTools>
                  <StudioImagePicker
                    unsupported={Boolean(
                      selectedModel && !selectedModel.capabilities.vision
                    )}
                    disabled={
                      !selectedModel ||
                      !selectedModel.capabilities.vision ||
                      send.isPending
                    }
                  />
                  <SkillPicker
                    open={activePicker === 'skill'}
                    onOpenChange={(open) => changePicker('skill', open)}
                    skills={skills}
                    value={composerValue.selectedSkillIds}
                    onInsert={(skill) =>
                      composerRef.current?.insertReference({
                        kind: 'skill',
                        id: skill.id,
                        label: skill.name,
                      })
                    }
                  />
                  <AssetPicker
                    open={activePicker === 'asset'}
                    onOpenChange={(open) => changePicker('asset', open)}
                    assets={[]}
                    value={composerValue.selectedAssets}
                    defaultTab='global'
                    onInsert={(asset, versionId) =>
                      composerRef.current?.insertReference({
                        kind: 'asset',
                        id: asset.id,
                        label: asset.name,
                        versionId,
                      })
                    }
                  />
                  <WorkflowPicker
                    open={activePicker === 'workflow'}
                    onOpenChange={(open) => changePicker('workflow', open)}
                    disabled={send.isPending}
                    onSelect={(workflow) =>
                      composerRef.current?.insertReference({
                        kind: 'workflow',
                        id: workflow.id,
                        label: workflow.name,
                      })
                    }
                  />
                  <PermissionPicker
                    value={permissionMode}
                    onChange={(value) => {
                      requestId.current = undefined
                      setPermissionMode(value)
                    }}
                  />
                </PromptInputTools>
                <PromptInputTools className='max-sm:w-full max-sm:justify-between'>
                  <ModelPicker
                    models={availableModels}
                    value={selectedModel?.id}
                    onChange={(value) => {
                      requestId.current = undefined
                      setModelId(value)
                    }}
                  />
                  <PromptInputSubmit
                    className='size-9 rounded-full'
                    size='icon-sm'
                    disabled={
                      !selectedModel ||
                      (Boolean(projectId) && !selectedProject) ||
                      send.isPending
                    }
                    aria-label={activity ?? '发送消息'}
                    status={send.isPending ? 'submitted' : undefined}
                  >
                    {send.isPending ? <Spinner data-icon='inline-start' /> : <ArrowUp data-icon='inline-start' />}
                  </PromptInputSubmit>
                </PromptInputTools>
              </PromptInputFooter>
            </PromptInput>
          </div>
          {activity ? (
            <div
              role='status'
              aria-live='polite'
              className='mt-3 flex justify-center text-sm text-muted-foreground'
            >
              <Shimmer>{activity}</Shimmer>
            </div>
          ) : null}
          {send.isError || attachmentError ? (
            <p role='alert' className='mt-3 text-sm text-destructive'>
              {attachmentError || send.error?.message}
            </p>
          ) : null}
        </div>
      </div>
      {projectDialogOpen ? (
        <StudioProjectDialog
          open
          onOpenChange={setProjectDialogOpen}
          onSaved={(project) => chooseProject(project.id)}
        />
      ) : null}
    </main>
  )
}
