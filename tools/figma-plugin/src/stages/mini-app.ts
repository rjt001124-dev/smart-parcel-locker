import { MINI_APP_SCREENS, PAGE_KEYS, REQUIRED_STATES } from "../domain/catalog";
import { createRunReport, recordResult, type RunReport } from "../domain/run-report";
import type { FigmaPort } from "../figma/port";
import { assertStageReady } from "./dependencies";

export async function runMiniApp(port: FigmaPort): Promise<RunReport> {
  await assertStageReady(port, "mini-app");
  let report = createRunReport("mini-app");
  const page = PAGE_KEYS[2];
  report = recordResult(report, await port.upsertPage({ key: page.key, name: page.name }));

  for (const screen of MINI_APP_SCREENS) {
    report = recordResult(report, await port.upsertScreen({
      ...screen,
      pageKey: page.key,
      layout: { direction: "VERTICAL", gap: 16, padding: 16 },
      sections: ["Status Bar", "Top Bar", "Primary Content", "Context Actions", "Bottom Navigation"],
      componentRefs: ["component/miniapp-top-bar", "component/button", "component/card", "component/miniapp-bottom-tab-bar"]
    }));
  }

  report = recordResult(report, await port.upsertScreen({
    key: "screen/miniapp/state-matrix", name: "Mini App / State Matrix", pageKey: page.key,
    width: 1600, height: 1200, states: REQUIRED_STATES,
    componentRefs: ["component/alert", "component/result", "component/empty-state", "component/error-state"]
  }));
  if (report.status === "success") await port.setStageMarker("mini-app");
  return report;
}
