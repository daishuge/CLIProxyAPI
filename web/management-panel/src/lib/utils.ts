import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

/**
 * Merge conditional class names and resolve Tailwind utility conflicts.
 * The single class-composition primitive used by every UI component.
 */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}

/** Format an integer with locale-aware grouping (defaults to compact grouping). */
export function formatNumber(value: number, locale?: string): string {
  if (!Number.isFinite(value)) return "—";
  return new Intl.NumberFormat(locale, { maximumFractionDigits: 0 }).format(value);
}

/** Format a 0..1 ratio as a percentage string with one decimal. */
export function formatPercent(ratio: number, locale?: string): string {
  if (!Number.isFinite(ratio)) return "—";
  return new Intl.NumberFormat(locale, {
    style: "percent",
    maximumFractionDigits: 1,
  }).format(ratio);
}

/** Compactly format large token/request counts (e.g. 12.4K, 3.1M). */
export function formatCompact(value: number, locale?: string): string {
  if (!Number.isFinite(value)) return "—";
  return new Intl.NumberFormat(locale, {
    notation: "compact",
    maximumFractionDigits: 1,
  }).format(value);
}
