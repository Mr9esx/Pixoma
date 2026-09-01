import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Reveal } from "@/components/motion-primitives";

export default function CallToAction() {
  return (
    <section className="py-16 md:py-20">
      <div className="mx-auto max-w-7xl px-6">
        <Reveal className="flex items-center justify-center gap-6 max-lg:flex-col max-lg:text-center lg:justify-between">
          <h2 className="max-w-4xl text-balance text-5xl font-semibold tracking-tight xl:text-6xl">
            加入我们的开源项目
          </h2>

          <Button
            size="lg"
            nativeButton={false}
            render={
              <Link
                href="https://github.com/Mr9esx/Pixoma"
                target="_blank"
                rel="noopener noreferrer"
              >
                View on Github
              </Link>
            }
          >
            View on Github
          </Button>
        </Reveal>
      </div>
    </section>
  );
}
