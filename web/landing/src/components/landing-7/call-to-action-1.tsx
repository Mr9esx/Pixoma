import { Button } from "@/components/landing-7/ui/button";
import Link from "next/link";

export default function CallToAction() {
  return (
    <section className="py-16 md:py-20">
      <div className="mx-auto max-w-7xl px-6">
        <div className="mx-auto max-w-4xl text-center">
          <h2 className="text-balance text-4xl font-semibold tracking-tight lg:text-5xl xl:text-6xl">
            让你随时随地
          </h2>

          <div className="mt-8 flex flex-wrap justify-center gap-3">
            <Button
              size="lg"
              nativeButton={false}
              render={
                <Link
                  href="https://github.com/Mr9esx/Pixoma"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Github
                </Link>
              }
            />

            <Button
              size="lg"
              variant="outline"
              nativeButton={false}
              render={<Link href="#features">查看功能</Link>}
            />
          </div>
        </div>
      </div>
    </section>
  );
}
