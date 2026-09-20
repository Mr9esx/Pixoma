import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Bot, BrainCircuit, Cable, CheckCircle2, Plus, Sparkles, Workflow } from 'lucide-react'
import { listStudioModels } from '@/lib/api/studio'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

export function StudioSettings() {
  const [tab, setTab] = useState('models')
  const models = useQuery({ queryKey: ['studio', 'models'], queryFn: listStudioModels })
  return (
    <main id='main-content' className='flex min-w-0 flex-1 flex-col bg-background'>
      <header className='flex min-h-16 items-center justify-between gap-3 border-b px-5 py-3'>
        <div>
          <h1 className='text-base font-semibold'>AI 设置</h1>
          <p className='text-xs text-muted-foreground'>配置 Agent 可以使用的模型与能力</p>
        </div>
        {tab === 'models' ? (
          <Button size='sm'>
            <Plus />
            添加模型
          </Button>
        ) : null}
      </header>
      <Tabs value={tab} onValueChange={setTab} className='min-h-0 flex-1 gap-0'>
        <div className='border-b px-5 py-3'>
          <TabsList>
            <TabsTrigger value='models'><BrainCircuit />模型</TabsTrigger>
            <TabsTrigger value='skills'><Sparkles />Skills</TabsTrigger>
            <TabsTrigger value='mcp'><Cable />MCP</TabsTrigger>
            <TabsTrigger value='workflows'><Workflow />工作流</TabsTrigger>
          </TabsList>
        </div>
        <ScrollArea className='min-h-0 flex-1'>
          <TabsContent value='models' className='m-0 p-5'>
            <div className='mx-auto max-w-5xl space-y-4'>
              <div>
                <h2 className='text-sm font-semibold'>模型配置</h2>
                <p className='mt-1 text-sm text-muted-foreground'>
                  支持 OpenAI Responses、OpenAI Chat Compatible 与 Anthropic Messages Compatible。
                </p>
              </div>
              {models.isLoading ? <p className='text-sm text-muted-foreground'>正在读取模型…</p> : null}
              {models.data?.length ? (
                <div className='grid gap-3 md:grid-cols-2'>
                  {models.data.map((model) => (
                    <Card key={model.id}>
                      <CardHeader className='pb-3'>
                        <div className='flex items-start justify-between gap-3'>
                          <span className='flex size-9 items-center justify-center rounded-lg bg-muted'><Bot className='size-4' /></span>
                          <div className='flex gap-1.5'>
                            {model.default ? <Badge>默认</Badge> : null}
                            <Badge variant={model.agent_enabled ? 'secondary' : 'outline'}>
                              {model.agent_enabled ? 'Agent 可用' : 'Agent 不可用'}
                            </Badge>
                          </div>
                        </div>
                        <CardTitle className='mt-3 text-base'>{model.name}</CardTitle>
                        <CardDescription className='truncate'>{model.model}</CardDescription>
                      </CardHeader>
                      <CardContent className='space-y-2 text-xs text-muted-foreground'>
                        <p className='truncate'>{model.base_url}</p>
                        <div className='flex items-center gap-2'>
                          <CheckCircle2 className='size-3.5 text-success' />
                          {model.enabled ? '已启用' : '已停用'} · 密钥 {model.api_key_masked ?? '未配置'}
                        </div>
                      </CardContent>
                    </Card>
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
          </TabsContent>
          <TabsContent value='skills' className='m-0 p-5'><EmptySetting icon={Sparkles} title='Skills' description='创建内联 Skill，让 Agent 在合适的任务中加载专门知识。本期不引入脚本和依赖图。' /></TabsContent>
          <TabsContent value='mcp' className='m-0 p-5'><EmptySetting icon={Cable} title='MCP 连接器' description='通过 Streamable HTTP 接入外部工具，凭据由系统统一管理。' /></TabsContent>
          <TabsContent value='workflows' className='m-0 p-5'><EmptySetting icon={Workflow} title='Agent 可用工作流' description='复用现有工作流名称、说明和输入输出定义，只需决定是否允许 Agent 调用。' /></TabsContent>
        </ScrollArea>
      </Tabs>
    </main>
  )
}

function EmptySetting({ icon: Icon, title, description }: { icon: typeof Bot; title: string; description: string }) {
  return (
    <div className='mx-auto flex min-h-80 max-w-xl flex-col items-center justify-center rounded-xl border border-dashed p-8 text-center'>
      <span className='mb-4 flex size-11 items-center justify-center rounded-xl bg-muted'><Icon className='size-5 text-muted-foreground' /></span>
      <h2 className='text-sm font-medium'>{title}</h2>
      <p className='mt-2 text-sm leading-6 text-muted-foreground'>{description}</p>
    </div>
  )
}
