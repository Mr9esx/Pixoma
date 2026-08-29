import { useTranslation } from 'react-i18next'
import { Pill } from '@/components/kibo-ui/pill'

type Props = {
  on: boolean
  children: string
}

export function StatusTag({ on, children }: Props) {
  return (
    <Pill
      dot={on ? 'success' : 'neutral'}
      className={
        on
          ? 'border-success/25 bg-success/10 text-success'
          : 'border-border bg-muted text-muted-foreground'
      }
    >
      {children}
    </Pill>
  )
}

type PresenceProps = {
  edgeOnline: boolean
  comfyRunning: boolean
}

export function PresenceTags({ edgeOnline, comfyRunning }: PresenceProps) {
  const { t } = useTranslation()
  return (
    <div className='flex items-center gap-1'>
      <StatusTag on={edgeOnline}>
        {edgeOnline ? t('edges.nodeOnline') : t('edges.nodeOffline')}
      </StatusTag>
      <StatusTag on={comfyRunning}>
        {comfyRunning ? t('edges.comfyRunning') : t('edges.comfyStopped')}
      </StatusTag>
    </div>
  )
}
