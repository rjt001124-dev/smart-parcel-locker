import { describe, expect, it, vi } from "vitest";
import { getCurrentCoordinates } from "./location";

describe("current location", () => {
  it("uses explicit preview coordinates for H5", async () => {
    const getLocation = vi.fn();

    await expect(getCurrentCoordinates({
      platform: "h5",
      previewLatitude: "31.2304",
      previewLongitude: "121.4737",
      getLocation
    })).resolves.toEqual({ latitude: 31.2304, longitude: 121.4737 });
    expect(getLocation).not.toHaveBeenCalled();
  });

  it("uses runtime positioning when the H5 preview pair is incomplete", async () => {
    const getLocation = vi.fn().mockResolvedValue({
      latitude: 30.2741,
      longitude: 120.1551
    });

    await expect(getCurrentCoordinates({
      platform: "h5",
      previewLatitude: "31.2304",
      previewLongitude: "",
      getLocation
    })).resolves.toEqual({ latitude: 30.2741, longitude: 120.1551 });
  });

  it("requests GCJ-02 coordinates for WeChat", async () => {
    const getLocation = vi.fn().mockResolvedValue({
      latitude: 31.2304,
      longitude: 121.4737
    });

    await expect(getCurrentCoordinates({
      platform: "weapp",
      previewLatitude: "",
      previewLongitude: "",
      getLocation
    })).resolves.toEqual({ latitude: 31.2304, longitude: 121.4737 });
    expect(getLocation).toHaveBeenCalledWith({ type: "gcj02" });
  });

  it("rejects invalid runtime coordinates", async () => {
    const getLocation = vi.fn().mockResolvedValue({
      latitude: Number.NaN,
      longitude: 121.4737
    });

    await expect(getCurrentCoordinates({
      platform: "h5",
      previewLatitude: "",
      previewLongitude: "",
      getLocation
    })).rejects.toThrow("Invalid coordinates");
  });
});
