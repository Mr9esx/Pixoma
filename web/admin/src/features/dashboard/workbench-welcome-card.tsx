import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { useQuery } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import MouseEffectCard from '@/components/kokonutui/mouse-effect-card'
import { fetchCurrentUser } from '@/lib/api/setup'

export function WorkbenchWelcomeCard() {
  const { t } = useTranslation()
  const { data: user } = useQuery({
    queryKey: ['current-user'],
    queryFn: fetchCurrentUser,
  })
  const name = user?.nickname || user?.username || ''

  return (
    <MouseEffectCard
      data-testid='workbench-welcome'
      className='h-[292px] w-full'
    >
      <h2 className='text-center text-2xl font-bold tracking-tight text-foreground'>
        {t('dashboard.workbench.welcomeTitle', { name })}
      </h2>
      <div className='flex flex-wrap items-center justify-center gap-3'>
        <Button data-testid='workbench-quick-create-case' asChild>
          <Link to='/cases'>{t('dashboard.workbench.quickCreateCase')}</Link>
        </Button>
        <Button
          data-testid='workbench-quick-manage-edges'
          asChild
          variant='outline'
        >
          <Link to='/edges'>{t('dashboard.workbench.quickManageEdges')}</Link>
        </Button>
      </div>
    </MouseEffectCard>
  )
}
