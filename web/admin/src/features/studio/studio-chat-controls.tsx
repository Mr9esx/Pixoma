import { useState } from 'react'
import {
  Boxes,
  Cable,
  Check,
  ChevronDown,
  ShieldCheck,
  TriangleAlert,
  Workflow,
} from 'lucide-react'
import type { StudioModel, StudioPermissionMode } from '@/lib/api/studio'
import { cn } from '@/lib/utils'
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
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { PromptInputButton } from '@/components/ai-elements/prompt-input'

export function ModelPicker({
  models,
  value,
  onChange,
}: {
  models: StudioModel[]
  value?: string
  onChange: (id: string) => void
}) {
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const selected = models.find((model) => model.id === value)

  return (
    <Popover
      open={open}
      onOpenChange={(nextOpen) => {
        setOpen(nextOpen)
        if (!nextOpen) setSearch('')
      }}
    >
      <PopoverTrigger asChild>
        <PromptInputButton
          aria-label={`选择模型：${selected?.name ?? '未选择模型'}`}
          className='max-w-52 font-normal'
        >
          <span className='truncate'>{selected?.name ?? '未选择模型'}</span>
          <ChevronDown className='size-3.5' />
        </PromptInputButton>
      </PopoverTrigger>
      <PopoverContent align='end' className='group/model-picker w-72 p-1'>
        <Command className='h-auto [&_[data-slot=command-input-wrapper]]:border-0'>
          <div className='order-last group-data-[side=bottom]/model-picker:order-first'>
            <CommandInput
              aria-label='搜索模型'
              placeholder='搜索模型'
              value={search}
              onValueChange={setSearch}
            />
          </div>
          <CommandList className='max-h-60'>
            <CommandEmpty>
              {models.length === 0 ? '没有可用模型' : '没有匹配的模型'}
            </CommandEmpty>
            <CommandGroup>
              {models.map((model) => (
                <CommandItem
                  key={model.id}
                  value={`${model.name} ${model.model} ${model.id}`}
                  onSelect={() => {
                    onChange(model.id)
                    setOpen(false)
                    setSearch('')
                  }}
                >
                  <span className='min-w-0 flex-1 truncate'>{model.name}</span>
                  {model.id === value ? (
                    <Check className='ms-auto size-4 text-foreground' />
                  ) : null}
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
}

const permissionLabels: Record<StudioPermissionMode, string> = {
  request_approval: '请求批准',
  auto_approve: '帮我批准',
  full_access: '完全访问',
}

const permissionOptions: { mode: StudioPermissionMode; description: string }[] =
  [
    { mode: 'request_approval', description: '每次执行前确认' },
    { mode: 'auto_approve', description: '仅高风险操作确认' },
    { mode: 'full_access', description: '自动执行所有操作' },
  ]

export function PermissionPicker({
  value,
  onChange,
}: {
  value: StudioPermissionMode
  onChange: (mode: StudioPermissionMode) => void
}) {
  const [confirmOpen, setConfirmOpen] = useState(false)

  return (
    <>
      <DropdownMenu modal={false}>
        <DropdownMenuTrigger asChild>
          <PromptInputButton
            aria-label={`Agent 操作权限：${permissionLabels[value]}`}
            className={cn(
              'font-normal',
              value === 'full_access' &&
                'text-warning-text hover:text-warning-text'
            )}
          >
            <ShieldCheck />
            {permissionLabels[value]}
            <ChevronDown className='size-3.5' />
          </PromptInputButton>
        </DropdownMenuTrigger>
        <DropdownMenuContent align='start' className='w-72'>
          {permissionOptions.map(({ mode, description }) => (
            <DropdownMenuItem
              key={mode}
              className={cn(
                'justify-between',
                mode === 'full_access' &&
                  'text-warning-text focus:text-warning-text'
              )}
              onSelect={() => {
                if (mode === value) return
                if (mode === 'full_access') {
                  setConfirmOpen(true)
                } else {
                  onChange(mode)
                }
              }}
            >
              <span className='min-w-0 truncate'>
                {permissionLabels[mode]} · {description}
              </span>
              {mode === value ? (
                <Check
                  className={cn(
                    'ms-auto size-4 text-foreground',
                    mode === 'full_access' && 'text-warning-text'
                  )}
                />
              ) : null}
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className='flex items-center gap-2'>
              <TriangleAlert className='size-5 text-warning-text' />
              要开启完全访问权限吗？
            </AlertDialogTitle>
            <AlertDialogDescription>
              开启后，Agent 可以在当前对话中直接执行已启用的操作，包括：
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className='space-y-3 rounded-xl bg-muted p-4 text-sm'>
            <div className='flex items-start gap-3'>
              <Boxes className='mt-0.5 size-4 shrink-0' />
              <div>
                <p className='font-medium'>会话资产</p>
                <p className='text-muted-foreground'>
                  无需逐次批准即可创建和修改会话资产
                </p>
              </div>
            </div>
            <div className='flex items-start gap-3'>
              <Workflow className='mt-0.5 size-4 shrink-0' />
              <div>
                <p className='font-medium'>已启用的工作流</p>
                <p className='text-muted-foreground'>
                  可以根据对话内容发起工作流
                </p>
              </div>
            </div>
            <div className='flex items-start gap-3'>
              <Cable className='mt-0.5 size-4 shrink-0' />
              <div>
                <p className='font-medium'>已连接的服务</p>
                <p className='text-muted-foreground'>
                  可以调用允许直接访问的连接器工具
                </p>
              </div>
            </div>
          </div>
          <p className='text-sm text-muted-foreground'>
            这可能修改已有内容，或将数据发送给已连接的服务。
          </p>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={() => onChange('full_access')}>
              确认开启
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
