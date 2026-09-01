import { useState } from "react";
import { AnimatePresence, motion } from "motion/react";
import { Check } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Reveal } from "@/components/motion-primitives";

const INSTALL_COMMAND =
  "curl -fsSL https://pixoma.miaoplus.com/install.sh | sh";
const INSTALL_WINDOWS_COMMAND =
  "irm https://pixoma.miaoplus.com/install.ps1 | iex";

type InstallPlatform = "unix" | "windows";

export default function InstallSection() {
  const [copied, setCopied] = useState(false);
  const [platform, setPlatform] = useState<InstallPlatform>(() =>
    /Windows/i.test(navigator.userAgent) ? "windows" : "unix",
  );
  const installCommand =
    platform === "windows" ? INSTALL_WINDOWS_COMMAND : INSTALL_COMMAND;

  const copyCommand = async () => {
    await navigator.clipboard.writeText(installCommand);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1600);
  };

  return (
    <section id="install" className="scroll-mt-24">
      <div className="py-16 md:py-20">
        <div className="mx-auto w-full max-w-7xl px-6">
          <Reveal>
            <h2 className="text-foreground text-4xl font-semibold">
              一条命令开始使用
            </h2>
          </Reveal>
          <Reveal className="mt-8" delay={0.08}>
            <div className="relative">
              <div className="bg-background ring-foreground/10 relative w-full overflow-hidden rounded-2xl ring-1">
                <div className="border-foreground/10 bg-foreground/3 flex items-center justify-between gap-4 border-b p-3">
                  <div className="flex gap-1">
                    <Button
                      size="sm"
                      variant={platform === "unix" ? "secondary" : "ghost"}
                      className={
                        platform === "unix"
                          ? "text-foreground"
                          : "text-muted-foreground"
                      }
                      aria-pressed={platform === "unix"}
                      onClick={() => setPlatform("unix")}
                    >
                      macOS / Linux
                    </Button>
                    <Button
                      size="sm"
                      variant={platform === "windows" ? "secondary" : "ghost"}
                      className={
                        platform === "windows"
                          ? "text-foreground"
                          : "text-muted-foreground"
                      }
                      aria-pressed={platform === "windows"}
                      onClick={() => setPlatform("windows")}
                    >
                      Windows
                    </Button>
                  </div>
                  <Button
                    size="sm"
                    variant="outline"
                    className="w-[5.5rem]"
                    onClick={copyCommand}
                  >
                    <AnimatePresence initial={false} mode="wait">
                      <motion.span
                        animate={{ opacity: 1, y: 0 }}
                        exit={{ opacity: 0, y: -4 }}
                        initial={{ opacity: 0, y: 4 }}
                        key={copied ? "copied" : "copy"}
                        transition={{ duration: 0.16 }}
                    >
                        {copied ? (
                          <span className="flex items-center gap-0.5">
                            <Check className="size-3.5"/>
                            已复制
                          </span>
                        ) : (
                          "复制命令"
                        )}
                      </motion.span>
                    </AnimatePresence>
                  </Button>
                </div>

                <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 p-6">
                  <span className="text-foreground/35 pt-0.5 font-mono text-xs">
                    01
                  </span>
                  <span className="text-muted-foreground text-[0.7rem] uppercase tracking-[0.08em]">
                    Install Pixoma
                  </span>
                  <span className="text-foreground/35 font-mono text-sm">
                    ~
                  </span>
                  <div className="bg-foreground/3 mt-1 overflow-x-auto rounded-xl px-4 py-3">
                    <code className="text-foreground/85 block font-mono text-sm whitespace-nowrap">
                      {installCommand}
                    </code>
                  </div>
                </div>
              </div>
            </div>

            <p className="text-muted-foreground mt-4 text-sm">
              {platform === "windows"
                ? "需要 Windows 10 或更新版本的 PowerShell。"
                : "支持 macOS 与 Linux。"}
            </p>
          </Reveal>
        </div>
      </div>
    </section>
  );
}
