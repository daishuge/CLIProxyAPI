import { describe, expect, it } from "vitest";
import { cn, formatPercent, formatNumber } from "./utils";

describe("cn", () => {
  it("merges class names and dedupes conflicting tailwind utilities", () => {
    expect(cn("px-2", "px-4")).toBe("px-4");
    expect(cn("text-sm", false, undefined, "font-medium")).toBe("text-sm font-medium");
  });
});

describe("formatNumber", () => {
  it("groups integers and guards against non-finite input", () => {
    expect(formatNumber(1234567, "en")).toBe("1,234,567");
    expect(formatNumber(Number.NaN)).toBe("—");
  });
});

describe("formatPercent", () => {
  it("formats a 0..1 ratio as a percentage", () => {
    expect(formatPercent(0.5, "en")).toBe("50%");
    expect(formatPercent(Number.POSITIVE_INFINITY)).toBe("—");
  });
});
