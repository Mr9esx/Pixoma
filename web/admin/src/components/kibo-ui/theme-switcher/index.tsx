"use client";

import { Moon, Sun } from "lucide-react";
import { motion } from "motion/react";
import { cn } from "@/lib/utils";

const themes = [
  {
    key: "light",
    icon: Sun,
    label: "Light theme",
  },
  {
    key: "dark",
    icon: Moon,
    label: "Dark theme",
  },
];

export type ThemeSwitcherProps = {
  value?: "light" | "dark";
  onChange?: (theme: "light" | "dark") => void;
  className?: string;
};

export const ThemeSwitcher = ({
  value,
  onChange,
  className,
}: ThemeSwitcherProps) => {
  return (
    <div
      className={cn(
        "relative isolate flex h-6 rounded-full bg-background p-0.5 ring-1 ring-inset ring-border",
        className
      )}
    >
      {themes.map(({ key, icon: Icon, label }) => {
        const isActive = value === key;

        return (
          <button
            aria-label={label}
            className="relative h-5 w-5 rounded-full"
            key={key}
            onClick={() => onChange?.(key as "light" | "dark")}
            type="button"
          >
            {isActive && (
              <motion.div
                className="absolute inset-0 rounded-full bg-secondary"
                layoutId="activeTheme"
                transition={{ type: "spring", duration: 0.5 }}
              />
            )}
            <Icon
              className={cn(
                "relative z-10 m-auto size-3.5",
                isActive ? "text-foreground" : "text-muted-foreground"
              )}
            />
          </button>
        );
      })}
    </div>
  );
};
