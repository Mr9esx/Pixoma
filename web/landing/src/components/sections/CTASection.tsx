import { useTranslation } from 'react-i18next'
import { site } from '@/constants/site'
import { Reveal } from '@/components/motion/Reveal'
import { Button } from '@/components/ui/Button'

export function CTASection() {
  const { t } = useTranslation()

  return (
    <section id="download" className="mx-auto max-w-6xl px-4 py-20 sm:px-6">
      <Reveal>
        <div className="flex flex-col items-center rounded-3xl border border-border bg-card px-6 py-16 text-center">
          <h2 className="max-w-2xl text-3xl font-semibold tracking-tight sm:text-4xl">
            {t('cta.title')}
          </h2>
          <p className="mt-4 max-w-xl text-muted-foreground">{t('cta.subtitle')}</p>
          <div className="mt-8 flex flex-col items-center gap-3 sm:flex-row">
            <Button href={site.downloadUrl}>{t('hero.cta_download')}</Button>
            <Button href={site.selfHostUrl} variant="outline">
              {t('hero.cta_selfhost')}
            </Button>
          </div>
        </div>
      </Reveal>
    </section>
  )
}
