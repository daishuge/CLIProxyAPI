import { Monitor, Moon, Sun } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useThemeStore, type ThemeMode } from "@/lib/theme";
import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui";

const ICONS: Record<ThemeMode, typeof Sun> = {
  light: Sun,
  dark: Moon,
  system: Monitor,
};

/** Theme selector cycling light / dark / system. */
export function ThemeToggle() {
  const { t } = useTranslation();
  const mode = useThemeStore((s) => s.mode);
  const isDark = useThemeStore((s) => s.isDark);
  const setMode = useThemeStore((s) => s.setMode);

  const Icon = mode === "system" ? Monitor : isDark ? Moon : Sun;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" aria-label={t("header.toggle_theme")}>
          <Icon className="size-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-36">
        {(["light", "dark", "system"] as const).map((value) => {
          const ItemIcon = ICONS[value];
          return (
            <DropdownMenuItem
              key={value}
              onSelect={() => setMode(value)}
              className={mode === value ? "text-primary" : undefined}
            >
              <ItemIcon className="size-4" />
              {t(`theme.${value}`)}
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
