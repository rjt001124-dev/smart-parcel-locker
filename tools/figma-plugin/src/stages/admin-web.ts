import { ADMIN_SCREENS, PAGE_KEYS } from "../domain/catalog";
import { createRunReport, recordResult, type RunReport } from "../domain/run-report";
import type { FigmaPort } from "../figma/port";
import { assertStageReady } from "./dependencies";
import { adminScreenSections } from "../domain/screen-content";

export async function runAdminWeb(port: FigmaPort): Promise<RunReport> {
  await assertStageReady(port, "admin-web");
  let report = createRunReport("admin-web");
  const page = PAGE_KEYS[2];
  report = recordResult(report, await port.upsertPage({ key: page.key, name: page.name }));

  for (const [index, screen] of ADMIN_SCREENS.entries()) {
    report = recordResult(report, await port.upsertScreen({
      ...screen,
      pageKey: page.key,
      x: 80 + (index % 3) * 1520,
      y: 80 + Math.floor(index / 3) * 980,
      navigationTheme: "dark",
      contentTheme: "light",
      layout: { direction: "HORIZONTAL", gap: 0, padding: 0 },
      sections: adminScreenSections(screen.key),
      componentRefs: ["component/admin-sidebar", "component/admin-header", "component/filter-bar", "component/data-table", "component/pagination"]
    }));
  }

  report = recordResult(report, await port.upsertScreen({
    key: "screen/admin/permission-denied", name: "Admin / Permission Denied", pageKey: page.key,
    width: 1440, height: 900, x: 80, y: 3040, navigationTheme: "dark", contentTheme: "light",
    actions: ["return", "request-permission"]
  }));
  report = recordResult(report, await port.upsertScreen({
    key: "screen/admin/risk-confirmations", name: "Admin / High-risk Confirmations", pageKey: page.key,
    width: 1440, height: 900, x: 1600, y: 3040, navigationTheme: "dark", contentTheme: "light",
    actions: ["remote-open", "restart", "isolate", "rollback"],
    fields: ["device", "reason", "risk-warning", "audit-notice"],
    requiresSecondConfirmation: true
  }));
  if (report.status === "success") await port.setStageMarker("admin-web");
  return report;
}
