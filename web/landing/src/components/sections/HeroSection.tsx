import { useTranslation } from "react-i18next";
import { site } from "@/constants/site";
import { Reveal } from "@/components/motion/Reveal";
import { GradientBackground } from "@/components/reactbits/GradientBackground";
import { ShinyText } from "@/components/reactbits/ShinyText";
import { Button } from "@/components/reui/Button";

export function HeroSection() {
  const { t } = useTranslation();

  return (
    <section className="relative isolate overflow-hidden">
      <GradientBackground />
      <div className="relative mx-auto flex max-w-6xl flex-col items-center px-4 pb-20 pt-24 text-center sm:px-6 md:pb-28 md:pt-32">
        <Reveal>
          <span className="inline-flex items-center rounded-full border border-border bg-card px-3 py-1 text-xs font-medium text-muted-foreground">
            {t("hero.badge")}
          </span>
        </Reveal>

        <Reveal delay={0.05}>
          <h1 className="mt-6 max-w-3xl text-4xl font-semibold tracking-tight sm:text-5xl md:text-6xl">
            <ShinyText>{t("hero.title")}</ShinyText>
          </h1>
        </Reveal>

        <Reveal delay={0.1}>
          <p className="mt-6 max-w-2xl text-base text-muted-foreground sm:text-lg">
            {t("hero.subtitle")}
          </p>
        </Reveal>

        <Reveal delay={0.15}>
          <div className="mt-8 flex flex-col items-center gap-3 sm:flex-row">
            <Button href={site.downloadUrl}>{t("hero.cta_download")}</Button>
            <Button href={site.selfHostUrl} variant="outline">
              {t("hero.cta_selfhost")}
            </Button>
          </div>
        </Reveal>

        <Reveal delay={0.2}>
          <p className="mt-8 text-xs text-muted-foreground sm:text-sm">
            {t("hero.platform")}
          </p>
        </Reveal>
      </div>
    </section>
  );
}
