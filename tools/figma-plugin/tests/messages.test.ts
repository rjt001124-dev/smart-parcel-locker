import { expect, it } from "vitest";
import { parsePluginMessage } from "../src/domain/messages";

it("validates plugin messages and confirmed stages", () => {
  expect(parsePluginMessage({ type: "run-stage", stage: "foundations", confirmed: true })).toEqual({ type: "run-stage", stage: "foundations", confirmed: true });
  expect(() => parsePluginMessage({ type: "run-stage", stage: "unknown", confirmed: true })).toThrow("Unknown stage");
  expect(() => parsePluginMessage({ type: "run-stage", stage: "components", confirmed: false })).toThrow("Confirm the target file first");
  expect(parsePluginMessage({ type: "close-plugin" })).toEqual({ type: "close-plugin" });
});
