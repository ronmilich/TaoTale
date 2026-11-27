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

test("creates, assigns, renames, recolors and deletes a topic shared across languages", async ({
  page,
  request,
}) => {
  const name = `Tea ${crypto.randomUUID().slice(0, 8)}`;
  await page.goto("/create");
  await page.getByLabel("Item language").selectOption("zh-CN");
  await page.getByLabel("Title").fill(`${name} Chinese`);
  await page.getByRole("button", { name: "Create topic", exact: true }).click();
  await page.getByLabel("Topic name").fill(name);
  await page.getByRole("radio", { name: "Sage" }).check();
  await page.getByRole("button", { name: "Save topic" }).click();
  await expect(page.getByRole("checkbox", { name, exact: true })).toBeChecked();
  await page.getByRole("button", { name: "Save Draft", exact: true }).click();
  await expect(page).toHaveURL(/draftId=/);
  const chineseID = new URL(page.url()).searchParams.get("draftId");

  await page.goto("/create");
  await page.getByLabel("Item language").selectOption("ko-KR");
  await page.getByLabel("Title").fill(`${name} Korean`);
  await page.getByRole("checkbox", { name, exact: true }).check();
  await page.getByRole("button", { name: "Save Draft", exact: true }).click();
  await expect(page).toHaveURL(/draftId=/);
  const koreanID = new URL(page.url()).searchParams.get("draftId");
  const chinese = (await (await request.get(`/api/v1/items/${chineseID}`)).json()) as Item;
  const korean = (await (await request.get(`/api/v1/items/${koreanID}`)).json()) as Item;
  expect(chinese.topicIds).toHaveLength(1);
  expect(korean.topicIds).toEqual(chinese.topicIds);

  await page.goto(`/library?languageCode=ko-KR&q=${encodeURIComponent(name)}`);
  await page.getByText("Manage topics", { exact: true }).click();
  await expect(page.getByRole("button", { name: "Delete topic History" })).toHaveCount(0);
  await page.getByRole("button", { name: `Edit topic ${name}`, exact: true }).click();
  await page.getByLabel("Topic name").fill("History");
  await page.getByRole("button", { name: "Save topic" }).click();
  await expect(page.getByRole("alert")).toHaveText("A topic already has that name.");
  await expect(page.getByLabel("Topic name")).toHaveValue("History");
  const renamed = `${name} renamed`;
  await page.getByLabel("Topic name").fill(renamed);
  await page.getByRole("radio", { name: "Amber" }).check();
  await page.getByRole("button", { name: "Save topic" }).click();
  const card = page
    .getByRole("article")
    .filter({ has: page.getByRole("heading", { name: `${name} Korean` }) });
  await expect(card.getByText(renamed, { exact: true })).toHaveCSS(
    "background-color",
    test.info().project.name === "tablet" ? "rgb(84, 66, 43)" : "rgb(255, 240, 218)",
  );
  page.once("dialog", async (dialog) => {
    expect(dialog.message()).toContain("2 items and 0 words");
    await dialog.accept();
  });
  await page.getByRole("button", { name: `Delete topic ${renamed}` }).click();
  await expect(card.getByText(renamed, { exact: true })).toHaveCount(0);
  for (const id of [chineseID, koreanID]) {
    const item = (await (await request.get(`/api/v1/items/${id}`)).json()) as Item;
    expect(item.topicIds).toEqual([]);
  }
});

test("combines Library filters and preserves favorite state, ordering and pagination", async ({
  page,
  request,
}) => {
  const prefix = `Filters ${crypto.randomUUID().slice(0, 8)}`;
  const titles = Array.from({ length: 9 }, (_, index) => `${prefix}%_ ${index}`);
  const catalog = await (await request.get("/api/v1/topics")).json();
  const [history, science] = ["history", "science"].map(
    (key) =>
      catalog.items.find((topic: { systemKey: string }) => topic.systemKey === key).id as string,
  );
  for (const [index, title] of [...titles, `${prefix}XY Decoy`].entries()) {
    const response = await request.post("/api/v1/items", {
      data: {
        languageCode: "zh-CN",
        title,
        isFavorite: true,
        topicIds: [index % 2 ? history : science],
      },
    });
    expect(response.status()).toBe(201);
  }
  await page.goto("/library?languageCode=zh-CN&page=1");
  const search = page.getByRole("searchbox", { name: "Search titles" });
  await search.fill(`${prefix}%_`);
  const articles = page.getByRole("article");
  await expect(articles.first().getByRole("heading")).toHaveText(titles[8]);
  await expect(page.getByText("Page 1 of 2")).toBeVisible();
  const favorite = articles.nth(1).getByRole("button", { name: "Remove from Favorites" });
  await page.route("**/api/v1/items/*/favorite", (route) => route.fulfill({ status: 503 }), {
    times: 1,
  });
  await favorite.click();
  await expect(articles.nth(1).getByRole("alert")).toContainText("Could not update favorite");
  await expect(favorite).toHaveAttribute("aria-pressed", "true");
  await favorite.click();
  const add = articles.nth(1).getByRole("button", { name: "Favorite", exact: true });
  await expect(add).toHaveAttribute("aria-pressed", "false");
  await add.click();
  await expect(favorite).toHaveAttribute("aria-pressed", "true");
  expect(await articles.getByRole("heading").allTextContents()).toEqual(titles.slice(1).reverse());

  await page.getByLabel("Difficulty", { exact: true }).selectOption("unspecified");
  await page.getByLabel("Status", { exact: true }).selectOption("draft");
  await page.getByText("Topics", { exact: true }).click();
  await page.getByRole("checkbox", { name: "History", exact: true }).check();
  await page.getByRole("checkbox", { name: "Science", exact: true }).check();
  await page.getByRole("checkbox", { name: "Favorites only" }).check();
  const sort = page.getByLabel("Sort", { exact: true });
  await sort.selectOption("title_asc");
  await expect(articles.first().getByRole("heading")).toHaveText(titles[0]);
  await page.goBack();
  await expect(sort).toHaveValue("");
  await page.getByRole("link", { name: "Next" }).click();
  await expect(page.getByText("Page 2 of 2")).toBeVisible();
  await expect(articles.getByRole("heading")).toHaveText(titles[0]);
  await articles.getByRole("button", { name: "Remove from Favorites" }).click();
  await expect(page).toHaveURL(/page=1$/);
  await expect(articles).toHaveCount(8);
  expect(await articles.getByRole("heading").allTextContents()).toEqual(titles.slice(1).reverse());
  await page.reload();
  await expect(articles).toHaveCount(8);
  await page.getByLabel("Status", { exact: true }).selectOption("published");
  await expect(page.getByRole("heading", { name: "No items match your filters" })).toBeVisible();
  await page.getByRole("button", { name: "Clear filters" }).click();
  await expect(page).toHaveURL(/\/library\?languageCode=zh-CN&page=1$/);
});
