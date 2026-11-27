import { expect, test } from "@playwright/test";
import type { Item } from "../src/api/client";

test("publishes a saved draft with its reviewed audio and replaces the editor in history", async ({
  page,
  request,
}) => {
  await page.goto("/library");
  await page.getByRole("link", { name: "New article" }).click();
  const publish = page.getByRole("button", { name: "Publish to Library" });
  await expect(publish).toBeDisabled();
  await expect(publish).toHaveAccessibleDescription(
    "Add a title and source text before publishing.",
  );
  await page.getByLabel("Title", { exact: true }).fill("Before publication");
  await page.getByLabel("Source text", { exact: true }).fill("今天的天气很好。");
  await page.getByRole("button", { name: "Save Draft", exact: true }).click();
  await expect(page).toHaveURL(/\/create\?draftId=/);
  const id = new URL(page.url()).searchParams.get("draftId");
  await expect(publish).toHaveAccessibleDescription("Generate matching audio before publishing.");
  await page.getByRole("button", { name: "Generate Audio" }).click();
  await expect(publish).toBeEnabled();
  const player = page.locator("audio");
  const pin = await player.getAttribute("src");
  await page.getByLabel("Source text", { exact: true }).fill("今天的天气很好。 ");
  await expect(publish).toBeDisabled();
  await page.getByLabel("Source text", { exact: true }).fill("今天的天气很好。");
  await page.getByLabel("Title", { exact: true }).fill("Published from the editor");

  await page.route(
    `**/api/v1/items/${id}/publish`,
    (route) =>
      route.fulfill({
        status: 503,
        json: {
          error: { code: "AUDIO_UNAVAILABLE", message: "The verified recording is unavailable." },
        },
      }),
    { times: 1 },
  );
  await publish.click();
  await expect(page.getByRole("alert")).toContainText("recording is unavailable");
  await expect(player).toHaveAttribute("src", pin!);
  const draft = (await (await request.get(`/api/v1/items/${id}`)).json()) as Item;
  expect(draft.status).toBe("draft");
  expect(draft.title).toBe("Published from the editor");

  await publish.click();
  await expect(page).toHaveURL(new RegExp(`/items/${id}$`));
  await expect(page.getByRole("heading", { name: draft.title! })).toBeVisible();
  const published = (await (await request.get(`/api/v1/items/${id}`)).json()) as Item;
  expect(published.status).toBe("published");
  expect(published.audioAssetId).toBe(draft.audioAssetId);
  expect((await request.get(pin!, { headers: { Range: "bytes=0-5" } })).status()).toBe(206);
  await page.goBack();
  await expect(page).toHaveURL(/\/library(?:\?|$)/);
});

test("recovers direct publication after a lost response using the original preview and key", async ({
  page,
  request,
}) => {
  await page.goto("/create");
  await page.getByRole("combobox", { name: "Item language" }).selectOption("ko-KR");
  await page.getByLabel("Title", { exact: true }).fill("A Korean preview");
  await page.getByLabel("Source text", { exact: true }).fill("안녕하세요.");
  await page.getByRole("button", { name: "Generate Preview" }).click();
  const publish = page.getByRole("button", { name: "Publish to Library" });
  await expect(publish).toBeEnabled();
  const preview = (await page.locator("audio").getAttribute("src"))!.replace(/\/audio$/, "");
  const before = await (await request.get(preview)).json();
  const attempts: { key: string; body: string | null; id: string }[] = [];
  await page.route("**/api/v1/items/publish", async (route) => {
    const response = await route.fetch();
    expect(response.status()).toBe(201);
    attempts.push({
      key: route.request().headers()["idempotency-key"],
      body: route.request().postData(),
      id: (await response.json()).itemId,
    });
    if (attempts.length === 1) await route.abort();
    else await route.fulfill({ response });
  });
  await publish.click();
  const recover = page.getByRole("button", { name: "Recover publication" });
  await expect(recover).toBeEnabled();
  await expect(page.getByLabel("Title", { exact: true })).toBeDisabled();
  await expect(page.getByRole("button", { name: "Save Draft", exact: true })).toBeDisabled();
  expect(attempts).toHaveLength(1);
  await recover.click();
  await expect(page).toHaveURL(new RegExp(`/items/${attempts[0].id}$`));
  await expect(page.getByRole("heading", { name: "A Korean preview" })).toBeVisible();
  expect(attempts).toHaveLength(2);
  expect(attempts[0].key).toMatch(/^[0-9a-f-]{36}$/);
  expect(attempts[1]).toEqual(attempts[0]);
  expect((await request.get(preview)).status()).toBe(404);
  const published = (await (await request.get(`/api/v1/items/${attempts[0].id}`)).json()) as Item;
  expect(published.audioAssetId).toBe(before.audioAssetId);
  expect(published.languageCode).toBe("ko-KR");
});
