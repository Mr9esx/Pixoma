import { useTranslation } from 'react-i18next'
import { kit } from './kit-classes'

type Props = {
  on: boolean
  children: string
}

export function StatusTag({ on, children }: Props) {
  return <span className={on ? kit.tagOn : kit.tagOff}>{children}</span>
}

type PresenceProps = {
  edgeOnline: boolean
  comfyRunning: boolean
}

export function PresenceTags({ edgeOnline, comfyRunning }: PresenceProps) {
  const { t } = useTranslation()
  return (
    <>
      <StatusTag on={edgeOnline}>
        {edgeOnline ? t('edges.nodeOnline') : t('edges.nodeOffline')}
      </StatusTag>
      <StatusTag on={comfyRunning}>
        {comfyRunning ? t('edges.comfyRunning') : t('edges.comfyStopped')}
      </StatusTag>
    </>
  )
}
