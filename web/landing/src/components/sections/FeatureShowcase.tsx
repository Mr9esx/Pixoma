import type { ReactNode } from "react";
import { ProductFrame } from "@/components/ui/ProductFrame";
import { cn } from "@/lib/utils";

type FeatureShowcaseProps = {
  title: string;
  description: string;
  offset?: "none" | "down" | "mid";
  children: ReactNode;
};

export function FeatureShowcase({
  title,
  description,
  offset = "none",
  children,
}: FeatureShowcaseProps) {
  return (
    <article
      className={cn(
        "flex min-w-0 flex-col gap-5",
        offset === "down" && "lg:translate-y-6",
        offset === "mid" && "lg:translate-y-2",
      )}
    >
      <ProductFrame>
        <div className="flex min-h-80 flex-col">
          <div className="min-h-0 flex-1">{children}</div>
          <div className="border-t border-border bg-muted/60 p-5">
            <h3 className="text-xl font-semibold tracking-tight">{title}</h3>
            <p className="mt-2 text-sm leading-6 text-muted-foreground">
              {description}
            </p>
          </div>
        </div>
      </ProductFrame>
    </article>
  );
}
