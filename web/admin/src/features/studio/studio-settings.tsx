import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Bot,
  BrainCircuit,
  Cable,
  CheckCircle2,
  CircleAlert,
  Plus,
  Sparkles,
  Workflow,
} from 'lucide-react'
import {
  createStudioModel,
  createStudioConnector,
  createStudioSkill,
  listStudioConnectors,
  listStudioAgentWorkflows,
  listStudioModels,
  listStudioSkills,
  testStudioModelConnection,
  type StudioConnectorPolicy,
  type StudioMCPConnector,
  type StudioAgentWorkflow,
  type StudioModel,
  type StudioModelConnectionTest,
  updateStudioAgentWorkflow,
  updateStudioConnector,
  updateStudioSkill,
} from '@/lib/api/studio'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'

type SettingSection = 'models' | 'skills' | 'mcp' | 'workflows'

const settingSections: Array<{
  id: SettingSection
  label: string
  description: string
  icon: typeof BrainCircuit
}> = [
  { id: 'models', label: '模型', description: '连接与思考配置', icon: BrainCircuit },
  { id: 'skills', label: 'Skills', description: '可选的专业知识', icon: Sparkles },
  { id: 'mcp', label: 'MCP 连接器', description: '外部工具与服务', icon: Cable },
  { id: 'workflows', label: '工作流', description: 'Agent 可调用范围', icon: Workflow },
]

