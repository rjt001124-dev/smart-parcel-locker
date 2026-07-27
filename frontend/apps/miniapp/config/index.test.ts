// @vitest-environment node

import { afterEach, describe, expect, it, vi } from "vitest";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

afterEach(() => {
  vi.unstubAllEnvs();
  vi.resetModules();
});

async function loadConfig() {
  return (await import("./index")).default as {
    outputRoot?: string;
    defineConstants?: Record<string, unknown>;
    h5?: {
      devServer?: {
        port?: number;
        proxy?: Record<string, { target?: string; changeOrigin?: boolean }>;
      };
    };
  };
}

describe("Taro build configuration", () => {
  it("keeps the WeChat API URL and output directory", async () => {
    vi.stubEnv("TARO_ENV", "weapp");
    const config = await loadConfig();

    expect(config.outputRoot).toBe("dist");
    expect(config.defineConstants?.TARO_APP_API_BASE_URL).toBe(
      JSON.stringify("http://127.0.0.1:8000")
    );
  });

  it("uses the H5 proxy and a separate output directory", async () => {
    vi.stubEnv("TARO_ENV", "h5");
    vi.stubEnv("TARO_APP_API_BASE_URL", "/api");
    vi.stubEnv("TARO_APP_PREVIEW_LATITUDE", "31.2304");
    vi.stubEnv("TARO_APP_PREVIEW_LONGITUDE", "121.4737");
    const config = await loadConfig();

    expect(config.outputRoot).toBe("dist-h5");
    expect(config.defineConstants).toMatchObject({
      TARO_APP_API_BASE_URL: JSON.stringify("/api"),
      TARO_APP_PREVIEW_LATITUDE: JSON.stringify("31.2304"),
      TARO_APP_PREVIEW_LONGITUDE: JSON.stringify("121.4737")
    });
    expect(config.h5?.devServer?.port).toBe(10086);
    expect(config.h5?.devServer?.proxy?.["/api"]).toMatchObject({
      target: "http://127.0.0.1:8000",
      changeOrigin: true
    });
  });

  it("provides Taro's H5 entry-script placeholder", () => {
    const html = readFileSync(resolve(__dirname, "../src/index.html"), "utf8");

    expect(html).toContain(
      "<script><%= htmlWebpackPlugin.options.script %></script>"
    );
    expect(html).toContain('<link rel="icon" href="data:,">');
  });
});
