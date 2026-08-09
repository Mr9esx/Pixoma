import { Check, Globe } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { setStoredLocale, type AppLocale } from '@/lib/i18n'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

export function LanguageSwitcher() {
  const { i18n, t } = useTranslation()
  const current: AppLocale = i18n.language.startsWith('en') ? 'en' : 'zh'

  function switchTo(locale: AppLocale) {
    void i18n.changeLanguage(locale)
    setStoredLocale(locale)
  }

  return (
    <div data-testid='language-switcher'>
      <DropdownMenu modal={false}>
        <DropdownMenuTrigger asChild>
          <Button
            variant='ghost'
            size='icon'
            className='scale-95 rounded-full'
            type='button'
          >
            <Globe className='size-[1.2rem]' />
            <span className='sr-only'>{t('lang.switch')}</span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align='end'>
          <DropdownMenuItem onClick={() => switchTo('zh')}>
            {t('lang.zh')}
            <Check
              size={14}
              className={cn('ms-auto', current !== 'zh' && 'hidden')}
            />
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => switchTo('en')}>
            {t('lang.en')}
            <Check
              size={14}
              className={cn('ms-auto', current !== 'en' && 'hidden')}
            />
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}
