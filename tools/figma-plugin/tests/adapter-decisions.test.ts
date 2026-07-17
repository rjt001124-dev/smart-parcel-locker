import { describe, expect, it } from "vitest";
import { normalizeColor, resourceDecision, toVariableScopes } from "../src/figma/adapter-helpers";

describe("Figma adapter decisions", () => {
  it("normalizes 0-255 colors", () => expect(normalizeColor({ r: 22, g: 119, b: 255 })).toEqual({ r: 22 / 255, g: 119 / 255, b: 1 }));
  it("keeps normalized colors", () => expect(normalizeColor({ r: 0.2, g: 0.4, b: 0.6 })).toEqual({ r: 0.2, g: 0.4, b: 0.6 }));
  it("keeps only supported scopes", () => expect(toVariableScopes(["FRAME_FILL", "TEXT_FILL", "UNKNOWN"])).toEqual(["FRAME_FILL", "TEXT_FILL"]));
  it("protects unknown same-name resources", () => expect(resourceDecision({ owner: "", key: "", nameMatches: true }, "page/admin-web")).toBe("conflict"));
  it("updates owned matching resources", () => expect(resourceDecision({ owner: "local-figma-generator", key: "page/admin-web", nameMatches: true }, "page/admin-web")).toBe("update"));
});
