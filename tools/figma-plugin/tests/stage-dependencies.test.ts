import { describe, expect, it } from "vitest";
import { assertStageReady } from "../src/stages/dependencies";
import { FakeFigmaPort } from "./fake-figma-port";

describe("stage dependencies", () => {
  it("rejects components before foundations", async () => {
    const port = new FakeFigmaPort();

    await expect(assertStageReady(port, "components")).rejects.toThrow("Run Foundations first");
  });

  it("accepts components after foundations", async () => {
    const port = new FakeFigmaPort();
    await port.setStageMarker("foundations");

    await expect(assertStageReady(port, "components")).resolves.toBeUndefined();
  });

  it("requires components before screen stages", async () => {
    const port = new FakeFigmaPort();
    await port.setStageMarker("foundations");

    await expect(assertStageReady(port, "mini-app")).rejects.toThrow("Run Components first");
    await expect(assertStageReady(port, "admin-web")).rejects.toThrow("Run Components first");
  });
});
