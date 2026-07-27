import { describe, expect, it, vi } from "vitest";
import { getCurrentCoordinates } from "./location";

describe("current location", () => {
  it("requests GCJ-02 coordinates for nearby-site discovery", async () => {
    const getLocation = vi.fn().mockResolvedValue({
      latitude: 31.2304,
      longitude: 121.4737
    });

    await expect(getCurrentCoordinates(getLocation)).resolves.toEqual({
      latitude: 31.2304,
      longitude: 121.4737
    });
    expect(getLocation).toHaveBeenCalledWith({ type: "gcj02" });
  });
});
