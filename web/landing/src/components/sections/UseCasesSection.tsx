import { useTranslation } from "react-i18next";
import { SectionShell } from "@/components/SectionShell";
import { Reveal } from "@/components/motion/Reveal";

const cards = [
  { key: "card_1" },
  { key: "card_2" },
  { key: "card_3" },
  { key: "card_4" },
] as const;

export function UseCasesSection() {
  const { t } = useTranslation();

  return (
    <SectionShell id="scenarios" title={t("nav.scenarios")}>
      <div className="grid gap-6 sm:grid-cols-2">
        {cards.map((card) => (
          <Reveal key={card.key}>
            <div className="rounded-xl border border-border bg-card p-6">
              <h3 className="text-lg font-semibold text-foreground">
                {t(`scenarios.${card.key}.title`)}
              </h3>
              <p className="mt-2 text-sm text-muted-foreground">
                {t(`scenarios.${card.key}.desc`)}
              </p>
            </div>
          </Reveal>
        ))}
      </div>
    </SectionShell>
  );
}
