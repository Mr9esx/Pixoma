import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/badge'

export function RequiredBadge() {
  const { t } = useTranslation()
  return <Badge variant='secondary'>{t('common.required')}</Badge>
}
