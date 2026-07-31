import Taro from "@tarojs/taro";

export type ScanResult = {
  ok: boolean;
  value: string;
  cancelled: boolean;
  error?: string;
};

export type ScanFn = () => Promise<ScanResult>;

/**
 * Adapter around the platform scan API. The mini program only uses the scan
 * result as a *convenience input* for the pickup code — the door still opens
 * ONLY through the server-validated order flow. This wrapper normalises the
 * platform differences (WeChat scanCode, unsupported H5, user cancellation)
 * into a single predictable result.
 */
export function createScanCode(deps: {
  scan: () => Promise<{ result: string }>;
}): ScanFn {
  return async function scanCode(): Promise<ScanResult> {
    try {
      const res = await deps.scan();
      const value = (res?.result ?? "").trim();
      return { ok: value.length > 0, value, cancelled: false };
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      const errMsg =
        (error as { errMsg?: string } | null)?.errMsg ?? "";
      const cancelled = /cancel/i.test(message) || /cancel/i.test(errMsg);
      if (cancelled) {
        return { ok: false, value: "", cancelled: true };
      }
      return { ok: false, value: "", cancelled: false, error: message };
    }
  };
}

const defaultScan = (): Promise<{ result: string }> =>
  Taro.scanCode({ onlyFromCamera: false, scanType: ["qrCode", "barCode"] });

export const scanCode = createScanCode({ scan: defaultScan });
