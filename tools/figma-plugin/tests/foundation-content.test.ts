import { expect, it } from "vitest";
import { FOUNDATION_SECTIONS, foundationItems } from "../src/domain/foundation-content";

it("defines visible examples for every Foundations section", () => {
  for (const section of FOUNDATION_SECTIONS) expect(foundationItems(section).length).toBeGreaterThanOrEqual(3);
  expect(foundationItems("颜色").some((item) => item.kind === "swatch" && item.label === "品牌蓝")).toBe(true);
  expect(foundationItems("排版").map((item) => item.value)).toEqual(expect.arrayContaining([32, 24, 16, 12]));
  expect(foundationItems("状态").map((item) => item.label)).toEqual(expect.arrayContaining(["成功", "处理中", "警告", "失败", "离线"]));
});
