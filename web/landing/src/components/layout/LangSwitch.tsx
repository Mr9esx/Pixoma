import { useTranslation } from "react-i18next";
import { useNavigate, useParams } from "react-router-dom";
import { cn } from "@/lib/utils";
import { langSegmentToLocale, supportedLangSegments } from "@/router/segments";

export function LangSwitch() {
  const { lang } = useParams();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const current = langSegmentToLocale[lang ?? "cn"] ?? "zh-CN";

  return (
    <div className="flex items-center gap-1 rounded-full border border-border bg-card p-1 text-sm">
      {supportedLangSegments.map((segment) => {
        const locale = langSegmentToLocale[segment];
        const active = current === locale;
        return (
          <button
            key={segment}
            type="button"
            aria-pressed={active}
            onClick={() => navigate(`/${segment}`)}
            className={cn(
              "rounded-full px-2.5 py-1 transition-colors",
              active
                ? "bg-primary text-primary-foreground"
                : "text-muted-foreground hover:text-foreground",
            )}
          >
            {segment === "cn" ? t("nav.lang_zh") : t("nav.lang_en")}
          </button>
        );
      })}
    </div>
  );
}
