import { expect, it } from "vitest";
import { ADMIN_SCREENS, MINI_APP_SCREENS, REQUIRED_STATES } from "../src/domain/catalog";
import { adminScreenSections, miniAppScreenSections, stateDescription } from "../src/domain/screen-content";

it("provides Chinese business content for every generated screen and state", () => {
  for (const screen of MINI_APP_SCREENS) expect(miniAppScreenSections(screen.key).length).toBeGreaterThanOrEqual(4);
  for (const screen of ADMIN_SCREENS) expect(adminScreenSections(screen.key).length).toBeGreaterThanOrEqual(4);
  for (const state of REQUIRED_STATES) expect(stateDescription(state)).toMatch(/｜/);
  expect(miniAppScreenSections("screen/miniapp/home").join(" ")).toContain("附近寄存点");
  expect(adminScreenSections("screen/admin/dashboard").join(" ")).toContain("设备在线率");
});
