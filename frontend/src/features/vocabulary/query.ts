import type { Language } from "../../preferences/storage";
import { single, trimSearch } from "../listQuery";
import { statuses, topicLimit } from "./words";

const topicID = /^[\da-f]{8}-[\da-f]{4}-[\da-f]{4}-[\da-f]{4}-[\da-f]{12}$/i;

export const sorts = [
  { value: "", label: "Newest" },
  { value: "oldest", label: "Oldest" },
  { value: "word_asc", label: "Word ascending" },
  { value: "word_desc", label: "Word descending" },
] as const;

export function vocabularyFilters(query: URLSearchParams) {
  const filters = new URLSearchParams();
  const text = trimSearch(single(query, "q"));
  const status = single(query, "status");
  const sort = single(query, "sort");
  const topics = [
    ...new Set(
      single(query, "topics")
        .toLowerCase()
        .split(",")
        .filter((id) => topicID.test(id)),
    ),
  ]
    .sort()
    .slice(0, topicLimit);
  if (text) filters.set("q", text);
  if (statuses.some((value) => value === status)) filters.set("status", status);
  if (sort && sorts.some((option) => option.value === sort)) filters.set("sort", sort);
  if (topics.length) filters.set("topics", topics.join(","));
  return filters;
}

export function vocabularyURL(language: Language, filters: URLSearchParams, page = 1) {
  const query = new URLSearchParams([
    ["languageCode", language],
    ...filters,
    ["page", String(page)],
  ]);
  return `/vocabulary?${query}`;
}
