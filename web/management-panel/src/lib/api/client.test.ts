import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  ApiError,
  apiClient,
  setTokenProvider,
  setUnauthorizedHandler,
} from "./client";

function jsonResponse(body: unknown, init: ResponseInit): Response {
  return new Response(JSON.stringify(body), {
    ...init,
    headers: { "content-type": "application/json", ...(init.headers ?? {}) },
  });
}

describe("apiClient", () => {
  beforeEach(() => {
    setTokenProvider(() => "test-key");
    setUnauthorizedHandler(null);
  });

  afterEach(() => {
    vi.restoreAllMocks();
    setTokenProvider(() => null);
  });

  it("injects the bearer token on requests", async () => {
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValue(jsonResponse({ status: "ok" }, { status: 200 }));

    await apiClient.get("/healthz");

    const [, requestInit] = fetchMock.mock.calls[0]!;
    const headers = new Headers(requestInit?.headers);
    expect(headers.get("Authorization")).toBe("Bearer test-key");
  });

  it("normalizes a 401 into an unauthorized ApiError and fires the handler", async () => {
    const onUnauthorized = vi.fn();
    setUnauthorizedHandler(onUnauthorized);
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      jsonResponse({ error: "unauthorized" }, { status: 401 }),
    );

    await expect(apiClient.get("/v0/management/config")).rejects.toMatchObject({
      kind: "unauthorized",
    });
    expect(onUnauthorized).toHaveBeenCalledOnce();
  });

  it("maps a network failure to a network ApiError", async () => {
    vi.spyOn(globalThis, "fetch").mockRejectedValue(new TypeError("offline"));

    const error = await apiClient.get("/healthz").catch((e: unknown) => e);
    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).kind).toBe("network");
  });
});
