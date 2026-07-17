import { describe, expect, it } from "vitest";
import { COMPONENT_KEYS } from "../src/domain/catalog";
import { runComponents } from "../src/stages/components";
import { runFoundations } from "../src/stages/foundations";
import { FakeFigmaPort } from "./fake-figma-port";

describe("Components stage", () => {
  it("requires foundations and creates every catalog component once", async () => {
    const port = new FakeFigmaPort();
    await expect(runComponents(port)).rejects.toThrow("Run Foundations first");

    await runFoundations(port);
    const first = await runComponents(port);
    expect(first.status).toBe("success");
    expect(port.componentKeys()).toEqual(expect.arrayContaining(COMPONENT_KEYS.map((key) => `component/${key}`)));
    expect(port.resource("component/button")).toMatchObject({ width: 160, height: 48, visualKind: "button" });
    expect(port.resource("component/input")).toMatchObject({ width: 320, height: 96, visualKind: "input" });
    expect(port.resource("component/site-card")).toMatchObject({ width: 360, height: 180, visualKind: "site-card" });
    expect(port.resource("component/data-table")).toMatchObject({ width: 640, height: 280, visualKind: "data-table" });
    const count = port.resourceCount();

    const second = await runComponents(port);
    expect(second.counts.updated).toBeGreaterThan(0);
    expect(port.resourceCount()).toBe(count);
    expect(await port.hasStageMarker("components")).toBe(true);
  });
});
