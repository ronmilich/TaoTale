import { expect, test, type Page } from "@playwright/test";

async function openNewDraft(page: Page) {
  await page.goto("/library");
  await page.getByRole("link", { name: "New article" }).click();
  await expect(page.getByRole("heading", { name: "New article" })).toBeVisible();
  await expect(page.getByRole("combobox", { name: "Item language" })).toBeEnabled();
}

function trackDraftWrites(page: Page) {
  const writes: string[] = [];
  page.on("request", (request) => {
    const method = request.method();
    const pathname = new URL(request.url()).pathname;
    if (pathname.startsWith("/api/v1/items") && (method === "POST" || method === "PATCH")) {
      writes.push(method);
    }
  });
  return writes;
}

test("creates, reloads, and continues a draft with exact source text", async ({
  page,
  request,
}) => {
  const writes = trackDraftWrites(page);
  await openNewDraft(page);
  const language = page.getByRole("combobox", { name: "Item language" });
  await language.selectOption("ko-KR");
  const title = page.getByLabel("Title", { exact: true });
  const source = page.getByLabel("Source text", { exact: true });
  const save = page.getByRole("button", { name: "Save Draft", exact: true });
  await title.fill("𠀀".repeat(121));
  await expect(title).toHaveAttribute("aria-invalid", "true");
  await expect(page.getByText("Use at most 120 characters.")).toBeVisible();
  await title.fill("𠀀".repeat(120));
  await expect(title).not.toHaveAttribute("aria-invalid", "true");

  await source.fill("𠀀".repeat(751));
  await expect(source).toHaveAttribute("aria-invalid", "true");
  await expect(page.getByText("1502 / 1500 UTF-16 units")).toBeVisible();
  await save.click();
  await expect(page).toHaveURL(/\/create$/);
  expect(writes).toEqual([]);

  const exactSource = "  𠀀 첫 줄\n\n다음 줄.  ";
  await title.fill("First draft");
  await source.fill(exactSource);
  await page.getByLabel("Description").fill("A short description");
  await page.getByLabel("Notes").fill("Draft notes");
  await page.getByLabel("Difficulty").selectOption("intermediate");
  const historyLength = await page.evaluate(() => history.length);
  await save.click();
  await expect(page).toHaveURL(/\/create\?draftId=[0-9a-f-]{36}$/);
  expect(await page.evaluate(() => history.length)).toBe(historyLength);
  await expect(page.getByRole("heading", { name: "Edit draft" })).toBeVisible();
  await expect(page.getByText("Draft saved", { exact: true })).toBeVisible();

  const draftId = new URL(page.url()).searchParams.get("draftId");
  await page.reload();
  await expect(title).toHaveValue("First draft");
  await expect(source).toHaveValue(exactSource);
  await expect(source).toHaveAttribute("rows", "9");
  await expect(page.getByLabel("Item language")).toHaveValue("Korean");
  await expect(page.getByLabel("Item language")).toHaveAttribute("readonly", "");
  await expect(page.getByLabel("Description")).toHaveValue("A short description");
  await expect(page.getByLabel("Notes")).toHaveValue("Draft notes");
  await expect(page.getByLabel("Difficulty")).toHaveValue("intermediate");
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  const textPanel = await page.getByRole("region", { name: "Article text" }).boundingBox();
  const detailsPanel = await page.getByRole("region", { name: "Article details" }).boundingBox();
  if (!textPanel || !detailsPanel) throw new Error("The editor panels are missing.");
  if (page.viewportSize()!.width > 940) {
    expect(detailsPanel.x).toBeGreaterThan(textPanel.x + textPanel.width);
  } else {
    expect(detailsPanel.y).toBeGreaterThan(textPanel.y + textPanel.height);
  }

  await source.fill(exactSource + "\n마지막 줄.");
  await save.click();
  await expect(page.getByText("Draft saved", { exact: true })).toBeVisible();
  const saved = await request.get(`/api/v1/items/${draftId}`);
  expect(saved.ok()).toBe(true);
  expect(await saved.json()).toMatchObject({
    languageCode: "ko-KR",
    sourceText: exactSource + "\n마지막 줄.",
    version: 2,
  });
  expect(writes).toEqual(["POST", "PATCH"]);
});

