import type { Stage } from "./catalog";

export type PluginMessage = { type: "close-plugin" } | { type: "run-stage"; stage: Stage; confirmed: true };
const STAGES = new Set<Stage>(["foundations", "components", "mini-app", "admin-web"]);

export function parsePluginMessage(value: unknown): PluginMessage {
  if (!value || typeof value !== "object") throw new Error("Invalid plugin message");
  const message = value as Record<string, unknown>;
  if (message.type === "close-plugin") return { type: "close-plugin" };
  if (message.type !== "run-stage") throw new Error("Unknown message type");
  if (!STAGES.has(message.stage as Stage)) throw new Error("Unknown stage");
  if (message.confirmed !== true) throw new Error("Confirm the target file first");
  return { type: "run-stage", stage: message.stage as Stage, confirmed: true };
}
