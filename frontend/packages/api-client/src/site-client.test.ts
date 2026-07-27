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

  it("normalizes protobuf JSON field names for domain consumers", async () => {
    const request = vi.fn().mockResolvedValue({
      sites: [{
        id: "2",
        siteNo: "SITE-SH-002",
        name: "南京东路寄存点",
        address: "上海市黄浦区南京东路 200 号",
        latitude: 31.2361,
        longitude: 121.4802,
        distanceM: 885,
        availability: [{ size: "CELL_SIZE_SMALL", availableCount: 2 }]
      }]
    });
    const client = createSiteClient({ request });

    const result = await client.listSites({ cityCode: "310100" });

    expect(result.sites[0]).toMatchObject({
      site_no: "SITE-SH-002",
      distance_m: 885,
      availability: [{ size: "CELL_SIZE_SMALL", available_count: 2 }]
    });
  });
});
