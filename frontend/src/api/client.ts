import type { components, paths } from "./schema";

export type LearningLanguage = components["schemas"]["LearningLanguage"];
type ErrorEnvelope = components["schemas"]["ErrorEnvelope"];
type JSONResult<Operation> = Operation extends {
  responses: { 200: { content: { "application/json": infer Payload } } };
}
  ? Payload
  : never;
type GetPath = {
  [P in keyof paths]: [JSONResult<paths[P]["get"]>] extends [never] ? never : P;
}[keyof paths];
type Result<P extends GetPath> = JSONResult<paths[P]["get"]>;

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
    public readonly requestId: string | null,
    public readonly details?: ErrorEnvelope["error"]["details"],
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export async function apiGet<P extends GetPath>(path: P, signal?: AbortSignal): Promise<Result<P>> {
  const response = await fetch(`/api/v1${path}`, {
    headers: { Accept: "application/json" },
    signal,
  });
  const requestId = response.headers.get("X-Request-Id");
  let body: unknown;
  try {
    body = await response.json();
  } catch (error) {
    if (signal?.aborted) throw error;
    if (response.ok) {
      throw new ApiError(
        response.status,
        "INVALID_RESPONSE",
        "TaoTale returned unreadable data.",
        requestId,
      );
    }
  }
  if (!response.ok) {
    const error = (body as ErrorEnvelope | null | undefined)?.error;
    throw new ApiError(
      response.status,
      error?.code ?? "REQUEST_FAILED",
      error?.message ?? "TaoTale could not complete the request.",
      requestId ?? error?.requestId ?? null,
      error?.details,
    );
  }
  return body as Result<P>;
}
