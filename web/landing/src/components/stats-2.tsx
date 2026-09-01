import {
  CountUp,
  Reveal,
  StaggerGroup,
  StaggerItem,
} from "@/components/motion-primitives";

export default function StatsSection() {
  return (
    <section className="py-16 md:py-20">
      <div className="mx-auto max-w-7xl px-6">
        <Reveal>
          <p className="text-muted-foreground max-w-4xl text-balance text-4xl font-medium tracking-tight lg:text-5xl">
            <span className="text-foreground">Scale with confidence.</span>{" "}
            Handle thousands of transactions per second.
          </p>
        </Reveal>

        <StaggerGroup className="mt-32 grid gap-12 md:grid-cols-3 xl:mt-44">
          <StaggerItem className="space-y-3 border-t pt-6">
            <div className="text-5xl font-semibold tracking-tight">
              <CountUp prefix="+" value={21200} />
            </div>
            <p className="text-muted-foreground">Stars on GitHub</p>
          </StaggerItem>
          <StaggerItem className="space-y-3 border-t pt-6">
            <div className="text-5xl font-semibold tracking-tight">
              <CountUp suffix=" Million" value={22} />
            </div>
            <p className="text-muted-foreground">Active Users</p>
          </StaggerItem>
          <StaggerItem className="space-y-3 border-t pt-6">
            <div className="text-5xl font-semibold tracking-tight">
              <CountUp prefix="+" value={500} />
            </div>
            <p className="text-muted-foreground">Powered Apps</p>
          </StaggerItem>
        </StaggerGroup>
      </div>
    </section>
  );
}
