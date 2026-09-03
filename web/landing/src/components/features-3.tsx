import { Card } from "@/components/ui/card";
import { Reveal } from "@/components/motion-primitives";
import { motion } from "motion/react";
import {
  CloudCog,
  Monitor,
  Rocket,
} from "lucide-react";
import Image from "next/image";
import { useState, type CSSProperties } from "react";
import { useTheme } from "@/context/theme-provider";
import { cn } from "@/lib/utils";

type DeployMode = "local" | "cloud";

const VIGNETTE_MASK =
  "radial-gradient(circle at center, transparent 0%, transparent 136px, black 100%)";

const IMAGE_CLASS = "absolute inset-0 size-full object-cover";

const CROSSFADE = { duration: 0.45, ease: "easeInOut" as const };

const BLUR_LAYER_STYLE = {
  filter: "blur(64px)",
  WebkitMaskMode: "alpha",
  maskMode: "alpha",
  WebkitMaskImage: VIGNETTE_MASK,
  maskImage: VIGNETTE_MASK,
} as CSSProperties;

const WHITE_TINT_STYLE = {
  backgroundColor: "rgba(255, 255, 255, 1)",
  WebkitMaskMode: "alpha",
  maskMode: "alpha",
  WebkitMaskImage: VIGNETTE_MASK,
  maskImage: VIGNETTE_MASK,
} as CSSProperties;

const DARK_TINT_STYLE = {
  backgroundColor: "rgba(0, 0, 0, 1)",
  WebkitMaskMode: "alpha",
  maskMode: "alpha",
  WebkitMaskImage: VIGNETTE_MASK,
  maskImage: VIGNETTE_MASK,
} as CSSProperties;

export default function Features() {
  const { theme } = useTheme();
  const [mode, setMode] = useState<DeployMode>("local");
  const showLocal = mode === "local";
  const showCloud = mode === "cloud";
  const isDark = theme === "dark";

  return (
    <section className="py-16 md:py-20">
      <div className="mx-auto max-w-7xl px-6">
        <Reveal>
          <h2 className="text-muted-foreground max-w-5xl text-balance text-4xl font-medium tracking-tight">
            <span className="text-foreground">
              文生图、文生视频、图片编辑、图生视频、TTS、人声模仿...
            </span>
            <br /> 创意随处轻松实现。
          </h2>
        </Reveal>
        <Reveal className="**:data-[slot=card]:bg-background mt-8 grid gap-x-3 gap-y-6 md:mt-16 md:grid-cols-2 lg:grid-cols-3">
          <div className="row-span-2 grid grid-cols-subgrid gap-4">
            <Card className="aspect-9/12 relative overflow-hidden">
              <Image
                src={isDark ? "/images/simplify-dark.png" : "/images/simplify-light.png"}
                alt="化繁为简：ComfyUI 节点被 Pixoma 简化为三段式工作流"
                width={1024}
                height={1365}
                className="absolute inset-0 size-full object-cover"
              />
            </Card>

            <p className="text-muted-foreground text-balance">
              <span className="text-foreground">化繁为简</span>
              <br />
              不再关心节点和参数，把灵感交给 Pixoma。
            </p>
          </div>

          <div className="row-span-2 grid grid-cols-subgrid gap-4">
            <Card className="aspect-9/12 bg-zinc-200! relative overflow-hidden">
              <DynamicIslandIllustration />
            </Card>

            <p className="text-muted-foreground text-balance">
              <span className="text-foreground">创意不设限</span>
              <br />
              随时随地在 Telegram 上继续你的灵感实现。
            </p>
          </div>

          <div className="row-span-2 grid grid-cols-subgrid gap-4">
            <Card className="aspect-9/12 relative overflow-hidden">
              <motion.div
                className="absolute inset-0"
                initial={false}
                animate={{ opacity: showLocal ? 1 : 0 }}
                transition={CROSSFADE}
              >
                <Image
                  src={isDark ? "/images/deploy-local-dark.jpg" : "/images/deploy-local.jpg"}
                  alt="单机部署示意"
                  width={1280}
                  height={1024}
                  className={IMAGE_CLASS}
                />
              </motion.div>
              <motion.div
                className="absolute inset-0"
                initial={false}
                animate={{ opacity: showCloud ? 1 : 0 }}
                transition={CROSSFADE}
              >
                <Image
                  src={isDark ? "/images/deploy-cloud-dark.jpg" : "/images/deploy-cloud.jpg"}
                  alt="云端多节点部署示意"
                  width={1280}
                  height={1024}
                  className={IMAGE_CLASS}
                />
              </motion.div>

              <motion.div
                className="absolute inset-0 z-[1]"
                initial={false}
                animate={{ opacity: showLocal ? 1 : 0 }}
                transition={CROSSFADE}
                aria-hidden
              >
                <Image
                  src={isDark ? "/images/deploy-local-dark.jpg" : "/images/deploy-local.jpg"}
                  alt=""
                  width={1280}
                  height={1024}
                  className={IMAGE_CLASS}
                  style={BLUR_LAYER_STYLE}
                />
              </motion.div>
              <motion.div
                className="absolute inset-0 z-[1]"
                initial={false}
                animate={{ opacity: showCloud ? 1 : 0 }}
                transition={CROSSFADE}
                aria-hidden
              >
                <Image
                  src={isDark ? "/images/deploy-cloud-dark.jpg" : "/images/deploy-cloud.jpg"}
                  alt=""
                  width={1280}
                  height={1024}
                  className={IMAGE_CLASS}
                  style={BLUR_LAYER_STYLE}
                />
              </motion.div>

              <div
                aria-hidden
                className="pointer-events-none absolute inset-0 z-[1]"
                style={isDark ? DARK_TINT_STYLE : WHITE_TINT_STYLE}
              />

              <DeployModePicker mode={mode} onChange={setMode} />
            </Card>

            <p className="text-muted-foreground text-balance">
              <span className="text-foreground">部署简单</span>
              <br />
              支持本地运行，依赖最小化。也支持云端多节点部署。
            </p>
          </div>
        </Reveal>
      </div>
    </section>
  );
}

