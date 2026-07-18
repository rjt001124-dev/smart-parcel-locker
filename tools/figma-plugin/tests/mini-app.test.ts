import { expect, it } from "vitest";
import { MINI_APP_SCREENS, REQUIRED_STATES } from "../src/domain/catalog";
import { runComponents } from "../src/stages/components";
import { runFoundations } from "../src/stages/foundations";
import { runMiniApp } from "../src/stages/mini-app";
import { FakeFigmaPort } from "./fake-figma-port";
import { stateDescription } from "../src/domain/screen-content";

it("generates all Mini App screens and required states idempotently", async () => {
  const port = new FakeFigmaPort();
  await expect(runMiniApp(port)).rejects.toThrow("Run Foundations first");
  await runFoundations(port);
  await runComponents(port);

  expect((await runMiniApp(port)).status).toBe("success");
  for (const screen of MINI_APP_SCREENS) expect(port.resource(screen.key)).toMatchObject({ width: 375, height: 812 });
  expect(port.resource("screen/miniapp/state-matrix")?.states).toEqual(REQUIRED_STATES.map(stateDescription));
  const count = port.resourceCount();
  await runMiniApp(port);
  expect(port.resourceCount()).toBe(count);
});
