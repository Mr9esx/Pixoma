import { CN, US } from 'country-flag-icons/react/3x2'
import { Check, Globe } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { setStoredLocale, type AppLocale } from '@/lib/i18n'
import { cn } from '@/lib/utils'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

const locales: Array<{
  value: AppLocale
  labelKey: string
  Flag: typeof CN
}> = [
  { value: 'zh', labelKey: 'lang.zh', Flag: CN },
  { value: 'en', labelKey: 'lang.en', Flag: US },
]

export function LocaleSwitcher({ className }: { className?: string }) {
  const { t, i18n } = useTranslation()
  const currentLocale: AppLocale = i18n.language.startsWith('en') ? 'en' : 'zh'

  function switchLocale(locale: AppLocale) {
    void i18n.changeLanguage(locale)
    setStoredLocale(locale)
  }

  return (
    <div
      className={cn('relative isolate flex', className)}
      data-testid='locale-switcher'
    >
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            aria-label={t('lang.label')}
            className='flex h-6 items-center gap-1.5 rounded-full bg-background px-2.5 ring-1 ring-border ring-inset'
            data-testid='locale-switcher-trigger'
            type='button'
          >
            <Globe className='size-3.5 text-muted-foreground' />
            <span className='text-xs text-foreground'>{t('lang.label')}</span>
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align='end' side='top'>
          {locales.map(({ value, labelKey, Flag }) => {
            const isActive = currentLocale === value

            return (
              <DropdownMenuItem
                className='gap-2'
                data-testid={`locale-switcher-${value}`}
                key={value}
                onClick={() => switchLocale(value)}
              >
                <Flag className='size-4' />
                <span>{t(labelKey)}</span>
                <Check
                  size={14}
                  className={cn('ms-auto', !isActive && 'hidden')}
                />
              </DropdownMenuItem>
            )
          })}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}
