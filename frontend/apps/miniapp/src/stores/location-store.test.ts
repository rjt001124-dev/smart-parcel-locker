import { beforeEach, describe, expect, it } from "vitest";
import { useLocationStore } from "./location-store";

describe("location store", () => {
  beforeEach(() => useLocationStore.getState().reset());

  it("stores only selected city and coordinates", () => {
    useLocationStore.getState().selectCity({ code: "310100", name: "上海市" });
    useLocationStore.getState().setCoordinates(31.2304, 121.4737);

    expect(useLocationStore.getState()).toMatchObject({
      cityCode: "310100",
      cityName: "上海市",
      latitude: 31.2304,
      longitude: 121.4737
    });
  });
});