export function StudioSettings() {
  const [section, setSection] = useState<SettingSection>('models')
  const [modelDialogOpen, setModelDialogOpen] = useState(false)
  const [skillDialogOpen, setSkillDialogOpen] = useState(false)
  const [connectorDialogOpen, setConnectorDialogOpen] = useState(false)
  const [modelTest, setModelTest] = useState<{
    modelId: string
    result?: StudioModelConnectionTest
    error?: string
  }>()
  const models = useQuery({
    queryKey: ['studio', 'models'],
    queryFn: listStudioModels,
  })
  const skills = useQuery({
    queryKey: ['studio', 'skills'],
    queryFn: listStudioSkills,
  })
  const connectors = useQuery({
    queryKey: ['studio', 'connectors'],
    queryFn: listStudioConnectors,
  })
  const workflows = useQuery({
    queryKey: ['studio', 'workflows'],
    queryFn: listStudioAgentWorkflows,
  })
  const testModel = useMutation({
    mutationFn: testStudioModelConnection,
    onSuccess: (result, modelId) => setModelTest({ modelId, result }),
    onError: (cause, modelId) =>
      setModelTest({
        modelId,
        error:
          cause instanceof Error ? cause.message : '模型连接失败，请检查配置。',
      }),
  })
  return (
    <main
      id='main-content'
      className='min-h-0 min-w-0 flex-1 bg-muted/30 p-3 sm:p-4'
    >
      <section className='flex h-full min-h-0 flex-col overflow-hidden rounded-2xl border bg-card'>
        <header className='flex min-h-16 items-center gap-3 border-b px-5'>
          <div>
            <h1 className='text-sm font-semibold'>AI 设置</h1>
            <p className='text-xs text-muted-foreground'>配置 Agent 可以使用的模型与能力</p>
          </div>
        </header>
        <div className='flex min-h-0 flex-1 flex-col md:flex-row'>
          <div className='flex gap-1 overflow-x-auto border-b p-2 md:hidden'>
            {settingSections.map((item) => {
              const Icon = item.icon
              return (
                <button
                  key={item.id}
                  type='button'
                  onClick={() => setSection(item.id)}
                  className={`flex min-h-10 shrink-0 items-center gap-2 rounded-md border px-3 text-sm ${section === item.id ? 'border-border bg-muted text-foreground' : 'border-transparent text-muted-foreground'}`}
                >
                  <Icon className='size-4' />
                  {item.label}
                </button>
              )
            })}
          </div>
          <nav className='hidden w-56 shrink-0 border-e bg-muted/20 p-3 md:block' aria-label='AI 设置分类'>
            <p className='px-2 pb-2 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground'>Agent 能力</p>
            <div className='space-y-1'>
              {settingSections.map((item) => {
                const Icon = item.icon
                const active = section === item.id
                return (
                  <button
                    key={item.id}
                    type='button'
                    onClick={() => setSection(item.id)}
                    className={`flex w-full items-start gap-3 rounded-lg border px-3 py-2.5 text-left transition-colors ${active ? 'border-border bg-background text-foreground' : 'border-transparent text-muted-foreground hover:border-border/70 hover:bg-background/70 hover:text-foreground'}`}
                  >
                    <Icon className='mt-0.5 size-4 shrink-0' />
                    <span className='min-w-0'>
                      <span className='block text-sm font-medium'>{item.label}</span>
                      <span className='mt-0.5 block truncate text-xs'>{item.description}</span>
                    </span>
                  </button>
                )
              })}
            </div>
          </nav>
          <div className='flex min-w-0 flex-1 flex-col'>
            <div className='flex min-h-16 items-center justify-between gap-3 border-b px-5'>
              <div>
                <h2 className='text-sm font-semibold'>{settingSections.find((item) => item.id === section)?.label}</h2>
                <p className='text-xs text-muted-foreground'>{settingSections.find((item) => item.id === section)?.description}</p>
              </div>
              {section === 'models' ? <Button size='sm' onClick={() => setModelDialogOpen(true)}><Plus />添加模型</Button> : null}
              {section === 'skills' ? <Button size='sm' onClick={() => setSkillDialogOpen(true)}><Plus />添加 Skill</Button> : null}
              {section === 'mcp' ? <Button size='sm' onClick={() => setConnectorDialogOpen(true)}><Plus />添加连接器</Button> : null}
            </div>
            <ScrollArea className='min-h-0 flex-1'>
              {section === 'models' ? <div className='p-5'>
                <div className='mx-auto max-w-5xl space-y-4'>
              <div>
                <h2 className='text-sm font-semibold'>模型配置</h2>
                <p className='mt-1 text-sm text-muted-foreground'>
                  支持 OpenAI Responses、OpenAI Chat Compatible 与 Anthropic
                  Messages Compatible。
                </p>
              </div>
              {models.isLoading ? (
                <p className='text-sm text-muted-foreground'>正在读取模型…</p>
              ) : null}
              {models.data?.length ? (
                <div className='overflow-hidden rounded-xl border bg-background'>
                  {models.data.map((model) => (
                    <div key={model.id} className='flex flex-wrap items-center gap-4 border-b p-4 last:border-b-0'>
                      <span className='flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted'>
                        <Bot className='size-4' />
                      </span>
                      <div className='min-w-0 flex-1'>
                        <div className='flex flex-wrap items-center gap-2'>
                          <p className='text-sm font-medium'>{model.name}</p>
                          {model.default ? <Badge>默认</Badge> : null}
                          <Badge variant={model.agent_enabled ? 'secondary' : 'outline'}>{model.agent_enabled ? 'Agent 可用' : 'Agent 不可用'}</Badge>
                        </div>
                        <p className='mt-1 truncate text-xs text-muted-foreground'>{model.model} · {model.base_url}</p>
                      </div>
                      <div className='flex shrink-0 items-center gap-2 text-xs text-muted-foreground'>
                        {model.enabled && model.agent_enabled && model.has_api_key ? (
                          <CheckCircle2 className='size-3.5 text-success' />
                        ) : (
                          <CircleAlert className='size-3.5 text-warning' />
                        )}
                        {model.enabled ? '已启用' : '已停用'} · {model.agent_enabled ? 'Agent 可用' : 'Agent 不可用'}
                      </div>
                      <div className='flex items-center gap-2'>
                        {modelTest?.modelId === model.id ? (
                          modelTest.result ? (
                            <span className='flex items-center gap-1 text-xs text-success'>
                              <CheckCircle2 className='size-3.5' />
                              连接成功 · {modelTest.result.latency_ms} ms
                            </span>
                          ) : modelTest.error ? (
                            <span className='flex max-w-56 items-center gap-1 text-xs text-warning'>
                              <CircleAlert className='size-3.5 shrink-0' />
                              <span className='truncate'>{modelTest.error}</span>
                            </span>
                          ) : null
                        ) : null}
                        <Button
                          variant='outline'
                          size='sm'
                          disabled={testModel.isPending}
                          onClick={() => {
                            setModelTest({ modelId: model.id })
                            testModel.mutate(model.id)
                          }}
                        >
                          {testModel.isPending && testModel.variables === model.id
                            ? '正在检测…'
                            : '测试连接'}
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              ) : !models.isLoading ? (
                <EmptySetting
                  icon={BrainCircuit}
                  title='还没有在线模型'
                  description='添加一个模型后，用户可以直接在输入框中选择。未配置时仍可使用内置 Mock Agent 验证完整创作流程。'
                />
              ) : null}
                </div>
              </div> : null}
              {section === 'skills' ? <div className='p-5'><SkillSettings skills={skills.data ?? []} loading={skills.isLoading} error={skills.isError} /></div> : null}
              {section === 'mcp' ? <div className='p-5'><ConnectorSettings connectors={connectors.data ?? []} loading={connectors.isLoading} error={connectors.isError} /></div> : null}
              {section === 'workflows' ? <div className='p-5'><WorkflowSettings workflows={workflows.data ?? []} loading={workflows.isLoading} error={workflows.isError} /></div> : null}
            </ScrollArea>
          </div>
        </div>
        <ModelDialog open={modelDialogOpen} onOpenChange={setModelDialogOpen} />
        <SkillDialog open={skillDialogOpen} onOpenChange={setSkillDialogOpen} />
        <ConnectorDialog open={connectorDialogOpen} onOpenChange={setConnectorDialogOpen} />
      </section>
    </main>
  )
}

function WorkflowSettings({
  workflows,
  loading,
  error,
}: {
  workflows: StudioAgentWorkflow[]
  loading: boolean
  error: boolean
}) {
  const queryClient = useQueryClient()
  const update = useMutation({
    mutationFn: ({ id, agentEnabled }: { id: string; agentEnabled: boolean }) =>
      updateStudioAgentWorkflow(id, agentEnabled),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ['studio', 'workflows'] }),
  })
  return (
    <div className='mx-auto max-w-5xl space-y-5'>
      <div>
        <h2 className='text-sm font-semibold'>Agent 可用工作流</h2>
        <p className='mt-1 text-sm text-muted-foreground'>
          这里仅决定当前 Studio Agent
          是否可调用已有工作流，不影响工作流在平台其他入口的启停。
        </p>
      </div>
      {loading ? (
        <p className='text-sm text-muted-foreground'>正在读取工作流…</p>
      ) : null}
      {error ? (
        <p role='alert' className='text-sm text-destructive'>
          工作流读取失败，刷新后重试。
        </p>
      ) : null}
      {!loading && !error && workflows.length === 0 ? (
        <p className='py-12 text-sm text-muted-foreground'>
          还没有可配置的工作流。请先在平台工作流中创建。
        </p>
      ) : null}
      {workflows.length > 0 ? (
        <div className='space-y-2'>
          {workflows.map((workflow) => (
            <Card key={workflow.id}>
              <CardContent className='flex items-center justify-between gap-4 p-4'>
                <div className='min-w-0'>
                  <div className='flex items-center gap-2'>
                    <p className='truncate text-sm font-medium'>
                      {workflow.name}
                    </p>
                    <Badge
                      variant={
                        workflow.workflow_enabled ? 'secondary' : 'outline'
                      }
                    >
                      {workflow.workflow_enabled
                        ? '工作流已启用'
                        : '工作流已停用'}
                    </Badge>
                  </div>
                  <p className='mt-1 line-clamp-1 text-xs text-muted-foreground'>
                    {workflow.description || '未填写说明'} · {workflow.inputs}{' '}
                    个输入 / {workflow.outputs} 个输出
                  </p>
                </div>
                <Switch
                  aria-label={`允许 Agent 调用 ${workflow.name}`}
                  checked={workflow.agent_enabled}
                  disabled={!workflow.workflow_enabled || update.isPending}
                  onCheckedChange={(agentEnabled) =>
                    update.mutate({ id: workflow.id, agentEnabled })
                  }
                />
              </CardContent>
            </Card>
          ))}
        </div>
      ) : null}
    </div>
  )
}

function ConnectorSettings({
  connectors,
  loading,
  error,
}: {
  connectors: StudioMCPConnector[]
  loading: boolean
  error: boolean
}) {
  const queryClient = useQueryClient()
  const update = useMutation({
    mutationFn: updateStudioConnector,
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ['studio', 'connectors'] }),
  })
  return (
    <div className='mx-auto max-w-5xl space-y-5'>
      <div>
        <h2 className='text-sm font-semibold'>MCP 连接器</h2>
        <p className='mt-1 text-sm text-muted-foreground'>
          用 Streamable HTTP 接入外部工具。凭据仅加密保存在服务端。
        </p>
      </div>
      {loading ? (
        <p className='text-sm text-muted-foreground'>正在读取连接器…</p>
      ) : null}
      {error ? (
        <p role='alert' className='text-sm text-destructive'>
          连接器读取失败，刷新后重试。
        </p>
      ) : null}
      {!loading && !error && connectors.length === 0 ? (
        <p className='py-12 text-sm text-muted-foreground'>
          还没有 MCP 连接器。添加后可集中管理其可用性和调用策略。
        </p>
      ) : null}
      {connectors.length > 0 ? (
        <div className='overflow-hidden rounded-xl border bg-background'>
          {connectors.map((connector) => (
            <div key={connector.id} className='flex flex-wrap items-center gap-4 border-b p-4 last:border-b-0'>
              <span className='flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted'>
                <Cable className='size-4' />
              </span>
              <div className='min-w-0 flex-1'>
                <p className='text-sm font-medium'>{connector.name}</p>
                <p className='mt-1 truncate text-xs text-muted-foreground'>{connector.url}</p>
                <p className='mt-1 text-xs text-muted-foreground'>调用策略：{connectorPolicyLabel(connector.policy)} · 凭据：{connector.credential_masked} · 已发现工具：{connector.tools.length}</p>
              </div>
              <Switch aria-label={`启用 ${connector.name}`} checked={connector.enabled} disabled={update.isPending} onCheckedChange={(enabled) => update.mutate({ id: connector.id, name: connector.name, url: connector.url, enabled, policy: connector.policy })} />
            </div>
          ))}
        </div>
      ) : null}
    </div>
  )
}

