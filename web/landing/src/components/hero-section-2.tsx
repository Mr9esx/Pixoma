import Image from "next/image";
import Link from "next/link";
import { HeroHeader } from "@/components/hero-section-2-header";
import { Reveal } from "@/components/motion-primitives";
import { Button } from "@/components/ui/button";
import { useTheme } from "@/context/theme-provider";

export default function HeroSection() {
  const { theme } = useTheme();

  const appScreenPaddingTop = "clamp(6rem, 24dvh, 13rem)";

  return (
    <>
      <HeroHeader />
      <main className="overflow-x-clip">
        <section style={{ paddingTop: appScreenPaddingTop }}>
          <div className="mx-auto w-full max-w-7xl px-6">
            <Reveal
              className="flex justify-between gap-6 max-md:flex-col md:items-end"
              duration={0.8}
              ease={[0.25, 0.1, 0.25, 1]}
              y={0}
            >
              <h1 className="max-w-2xl text-balance text-4xl font-medium tracking-tight md:text-5xl">
                <span className="block">让你随时随地使用 ComfyUI</span>
                <span className="block">把灵感快速变成作品</span>
              </h1>

              <div className="flex flex-wrap items-center gap-3">
                <Button
                  className="w-fit"
                  size="lg"
                  nativeButton={false}
                  render={<Link href="#install">Get Started</Link>}
                >
                  Get Started
                </Button>
                <Button className="w-fit" size="lg" variant="ghost">
                  Live Demo
                </Button>
              </div>
            </Reveal>

            <Reveal duration={0.8} ease={[0.25, 0.1, 0.25, 1]} y={0}>
              <div className="relative -mx-2 mt-8 overflow-hidden rounded-3xl bg-black p-2 sm:mt-12">
                <div className="bg-background ring-foreground/6.5 before:mask-radial-at-top-left before:mask-radial-from-65% before:mask-radial-[100%_60%] before:ring-foreground before:border-foreground/10 relative rounded-2xl p-2 ring before:absolute before:-inset-px before:z-10 before:size-56 before:rounded-tl-2xl before:border-l before:border-t">
                  <div className="bg-foreground/2 z-1 absolute inset-0 rounded-2xl"></div>
                  <Image
                    className="relative w-full rounded-2xl"
                    src={
                      theme === "dark"
                        ? "/images/app-dashboard-dark.png"
                        : "/images/app-dashboard-light.png"
                    }
                    alt={
                      theme === "dark"
                        ? "Pixoma 深色管理后台界面"
                        : "Pixoma 浅色管理后台界面"
                    }
                    width={1470}
                    height={956}
                  />
                </div>
              </div>
            </Reveal>
          </div>
        </section>
      </main>
    </>
  );
}
