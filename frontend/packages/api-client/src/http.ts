import { request as taroRequest } from "@tarojs/taro";

export type AppErrorCode =
  | "NETWORK_FAILURE"
  | "NOT_FOUND"
  | "DEVICE_OFFLINE"
  | "PERMISSION_DENIED"
  | "UNKNOWN";

export class AppError extends Error {
  constructor(
    public readonly code: AppErrorCode,
    message: string,
    public readonly traceId?: string
  ) {
    super(message);
    this.name = "AppError";
  }
}

export interface RequestInput {
  path: string;
  method?: "GET" | "POST";
  query?: Record<string, string | number | boolean | undefined>;
  data?: Record<string, unknown>;
}

export type RequestFn = <T>(input: RequestInput) => Promise<T>;

const knownErrorCodes = new Set<AppErrorCode>([
  "NETWORK_FAILURE",
  "NOT_FOUND",
  "DEVICE_OFFLINE",
  "PERMISSION_DENIED",
  "UNKNOWN"
]);

function readErrorBody(data: unknown): { code?: AppErrorCode; message?: string } {
  if (typeof data !== "object" || data === null) return {};
  const body = data as Record<string, unknown>;
  const rawCode = typeof body.reason === "string"
    ? body.reason
    : typeof body.code === "string"
      ? body.code
      : undefined;
  const message = typeof body.message === "string" ? body.message : undefined;
  return {
    ...(rawCode && knownErrorCodes.has(rawCode as AppErrorCode)
      ? { code: rawCode as AppErrorCode }
      : {}),
    ...(message ? { message } : {})
  };
}

export function createTaroRequest(baseUrl: string): RequestFn {
  const normalizedBaseUrl = baseUrl.replace(/\/$/, "");

  return async <T>(input: RequestInput): Promise<T> => {
    const query = Object.entries(input.query ?? {})
      .filter((entry): entry is [string, string | number | boolean] => entry[1] !== undefined)
      .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`)
      .join("&");
    const url = `${normalizedBaseUrl}${input.path}${query ? `?${query}` : ""}`;

    try {
      const method = input.method ?? "GET";
      const response = await taroRequest<T>(
        method === "POST"
          ? {
              url,
              method,
              data: input.data ?? {},
              header: { "content-type": "application/json" }
            }
          : { url, method }
      );
      const traceId = String(response.header["x-request-id"] ?? "") || undefined;

      if (response.statusCode < 200 || response.statusCode >= 300) {
        const body = readErrorBody(response.data);
        const fallbackCode: AppErrorCode = response.statusCode === 404
          ? "NOT_FOUND"
          : response.statusCode === 403
            ? "PERMISSION_DENIED"
            : "UNKNOWN";
        throw new AppError(body.code ?? fallbackCode, body.message ?? "请求失败", traceId);
      }

      return response.data;
    } catch (error) {
      if (error instanceof AppError) throw error;
      throw new AppError("NETWORK_FAILURE", "网络连接失败");
    }
  };
}
