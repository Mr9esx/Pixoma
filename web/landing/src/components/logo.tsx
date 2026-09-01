import type { HTMLAttributes } from "react";
import { cn } from "@/lib/utils";
import { useTheme } from "@/context/theme-provider";

type LogoProps = Omit<HTMLAttributes<HTMLSpanElement>, "children"> & {
  alt?: string;
  className?: string;
  uniColor?: boolean;
};

export function Logo({ className, alt = "Pixoma" }: LogoProps) {
  const { theme } = useTheme();

  return (
    <img
      alt={alt}
      className={cn("h-[26px] w-auto shrink-0 object-contain", className)}
      src={
        theme === "dark" ? "/images/logo-dark.svg" : "/images/logo-light.svg"
      }
    />
  );
}

export function LogoIcon({ className, alt = "Pixoma" }: LogoProps) {
  return (
    <img
      src="/images/logo.png"
      alt={alt}
      className={cn("size-8 shrink-0 rounded-md object-contain", className)}
    />
  );
}
