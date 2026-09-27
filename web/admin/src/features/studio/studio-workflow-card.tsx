import { useState } from 'react'
import { Sparkles } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardFooter, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { useMediaObjectUrl } from '@/features/cases/lib/use-media-object-url'
import { resolveMediaKey } from '@/lib/api/media'
import type { StudioAsset, StudioWorkflowRequest } from '@/lib/api/studio'

type Property = { type?: string; title?: string; description?: string; enum?: unknown[] }

export function StudioWorkflowCard({
  workflow,
  assets,
  disabled = false,
  submitting = false,
  onUploadAsset,
  onSubmit,
  onSkip,
}: {
  workflow: StudioWorkflowRequest
  assets: StudioAsset[]
  disabled?: boolean
  submitting?: boolean
  onUploadAsset?: (file: File) => Promise<StudioAsset>
  onSubmit: (inputs: Record<string, unknown>) => void
  onSkip: () => void
}) {
  const [values, setValues] = useState<Record<string, unknown>>(workflow.suggested_inputs ?? {})
  const [uploadedAssets, setUploadedAssets] = useState<StudioAsset[]>([])
  const [uploadingField, setUploadingField] = useState<string>()
  const [uploadError, setUploadError] = useState<string>()
  const previewUrl = useMediaObjectUrl(workflow.preview)
  const previewIsVideo = /\.(mp4|webm)$/i.test(resolveMediaKey(workflow.preview) ?? '')
  const schema = workflow.input_schema
  const properties = (schema.properties ?? {}) as Record<string, Property>
  const required = new Set(Array.isArray(schema.required) ? schema.required as string[] : [])
  const fields = workflow.input_fields?.length
    ? workflow.input_fields
    : Object.entries(properties).map(([key, property]) => ({
        key, type: property.type ?? 'string', required: required.has(key), description: property.description,
      }))
  const complete = fields.every((field) => {
    if (!field.required) return true
    const value = values[field.key]
    return value !== undefined && value !== null && value !== ''
  })
  const setValue = (key: string, value: unknown) => setValues((current) => ({ ...current, [key]: value }))

  return (
    <Card
      data-testid='studio-workflow-card'
      className='grid grid-cols-[minmax(0,2fr)_minmax(0,3fr)] gap-x-4 gap-y-4 overflow-hidden px-3 py-3'
    >
      <CardContent data-testid='studio-workflow-info' className='flex flex-col gap-3 p-0'>
        <div className='relative h-44 w-full shrink-0 overflow-hidden rounded-lg bg-muted'>
          {previewUrl && previewIsVideo ? (
            <video src={`${previewUrl}#t=0.1`} muted playsInline preload='metadata' disablePictureInPicture className='h-full w-full object-cover' />
          ) : previewUrl ? (
            <img src={previewUrl} alt={workflow.name} className='h-full w-full object-cover' />
          ) : (
            <div className='flex h-full w-full items-center justify-center'>
              <Sparkles aria-hidden='true' className='size-8 text-muted-foreground' />
            </div>
          )}
        </div>
        <div className='flex flex-col gap-1'>
          <CardTitle className='text-sm'>{workflow.name}</CardTitle>
          {workflow.description ? <p className='line-clamp-2 text-sm text-muted-foreground'>{workflow.description}</p> : null}
        </div>
      </CardContent>
      <div className='relative min-w-0'>
        <CardContent data-testid='studio-workflow-inputs' className='absolute inset-0 flex flex-col gap-4 overflow-y-auto overscroll-contain p-0 pr-2'>
          {fields.map((field) => {
            const property = properties[field.key] ?? {}
            const label = property.title || field.description || field.key
            const options = property.enum?.filter((option): option is string => typeof option === 'string') ?? []
            const availableAssets = [...assets, ...uploadedAssets.filter((asset) => !assets.some((current) => current.id === asset.id))]
              .filter((asset) => asset.kind === field.type)
            const value = values[field.key]
            const isMedia = field.type === 'image' || field.type === 'video'
            return (
              <div key={field.key} className='flex flex-col gap-2'>
                <label htmlFor={`workflow-${workflow.id}-${field.key}`} className='text-sm font-medium'>
                  {label}{field.required ? ' *' : ''}
                </label>
                {field.type === 'boolean' ? (
                  <Checkbox
                    id={`workflow-${workflow.id}-${field.key}`}
                    checked={value === true}
                    disabled={disabled}
                    onCheckedChange={(checked) => setValue(field.key, checked === true)}
                  />
                ) : options.length || isMedia ? (
                  <Select value={typeof value === 'string' ? value : undefined} disabled={disabled} onValueChange={(next) => setValue(field.key, next)}>
                    <SelectTrigger id={`workflow-${workflow.id}-${field.key}`} className='w-full'>
                      <SelectValue placeholder='选择输入' />
                    </SelectTrigger>
                    <SelectContent>
                      {options.map((option) => <SelectItem key={option} value={option}>{option}</SelectItem>)}
                      {availableAssets.map((asset) => <SelectItem key={asset.id} value={asset.id}>{asset.name}</SelectItem>)}
                    </SelectContent>
                  </Select>
                ) : field.type === 'number' || property.type === 'number' ? (
                  <Input
                    id={`workflow-${workflow.id}-${field.key}`}
                    type='number'
                    value={typeof value === 'number' || typeof value === 'string' ? value : ''}
                    disabled={disabled}
                    onChange={(event) => setValue(field.key, event.target.value === '' ? '' : Number(event.target.value))}
                  />
                ) : (
                  <Textarea
                    id={`workflow-${workflow.id}-${field.key}`}
                    value={typeof value === 'string' ? value : ''}
                    disabled={disabled}
                    placeholder={property.description}
                    onChange={(event) => setValue(field.key, event.target.value)}
                  />
                )}
                {isMedia && onUploadAsset ? (
                  <Input
                    aria-label={`上传${label}`}
                    type='file'
                    accept={field.type === 'image' ? 'image/*' : 'video/*'}
                    disabled={disabled || Boolean(uploadingField)}
                    onChange={(event) => {
                      const file = event.target.files?.[0]
                      if (!file) return
                      event.currentTarget.value = ''
                      setUploadingField(field.key)
                      setUploadError(undefined)
                      void onUploadAsset(file).then((asset) => {
                        setUploadedAssets((current) => [...current, asset])
                        setValue(field.key, asset.id)
                      }).catch((error: unknown) => {
                        setUploadError(error instanceof Error ? error.message : '上传素材失败')
                      }).finally(() => setUploadingField(undefined))
                    }}
                  />
                ) : null}
              </div>
            )
          })}
          {uploadError ? <p role='alert' className='text-sm text-destructive'>{uploadError}</p> : null}
        </CardContent>
      </div>
      <CardFooter className='col-span-2 justify-end gap-2 p-0'>
        <Button type='button' variant='outline' disabled={disabled || Boolean(uploadingField)} onClick={onSkip}>跳过</Button>
        <Button type='button' disabled={disabled || Boolean(uploadingField) || !complete} onClick={() => onSubmit(values)}>
          {submitting ? '提交中…' : '提交工作流'}
        </Button>
      </CardFooter>
    </Card>
  )
}
