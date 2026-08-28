import { useTranslation } from "react-i18next";
import { useInView } from "motion/react";
import { useRef } from "react";
import {
  LazyAdmin,
  LazyBotChat,
  LazyCaseDir,
} from "@/components/demos/LazyDemos";
import { SectionShell } from "@/components/SectionShell";
import { FeatureBlock } from "./FeatureBlock";

export function FeaturesSection() {
  const { t } = useTranslation();
  const gridRef = useRef<HTMLDivElement>(null);
  const gridInView = useInView(gridRef, {
    once: true,
    margin: "0px 0px -10% 0px",
  });

  return (
    <SectionShell id="features" title={t("nav.features")}>
      <div
        ref={gridRef}
        className="mt-10 grid gap-8 md:grid-cols-2 lg:grid-cols-3"
      >
        <FeatureBlock
          title={t("features.bot.title")}
          desc={t("features.bot.desc")}
        >
          {gridInView ? (
            <LazyBotChat />
          ) : (
            <div className="aspect-[4/3] bg-muted" />
          )}
        </FeatureBlock>
        <FeatureBlock
          title={t("features.case.title")}
          desc={t("features.case.desc")}
        >
          {gridInView ? (
            <LazyCaseDir />
          ) : (
            <div className="aspect-[4/3] bg-muted" />
          )}
        </FeatureBlock>
        <FeatureBlock
          title={t("features.admin.title")}
          desc={t("features.admin.desc")}
        >
          {gridInView ? (
            <LazyAdmin />
          ) : (
            <div className="aspect-[4/3] bg-muted" />
          )}
        </FeatureBlock>
      </div>
    </SectionShell>
  );
}
