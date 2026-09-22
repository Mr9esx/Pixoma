import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { HttpAgent } from '@ag-ui/client'
import {
  AssistantRuntimeProvider,
  ComposerPrimitive,
  MessagePrimitive,
  ThreadPrimitive,
  useAui,
} from '@assistant-ui/react'
import { useAgUiRuntime } from '@assistant-ui/react-ag-ui'
import {
  Bot,
  Check,
  ChevronDown,
  Paperclip,
  Send,
  ShieldCheck,
  Sparkles,
  User,
} from 'lucide-react'
import { baseURL, sessionToken } from '@/lib/api/client'
import {
  listStudioLibraryAssets,
  type StudioAsset,
  type StudioMessage,
  type StudioModel,
  type StudioPermissionMode,
  type StudioSkill,
} from '@/lib/api/studio'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

type Props = {
  sessionId: string
  messages: StudioMessage[]
  models: StudioModel[]
  modelConfigId?: string
  permissionMode: StudioPermissionMode
  skills: StudioSkill[]
  assets: StudioAsset[]
  selectedSkillIds: string[]
  selectedAssets: SelectedAsset[]
  onModelChange: (id: string) => void
  onPermissionChange: (mode: StudioPermissionMode) => void
  onSkillChange: (ids: string[]) => void
  onAssetChange: (assets: SelectedAsset[]) => void
	 onImportLibraryAsset: (asset: SelectedAsset) => Promise<StudioAsset>
}

type SelectedAsset = { assetId: string; assetVersionId: string }

function toAGUIMessages(messages: StudioMessage[]) {
  return messages
    .filter(
      (message) => message.role === 'user' || message.role === 'assistant'
    )
    .map((message) => ({
      id: message.id,
      role: message.role,
      content: message.content
        .filter((part) => part.type === 'text')
        .map((part) => part.text ?? '')
        .join(''),
    }))
}

export function StudioChat(props: Props) {
  const selectedModel =
    props.models.find((model) => model.id === props.modelConfigId) ??
    props.models.find((model) => model.default) ??
    props.models[0]

  const agent = useMemo(() => {
    return new HttpAgent({
      url: `${baseURL()}/api/v1/studio/agui`,
      threadId: props.sessionId,
      initialMessages: toAGUIMessages(props.messages) as never[],
      fetch: async (url, init) => {
        const body = JSON.parse(String(init.body ?? '{}')) as Record<
          string,
          unknown
        >
        body.forwardedProps = {
          ...((body.forwardedProps as Record<string, unknown>) ?? {}),
          runConfig: {
            modelConfigId: selectedModel?.id ?? '',
            permissionMode: props.permissionMode,
            selectedSkillIds: props.selectedSkillIds,
            selectedAssets: props.selectedAssets,
          },
        }
        const token = sessionToken()
        return fetch(url, {
          ...init,
          credentials: 'include',
          headers: {
            ...((init.headers as Record<string, string>) ?? {}),
            'Content-Type': 'application/json',
            ...(token ? { Authorization: `Bearer ${token}` } : {}),
          },
          body: JSON.stringify(body),
        })
      },
    })
  }, [
    props.sessionId,
    selectedModel?.id,
    props.permissionMode,
    props.selectedSkillIds,
    props.selectedAssets,
  ])

  const runtime = useAgUiRuntime({ agent, showThinking: true })

  return (
    <AssistantRuntimeProvider runtime={runtime}>
      <ThreadPrimitive.Root className='flex min-h-0 flex-1 flex-col'>
        <ThreadPrimitive.Viewport className='min-h-0 flex-1 overflow-y-auto scroll-smooth'>
          <div className='mx-auto flex min-h-full w-full max-w-3xl flex-col px-5 py-8'>
            <ThreadPrimitive.Empty>
              <StudioWelcome />
            </ThreadPrimitive.Empty>
            <ThreadPrimitive.Messages
              components={{
                UserMessage: StudioUserMessage,
                AssistantMessage: StudioAssistantMessage,
              }}
            />
            <div className='min-h-6 flex-1' />
          </div>
        </ThreadPrimitive.Viewport>
        <div className='shrink-0 px-4 pt-2 pb-5'>
          <ComposerPrimitive.Root className='mx-auto w-full max-w-3xl rounded-2xl border bg-card p-2 focus-within:border-ring focus-within:ring-3 focus-within:ring-ring/20'>
            <ComposerPrimitive.Input
              autoFocus
              rows={3}
              placeholder='描述你想创作的内容，或让 Agent 调用工作流…'
              className='max-h-40 min-h-20 w-full resize-none bg-transparent px-3 py-2 text-sm leading-6 outline-none placeholder:text-muted-foreground'
            />
            <div className='flex flex-wrap items-center justify-between gap-2 px-1 pb-1'>
              <div className='flex flex-wrap items-center gap-1'>
                <ModelPicker
                  models={props.models}
                  value={selectedModel?.id}
                  onChange={props.onModelChange}
                />
                <SkillPicker
                  skills={props.skills}
                  value={props.selectedSkillIds}
                  onChange={props.onSkillChange}
                />
                <AssetPicker
                  assets={props.assets}
                  value={props.selectedAssets}
                  onChange={props.onAssetChange}
				  onImportLibraryAsset={props.onImportLibraryAsset}
                />
                <PermissionPicker
                  value={props.permissionMode}
                  onChange={props.onPermissionChange}
                />
              </div>
              <ComposerPrimitive.Send asChild>
                <Button
                  size='icon'
                  className='rounded-xl'
                  aria-label='发送消息'
                >
                  <Send />
                </Button>
              </ComposerPrimitive.Send>
            </div>
          </ComposerPrimitive.Root>
          <p className='mx-auto mt-2 max-w-3xl text-center text-xs text-muted-foreground'>
            Agent 可能会调用模型、Skill、连接器和工作流，请核对重要结果。
          </p>
        </div>
      </ThreadPrimitive.Root>
    </AssistantRuntimeProvider>
  )
}

