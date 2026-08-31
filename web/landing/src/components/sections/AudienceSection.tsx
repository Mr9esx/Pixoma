import { useTranslation } from "react-i18next";
import { SectionShell } from "@/components/SectionShell";
import { audienceIds } from "@/constants/content";

export function AudienceSection() {
  const { t } = useTranslation();

  return (
    <SectionShell
      id="audience"
      title={t("audience.title")}
      className="surface-glow overflow-hidden bg-background"
    >
      <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
        {audienceIds.map((id, index) => (
          <article key={id} className="rounded-[1.35rem] bg-muted p-2.5">
            <div className="flex min-h-64 min-w-0 flex-col gap-5 rounded-[1.1rem] bg-card p-6">
              <span className="flex size-11 items-center justify-center rounded-full bg-foreground text-sm font-medium text-background">
                {String(index + 1).padStart(2, "0")}
              </span>
              <h3 className="text-xl font-semibold">
                {t(`audience.${id}.title`)}
              </h3>
              <p className="leading-7 text-foreground">
                {t(`audience.${id}.benefit`)}
              </p>
              <p className="text-sm leading-6 text-muted-foreground">
                {t(`audience.${id}.usage`)}
              </p>
            </div>
          </article>
        ))}
      </div>
    </SectionShell>
  );
}
