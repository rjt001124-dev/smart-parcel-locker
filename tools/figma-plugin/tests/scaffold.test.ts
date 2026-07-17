import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

describe("plugin scaffold", () => {
  it("declares the generated main and UI files", () => {
    const manifest = JSON.parse(readFileSync("manifest.json", "utf8"));

    expect(manifest.name).toBe("Smart Parcel Locker Generator");
    expect(manifest.main).toBe("dist/code.js");
    expect(manifest.ui).toBe("dist/ui.html");
    expect(manifest.editorType).toEqual(["figma"]);
  });
});
