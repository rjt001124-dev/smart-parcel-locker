import { COMPONENT_KEYS, PAGE_KEYS } from "../domain/catalog";
import { createRunReport, recordResult, type RunReport } from "../domain/run-report";
import type { FigmaPort } from "../figma/port";
import { assertStageReady } from "./dependencies";

const DISPLAY_NAMES: Readonly<Record<(typeof COMPONENT_KEYS)[number], string>> = {
  button: "Button", "icon-button": "Icon Button", input: "Input", search: "Search", select: "Select",
  "status-pill": "Status Pill", alert: "Alert", result: "Result", "empty-state": "Empty State",
  "error-state": "Error State", skeleton: "Skeleton", card: "Card", "site-card": "Site Card",
  "locker-size-card": "Locker Size Card", "order-card": "Order Card", modal: "Modal",
  "bottom-sheet": "Bottom Sheet", "confirm-dialog": "Confirm Dialog", "miniapp-top-bar": "Mini App Top Bar",
  "miniapp-bottom-tab-bar": "Mini App Bottom Tab Bar", "admin-sidebar": "Admin Sidebar",
  "admin-header": "Admin Header", "filter-bar": "Filter Bar", "data-table": "Data Table", pagination: "Pagination"
};

function variantsFor(key: (typeof COMPONENT_KEYS)[number]): readonly Record<string, string>[] {
  if (key === "button" || key === "icon-button") return [
    { Style: "Primary", Size: "Medium", State: "Default" },
    { Style: "Primary", Size: "Medium", State: "Loading" },
    { Style: "Primary", Size: "Medium", State: "Disabled" },
    { Style: "Secondary", Size: "Medium", State: "Default" }
  ];
  if (["input", "search", "select"].includes(key)) return [
    { State: "Default" }, { State: "Focus" }, { State: "Filled" }, { State: "Error" }, { State: "Disabled" }
  ];
  if (["site-card", "locker-size-card", "order-card", "card"].includes(key)) return [
    { State: "Default" }, { State: "Selected" }, { State: "Disabled" }
  ];
  return [{ State: "Default" }];
}

function visualSpec(key: (typeof COMPONENT_KEYS)[number]): { width: number; height: number; visualKind: string } {
  if (key === "button") return { width: 160, height: 48, visualKind: "button" };
  if (key === "icon-button") return { width: 48, height: 48, visualKind: "icon-button" };
  if (["input", "search", "select"].includes(key)) return { width: 320, height: 96, visualKind: key };
  if (["site-card", "locker-size-card", "order-card", "card"].includes(key)) return { width: 360, height: 180, visualKind: key };
  if (key === "data-table") return { width: 640, height: 280, visualKind: "data-table" };
  if (["admin-header", "filter-bar"].includes(key)) return { width: 640, height: 96, visualKind: key };
  if (key === "admin-sidebar") return { width: 240, height: 360, visualKind: key };
  if (["modal", "bottom-sheet", "confirm-dialog"].includes(key)) return { width: 420, height: 240, visualKind: key };
  if (["miniapp-top-bar", "miniapp-bottom-tab-bar"].includes(key)) return { width: 375, height: 80, visualKind: key };
  return { width: 280, height: 112, visualKind: key };
}

export async function runComponents(port: FigmaPort): Promise<RunReport> {
  await assertStageReady(port, "components");
  let report = createRunReport("components");
  const page = PAGE_KEYS[0];
  report = recordResult(report, await port.upsertPage({ key: page.key, name: page.name }));

  for (const [index, key] of COMPONENT_KEYS.entries()) {
    const visual = visualSpec(key);
    report = recordResult(report, await port.upsertComponentFamily({
      key: `component/${key}`,
      name: DISPLAY_NAMES[key],
      pageKey: page.key,
      x: 1600 + (index % 2) * 720,
      y: 80 + Math.floor(index / 2) * 340,
      ...visual,
      variants: variantsFor(key),
      layout: { direction: "HORIZONTAL", gap: 12, padding: 12 },
      tokens: ["color/action/primary", "color/text/primary", "space/12", "radius/8"]
    }));
  }

  if (report.status === "success") await port.setStageMarker("components");
  return report;
}
