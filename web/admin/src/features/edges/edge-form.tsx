import { useState, type FormEvent } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { RefreshCcw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { createEdge, getEdge, patchEdge } from '@/lib/api/edges'
import { queryKeys } from '@/lib/api/query-keys'
import type { ComfyEdge, EdgeHardwareGPU } from '@/lib/api/types'
import { Button } from '@/components/ui/button'
import { DialogFooter } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { DEFAULT_TOPIC_KEY } from '@/features/task-flow/types'
import { NodeTopicPicker } from './node-topic-picker'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function parseCapabilities(raw: string): string[] {
  return raw
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
}

type CreateProps = {
  mode: 'create'
  initial?: undefined
  onSaved: (edge: ComfyEdge) => void
  onDeleted?: undefined
  /** 对话框页脚在提交旁显示「取消」。 */
  onCancel?: () => void
  /** 页面级（新建向导）与对话框级（编辑）使用不同的页脚布局。 */
  layout?: 'page' | 'dialog'
}

type EditProps = {
  mode: 'edit'
  initial: ComfyEdge
  onSaved: (edge: ComfyEdge) => void
  onDeleted: () => void
  /** 对话框页脚在提交旁显示「取消」。 */
  onCancel?: () => void
  /** 页面级（新建向导）与对话框级（编辑）使用不同的页脚布局。 */
  layout?: 'page' | 'dialog'
}

type Props = CreateProps | EditProps

export function EdgeForm(props: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const [name, setName] = useState(
    props.mode === 'edit' ? props.initial.name : ''
  )
  const [description, setDescription] = useState(
    props.mode === 'edit' ? (props.initial.description ?? '') : ''
  )
  const [enabled, setEnabled] = useState(
    props.mode === 'edit' ? props.initial.enabled : true
  )
  const [capabilitiesRaw, setCapabilitiesRaw] = useState(
    props.mode === 'edit' ? props.initial.capabilities.join(', ') : ''
  )
  const [cpuModel, setCpuModel] = useState(
    props.mode === 'edit' ? (props.initial.hardware?.cpu_model ?? '') : ''
  )
  const [cpuCores, setCpuCores] = useState(
    props.mode === 'edit' ? String(props.initial.hardware?.cpu_cores ?? '') : ''
  )
  const [ramBytes, setRamBytes] = useState(
    props.mode === 'edit' ? String(props.initial.hardware?.ram_bytes ?? '') : ''
  )
  const [gpus, setGpus] = useState<EdgeHardwareGPU[]>(
    props.mode === 'edit' ? (props.initial.hardware?.gpus ?? []) : []
  )
  const [subscribeTopics, setSubscribeTopics] = useState<string[]>(
    props.mode === 'edit'
      ? (props.initial.subscribe_topics ?? [])
      : [DEFAULT_TOPIC_KEY]
  )

  const createMutation = useMutation({
    mutationFn: createEdge,
    onSuccess: async (next) => {
      if (subscribeTopics.length > 0) {
        try {
          await patchEdge(next.id, { subscribe_topics: subscribeTopics })
        } catch (err) {
          toast.error(
            errorMessage(err) ?? t('edges.subscribeTopicsPatchFailed')
          )
        }
      }
      await queryClient.invalidateQueries({ queryKey: queryKeys.edges.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.linkHealth })
      props.onSaved(next)
    },
  })

  const updateMutation = useMutation({
    mutationFn: (body: Parameters<typeof patchEdge>[1]) => {
      if (props.mode !== 'edit') {
        throw new Error('update requires edit mode')
      }
      return patchEdge(props.initial.id, body)
    },
    onSuccess: async (next) => {
      if (props.mode !== 'edit') return
      await queryClient.invalidateQueries({ queryKey: queryKeys.edges.all })
      await queryClient.invalidateQueries({
        queryKey: queryKeys.edges.detail(props.initial.id),
      })
      void queryClient.invalidateQueries({ queryKey: queryKeys.linkHealth })
      toast.success(t('common.successSaved'))
      props.onSaved(next)
    },
  })

  const refreshMutation = useMutation({
    mutationFn: () => {
      if (props.mode !== 'edit') {
        throw new Error('refresh requires edit mode')
      }
      return patchEdge(props.initial.id, { refresh_hardware: true })
    },
    onSuccess: async () => {
      if (props.mode !== 'edit') return
      await queryClient.invalidateQueries({
        queryKey: queryKeys.edges.detail(props.initial.id),
      })
      toast.success(t('edges.refreshRequested'))
      try {
        const fresh = await getEdge(props.initial.id)
        setCpuModel(fresh.hardware?.cpu_model ?? '')
        setCpuCores(String(fresh.hardware?.cpu_cores ?? ''))
        setRamBytes(String(fresh.hardware?.ram_bytes ?? ''))
        setGpus(fresh.hardware?.gpus ?? [])
      } catch {
        // hardware will arrive on the next heartbeat; keep current form values
      }
    },
  })

  const pending =
    createMutation.isPending ||
    updateMutation.isPending ||
    refreshMutation.isPending

  function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (!name.trim()) return
    const capabilities = parseCapabilities(capabilitiesRaw)
    if (props.mode === 'create') {
      createMutation.mutate({
        name: name.trim(),
        description: description.trim(),
        enabled,
        capabilities,
      })
      return
    }
    const cores = Number.parseInt(cpuCores, 10)
    const ram = Number.parseInt(ramBytes, 10)
    updateMutation.mutate({
      name: name.trim(),
      description: description.trim(),
      enabled,
      capabilities,
      subscribe_topics: subscribeTopics,
      hardware: {
        cpu_model: cpuModel.trim() || undefined,
        cpu_cores: Number.isFinite(cores) && cores > 0 ? cores : undefined,
        ram_bytes: Number.isFinite(ram) && ram > 0 ? ram : undefined,
        gpus: gpus.filter((g) => g.name.trim()),
      },
    })
  }

  const formFields = (
    <form
      id='edge-form'
      onSubmit={onSubmit}
      className='flex flex-col gap-4'
      data-testid='edge-form'
    >
      <div className='flex flex-col gap-2'>
        <Label htmlFor='edge-name'>{t('edges.fieldName')}</Label>
        <Input
          id='edge-name'
          value={name}
          onChange={(e) => setName(e.target.value)}
          disabled={pending}
          required
          autoComplete='off'
        />
      </div>

      <NodeTopicPicker
        id='edge-subscribe-topics'
        value={subscribeTopics}
        onChange={setSubscribeTopics}
        disabled={pending}
      />

      <div className='flex flex-col gap-2'>
        <Label htmlFor='edge-description'>{t('edges.fieldDescription')}</Label>
        <Textarea
          id='edge-description'
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          disabled={pending}
        />
      </div>

      <div className='flex flex-col gap-2'>
        <Label htmlFor='edge-capabilities'>
          {t('edges.fieldCapabilities')}
        </Label>
        <Input
          id='edge-capabilities'
          value={capabilitiesRaw}
          onChange={(e) => setCapabilitiesRaw(e.target.value)}
          disabled={pending}
          autoComplete='off'
        />
      </div>

      <div className='flex items-center justify-between gap-3'>
        <Label htmlFor='edge-enabled'>{t('edges.fieldEnabled')}</Label>
        <Switch
          id='edge-enabled'
          checked={enabled}
          onCheckedChange={setEnabled}
          disabled={pending}
        />
      </div>

      {props.mode === 'edit' ? (
        <>
          <Button
            type='button'
            variant='outline'
            disabled={refreshMutation.isPending}
            onClick={() => refreshMutation.mutate()}
          >
            <RefreshCcw className='size-3.5' />
            {t('edges.refreshHardware')}
          </Button>
          <div className='flex flex-col gap-2'>
            <Label htmlFor='edge-cpu'>{t('edges.fieldCpu')}</Label>
            <Input
              id='edge-cpu'
              value={cpuModel}
              onChange={(e) => setCpuModel(e.target.value)}
              disabled={pending}
              autoComplete='off'
            />
          </div>
          <div className='flex flex-col gap-2'>
            <Label htmlFor='edge-cpu-cores'>{t('edges.fieldCpuCores')}</Label>
            <Input
              id='edge-cpu-cores'
              value={cpuCores}
              onChange={(e) => setCpuCores(e.target.value)}
              disabled={pending}
              autoComplete='off'
            />
          </div>
          <div className='flex flex-col gap-2'>
            <Label htmlFor='edge-ram-bytes'>{t('edges.fieldRamBytes')}</Label>
            <Input
              id='edge-ram-bytes'
              value={ramBytes}
              onChange={(e) => setRamBytes(e.target.value)}
              disabled={pending}
              autoComplete='off'
            />
          </div>
          <div className='flex flex-col gap-2'>
            <Label>{t('edges.fieldGpu')}</Label>
            {gpus.map((gpu, index) => (
              <div key={index} className='flex gap-2'>
                <Input
                  value={gpu.name}
                  onChange={(e) => {
                    const next = [...gpus]
                    next[index] = { ...gpu, name: e.target.value }
                    setGpus(next)
                  }}
                  disabled={pending}
                  autoComplete='off'
                />
                <Input
                  value={gpu.vram_bytes ? String(gpu.vram_bytes) : ''}
                  onChange={(e) => {
                    const next = [...gpus]
                    const n = Number.parseInt(e.target.value, 10)
                    next[index] = {
                      ...gpu,
                      vram_bytes: Number.isFinite(n) ? n : undefined,
                    }
                    setGpus(next)
                  }}
                  disabled={pending}
                  autoComplete='off'
                />
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  onClick={() => setGpus(gpus.filter((_, i) => i !== index))}
                >
                  {t('common.delete')}
                </Button>
              </div>
            ))}
            <Button
              type='button'
              variant='outline'
              size='sm'
              onClick={() => setGpus([...gpus, { name: '' }])}
            >
              {t('edges.addGpu')}
            </Button>
          </div>
        </>
      ) : null}
    </form>
  )

  if (props.layout === 'page') {
    return (
      <div className='flex flex-1 flex-col'>
        {formFields}
        <div className='sticky bottom-0 z-10 mt-auto flex shrink-0 items-center justify-end gap-2 border-t bg-card px-6 py-3'>
          {props.onCancel ? (
            <Button type='button' variant='outline' onClick={props.onCancel}>
              {t('common.cancel')}
            </Button>
          ) : null}
          <Button type='submit' form='edge-form' disabled={pending}>
            {props.mode === 'create'
              ? t('edges.createAndContinue')
              : t('common.save')}
          </Button>
        </div>
      </div>
    )
  }

  return (
    <>
      <div className='min-h-0 flex-1 overflow-y-auto px-1'>{formFields}</div>
      <DialogFooter className='shrink-0'>
        {props.onCancel ? (
          <Button type='button' variant='outline' onClick={props.onCancel}>
            {t('common.cancel')}
          </Button>
        ) : null}
        <Button type='submit' form='edge-form' disabled={pending}>
          {props.mode === 'create'
            ? t('edges.createAndContinue')
            : t('common.save')}
        </Button>
      </DialogFooter>
    </>
  )
}
