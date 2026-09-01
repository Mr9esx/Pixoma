import { Check, Globe } from "lucide-react";
import { cn } from "@/lib/utils";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

const locales = [
  { value: "zh", label: "中文" },
  { value: "en", label: "English" },
] as const;

type Locale = (typeof locales)[number]["value"];

export function LocaleSwitcher({
  value = "zh",
  onChange,
  className,
}: {
  value?: Locale;
  onChange?: (locale: Locale) => void;
  className?: string;
}) {
  const current = locales.find(({ value: itemValue }) => itemValue === value);

  return (
    <div className={cn("relative isolate flex", className)}>
      <DropdownMenu>
        <DropdownMenuTrigger
          aria-label="切换语言 / Language"
          className="flex h-6 items-center gap-1.5 rounded-full bg-background px-2.5 text-xs ring-1 ring-border ring-inset outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          <Globe className="size-3.5 text-muted-foreground" />
          <span className="text-foreground">{current?.label}</span>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          {locales.map(({ value, label }) => {
            const isActive = value === current?.value;

            return (
              <DropdownMenuItem
                className="gap-2"
                key={value}
                onClick={() => onChange?.(value)}
              >
                <span>{label}</span>
                <Check
                  size={14}
                  className={cn("ms-auto", !isActive && "hidden")}
                />
              </DropdownMenuItem>
            );
          })}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
