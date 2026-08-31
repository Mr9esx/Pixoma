import type { ImgHTMLAttributes } from "react";
import { cn } from "@/lib/utils";

type LogoProps = ImgHTMLAttributes<HTMLImageElement> & {
  className?: string;
  uniColor?: boolean;
};

export function Logo({ className, alt = "Pixoma" }: LogoProps) {
  return (
    <img
      src="/images/logo.png"
      alt={alt}
      className={cn("size-8 shrink-0 rounded-xl object-contain", className)}
    />
  );
}

export function LogoIcon({ className, alt = "Pixoma" }: LogoProps) {
  return (
    <img
      src="/images/logo.png"
      alt={alt}
      className={cn("size-8 shrink-0 rounded-xl object-contain", className)}
    />
  );
}
