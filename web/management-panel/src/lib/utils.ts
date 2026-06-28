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

/** Human-readable byte size (e.g. 1.2 KB, 3.4 MB). */
export function formatBytes(bytes: number, locale?: string): string {
  if (!Number.isFinite(bytes) || bytes < 0) return "—";
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const exponent = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  const value = bytes / Math.pow(1024, exponent);
  return `${new Intl.NumberFormat(locale, { maximumFractionDigits: 1 }).format(value)} ${units[exponent]}`;
}

/** Locale-aware date-time, tolerant of empty/invalid input. */
export function formatDateTime(value: string | number | undefined | null, locale?: string): string {
  if (value === undefined || value === null || value === "") return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short" }).format(date);
}

/** Mask a secret, keeping a short readable suffix (e.g. ••••••3f9a). */
export function maskSecret(secret: string, visible = 4): string {
  const trimmed = secret.trim();
  if (trimmed.length <= visible) return "•".repeat(Math.max(trimmed.length, 4));
  return `${"•".repeat(8)}${trimmed.slice(-visible)}`;
}

/** Trigger a browser download for an in-memory blob. */
export function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}
