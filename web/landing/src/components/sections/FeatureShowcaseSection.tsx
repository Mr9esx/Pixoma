import { useTranslation } from "react-i18next";
import {
  LazyAdmin,
  LazyBotChat,
  LazyCaseDir,
} from "@/components/demos/LazyDemos";
import { SectionShell } from "@/components/SectionShell";
import { FeatureShowcase } from "./FeatureShowcase";

export function FeatureShowcaseSection() {
  const { t } = useTranslation();

  return (
    <SectionShell
      id="features"
      title={t("nav.features")}
      className="relative z-10 -mt-10 rounded-t-[2.5rem] bg-background lg:-mt-14 lg:rounded-t-[3.5rem]"
    >
      <div className="grid gap-6 lg:grid-cols-3 lg:gap-8">
        <FeatureShowcase
          title={t("features.bot.title")}
          description={t("features.bot.desc")}
        >
          <LazyBotChat />
        </FeatureShowcase>
        <FeatureShowcase
          title={t("features.case.title")}
          description={t("features.case.desc")}
          offset="down"
        >
          <LazyCaseDir />
        </FeatureShowcase>
        <FeatureShowcase
          title={t("features.admin.title")}
          description={t("features.admin.desc")}
          offset="mid"
        >
          <LazyAdmin />
        </FeatureShowcase>
      </div>
    </SectionShell>
  );
}
