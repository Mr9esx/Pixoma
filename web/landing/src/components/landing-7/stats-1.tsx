export default function StatsSection() {
  return (
    <section className="py-16 md:py-20">
      <div className="mx-auto max-w-7xl px-6">
        <div className="grid gap-4 md:grid-cols-2 md:gap-6">
          <h2 className="text-muted-foreground max-w-4xl text-balance text-4xl font-medium tracking-tight lg:text-5xl">
            <span className="text-foreground">更快出图。</span> <br />{" "}
            让每个节点都清晰可控。
          </h2>
          <div className="flex flex-col gap-32 md:mx-auto xl:gap-44">
            <p className="text-muted-foreground text-balance text-lg">
              当 Case、对话、任务和节点状态都在一个工作台里，创作节奏会明显
              变快。Pixoma 让你专注于选择风格与调整灵感，把执行交给可靠的
              运行时。
            </p>

            <div className="grid gap-12 md:grid-cols-3 md:gap-12">
              <div className="space-y-3 border-t pt-6">
                <div className="text-4xl font-semibold tracking-tight">21k</div>
                <p className="text-muted-foreground">任务可追踪</p>
              </div>
              <div className="space-y-3 border-t pt-6">
                <div className="text-4xl font-semibold tracking-tight">22m</div>
                <p className="text-muted-foreground">节点信号</p>
              </div>
              <div className="space-y-3 border-t pt-6">
                <div className="text-4xl font-semibold tracking-tight">
                  +500
                </div>
                <p className="text-muted-foreground">创作者</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