test("guards reload and supports Stay and Leave Without Saving without autosave", async ({
  page,
}) => {
  const writes = trackDraftWrites(page);
  await openNewDraft(page);
  const title = page.getByLabel("Title", { exact: true });
  await title.click();
  await title.fill("Unsaved title");

  const reloadWarning = page.waitForEvent("dialog").then(async (dialog) => {
    expect(dialog.type()).toBe("beforeunload");
    await dialog.dismiss();
  });
  await page.evaluate(() => {
    setTimeout(() => window.location.reload(), 0);
  });
  await reloadWarning;
  await expect(title).toHaveValue("Unsaved title");

  const leave = page.getByRole("dialog", { name: "Leave this draft?" });
  await page.getByRole("link", { name: "Vocabulary" }).click();
  await expect(leave).toBeVisible();
  await leave.getByRole("button", { name: "Stay", exact: true }).click();
  await expect(leave).toBeHidden();
  await expect(page).toHaveURL(/\/create$/);
  await expect(title).toHaveValue("Unsaved title");

  await page.evaluate(() => history.back());
  await expect(leave).toBeVisible();
  await expect(page).toHaveURL(/\/create$/);
  await leave.getByRole("button", { name: "Leave Without Saving" }).click();
  await expect(page).toHaveURL(/\/library$/);
  expect(writes).toEqual([]);
});

test("Save Draft & Leave waits for a successful save and replaces the history entry", async ({
  page,
}) => {
  const writes = trackDraftWrites(page);
  await openNewDraft(page);
  const title = page.getByLabel("Title", { exact: true });
  await title.fill("A title with an empty source");
  const vocabulary = page.getByRole("link", { name: "Vocabulary" });
  const saveAndLeave = page.getByRole("button", { name: "Save Draft & Leave" });
  await page.route(
    "**/api/v1/items",
    (route) =>
      route.fulfill({
        status: 503,
        json: { error: { code: "SERVICE_UNAVAILABLE", message: "Backend storage unavailable." } },
      }),
    { times: 1 },
  );
  await vocabulary.click();
  await saveAndLeave.click();
  await expect(page.getByRole("alert")).toHaveText("Backend storage unavailable.");
  await expect(title).toHaveValue("A title with an empty source");
  await expect(page).toHaveURL(/\/create$/);

  await vocabulary.click();
  await saveAndLeave.click();
  await expect(page).toHaveURL(/\/vocabulary$/);
  await page.goBack();
  await expect(page).toHaveURL(/\/create\?draftId=[0-9a-f-]{36}$/);
  await expect(title).toHaveValue("A title with an empty source");
  await expect(page.getByLabel("Source text")).toHaveValue("");
  await page.goBack();
  await expect(page).toHaveURL(/\/library$/);
  expect(writes).toEqual(["POST", "POST"]);
});

test("reviews version conflicts and preserves exact existing line endings on edit", async ({
  page,
  request,
}) => {
  const originalSource = "甲。\r\n\r乙。 ";
  const created = await request.post("/api/v1/items", {
    data: { languageCode: "zh-CN", title: "Original", sourceText: originalSource },
  });
  expect(created.status()).toBe(201);
  const { id } = await created.json();
  const apiPath = `/api/v1/items/${id}`;
  await page.goto(`/create?draftId=${id}`);
  const title = page.getByLabel("Title", { exact: true });
  const source = page.getByLabel("Source text");
  await expect(title).toHaveValue("Original");
  await expect(source).toHaveValue("甲。\n\n乙。 ");
  await title.fill("My title");
  await source.focus();
  await source.evaluate((element: HTMLTextAreaElement) => element.setSelectionRange(0, 0));
  await page.keyboard.insertText("前");

  const concurrent = await request.patch(apiPath, {
    data: { expectedVersion: 1, title: "Other title", description: "Other description" },
  });
  expect(concurrent.ok()).toBe(true);
  const save = page.getByRole("button", { name: "Save Draft", exact: true });
  await save.click();
  const review = page.getByRole("region", { name: "Review the latest draft" });
  await expect(review).toBeVisible();
  await expect(title).toHaveValue("My title");
  await expect(save).toBeDisabled();
  await review.getByRole("button", { name: "Load latest version" }).click();
  await expect(review.getByText("Other title", { exact: true })).toBeVisible();
  await expect(review.getByText("My title", { exact: true })).toBeVisible();
  await review.getByRole("button", { name: "Keep my edits" }).click();
  await expect(review).toBeHidden();
  await expect(title).toHaveValue("My title");
  await expect(page.getByLabel("Description")).toHaveValue("Other description");
  expect(await (await request.get(apiPath)).json()).toMatchObject({
    title: "Other title",
    version: 2,
  });

  await save.click();
  await expect(page.getByText("Draft saved", { exact: true })).toBeVisible();
  expect(await (await request.get(apiPath)).json()).toMatchObject({
    title: "My title",
    description: "Other description",
    sourceText: "前" + originalSource,
    version: 3,
  });
});
