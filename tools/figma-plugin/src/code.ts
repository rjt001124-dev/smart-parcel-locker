import { parsePluginMessage } from "./domain/messages";
import { createRunReport, recordResult } from "./domain/run-report";
import { RealFigmaAdapter } from "./figma/adapter";
import { runAdminWeb } from "./stages/admin-web";
import { runComponents } from "./stages/components";
import { runFoundations } from "./stages/foundations";
import { runMiniApp } from "./stages/mini-app";

figma.showUI(__html__, { width: 420, height: 620 });
const adapter = new RealFigmaAdapter();
void adapter.getFileName().then((fileName) => figma.ui.postMessage({ type: "ready", fileName }));

figma.ui.onmessage = async (raw: unknown) => {
  try {
    const message = parsePluginMessage(raw);
    if (message.type === "close-plugin") return figma.closePlugin();
    figma.ui.postMessage({ type: "running", stage: message.stage });
    const runners = { foundations: runFoundations, components: runComponents, "mini-app": runMiniApp, "admin-web": runAdminWeb } as const;
    const report = await runners[message.stage](adapter);
    figma.ui.postMessage({ type: "result", report });
  } catch (error) {
    let report = createRunReport("foundations");
    report = recordResult(report, { key: "plugin/run", outcome: "error", message: error instanceof Error ? error.message : String(error) });
    figma.ui.postMessage({ type: "result", report });
  }
};
