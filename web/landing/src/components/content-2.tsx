import { Cpu, Zap } from "lucide-react";
import { Reveal } from "@/components/motion-primitives";

export default function ContentSection() {
  return (
    <section className="py-16 md:py-20">
      <div className="mx-auto max-w-7xl px-6">
        <div className="grid gap-4 md:grid-cols-2 md:gap-6 lg:gap-12">
          <Reveal>
            <h2 className="max-w-md text-balance text-4xl font-medium tracking-tight lg:text-5xl">
              Pixoma 是一个什么样的项目？
            </h2>
          </Reveal>
          <Reveal className="space-y-6 lg:space-y-6" delay={0.08}>
            <p className="text-muted-foreground text-balance text-lg">
            ComfyUI 高度依赖本地电脑环境，出门在外就没法使用工作流。
            Pixoma 是 ComfyUI 的代理抽象层，把你的工作流打包成可调用服务。随时随地可以调用，团队内也能一键共享复用，不再被设备束缚。
            </p>

            <div className="grid gap-4 sm:grid-cols-2">
              <p className="text-muted-foreground text-balance text-lg">
                <span className="text-foreground font-medium">
                  <Zap className="inline size-4 -translate-y-0.5" /> 一次构建，随处可用。
                </span>{" "}
              </p>

              <p className="text-muted-foreground text-balance text-lg">
                <span className="text-foreground font-medium">
                  <Cpu className="inline size-4 -translate-y-0.5" /> 对外分享 AI 能力。
                </span>{" "}
              </p>
            </div>
          </Reveal>
        </div>
      </div>
    </section>
  );
}
