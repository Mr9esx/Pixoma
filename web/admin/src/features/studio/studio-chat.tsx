import { useMemo } from 'react'
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
  Send,
  ShieldCheck,
  Sparkles,
  User,
} from 'lucide-react'
import { baseURL, sessionToken } from '@/lib/api/client'
import type {
  StudioMessage,
  StudioModel,
  StudioPermissionMode,
  StudioSkill,
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
  selectedSkillIds: string[]
  onModelChange: (id: string) => void
  onPermissionChange: (mode: StudioPermissionMode) => void
  onSkillChange: (ids: string[]) => void
}

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
