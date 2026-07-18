import { describe, expect, it } from "vitest";
import {
  ADMIN_SCREENS,
  COMPONENT_KEYS,
  MINI_APP_SCREENS,
  PAGE_KEYS,
  REQUIRED_STATES,
  assertUniqueKeys,
  prerequisitesFor
} from "../src/domain/catalog";

describe("design catalog", () => {
  it("has unique stable keys", () => {
    expect(() => assertUniqueKeys()).not.toThrow();
  });

  it("covers required pages and screen sizes", () => {
    expect(PAGE_KEYS.map((item) => item.name)).toEqual(["00 Design System", "01 Mini App", "02 Admin Web"]);
    expect(PAGE_KEYS.length).toBeLessThanOrEqual(3);
    expect(MINI_APP_SCREENS.every((item) => item.width === 375 && item.height === 812)).toBe(true);
    expect(ADMIN_SCREENS.every((item) => item.width === 1440 && item.height === 900)).toBe(true);
  });

  it("covers required failure and processing states", () => {
    expect(REQUIRED_STATES).toEqual(
      expect.arrayContaining([
        "loading",
        "empty",
        "network-failure",
        "payment-processing",
        "payment-failure",
        "payment-success",
        "cell-contention",
        "device-offline",
        "door-open-failure",
        "door-not-closed",
        "overdue-payment",
        "permission-denied"
      ])
    );
  });

  it("requires stages in order", () => {
    expect(prerequisitesFor("components")).toEqual(["foundations"]);
    expect(prerequisitesFor("mini-app")).toEqual(["foundations", "components"]);
    expect(prerequisitesFor("admin-web")).toEqual(["foundations", "components"]);
  });

  it("includes the agreed core component families", () => {
    expect(COMPONENT_KEYS).toEqual(
      expect.arrayContaining([
        "button",
        "input",
        "site-card",
        "locker-size-card",
        "order-card",
        "data-table",
        "admin-sidebar"
      ])
    );
  });
});
