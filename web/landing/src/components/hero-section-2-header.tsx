import Link from "next/link";
import { Logo } from "@/components/logo";
import { Star, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Github } from "@/components/ui/svgs/github";
import { ThemeSwitcher } from "@/components/kibo-ui/theme-switcher";
import { useTheme } from "@/context/theme-provider";
import { LocaleSwitcher } from "@/components/locale-switcher";
import { Reveal } from "@/components/motion-primitives";
import React from "react";

export const HeroHeader = () => {
  const [menuState, setMenuState] = React.useState(false);
  const [locale, setLocale] = React.useState<"zh" | "en">("zh");
  const { theme, setTheme } = useTheme();
  const [starCount, setStarCount] = React.useState<number | null>(null);

  React.useEffect(() => {
    let active = true;

    fetch("https://api.github.com/repos/Mr9esx/Pixoma")
      .then((response) => (response.ok ? response.json() : null))
      .then((data) => {
        if (!active || !data) return;
        setStarCount(
          typeof data.stargazers_count === "number"
            ? data.stargazers_count
            : null,
        );
      })
      .catch(() => {});

    return () => {
      active = false;
    };
  }, []);

  React.useEffect(() => {
    if (!menuState) return;

    const mediaQuery = window.matchMedia("(max-width: 1023px)");
    const updateOverflow = () => {
      document.documentElement.classList.toggle(
        "overflow-hidden",
        mediaQuery.matches,
      );
    };

    updateOverflow();
    mediaQuery.addEventListener("change", updateOverflow);

    return () => {
      mediaQuery.removeEventListener("change", updateOverflow);
      document.documentElement.classList.remove("overflow-hidden");
    };
  }, [menuState]);

  return (
    <header>
      <nav
        data-state={menuState && "active"}
        className="bg-background/80 backdrop-blur-sm fixed top-0 z-20 w-full max-lg:data-[state=active]:bottom-0"
      >
        <Reveal className="mx-auto max-w-7xl px-6" duration={0.8} y={0}>
          <div className="relative flex flex-wrap items-center justify-between max-lg:gap-6">
            <div className="max-lg:in-data-[state=active]:border-b flex w-full items-center justify-between gap-12 py-4 lg:w-auto lg:py-5">
              <Link
                href="/"
                aria-label="goto home"
                className="flex items-center gap-2"
              >
                <Logo />
              </Link>

              <button
                onClick={() => setMenuState(!menuState)}
                aria-label={menuState == true ? "Close Menu" : "Open Menu"}
                className="relative z-20 block cursor-pointer after:absolute after:-inset-4 lg:hidden"
              >
                <div className="in-data-[state=active]:rotate-180 in-data-[state=active]:scale-0 in-data-[state=active]:opacity-0 size-4.5 m-auto flex flex-col items-center justify-center gap-[7px] duration-200">
                  <span className="bg-foreground h-0.5 w-full rounded-full" />
                  <span className="bg-foreground h-0.5 w-full rounded-full" />
                </div>

                <X className="in-data-[state=active]:rotate-0 in-data-[state=active]:scale-100 in-data-[state=active]:opacity-100 absolute inset-0 m-auto size-6 translate-x-[-3px] -rotate-180 scale-0 opacity-0 duration-200" />
              </button>
            </div>

            <div className="in-data-[state=active]:block lg:in-data-[state=active]:flex mb-6 hidden w-full flex-wrap items-center justify-end max-lg:space-y-8 md:flex-nowrap lg:m-0 lg:flex lg:w-fit lg:gap-6">
              <div className="flex w-full flex-col gap-3 sm:flex-row sm:items-center sm:gap-3 md:w-fit">
                <div className="flex items-center justify-center gap-3">
                  <LocaleSwitcher onChange={setLocale} value={locale} />
                  <ThemeSwitcher value={theme} onChange={setTheme} />
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  nativeButton={false}
                  render={
                    <Link
                      href="https://github.com/Mr9esx/Pixoma"
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      <Star className="size-4" />
                      <span>Star</span>
                      {starCount !== null ? (
                        <span className="tabular-nums">
                          {starCount.toLocaleString()}
                        </span>
                      ) : null}
                    </Link>
                  }
                />
                <Button
                  variant="ghost"
                  size="sm"
                  nativeButton={false}
                  render={
                    <Link
                      href="https://github.com/Mr9esx/Pixoma"
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      <Github className="size-4" />
                      <span>Github</span>
                    </Link>
                  }
                />
              </div>
            </div>
          </div>
        </Reveal>
      </nav>
    </header>
  );
};
