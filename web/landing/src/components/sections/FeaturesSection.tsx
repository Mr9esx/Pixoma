import { useTranslation } from 'react-i18next'
import { BotChatDemo } from '@/components/demos/BotChatDemo'
import { Reveal } from '@/components/motion/Reveal'
import { FeatureBlock } from './FeatureBlock'

export function FeaturesSection() {
  const { t } = useTranslation()

  return (
    <section id="features" className="mx-auto max-w-6xl px-4 py-20 sm:px-6">
      <Reveal>
        <h2 className="text-3xl font-semibold tracking-tight sm:text-4xl">
          {t('nav.features')}
        </h2>
      </Reveal>

      <div className="mt-10 grid gap-8 md:grid-cols-2 lg:grid-cols-3">
        <FeatureBlock title={t('features.bot.title')} desc={t('features.bot.desc')}>
          <BotChatDemo />
        </FeatureBlock>
        <FeatureBlock title={t('features.case.title')} desc={t('features.case.desc')}>
          <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
            Case
          </div>
        </FeatureBlock>
        <FeatureBlock title={t('features.admin.title')} desc={t('features.admin.desc')}>
          <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
            Admin
          </div>
        </FeatureBlock>
      </div>
    </section>
  )
}
