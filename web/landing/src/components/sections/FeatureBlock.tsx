import { type ReactNode } from "react";
import { Reveal } from "@/components/motion/Reveal";

type FeatureBlockProps = {
  title: string;
  desc: string;
  children: ReactNode;
};

export function FeatureBlock({ title, desc, children }: FeatureBlockProps) {
  return (
    <Reveal className="flex flex-col gap-5">
      <div>
        <h3 className="text-2xl font-semibold tracking-tight text-foreground">
          {title}
        </h3>
        <p className="mt-2 max-w-md text-muted-foreground">{desc}</p>
      </div>
      <div className="aspect-[4/3] overflow-hidden rounded-xl border border-border bg-card">
        {children}
      </div>
    </Reveal>
  );
}
