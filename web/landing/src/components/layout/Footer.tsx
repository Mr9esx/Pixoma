import { useTranslation } from 'react-i18next'
import { LangSwitch } from './LangSwitch'

export function Footer() {
  const { t } = useTranslation()

  return (
    <footer className="border-t border-border">
      <div className="mx-auto flex max-w-6xl flex-col gap-6 px-4 py-10 sm:px-6 md:flex-row md:items-center md:justify-between">
        <div>
          <div className="text-base font-semibold text-foreground">Pixoma</div>
          <p className="mt-1 text-sm text-muted-foreground">{t('hero.subtitle')}</p>
        </div>

        <nav className="flex flex-col gap-2 text-sm text-muted-foreground">
          <a href="#features" className="hover:text-foreground">
            {t('nav.features')}
          </a>
          <a href="#scenarios" className="hover:text-foreground">
            {t('nav.scenarios')}
          </a>
        </nav>

        <div className="flex items-center gap-4">
          <LangSwitch />
          <span className="text-sm text-muted-foreground">{t('footer.copyright')}</span>
        </div>
      </div>
    </footer>
  )
}
