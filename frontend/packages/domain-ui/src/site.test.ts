import { describe, expect, it } from "vitest";
import { toSiteCardView } from "./site";

describe("site view model", () => {
  it("formats distance and availability without inventing data", () => {
    expect(toSiteCardView({
      id: "1",
      site_no: "SITE-SH-001",
      name: "万象城智能寄存点",
      address: "世纪大道88号B1层",
      latitude: 31.2304,
      longitude: 121.4737,
      distance_m: 320,
      availability: [
        { size: "CELL_SIZE_SMALL", available_count: 12 },
        { size: "CELL_SIZE_MEDIUM", available_count: 8 },
        { size: "CELL_SIZE_LARGE", available_count: 4 }
      ]
    })).toEqual({
      id: "1",
      name: "万象城智能寄存点",
      address: "世纪大道88号B1层",
      distanceLabel: "320m",
      availabilityLabel: "可用24格",
      statusLabel: "有空柜",
      statusTone: "success"
    });
  });

  it("shows an honest empty state when availability is zero", () => {
    const result = toSiteCardView({
      id: "2",
      site_no: "SITE-SH-002",
      name: "地铁站寄存点",
      address: "地铁站2号口",
      latitude: 31.2,
      longitude: 121.4,
      distance_m: 1250,
      availability: []
    });

    expect(result.distanceLabel).toBe("1.3km");
    expect(result.availabilityLabel).toBe("可用0格");
    expect(result.statusLabel).toBe("暂无空柜");
    expect(result.statusTone).toBe("warning");
  });
});
