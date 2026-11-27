const statuses = ["draft", "published"] as const;
const difficulties = ["beginner", "intermediate", "advanced", "unspecified"] as const;
const sorts = ["oldest", "title_asc", "title_desc"] as const;
const topicID = /^[\da-f]{8}-[\da-f]{4}-[\da-f]{4}-[\da-f]{4}-[\da-f]{12}$/i;

export const searchLimit = 120;

export function trimSearch(value: string) {
  return [...value.replaceAll("\0", "")].slice(0, searchLimit).join("").trim();
}

export function readFilters(query: URLSearchParams) {
  const result = new URLSearchParams();
  const text = trimSearch(single(query, "q"));
  if (text) result.set("q", text);
  for (const [key, options] of [
    ["status", statuses],
    ["difficulty", difficulties],
    ["sort", sorts],
  ] as const) {
    const value = single(query, key);
    if (options.some((choice) => choice === value)) result.set(key, value);
  }
  if (single(query, "favoritesOnly") === "true") result.set("favoritesOnly", "true");
  const topics = [
    ...new Set(
      single(query, "topics")
        .toLowerCase()
        .split(",")
        .filter((id) => topicID.test(id)),
    ),
  ].sort();
  if (topics.length) result.set("topics", topics.join(","));
  return result;
}

export function readPage(query: URLSearchParams) {
  const value = single(query, "page");
  return /^\d+$/.test(value) ? Math.max(1, Math.min(Number(value), 2147483647)) : 1;
}

function single(query: URLSearchParams, key: string) {
  const values = query.getAll(key);
  return values.length === 1 ? values[0] : "";
}