function DeployModePicker({
  mode,
  onChange,
}: {
  mode: DeployMode;
  onChange: (mode: DeployMode) => void;
}) {
  return (
    <div className="z-[2] absolute bottom-3 left-3 flex w-[min(15rem,calc(100%-1.5rem))] flex-col gap-2">
      <div className="bg-background/40 ring-foreground/10 flex w-fit items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium ring backdrop-blur-md">
        <Rocket className="opacity-70 size-3.5" />
        部署方式
      </div>

      <div className="bg-background/70 ring-foreground/10 rounded-2xl p-1 ring backdrop-blur-md">
        <button
          type="button"
          onClick={() => onChange("local")}
          aria-pressed={mode === "local"}
          className={cn(
            "flex w-full cursor-pointer items-center gap-2 rounded-xl px-2.5 py-1.5 text-left transition-colors",
            mode === "local" ? "bg-foreground/8" : "hover:bg-foreground/5",
          )}
        >
          <Monitor className="size-3.5 shrink-0 text-foreground/80" />
          <div className="min-w-0 flex-1 leading-tight">
            <div className="text-xs font-medium text-foreground">单机部署</div>
            <div className="truncate text-[10px] text-foreground/55">一台本地电脑即可运行</div>
          </div>
        </button>

        <button
          type="button"
          onClick={() => onChange("cloud")}
          aria-pressed={mode === "cloud"}
          className={cn(
            "flex w-full cursor-pointer items-center gap-2 rounded-xl px-2.5 py-1.5 text-left transition-colors",
            mode === "cloud" ? "bg-foreground/8" : "hover:bg-foreground/5",
          )}
        >
          <CloudCog className="size-3.5 shrink-0 text-foreground/80" />
          <div className="min-w-0 flex-1 leading-tight">
            <div className="text-xs font-medium text-foreground">云端多节点</div>
            <div className="truncate text-[10px] text-foreground/55">云端多节点任务调度</div>
          </div>
        </button>
      </div>
    </div>
  );
}


function DynamicIslandIllustration() {
  return (
    <div
      aria-hidden
      className="z-1 bg-black/2.5 absolute inset-x-8 bottom-0 mx-auto mt-auto h-2/3 w-10/12 origin-bottom scale-95 rounded-t-[4rem] border border-black/5 px-4 pt-4"
    >
      <div className="h-full overflow-hidden rounded-t-[3rem] bg-white p-3 ring ring-black/10">
        <div className="relative">
          <Image
            src="https://images.unsplash.com/photo-1782366951390-d6798e902db7?q=80&w=1015&auto=format&fit=crop&ixlib=rb-4.1.0&ixid=M3wxMjA3fDB8MHxwaG90by1wYWdlfHx8fGVufDB8fHx8fA%3D%3D"
            alt="Théo Balick"
            width={500}
            height={500}
            className="absolute inset-0 top-0 size-full object-cover opacity-45 blur-xl contrast-200"
          />
          <div className="relative rounded-[2.25rem] bg-white p-2 ring ring-black/10">
            <div className="flex gap-2">
              <div className="size-18 relative overflow-hidden rounded-[1.75rem] before:absolute before:inset-0 before:rounded-[1.75rem] before:border before:border-black/20">
                <Image
                  src="https://images.unsplash.com/photo-1782366951390-d6798e902db7?q=80&w=1015&auto=format&fit=crop&ixlib=rb-4.1.0&ixid=M3wxMjA3fDB8MHxwaG90by1wYWdlfHx8fGVufDB8fHx8fA%3D%3D"
                  alt="Théo Balick"
                  width={136}
                  height={136}
                />
              </div>
              <div className="py-1 pr-4">
                <div className="text-sm font-medium text-black">
                  Théo Balick
                </div>
                <div className="mt-1.5 flex items-center gap-3">
                  <div>
                    <div className="text-xs text-black/50">Expenses</div>
                    <div className="mt-0.5 text-sm font-semibold text-black">
                      $32.65k
                    </div>
                  </div>
                  <div className="bg-border h-7 w-px" />
                  <div>
                    <div className="text-xs text-black/50">Income</div>
                    <div className="mt-0.5 text-sm font-semibold text-black">
                      $2.65k
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
