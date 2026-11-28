import { single, trimSearch } from "../listQuery";

export { readPage, searchLimit } from "../listQuery";

const statuses = ["draft", "published"] as const;
const difficulties = ["beginner", "intermediate", "advanced", "unspecified"] as const;
const sorts = ["oldest", "title_asc", "title_desc"] as const;
const topicID = /^[\da-f]{8}-[\da-f]{4}-[\da-f]{4}-[\da-f]{4}-[\da-f]{12}$/i;

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
