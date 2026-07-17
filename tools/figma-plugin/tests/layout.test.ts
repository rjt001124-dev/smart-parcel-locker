import { expect, it } from "vitest";
import { COMPONENT_KEYS, MINI_APP_SCREENS } from "../src/domain/catalog";
import { runComponents } from "../src/stages/components";
import { runFoundations } from "../src/stages/foundations";
import { runMiniApp } from "../src/stages/mini-app";
import { FakeFigmaPort } from "./fake-figma-port";

it("places design-system resources and screens in reviewable grids", async () => {
  const port = new FakeFigmaPort();
  await runFoundations(port);
  await runComponents(port);
  await runMiniApp(port);

  expect(port.resource("screen/foundations/documentation")).toMatchObject({ x: 80, y: 80 });
  const componentPositions = COMPONENT_KEYS.map((key) => {
    const item = port.resource(`component/${key}`);
    return `${item?.x},${item?.y}`;
  });
  expect(new Set(componentPositions).size).toBe(COMPONENT_KEYS.length);
  expect(Math.max(...COMPONENT_KEYS.map((key) => Number(port.resource(`component/${key}`)?.x)))).toBeLessThan(3000);

  const screenPositions = MINI_APP_SCREENS.map((screen) => {
    const item = port.resource(screen.key);
    return `${item?.x},${item?.y}`;
  });
  expect(new Set(screenPositions).size).toBe(MINI_APP_SCREENS.length);
});
