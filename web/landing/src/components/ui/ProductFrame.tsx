import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

type ProductFrameProps = {
  children: ReactNode;
  className?: string;
  innerClassName?: string;
  stacked?: boolean;
};

export function ProductFrame({
  children,
  className,
  innerClassName,
  stacked = false,
}: ProductFrameProps) {
  return (
    <div className={cn(stacked && "product-stack", className)}>
      {stacked ? (
        <div className="product-stack-plate" aria-hidden="true" />
      ) : null}
      <div className="product-frame">
        <div className={cn("product-frame-inner", innerClassName)}>
          {children}
        </div>
      </div>
    </div>
  );
}
