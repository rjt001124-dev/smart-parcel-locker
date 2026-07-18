import { expect, it } from "vitest";
import { ADMIN_SCREENS } from "../src/domain/catalog";
import { runAdminWeb } from "../src/stages/admin-web";
import { runComponents } from "../src/stages/components";
import { runFoundations } from "../src/stages/foundations";
import { FakeFigmaPort } from "./fake-figma-port";

it("generates every Admin Web screen and high-risk confirmations idempotently", async () => {
  const port = new FakeFigmaPort();
  await expect(runAdminWeb(port)).rejects.toThrow("Run Foundations first");
  await runFoundations(port);
  await runComponents(port);

  expect((await runAdminWeb(port)).status).toBe("success");
  for (const screen of ADMIN_SCREENS) {
    expect(port.resource(screen.key)).toMatchObject({ width: 1440, height: 900, navigationTheme: "dark", contentTheme: "light" });
  }
  expect(port.resource("screen/admin/risk-confirmations")?.actions).toEqual(["remote-open", "restart", "isolate", "rollback"]);
  expect(port.resource("screen/admin/risk-confirmations")?.fields).toEqual(["device", "reason", "risk-warning", "audit-notice"]);
  expect(port.resource("screen/admin/permission-denied")).toBeDefined();
  const count = port.resourceCount();
  await runAdminWeb(port);
  expect(port.resourceCount()).toBe(count);
});
