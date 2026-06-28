import type { KeyValuePair } from "./key-value-field";

/** Convert a header record into editable pairs. */
export function recordToPairs(record?: Record<string, string>): KeyValuePair[] {
  if (!record) return [];
  return Object.entries(record).map(([key, value]) => ({ key, value }));
}

/** Serialize editable pairs back into a record, dropping blank keys. */
export function pairsToRecord(pairs: KeyValuePair[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (const { key, value } of pairs) {
    const trimmed = key.trim();
    if (trimmed) out[trimmed] = value;
  }
  return out;
}
