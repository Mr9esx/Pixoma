import { useTranslation } from 'react-i18next'
import { setStoredLocale, type AppLocale } from '@/lib/i18n'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'

export function LanguageSwitcher() {
  const { i18n, t } = useTranslation()
  const current: AppLocale = i18n.language.startsWith('en') ? 'en' : 'zh'

  function switchTo(locale: AppLocale) {
    void i18n.changeLanguage(locale)
    setStoredLocale(locale)
  }

  return (
    <div className='flex items-center gap-1' data-testid='language-switcher'>
      <Button
        type='button'
        variant={current === 'zh' ? 'secondary' : 'ghost'}
        size='sm'
        className={cn('h-8 px-2')}
        onClick={() => switchTo('zh')}
        aria-pressed={current === 'zh'}
      >
        {t('lang.zh')}
      </Button>
      <Button
        type='button'
        variant={current === 'en' ? 'secondary' : 'ghost'}
        size='sm'
        className={cn('h-8 px-2')}
        onClick={() => switchTo('en')}
        aria-pressed={current === 'en'}
      >
        {t('lang.en')}
      </Button>
    </div>
  )
}
