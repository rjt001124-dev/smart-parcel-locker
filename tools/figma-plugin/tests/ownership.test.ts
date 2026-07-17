import { describe, expect, it } from "vitest";
import { decideMutation, OWNERSHIP } from "../src/domain/ownership";

describe("ownership", () => {
  it("updates a matching owned resource", () => {
    expect(decideMutation({ nameMatches: true, owner: OWNERSHIP.owner, keyMatches: true })).toBe("update");
  });

  it("blocks an unowned name conflict", () => {
    expect(decideMutation({ nameMatches: true, owner: "", keyMatches: false })).toBe("conflict");
  });

  it("creates when no resource exists", () => {
    expect(decideMutation(null)).toBe("create");
  });
});