function AssetPicker({
  assets,
  value,
  onChange,
	 onImportLibraryAsset,
}: {
  assets: StudioAsset[]
  value: SelectedAsset[]
  onChange: (assets: SelectedAsset[]) => void
	 onImportLibraryAsset: (asset: SelectedAsset) => Promise<StudioAsset>
}) {
  const libraryAssets = useQuery({
    queryKey: ['studio', 'library', 'assets'],
    queryFn: () => listStudioLibraryAssets(),
  })
  const sessionAssetIDs = new Set(assets.map((asset) => asset.id))
  const assetsByID = new Map<string, StudioAsset>()
  for (const asset of assets) assetsByID.set(asset.id, asset)
  for (const asset of libraryAssets.data ?? []) {
    assetsByID.set(asset.id, asset)
  }
  const selected = new Map(value.map((item) => [item.assetId, item.assetVersionId]))
  const toggle = async (asset: StudioAsset, fromLibrary = false) => {
    const version = asset.versions[asset.versions.length - 1]
    if (!version) return
    if (selected.get(asset.id) === version.id) {
      onChange(value.filter((item) => item.assetId !== asset.id))
      return
    }
    let next = { assetId: asset.id, assetVersionId: version.id }
    if (fromLibrary) {
      const imported = await onImportLibraryAsset(next)
      const importedVersion = imported.versions[imported.versions.length - 1]
      if (!importedVersion) return
      next = { assetId: imported.id, assetVersionId: importedVersion.id }
    }
    onChange([
      ...value.filter((item) => item.assetId !== asset.id),
      next,
    ])
  }
  const unavailableSelections = value.filter((selection) => {
    const asset = assetsByID.get(selection.assetId)
    return !asset || !asset.versions.some((version) => version.id === selection.assetVersionId)
  })
  const currentAssets = assets
  const reusableAssets = (libraryAssets.data ?? []).filter(
    (asset) => !sessionAssetIDs.has(asset.id)
  )

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant='ghost' size='sm' className='rounded-lg'>
          <Paperclip />
          {value.length === 0 ? '资产' : `资产 · ${value.length}`}
          <ChevronDown className='size-3.5' />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align='start' className='w-72'>
        <DropdownMenuLabel>本轮使用的资产</DropdownMenuLabel>
        <DropdownMenuSeparator />
        <AssetPickerSection
          label='当前 Session'
          assets={currentAssets}
          selected={selected}
          onToggle={toggle}
          emptyText='当前 Session 还没有资产'
        />
        <DropdownMenuSeparator />
        <AssetPickerSection
          label='资产库'
          assets={reusableAssets}
          selected={selected}
		  onToggle={(asset) => void toggle(asset, true)}
          emptyText={libraryAssets.isLoading ? '正在读取资产库…' : '资产库还没有可复用资产'}
        />
        {unavailableSelections.length > 0 ? (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem disabled>
              {unavailableSelections.length} 项已选资产不可用
            </DropdownMenuItem>
          </>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

function AssetPickerSection({
  label,
  assets,
  selected,
  onToggle,
  emptyText,
}: {
  label: string
  assets: StudioAsset[]
  selected: Map<string, string>
  onToggle: (asset: StudioAsset) => void | Promise<void>
  emptyText: string
}) {
  return (
    <>
      <DropdownMenuLabel className='text-xs font-medium text-muted-foreground'>
        {label}
      </DropdownMenuLabel>
      {assets.length === 0 ? (
        <DropdownMenuItem disabled>{emptyText}</DropdownMenuItem>
      ) : (
        assets.map((asset) => {
          const version = asset.versions[asset.versions.length - 1]
          return (
            <DropdownMenuCheckboxItem
              key={asset.id}
              checked={selected.get(asset.id) === version?.id}
              onSelect={(event) => event.preventDefault()}
			  onCheckedChange={() => void onToggle(asset)}
            >
              <span className='min-w-0 flex-1'>
                <span className='block truncate'>{asset.name}</span>
                <span className='block text-xs text-muted-foreground'>
                  {asset.kind === 'image' ? '图片' : asset.kind === 'document' ? '文档' : '文件'}{version ? ` · v${version.version}` : ''}
                </span>
              </span>
            </DropdownMenuCheckboxItem>
          )
        })
      )}
    </>
  )
}

function SkillPicker({
  skills,
  value,
  onChange,
}: {
  skills: StudioSkill[]
  value: string[]
  onChange: (ids: string[]) => void
}) {
  const selected = new Set(value)
  const enabledSkills = skills.filter((skill) => skill.enabled)
  const toggle = (id: string) =>
    onChange(
      selected.has(id)
        ? value.filter((selectedID) => selectedID !== id)
        : [...value, id]
    )
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant='ghost' size='sm' className='rounded-lg'>
          <Sparkles />
          {value.length === 0 ? 'Skills' : `Skills · ${value.length}`}
          <ChevronDown className='size-3.5' />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align='start' className='w-72'>
        <DropdownMenuLabel>本轮使用的 Skills</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {enabledSkills.length === 0 ? (
          <DropdownMenuItem disabled>没有已启用的 Skill</DropdownMenuItem>
        ) : null}
        {enabledSkills.map((skill) => (
          <DropdownMenuCheckboxItem
            key={skill.id}
            checked={selected.has(skill.id)}
            onSelect={(event) => event.preventDefault()}
            onCheckedChange={() => toggle(skill.id)}
          >
            <span className='min-w-0 flex-1'>
              <span className='block truncate'>{skill.name}</span>
              <span className='block truncate text-xs text-muted-foreground'>
                {skill.description}
              </span>
            </span>
          </DropdownMenuCheckboxItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

function StudioWelcome() {
  const aui = useAui()
  return (
    <div className='flex flex-1 flex-col items-center justify-center py-16 text-center'>
      <span className='mb-5 flex size-12 items-center justify-center rounded-2xl bg-primary text-primary-foreground'>
        <Bot className='size-6' />
      </span>
      <h2 className='text-xl font-semibold tracking-tight'>从一个想法开始</h2>
      <p className='mt-2 max-w-md text-sm leading-6 text-muted-foreground'>
        和 Agent
        一起构思内容、生成资产，或让它调用现有工作流。每一项产出都会沉淀在当前
        Session 的资产路线中。
      </p>
      <div className='mt-6 grid w-full max-w-xl gap-2 sm:grid-cols-2'>
        {['为雨夜侦探构思漫画并生成分镜', '根据一张角色图生成三视图'].map(
          (prompt) => (
            <button
              key={prompt}
              type='button'
              onClick={() => {
                const composer = aui.thread.composer()
                composer.setText(prompt)
                composer.send()
              }}
              className='min-h-11 rounded-xl border bg-card px-4 py-3 text-left text-sm transition-colors hover:bg-accent'
            >
              {prompt}
            </button>
          )
        )}
      </div>
    </div>
  )
}

function StudioUserMessage() {
  return (
    <MessagePrimitive.Root className='mb-6 flex justify-end gap-3'>
      <div className='max-w-[82%] rounded-2xl rounded-tr-md bg-primary px-4 py-3 text-sm leading-6 text-primary-foreground'>
        <MessagePrimitive.Parts />
      </div>
      <Avatar className='mt-1 size-8'>
        <AvatarFallback>
          <User className='size-4' />
        </AvatarFallback>
      </Avatar>
    </MessagePrimitive.Root>
  )
}

function StudioAssistantMessage() {
  return (
    <MessagePrimitive.Root className='mb-7 flex gap-3'>
      <Avatar className='mt-1 size-8 border'>
        <AvatarFallback>
          <Bot className='size-4' />
        </AvatarFallback>
      </Avatar>
      <div className='max-w-[88%] min-w-0 pt-1 text-sm leading-7'>
        <MessagePrimitive.Parts />
      </div>
    </MessagePrimitive.Root>
  )
}

function ModelPicker({
  models,
  value,
  onChange,
}: {
  models: StudioModel[]
  value?: string
  onChange: (id: string) => void
}) {
  const selected = models.find((model) => model.id === value)
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant='ghost' size='sm' className='max-w-52 rounded-lg'>
          <span className='truncate'>{selected?.name ?? 'Mock Agent'}</span>
          <ChevronDown className='size-3.5' />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align='start' className='w-64'>
        <DropdownMenuLabel>本次对话使用的模型</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {models.length === 0 ? (
          <DropdownMenuItem disabled>
            未配置在线模型，将使用 Mock Agent
          </DropdownMenuItem>
        ) : (
          <DropdownMenuRadioGroup value={value} onValueChange={onChange}>
            {models
              .filter((model) => model.enabled && model.agent_enabled)
              .map((model) => (
                <DropdownMenuRadioItem key={model.id} value={model.id}>
                  <span className='min-w-0 flex-1 truncate'>{model.name}</span>
                  {model.default ? <Check className='size-3.5' /> : null}
                </DropdownMenuRadioItem>
              ))}
          </DropdownMenuRadioGroup>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

const permissionLabels: Record<StudioPermissionMode, string> = {
  request_approval: '请求批准',
  auto_approve: '帮我批准',
  full_access: '完全访问',
}

function PermissionPicker({
  value,
  onChange,
}: {
  value: StudioPermissionMode
  onChange: (mode: StudioPermissionMode) => void
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant='ghost' size='sm' className='rounded-lg'>
          <ShieldCheck />
          {permissionLabels[value]}
          <ChevronDown className='size-3.5' />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align='start' className='w-72'>
        <DropdownMenuLabel>Agent 操作权限</DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuRadioGroup
          value={value}
          onValueChange={(next) => onChange(next as StudioPermissionMode)}
        >
          <DropdownMenuRadioItem value='request_approval'>
            请求批准 · 每次执行前确认
          </DropdownMenuRadioItem>
          <DropdownMenuRadioItem value='auto_approve'>
            帮我批准 · 仅高风险操作确认
          </DropdownMenuRadioItem>
          <DropdownMenuRadioItem value='full_access'>
            完全访问 · 自动执行所有操作
          </DropdownMenuRadioItem>
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
