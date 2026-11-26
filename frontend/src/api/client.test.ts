import { afterEach, describe, expect, it, vi } from "vitest";
import { apiGet } from "./client";

afterEach(() => vi.unstubAllGlobals());

describe("JSON API", () => {
  it("uses the API prefix and forwards cancellation", async () => {
    const fetch = vi.fn(async () => Response.json({ items: [] }));
    vi.stubGlobal("fetch", fetch);
    const controller = new AbortController();
    await expect(apiGet("/languages", controller.signal)).resolves.toEqual({ items: [] });
    expect(fetch).toHaveBeenCalledWith("/api/v1/languages", {
      signal: controller.signal,
      headers: { Accept: "application/json" },
    });
  });

  it("reports the structured error and request ID", async () => {
    vi.stubGlobal("fetch", async () =>
      Response.json(
        {
          error: {
            code: "SERVICE_UNAVAILABLE",
            message: "Storage unavailable",
            requestId: "fallback",
          },
        },
        {
          status: 503,
          headers: { "X-Request-Id": "request-id" },
        },
      ),
    );
    await expect(apiGet("/topics")).rejects.toMatchObject({
      status: 503,
      code: "SERVICE_UNAVAILABLE",
      message: "Storage unavailable",
      requestId: "request-id",
    });
  });
});
