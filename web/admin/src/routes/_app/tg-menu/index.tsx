import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { TgMenuEditor } from '@/features/tg-menu/menu-editor'

export const Route = createFileRoute('/_app/tg-menu/')({
  component: TgMenuPage,
})

function TgMenuPage() {
  const { t } = useTranslation()
  return (
    <div className='space-y-3' data-testid='tg-menu-page'>
      <div>
        <h1 className='text-2xl font-bold tracking-tight'>
          {t('tgMenu.title')}
        </h1>
        <p className='text-muted-foreground text-sm'>
          {t('tgMenu.description')}
        </p>
      </div>
      <TgMenuEditor />
    </div>
  )
}
