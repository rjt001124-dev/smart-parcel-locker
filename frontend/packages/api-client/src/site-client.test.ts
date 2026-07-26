import { describe, expect, it, vi } from "vitest";
import { createSiteClient } from "./site-client";

describe("site client", () => {
  it("sends the approved nearby-site query", async () => {
    const request = vi.fn().mockResolvedValue({ sites: [] });
    const client = createSiteClient({ request });

    await client.listSites({
      cityCode: "310100",
      latitude: 31.2304,
      longitude: 121.4737,
      radiusM: 5000
    });

    expect(request).toHaveBeenCalledWith({
      path: "/v1/sites",
      query: {
        city_code: "310100",
        latitude: 31.2304,
        longitude: 121.4737,
        radius_m: 5000
      }
    });
  });

  it("never sends the internal token from a public client", async () => {
    const request = vi.fn().mockResolvedValue({ cities: [] });
    const client = createSiteClient({ request });

    await client.listCities();

    expect(JSON.stringify(request.mock.calls)).not.toContain("X-Internal-Token");
  });
});
