import { useTranslation } from "react-i18next";
import { LazyCaseDir } from "@/components/demos/LazyDemos";
import { SectionShell } from "@/components/SectionShell";
import { ProductFrame } from "@/components/ui/ProductFrame";

export function CrossDeviceSection() {
  const { t } = useTranslation();

  return (
    <SectionShell
      id="devices"
      title={t("crossDevice.title")}
      className="surface-grid relative overflow-hidden bg-muted pb-28 lg:pb-36"
    >
      <div className="relative mx-auto max-w-5xl">
        <ProductFrame
          className="mx-auto max-w-4xl"
          innerClassName="device-desktop min-h-72"
        >
          <LazyCaseDir />
        </ProductFrame>
        <div className="relative z-20 mt-6 flex flex-col gap-6 sm:flex-row sm:items-end lg:absolute lg:-bottom-8 lg:left-8 lg:mt-0">
          <div className="device-phone bg-card">
            <div className="flex h-full flex-col justify-between rounded-[1.2rem] bg-background p-4">
              <h3 className="text-xs font-medium text-muted-foreground">
                {t("crossDevice.mobile.title")}
              </h3>
              <p className="text-sm leading-6 text-foreground">
                {t("crossDevice.mobile.desc")}
              </p>
            </div>
          </div>
          <div className="max-w-sm pb-2">
            <h3 className="text-xl font-semibold">
              {t("crossDevice.desktop.title")}
            </h3>
            <p className="mt-2 leading-7 text-muted-foreground">
              {t("crossDevice.desktop.desc")}
            </p>
          </div>
        </div>
      </div>
    </SectionShell>
  );
}
