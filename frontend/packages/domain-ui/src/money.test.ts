import { describe, expect, it } from "vitest";

import { addFen, formatFen } from "./money";

describe("formatFen", () => {
  it("formats integer fen as yuan with two decimals", () => {
    expect(formatFen(0)).toBe("¥0.00");
    expect(formatFen(1)).toBe("¥0.01");
    expect(formatFen(300)).toBe("¥3.00");
    expect(formatFen(1250)).toBe("¥12.50");
    expect(formatFen(123456)).toBe("¥1234.56");
  });

  it("formats negative fen with sign", () => {
    expect(formatFen(-500)).toBe("-¥5.00");
  });

  it("rejects non-integer input to forbid floating point money", () => {
    expect(() => formatFen(3.5)).toThrow(/integer/i);
    expect(() => formatFen(Number.NaN)).toThrow(/integer/i);
    expect(() => formatFen(Number.POSITIVE_INFINITY)).toThrow(/integer/i);
  });
});

describe("addFen", () => {
  it("adds integer fen amounts", () => {
    expect(addFen(300, 500)).toBe(800);
    expect(addFen(0, 0)).toBe(0);
  });

  it("rejects non-integer operands", () => {
    expect(() => addFen(1.2, 3)).toThrow(/integer/i);
    expect(() => addFen(3, Number.NaN)).toThrow(/integer/i);
  });
});
