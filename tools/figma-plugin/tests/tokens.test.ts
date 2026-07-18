import { describe, expect, it } from "vitest";
import {
  COLOR_TOKENS,
  DIMENSION_TOKENS,
  tokenCssSyntax,
  validateTokens
} from "../src/domain/tokens";

describe("design tokens", () => {
  it("uses unique slash-delimited names", () => {
    expect(validateTokens()).toEqual([]);
  });

  it("contains project semantic states", () => {
    const names = COLOR_TOKENS.map((token) => token.name);
    expect(names).toEqual(
      expect.arrayContaining([
        "color/action/primary",
        "color/status/success",
        "color/status/warning",
        "color/status/danger",
        "color/status/offline",
        "color/status/processing"
      ])
    );
  });

  it("uses a 4px spacing scale", () => {
    expect(
      DIMENSION_TOKENS.filter((token) => token.name.startsWith("space/")).every(
        (token) => Number(token.value) % 4 === 0
      )
    ).toBe(true);
  });

  it("derives stable web syntax", () => {
    expect(tokenCssSyntax("color/bg/primary")).toBe("var(--color-bg-primary)");
  });
});
