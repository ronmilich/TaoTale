import { expect, test } from "@playwright/test";
import type { Item, ItemPage } from "../src/api/client";

test("lists seeded drafts using one list request per page and clamps an out-of-range page", async ({
  page,
  request,
}) => {
  for (let number = 1; number <= 10; number++) {
    const response = await request.post("/api/v1/items", {
      data: {
        languageCode: "ko-KR",
        title: `Korean draft ${number}`,
        description: "Library description",
        difficulty: "beginner",
      },
    });
    expect(response.status()).toBe(201);
  }
  const calls: string[] = [];
  page.on("request", (request) => {
    const url = new URL(request.url());
    if (url.pathname.startsWith("/api/v1/items")) calls.push(url.pathname + url.search);
  });
  const read = page.waitForResponse((response) => response.url().includes("/api/v1/items?"));
  await page.goto("/library?languageCode=ko-KR&page=1");
  const first = (await (await read).json()) as ItemPage;
  expect(first.total).toBeGreaterThanOrEqual(10);
  expect(first.items.every((item) => item.languageCode === "ko-KR")).toBe(true);
  const articles = page.getByRole("article");
  await expect(articles).toHaveCount(8);
  await expect(page.getByLabel("Reading language")).toHaveValue("ko-KR");
  await expect(articles.first().getByRole("heading")).toHaveText(
    first.items[0].title || "Untitled draft",
  );
  const draft = first.items.find((item) => item.status === "draft");
  expect(draft).toBeDefined();
  const editLink = page.locator(`a[href="/create?draftId=${draft!.id}"]`);
  await expect(editLink).toHaveAccessibleName("Continue Editing");
  expect(calls).toEqual(["/api/v1/items?languageCode=ko-KR&page=1&pageSize=8"]);

  let release!: () => void;
  const paused = new Promise<void>((resolve) => {
    release = resolve;
  });
  await page.route(
    "**/api/v1/items?*page=2&*",
    async (route) => {
      await paused;
      await route.continue();
    },
    { times: 1 },
  );
  await page.getByRole("link", { name: "Next" }).click();
  await expect(page).toHaveURL(/page=2$/);
  await expect(page.getByText("Refreshing library…")).toBeVisible();
  await expect(articles.first().getByRole("heading")).toHaveText(
    first.items[0].title || "Untitled draft",
  );
  release();
  await expect(page.getByText(/^Page 2 of /)).toBeVisible();
  expect(calls).toEqual([
    "/api/v1/items?languageCode=ko-KR&page=1&pageSize=8",
    "/api/v1/items?languageCode=ko-KR&page=2&pageSize=8",
  ]);

  await page.goBack();
  await editLink.click();
  await expect(page.getByRole("heading", { name: "Edit draft" })).toBeVisible();

  await page.goto("/library?languageCode=ko-KR&page=999");
  await expect(page).not.toHaveURL(/page=999$/);
  await expect(articles.first()).toBeVisible();
});

test("shows recoverable Library errors, a true empty list, and a working published Reader link", async ({
  page,
  request,
}) => {
  const response = await request.post("/api/v1/items", {
    data: {
      languageCode: "ko-KR",
      title: "Published fixture",
      sourceText: "원문.",
      difficulty: "advanced",
    },
  });
  expect(response.status()).toBe(201);
  const item = (await response.json()) as Item;
  const published: Item = { ...item, status: "published", publishedAt: item.createdAt };
  let lists = 0;
  await page.route("**/api/v1/items?*", (route) => {
    lists++;
    if (lists === 1)
      return route.fulfill({ status: 503, json: { error: { code: "SERVICE_UNAVAILABLE" } } });
    const language = new URL(route.request().url()).searchParams.get("languageCode");
    const items = language === "ko-KR" ? [published] : [];
    return route.fulfill({ json: { items, total: items.length, page: 1, pageSize: 8 } });
  });
  await page.route(`**/api/v1/items/${item.id}`, (route) => route.fulfill({ json: published }));
  await page.goto("/library?languageCode=zh-CN");
  await expect(page.getByRole("alert")).toContainText("Could not load the Library");
  await page.getByRole("button", { name: "Retry" }).click();
  await expect(
    page.getByRole("heading", { name: "No saved articles in this language" }),
  ).toBeVisible();
  await expect(page.getByRole("link", { name: "Create New Item" })).toHaveAttribute(
    "href",
    "/create",
  );
  await page.getByLabel("Reading language").selectOption("ko-KR");
  const card = page.getByRole("article");
  await expect(card.getByText("Published", { exact: true })).toBeVisible();
  await expect(card.getByText("Advanced")).toBeVisible();
  await card.getByRole("link", { name: "Open in Reader" }).click();
  await expect(page.getByRole("heading", { name: "Published fixture" })).toBeVisible();
  await expect(page.getByText("원문.", { exact: true })).toBeVisible();
});
