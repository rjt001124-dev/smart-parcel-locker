// @vitest-environment node

import { describe, expect, it } from "vitest";
import config from "./index";

const staticConfig = config as {
  defineConstants?: Record<string, unknown>;
};

describe("Taro build constants", () => {
  it("defines the public API base URL as a compile-time constant", () => {
    expect(staticConfig.defineConstants?.TARO_APP_API_BASE_URL).toBe(
      JSON.stringify("http://127.0.0.1:8000")
    );
  });
});
