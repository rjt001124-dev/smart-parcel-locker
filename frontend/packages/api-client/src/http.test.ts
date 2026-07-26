import Taro from "@tarojs/taro";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AppError, createTaroRequest } from "./http";

vi.mock("@tarojs/taro", () => ({
  default: { request: vi.fn() }
}));

describe("public Taro request", () => {
  beforeEach(() => {
    vi.mocked(Taro.request).mockReset();
  });

  it("encodes query parameters without internal headers", async () => {
    vi.mocked(Taro.request).mockResolvedValue({
      data: { sites: [] },
      statusCode: 200,
      header: {},
      cookies: [],
      errMsg: "request:ok"
    } as never);
    const request = createTaroRequest("https://api.example.com/");

    await request({ path: "/v1/sites", query: { city_code: "310100", radius_m: 5000 } });

    expect(Taro.request).toHaveBeenCalledWith({
      url: "https://api.example.com/v1/sites?city_code=310100&radius_m=5000",
      method: "GET"
    });
    expect(JSON.stringify(vi.mocked(Taro.request).mock.calls)).not.toContain("X-Internal-Token");
  });

  it("preserves the request trace ID for not-found errors", async () => {
    vi.mocked(Taro.request).mockResolvedValue({
      data: {},
      statusCode: 404,
      header: { "x-request-id": "trace-404" },
      cookies: [],
      errMsg: "request:ok"
    } as never);
    const request = createTaroRequest("https://api.example.com");

    await expect(request({ path: "/v1/sites/missing" })).rejects.toEqual(
      new AppError("NOT_FOUND", "请求失败", "trace-404")
    );
  });

  it("normalizes transport rejection as a network failure", async () => {
    vi.mocked(Taro.request).mockRejectedValue(new Error("socket closed"));
    const request = createTaroRequest("https://api.example.com");

    await expect(request({ path: "/v1/cities" })).rejects.toMatchObject({
      code: "NETWORK_FAILURE",
      message: "网络连接失败"
    });
  });

  it("maps a stable backend reason to the offline error code", async () => {
    vi.mocked(Taro.request).mockResolvedValue({
      data: { reason: "DEVICE_OFFLINE", message: "柜机离线" },
      statusCode: 503,
      header: { "x-request-id": "trace-offline" },
      cookies: [],
      errMsg: "request:ok"
    } as never);
    const request = createTaroRequest("https://api.example.com");

    await expect(request({ path: "/v1/sites/site-1/cells" })).rejects.toMatchObject({
      code: "DEVICE_OFFLINE",
      message: "柜机离线",
      traceId: "trace-offline"
    });
  });
});
