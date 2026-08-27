import { useCallback, useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { WorkflowEditor, type WorkflowEditorProps } from './workflow-editor'

/**
 * 独立编辑页入口。编辑逻辑统一在 WorkflowEditor，本组件仅补「创建/取消」底部操作区，
 * 与 Quick Config 第一步共用同一编辑器。
 */
export function CaseForm(props: WorkflowEditorProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [pending, setPending] = useState(false)

  const handlePendingChange = useCallback(
    (next: boolean) => {
      setPending(next)
      props.onPendingChange?.(next)
    },
    [props.onPendingChange]
  )

  const showFooter = props.mode === 'create' && !props.hideActions
  const footer = showFooter ? (
    <div className='sticky bottom-0 z-10 -mx-6 -mb-7 flex flex-wrap gap-2 border-t bg-card px-6 py-3 md:-mx-8 md:px-8'>
      <Button type='submit' disabled={pending}>
        {pending ? <Loader2 className='size-4 animate-spin' /> : null}
        {t('common.create')}
      </Button>
      <Button
        type='button'
        variant='outline'
        disabled={pending}
        onClick={() => void navigate({ to: '/cases' })}
      >
        {t('common.cancel')}
      </Button>
    </div>
  ) : null

  const Wrapper = WorkflowEditor
  return <Wrapper {...props} footer={footer} onPendingChange={handlePendingChange} />
}
