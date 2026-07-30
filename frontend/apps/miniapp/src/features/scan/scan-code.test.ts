import { describe, expect, it } from "vitest";
import { createScanCode } from "./scan-code";

describe("createScanCode", () => {
  it("returns the trimmed scan value on success", async () => {
    const scanCode = createScanCode({ scan: async () => ({ result: " A-01 " }) });
    const result = await scanCode();
    expect(result).toEqual({ ok: true, value: "A-01", cancelled: false });
  });

  it("treats a cancelled scan as not-ok without surfacing an error", async () => {
    const scanCode = createScanCode({
      scan: async () => {
        throw new Error("scanCode:fail cancel");
      }
    });
    const result = await scanCode();
    expect(result.ok).toBe(false);
    expect(result.cancelled).toBe(true);
    expect(result.value).toBe("");
  });

  it("surfaces a non-cancel error message", async () => {
    const scanCode = createScanCode({
      scan: async () => {
        throw new Error("scanCode is not supported in this environment");
      }
    });
    const result = await scanCode();
    expect(result.ok).toBe(false);
    expect(result.cancelled).toBe(false);
    expect(result.error).toBe("scanCode is not supported in this environment");
  });
});
