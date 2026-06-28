/**
 * Typed management API client.
 *
 * The panel is served from the same origin as the backend (`/management.html`),
 * so the base URL is auto-detected from `window.location`. Every management
 * request carries `Authorization: Bearer <management-key>`; a 401 response is
 * surfaced as an `ApiError` with `unauthorized = true` so the auth layer can
 * force a logout. All non-2xx responses are normalized into `ApiError`.
 */

export const MANAGEMENT_PREFIX = "/v0/management";
export const HEALTH_PATH = "/healthz";

/** Resolve the API origin. Falls back to current origin when running in-panel. */
export function resolveBaseUrl(): string {
  if (typeof window !== "undefined" && window.location?.origin) {
    return window.location.origin.replace(/\/$/, "");
  }
  return "";
}

export type ApiErrorKind =
  | "unauthorized"
  | "forbidden"
  | "not_found"
  | "server"
  | "network"
  | "timeout"
  | "parse"
  | "unknown";

export class ApiError extends Error {
  readonly status: number;
  readonly kind: ApiErrorKind;
  readonly code: string | undefined;
  readonly details: unknown;

  constructor(params: {
    message: string;
    status: number;
    kind: ApiErrorKind;
    code?: string;
    details?: unknown;
  }) {
    super(params.message);
    this.name = "ApiError";
    this.status = params.status;
    this.kind = params.kind;
    this.code = params.code;
    this.details = params.details;
  }

  get unauthorized(): boolean {
    return this.kind === "unauthorized";
  }
}

function kindForStatus(status: number): ApiErrorKind {
  if (status === 401) return "unauthorized";
  if (status === 403) return "forbidden";
  if (status === 404) return "not_found";
  if (status >= 500) return "server";
  return "unknown";
}

/** Extracts a human message + machine code from a normalized error envelope. */
function parseErrorBody(body: unknown): { message?: string; code?: string } {
  if (body && typeof body === "object") {
    const record = body as Record<string, unknown>;
    const message =
      typeof record.message === "string"
        ? record.message
        : typeof record.error === "string"
          ? record.error
          : undefined;
    const code =
      typeof record.code === "string"
        ? record.code
        : typeof record.error === "string"
          ? record.error
          : undefined;
    return {
      ...(message !== undefined ? { message } : {}),
      ...(code !== undefined ? { code } : {}),
    };
  }
  return {};
}

export interface RequestOptions extends Omit<RequestInit, "body" | "method"> {
  /** Request body; objects are JSON-encoded automatically. */
  body?: unknown;
  /** Query string parameters. */
  query?: Record<string, string | number | boolean | undefined>;
  /** Override the request timeout in milliseconds (default 20s). */
  timeoutMs?: number;
  /** Skip automatic JSON parsing and return the raw Response. */
  raw?: boolean;
}

/** Callback invoked when any request returns 401, used to drive logout. */
type UnauthorizedHandler = () => void;

let unauthorizedHandler: UnauthorizedHandler | null = null;
export function setUnauthorizedHandler(handler: UnauthorizedHandler | null): void {
  unauthorizedHandler = handler;
}

/** Token provider — wired to the auth store so the client stays decoupled. */
let tokenProvider: () => string | null = () => null;
export function setTokenProvider(provider: () => string | null): void {
  tokenProvider = provider;
}

const DEFAULT_TIMEOUT_MS = 20_000;

function buildUrl(path: string, query?: RequestOptions["query"]): string {
  const base = resolveBaseUrl();
  const normalizedPath = path.startsWith("/") ? path : `/${path}`;
  const url = new URL(`${base}${normalizedPath}`, base || window.location.href);
  if (query) {
    for (const [key, value] of Object.entries(query)) {
      if (value !== undefined) url.searchParams.set(key, String(value));
    }
  }
  return url.toString();
}

async function request<T>(method: string, path: string, options: RequestOptions = {}): Promise<T> {
  const { body, query, timeoutMs = DEFAULT_TIMEOUT_MS, raw, headers, signal, ...rest } = options;

  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), timeoutMs);
  if (signal) {
    signal.addEventListener("abort", () => controller.abort(), { once: true });
  }

  const finalHeaders = new Headers(headers);
  const token = tokenProvider();
  if (token) finalHeaders.set("Authorization", `Bearer ${token}`);

  let payload: BodyInit | undefined;
  if (body !== undefined) {
    if (body instanceof FormData || body instanceof Blob || typeof body === "string") {
      payload = body as BodyInit;
    } else {
      finalHeaders.set("Content-Type", "application/json");
      payload = JSON.stringify(body);
    }
  }

  let response: Response;
  try {
    response = await fetch(buildUrl(path, query), {
      method,
      headers: finalHeaders,
      signal: controller.signal,
      ...(payload !== undefined ? { body: payload } : {}),
      ...rest,
    });
  } catch (err) {
    clearTimeout(timeout);
    const aborted = err instanceof DOMException && err.name === "AbortError";
    throw new ApiError({
      message: aborted ? "Request timed out" : "Network request failed",
      status: 0,
      kind: aborted ? "timeout" : "network",
      details: err,
    });
  } finally {
    clearTimeout(timeout);
  }

  if (response.status === 401) {
    unauthorizedHandler?.();
  }

  if (!response.ok) {
    let parsed: unknown = undefined;
    try {
      const text = await response.text();
      parsed = text ? JSON.parse(text) : undefined;
    } catch {
      // Non-JSON error body; keep details undefined.
    }
    const { message, code } = parseErrorBody(parsed);
    throw new ApiError({
      message: message ?? `Request failed with status ${response.status}`,
      status: response.status,
      kind: kindForStatus(response.status),
      ...(code !== undefined ? { code } : {}),
      details: parsed,
    });
  }

  if (raw) {
    return response as unknown as T;
  }

  if (response.status === 204) {
    return undefined as T;
  }

  const contentType = response.headers.get("content-type") ?? "";
  if (contentType.includes("application/json")) {
    try {
      return (await response.json()) as T;
    } catch (err) {
      throw new ApiError({
        message: "Failed to parse server response",
        status: response.status,
        kind: "parse",
        details: err,
      });
    }
  }

  return (await response.text()) as unknown as T;
}

export const apiClient = {
  get: <T>(path: string, options?: RequestOptions) => request<T>("GET", path, options),
  post: <T>(path: string, options?: RequestOptions) => request<T>("POST", path, options),
  put: <T>(path: string, options?: RequestOptions) => request<T>("PUT", path, options),
  patch: <T>(path: string, options?: RequestOptions) => request<T>("PATCH", path, options),
  delete: <T>(path: string, options?: RequestOptions) => request<T>("DELETE", path, options),
};

/** Convenience wrapper that prefixes the management API namespace. */
export const managementApi = {
  get: <T>(path: string, options?: RequestOptions) =>
    apiClient.get<T>(`${MANAGEMENT_PREFIX}${path}`, options),
  post: <T>(path: string, options?: RequestOptions) =>
    apiClient.post<T>(`${MANAGEMENT_PREFIX}${path}`, options),
  put: <T>(path: string, options?: RequestOptions) =>
    apiClient.put<T>(`${MANAGEMENT_PREFIX}${path}`, options),
  patch: <T>(path: string, options?: RequestOptions) =>
    apiClient.patch<T>(`${MANAGEMENT_PREFIX}${path}`, options),
  delete: <T>(path: string, options?: RequestOptions) =>
    apiClient.delete<T>(`${MANAGEMENT_PREFIX}${path}`, options),
};