function connectorPolicyLabel(policy: StudioConnectorPolicy) {
  switch (policy) {
    case 'auto':
      return '自动执行'
    case 'approval':
      return '请求确认'
    case 'forbidden':
      return '禁止调用'
  }
}

function ConnectorDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const queryClient = useQueryClient()
  const [form, setForm] = useState({
    name: '',
    url: '',
    credential: '',
    enabled: true,
    policy: 'approval' as StudioConnectorPolicy,
  })
  const [error, setError] = useState('')
  const create = useMutation({
    mutationFn: () =>
      createStudioConnector({
        ...form,
        name: form.name.trim(),
        url: form.url.trim(),
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ['studio', 'connectors'],
      })
      setForm({
        name: '',
        url: '',
        credential: '',
        enabled: true,
        policy: 'approval',
      })
      setError('')
      onOpenChange(false)
    },
    onError: (cause) =>
      setError(
        cause instanceof Error ? cause.message : '添加连接器失败，请稍后重试。'
      ),
  })
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-xl'>
        <DialogHeader>
          <DialogTitle>添加 MCP 连接器</DialogTitle>
          <DialogDescription>
            使用 Streamable HTTP 地址。凭据创建后不会再次显示。
          </DialogDescription>
        </DialogHeader>
        <form
          className='grid gap-4'
          onSubmit={(event) => {
            event.preventDefault()
            setError('')
            create.mutate()
          }}
        >
          <div className='grid gap-2'>
            <Label htmlFor='studio-connector-name'>名称</Label>
            <Input
              id='studio-connector-name'
              required
              value={form.name}
              onChange={(event) =>
                setForm({ ...form, name: event.target.value })
              }
              placeholder='例如：内部知识库'
            />
          </div>
          <div className='grid gap-2'>
            <Label htmlFor='studio-connector-url'>服务地址</Label>
            <Input
              id='studio-connector-url'
              required
              type='url'
              value={form.url}
              onChange={(event) =>
                setForm({ ...form, url: event.target.value })
              }
              placeholder='https://mcp.example.com'
            />
          </div>
          <div className='grid gap-2'>
            <Label htmlFor='studio-connector-credential'>访问凭据</Label>
            <Input
              id='studio-connector-credential'
              required
              type='password'
              autoComplete='new-password'
              value={form.credential}
              onChange={(event) =>
                setForm({ ...form, credential: event.target.value })
              }
            />
          </div>
          <div className='grid gap-2'>
            <Label>调用策略</Label>
            <Select
              value={form.policy}
              onValueChange={(policy: StudioConnectorPolicy) =>
                setForm({ ...form, policy })
              }
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='approval'>请求确认</SelectItem>
                <SelectItem value='auto'>自动执行</SelectItem>
                <SelectItem value='forbidden'>禁止调用</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <ToggleRow
            label='启用连接器'
            description='关闭后 Agent 不会调用此连接器。'
            checked={form.enabled}
            onCheckedChange={(enabled) => setForm({ ...form, enabled })}
          />
          {error ? (
            <p role='alert' className='text-sm text-destructive'>
              {error}
            </p>
          ) : null}
          <DialogFooter>
            <Button
              type='button'
              variant='outline'
              onClick={() => onOpenChange(false)}
            >
              取消
            </Button>
            <Button type='submit' disabled={create.isPending}>
              {create.isPending ? '正在保存…' : '保存连接器'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function SkillSettings({
  skills,
  loading,
  error,
}: {
  skills: Awaited<ReturnType<typeof listStudioSkills>>
  loading: boolean
  error: boolean
}) {
  const queryClient = useQueryClient()
  const update = useMutation({
    mutationFn: updateStudioSkill,
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ['studio', 'skills'] }),
  })
  return (
    <div className='mx-auto max-w-5xl space-y-5'>
      <div>
        <h2 className='text-sm font-semibold'>Skills</h2>
        <p className='mt-1 text-sm text-muted-foreground'>
          为 Agent 添加本轮可选的专门知识与工作方式。
        </p>
      </div>
      {loading ? (
        <p className='text-sm text-muted-foreground'>正在读取 Skills…</p>
      ) : null}
      {error ? (
        <p role='alert' className='text-sm text-destructive'>
          Skills 读取失败，刷新后重试。
        </p>
      ) : null}
      {!loading && !error && skills.length === 0 ? (
        <p className='py-12 text-sm text-muted-foreground'>
          还没有 Skill。添加后可在聊天输入框中选择。
        </p>
      ) : null}
      {skills.length > 0 ? (
        <div className='overflow-hidden rounded-xl border bg-background'>
          {skills.map((skill) => (
            <div key={skill.id} className='flex flex-wrap items-start gap-4 border-b p-4 last:border-b-0'>
              <span className='flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted'>
                <Sparkles className='size-4' />
              </span>
              <div className='min-w-0 flex-1'>
                <p className='text-sm font-medium'>{skill.name}</p>
                <p className='mt-1 text-xs text-muted-foreground'>{skill.description || '未填写说明'}</p>
                <p className='mt-2 line-clamp-2 text-xs leading-5 text-muted-foreground'>{skill.prompt}</p>
              </div>
              <Switch aria-label={`启用 ${skill.name}`} checked={skill.enabled} disabled={update.isPending} onCheckedChange={(enabled) => update.mutate({ ...skill, enabled })} />
            </div>
          ))}
        </div>
      ) : null}
    </div>
  )
}

function SkillDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const queryClient = useQueryClient()
  const [form, setForm] = useState({
    name: '',
    description: '',
    prompt: '',
    enabled: true,
  })
  const [error, setError] = useState('')
  const create = useMutation({
    mutationFn: () =>
      createStudioSkill({
        ...form,
        name: form.name.trim(),
        description: form.description.trim(),
        prompt: form.prompt.trim(),
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['studio', 'skills'] })
      setForm({ name: '', description: '', prompt: '', enabled: true })
      setError('')
      onOpenChange(false)
    },
    onError: (cause) =>
      setError(
        cause instanceof Error ? cause.message : '添加 Skill 失败，请稍后重试。'
      ),
  })
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-xl'>
        <DialogHeader>
          <DialogTitle>添加 Skill</DialogTitle>
          <DialogDescription>
            启用后，创作输入框可以按需选择它。
          </DialogDescription>
        </DialogHeader>
        <form
          className='grid gap-4'
          onSubmit={(event) => {
            event.preventDefault()
            setError('')
            create.mutate()
          }}
        >
          <div className='grid gap-2'>
            <Label htmlFor='studio-skill-name'>名称</Label>
            <Input
              id='studio-skill-name'
              required
              value={form.name}
              onChange={(event) =>
                setForm({ ...form, name: event.target.value })
              }
              placeholder='例如：漫画分镜'
            />
          </div>
          <div className='grid gap-2'>
            <Label htmlFor='studio-skill-description'>说明</Label>
            <Input
              id='studio-skill-description'
              value={form.description}
              onChange={(event) =>
                setForm({ ...form, description: event.target.value })
              }
              placeholder='说明 Agent 何时该用它'
            />
          </div>
          <div className='grid gap-2'>
            <Label htmlFor='studio-skill-prompt'>Skill 内容</Label>
            <Textarea
              id='studio-skill-prompt'
              required
              value={form.prompt}
              onChange={(event) =>
                setForm({ ...form, prompt: event.target.value })
              }
              placeholder='写入专业约束、步骤或输出格式'
            />
          </div>
          <ToggleRow
            label='启用 Skill'
            description='关闭后不会出现在创作输入框中。'
            checked={form.enabled}
            onCheckedChange={(enabled) => setForm({ ...form, enabled })}
          />
          {error ? (
            <p role='alert' className='text-sm text-destructive'>
              {error}
            </p>
          ) : null}
          <DialogFooter>
            <Button
              type='button'
              variant='outline'
              onClick={() => onOpenChange(false)}
            >
              取消
            </Button>
            <Button type='submit' disabled={create.isPending}>
              {create.isPending ? '正在保存…' : '保存 Skill'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

const initialModel = {
  name: '',
  protocol: 'openai_chat_compatible' as StudioModel['protocol'],
  baseUrl: '',
  model: '',
  apiKey: '',
  enabled: true,
  agentEnabled: true,
  default: false,
  thinkingEnabled: false,
  thinkingEffort: 'medium',
  thinkingBudget: '2048',
}

function ModelDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const queryClient = useQueryClient()
  const [form, setForm] = useState(initialModel)
  const [error, setError] = useState('')
  const create = useMutation({
    mutationFn: () =>
      createStudioModel({
        name: form.name.trim(),
        protocol: form.protocol,
        baseUrl: form.baseUrl.trim(),
        model: form.model.trim(),
        apiKey: form.apiKey,
        enabled: form.enabled,
        agentEnabled: form.agentEnabled,
        default: form.default,
        thinking: form.thinkingEnabled
          ? {
              enabled: true,
              effort: form.thinkingEffort,
              budget_tokens: Number(form.thinkingBudget) || undefined,
            }
          : { enabled: false },
        capabilities: {
          tools: true,
          vision: false,
          image_output: false,
          streaming: true,
        },
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['studio', 'models'] })
      setForm(initialModel)
      setError('')
      onOpenChange(false)
    },
    onError: (cause) =>
      setError(
        cause instanceof Error ? cause.message : '添加模型失败，请稍后重试。'
      ),
  })
  const submit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setError('')
    create.mutate()
  }
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-h-[calc(100vh-2rem)] overflow-y-auto sm:max-w-xl'>
        <DialogHeader>
          <DialogTitle>添加模型</DialogTitle>
          <DialogDescription>
            密钥仅加密保存于服务端，创建后不会再次展示。
          </DialogDescription>
        </DialogHeader>
        <form className='grid gap-4' onSubmit={submit}>
          <div className='grid gap-2'>
            <Label htmlFor='studio-model-name'>名称</Label>
            <Input
              id='studio-model-name'
              required
              value={form.name}
              onChange={(event) =>
                setForm({ ...form, name: event.target.value })
              }
              placeholder='例如：我的 Claude'
            />
          </div>
          <div className='grid gap-2'>
            <Label>接口协议</Label>
            <Select
              value={form.protocol}
              onValueChange={(protocol: StudioModel['protocol']) =>
                setForm({ ...form, protocol })
              }
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='openai_chat_compatible'>
                  OpenAI Chat Compatible
                </SelectItem>
                <SelectItem value='openai_responses'>
                  OpenAI Responses
                </SelectItem>
                <SelectItem value='anthropic_messages_compatible'>
                  Anthropic Messages Compatible
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className='grid gap-2'>
            <Label htmlFor='studio-model-url'>Base URL</Label>
            <Input
              id='studio-model-url'
              required
              type='url'
              value={form.baseUrl}
              onChange={(event) =>
                setForm({ ...form, baseUrl: event.target.value })
              }
              placeholder='https://api.example.com/v1'
            />
          </div>
          <div className='grid gap-2'>
            <Label htmlFor='studio-model-id'>模型 ID</Label>
            <Input
              id='studio-model-id'
              required
              value={form.model}
              onChange={(event) =>
                setForm({ ...form, model: event.target.value })
              }
              placeholder='例如：claude-sonnet-4-5'
            />
          </div>
          <div className='grid gap-2'>
            <Label htmlFor='studio-model-key'>API Key</Label>
            <Input
              id='studio-model-key'
              required
              type='password'
              autoComplete='new-password'
              value={form.apiKey}
              onChange={(event) =>
                setForm({ ...form, apiKey: event.target.value })
              }
            />
          </div>
          <div className='grid gap-3 rounded-lg border bg-muted/30 p-3'>
            <ToggleRow
              label='允许 Agent 使用'
              description='关闭后不会出现在创作输入框中。'
              checked={form.agentEnabled}
              onCheckedChange={(agentEnabled) =>
                setForm({ ...form, agentEnabled })
              }
            />
            <ToggleRow
              label='设为默认模型'
              description='新建会话优先选择此模型。'
              checked={form.default}
              onCheckedChange={(value) => setForm({ ...form, default: value })}
            />
            <ToggleRow
              label='启用思考'
              description='按所选协议传递推理参数。'
              checked={form.thinkingEnabled}
              onCheckedChange={(thinkingEnabled) =>
                setForm({ ...form, thinkingEnabled })
              }
            />
            {form.thinkingEnabled ? (
              <div className='grid grid-cols-2 gap-3 pt-1'>
                <div className='grid gap-2'>
                  <Label htmlFor='studio-model-effort'>思考强度</Label>
                  <Select
                    value={form.thinkingEffort}
                    onValueChange={(thinkingEffort) =>
                      setForm({ ...form, thinkingEffort })
                    }
                  >
                    <SelectTrigger id='studio-model-effort'>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value='low'>低</SelectItem>
                      <SelectItem value='medium'>中</SelectItem>
                      <SelectItem value='high'>高</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className='grid gap-2'>
                  <Label htmlFor='studio-model-budget'>Token 预算</Label>
                  <Input
                    id='studio-model-budget'
                    inputMode='numeric'
                    value={form.thinkingBudget}
                    onChange={(event) =>
                      setForm({ ...form, thinkingBudget: event.target.value })
                    }
                  />
                </div>
              </div>
            ) : null}
          </div>
          {error ? (
            <p role='alert' className='text-sm text-destructive'>
              {error}
            </p>
          ) : null}
          <DialogFooter>
            <Button
              type='button'
              variant='outline'
              onClick={() => onOpenChange(false)}
            >
              取消
            </Button>
            <Button type='submit' disabled={create.isPending}>
              {create.isPending ? '正在保存…' : '保存模型'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function ToggleRow({
  label,
  description,
  checked,
  onCheckedChange,
}: {
  label: string
  description: string
  checked: boolean
  onCheckedChange: (checked: boolean) => void
}) {
  return (
    <div className='flex items-center justify-between gap-3'>
      <div>
        <p className='text-sm font-medium'>{label}</p>
        <p className='text-xs text-muted-foreground'>{description}</p>
      </div>
      <Switch checked={checked} onCheckedChange={onCheckedChange} />
    </div>
  )
}

function EmptySetting({
  icon: Icon,
  title,
  description,
}: {
  icon: typeof Bot
  title: string
  description: string
}) {
  return (
    <div className='mx-auto flex min-h-80 max-w-xl flex-col items-center justify-center rounded-xl border border-dashed p-8 text-center'>
      <span className='mb-4 flex size-11 items-center justify-center rounded-xl bg-muted'>
        <Icon className='size-5 text-muted-foreground' />
      </span>
      <h2 className='text-sm font-medium'>{title}</h2>
      <p className='mt-2 text-sm leading-6 text-muted-foreground'>
        {description}
      </p>
    </div>
  )
}
