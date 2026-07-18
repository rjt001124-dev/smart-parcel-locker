import { describe, expect, it } from "vitest";
import { runFoundations } from "../src/stages/foundations";
import { FakeFigmaPort } from "./fake-figma-port";

describe("Foundations stage", () => {
  it("creates foundations once and updates on rerun", async () => {
    const port = new FakeFigmaPort();

    const first = await runFoundations(port);
    expect(first.status).toBe("success");
    expect(first.counts.created).toBeGreaterThan(0);
    const sizeAfterFirstRun = port.resourceCount();

    const second = await runFoundations(port);
    expect(second.counts.updated).toBeGreaterThan(0);
    expect(port.resourceCount()).toBe(sizeAfterFirstRun);
    expect(await port.hasStageMarker("foundations")).toBe(true);
  });

  it("stops before writes when a required font is unavailable", async () => {
    const port = new FakeFigmaPort();
    port.setFonts([{ family: "Inter", style: "Regular" }]);

    await expect(runFoundations(port)).rejects.toThrow("Missing required fonts: Inter Medium, Inter Semi Bold, Inter Bold");
    expect(port.resourceCount()).toBe(0);
  });
});
