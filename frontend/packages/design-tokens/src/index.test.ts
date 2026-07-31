import { describe, expect, it } from "vitest";
import { colors, radii, spacing } from "./index";

describe("approved design tokens", () => {
  it("keeps the approved blue and pure-white surfaces", () => {
    expect(colors.primary).toBe("#1769E0");
    expect(colors.surface).toBe("#FFFFFF");
    expect(colors.background).toBe("#F7F9FC");
  });

  it("uses the 8px spacing baseline and restrained radii", () => {
    expect(spacing[2]).toBe(8);
    expect(radii.card).toBe(16);
    expect(radii.button).toBe(12);
  });
});
