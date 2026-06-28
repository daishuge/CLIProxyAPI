import { create } from "zustand";

export type ThemeMode = "light" | "dark" | "system";

const THEME_STORAGE = "ppap.theme";

function readStoredTheme(): ThemeMode {
  try {
    const value = window.localStorage.getItem(THEME_STORAGE);
    if (value === "light" || value === "dark" || value === "system") return value;
  } catch {
    // ignore
  }
  return "system";
}

function systemPrefersDark(): boolean {
  return typeof window !== "undefined" && window.matchMedia("(prefers-color-scheme: dark)").matches;
}

/** Apply the resolved theme by toggling the `.dark` class on <html>. */
function applyTheme(mode: ThemeMode): void {
  const root = document.documentElement;
  const dark = mode === "dark" || (mode === "system" && systemPrefersDark());
  root.classList.toggle("dark", dark);
}

interface ThemeState {
  mode: ThemeMode;
  /** Whether the effective theme currently renders dark. */
  isDark: boolean;
  setMode: (mode: ThemeMode) => void;
  /** Cycle between light and dark (ignores system). */
  toggle: () => void;
}

export const useThemeStore = create<ThemeState>((set, get) => ({
  mode: readStoredTheme(),
  isDark: false,
  setMode: (mode) => {
    try {
      window.localStorage.setItem(THEME_STORAGE, mode);
    } catch {
      // ignore
    }
    applyTheme(mode);
    set({ mode, isDark: document.documentElement.classList.contains("dark") });
  },
  toggle: () => {
    const next: ThemeMode = get().isDark ? "light" : "dark";
    get().setMode(next);
  },
}));

/** Initialise the theme on boot and keep `system` mode in sync with the OS. */
export function initTheme(): void {
  const store = useThemeStore.getState();
  applyTheme(store.mode);
  useThemeStore.setState({ isDark: document.documentElement.classList.contains("dark") });

  if (typeof window !== "undefined") {
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    media.addEventListener("change", () => {
      if (useThemeStore.getState().mode === "system") {
        applyTheme("system");
        useThemeStore.setState({ isDark: document.documentElement.classList.contains("dark") });
      }
    });
  }
}
