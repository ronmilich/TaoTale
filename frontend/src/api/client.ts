import type { components, operations, paths } from "./schema";

export type LearningLanguage = components["schemas"]["LearningLanguage"];
export type Item = components["schemas"]["Item"];
export type ItemCreate = components["schemas"]["ItemCreate"];
export type ItemPatch = components["schemas"]["ItemPatch"];
export type ItemPage = components["schemas"]["ItemPage"];
export type Voice = components["schemas"]["TTSVoice"];
export type PreviewInput = components["schemas"]["PreviewInput"];
export type PreviewStatus = components["schemas"]["PreviewStatus"];
export type DraftGeneration = JSONResult<operations["getDraftGeneration"]>;
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

export function apiGet<P extends GetPath>(path: P, signal?: AbortSignal): Promise<Result<P>> {
  return readJSON(path, signal);
}

export function getItem(id: string, signal?: AbortSignal): Promise<Item> {
  return readJSON(itemPath(id), signal);
}

export const libraryPageSize = 8;

export function listItems(language: LearningLanguage["code"], page: number, signal: AbortSignal) {
  const query = new URLSearchParams({
    languageCode: language,
    page: String(page),
    pageSize: String(libraryPageSize),
  });
  return readJSON<ItemPage>(`/items?${query}`, signal);
}

export function listVoices(language: LearningLanguage["code"], signal: AbortSignal) {
  return readJSON<Result<"/tts/voices">>(`/tts/voices?languageCode=${language}`, signal);
}

type GenerationReceipt = { generationId: string };

export function generateDraft(id: string, version: number, key: string, signal: AbortSignal) {
  return postGeneration(
    `${itemPath(id)}/audio/generate`,
    { expectedVersion: version },
    key,
    signal,
  );
}

export function generatePreview(input: PreviewInput, key: string, signal: AbortSignal) {
  return postGeneration("/tts/previews", input, key, signal);
}

export function getDraftGeneration(itemId: string, id: string, signal: AbortSignal) {
  return readJSON<DraftGeneration>(
    `${itemPath(itemId)}/audio/generations/${encodeURIComponent(id)}`,
    signal,
  );
}

export function getPreview(id: string, signal: AbortSignal) {
  return readJSON<PreviewStatus>(previewPath(id), signal);
}

export function draftAudioURLs(itemId: string, assetId: string) {
  const path = `/api/v1${itemPath(itemId)}/audio`;
  const pin = `?audioAssetId=${encodeURIComponent(assetId)}`;
  return { src: path + pin, download: path + "/download" + pin };
}

export function previewAudioURLs(id: string) {
  const src = `/api/v1${previewPath(id)}/audio`;
  return { src, download: src + "/download" };
}

export function createItem(item: ItemCreate): Promise<Item> {
  return writeItem("/items", "POST", item);
}

export function patchItem(id: string, patch: ItemPatch): Promise<Item> {
  return writeItem(itemPath(id), "PATCH", patch);
}

function itemPath(id: string) {
  return `/items/${encodeURIComponent(id)}`;
}

function previewPath(id: string) {
  return `/tts/previews/${encodeURIComponent(id)}`;
}

function readJSON<T>(path: string, signal?: AbortSignal) {
  return requestJSON<T>(path, { headers: { Accept: "application/json" }, signal });
}

function postGeneration(
  path: string,
  body: PreviewInput | { expectedVersion: number },
  key: string,
  signal: AbortSignal,
) {
  return requestJSON<GenerationReceipt>(path, {
    method: "POST",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
      "Idempotency-Key": key,
    },
    body: JSON.stringify(body),
    signal,
  });
}

function writeItem(path: string, method: "POST" | "PATCH", body: ItemCreate | ItemPatch) {
  return requestJSON<Item>(path, {
    method,
    headers: { Accept: "application/json", "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

async function requestJSON<T>(path: string, init: RequestInit): Promise<T> {
  const response = await fetch(`/api/v1${path}`, init);
  const requestId = response.headers.get("X-Request-Id");
  let body: unknown;
  try {
    body = await response.json();
  } catch (error) {
    if (init.signal?.aborted) throw error;
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
  return body as T;
}
