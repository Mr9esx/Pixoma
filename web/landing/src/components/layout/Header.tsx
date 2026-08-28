import { useState } from "react";
import { useTranslation } from "react-i18next";
import { LangSwitch } from "./LangSwitch";
import { MobileMenu } from "./MobileMenu";
import { cn } from "@/lib/utils";

export function Header() {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);

  return (
    <header className="sticky top-0 z-50 border-b border-border bg-background/80 backdrop-blur">
      <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4 sm:px-6">
        <a
          href="/cn"
          className="text-base font-semibold tracking-tight text-foreground"
        >
          Pixoma
        </a>

        <nav className="hidden items-center gap-6 text-sm text-muted-foreground md:flex">
          <a
            href="#features"
            className="transition-colors hover:text-foreground"
          >
            {t("nav.features")}
          </a>
          <a
            href="#scenarios"
            className="transition-colors hover:text-foreground"
          >
            {t("nav.scenarios")}
          </a>
          <a
            href="#download"
            className="transition-colors hover:text-foreground"
          >
            {t("nav.download")}
          </a>
        </nav>

        <div className="hidden md:block">
          <LangSwitch />
        </div>

        <button
          type="button"
          aria-label="menu"
          onClick={() => setOpen((v) => !v)}
          className={cn(
            "flex h-9 w-9 items-center justify-center rounded-md border border-border md:hidden",
          )}
        >
          <span className="block h-0.5 w-4 bg-foreground" />
          <span className="mt-1 block h-0.5 w-4 bg-foreground" />
        </button>
      </div>

      <MobileMenu open={open} onClose={() => setOpen(false)} />
    </header>
  );
}
