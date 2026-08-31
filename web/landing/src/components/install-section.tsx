import { useState } from "react";
import { Check, Copy } from "lucide-react";
import { Button } from "@/components/ui/button";

const INSTALL_COMMAND =
  "curl -fsSL https://pixoma.miaoplus.com/install.sh | sh";

export default function InstallSection() {
  const [copied, setCopied] = useState(false);

  const copyCommand = async () => {
    await navigator.clipboard.writeText(INSTALL_COMMAND);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1600);
  };

  return (
    <section id="install" className="scroll-mt-24">
      <div className="py-16 md:py-20">
        <div className="mx-auto w-full max-w-5xl px-6">
          <h2 className="text-foreground text-4xl font-semibold">
            一条命令开始使用
          </h2>
          <div className="mt-8">
            <div className="bg-background shadow-foreground/5 ring-foreground/5 w-full overflow-hidden rounded-xl p-6 shadow-md ring-1">
              <div className="mb-6 flex items-center justify-between gap-4">
                <div className="text-lg font-medium">安装 Pixoma</div>
                <Button size="sm" variant="outline" onClick={copyCommand}>
                  {copied ? <Check /> : <Copy />}
                  {copied ? "已复制" : "复制"}
                </Button>
              </div>

              <code className="bg-foreground/5 ring-foreground/10 block overflow-x-auto rounded-2xl p-4 font-mono text-sm ring-1">
                {INSTALL_COMMAND}
              </code>
            </div>

            <p className="text-muted-foreground mt-4 text-sm">
              支持 macOS、Linux 与 Windows。
            </p>
          </div>
        </div>
      </div>
    </section>
  );
}
