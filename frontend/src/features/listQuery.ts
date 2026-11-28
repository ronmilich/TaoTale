export const searchLimit = 120;

export function trimSearch(value: string) {
  return [...value.replaceAll("\0", "")].slice(0, searchLimit).join("").trim();
}

export function readPage(query: URLSearchParams) {
  const value = single(query, "page");
  return /^\d+$/.test(value) ? Math.max(1, Math.min(Number(value), 2147483647)) : 1;
}

export function single(query: URLSearchParams, key: string) {
  const values = query.getAll(key);
  return values.length === 1 ? values[0] : "";
}
