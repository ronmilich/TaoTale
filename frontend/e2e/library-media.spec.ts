import { expect, test, type Locator } from "@playwright/test";
import type { Item } from "../src/api/client";

function selectCover(
  input: Locator,
  options: { width?: number; size?: number; invalid?: boolean } = {},
) {
  return input.evaluate((input: HTMLInputElement, { width = 1, size, invalid }) => {
    const canvas = document.createElement("canvas");
    canvas.width = width;
    canvas.height = 1;
    const png = Uint8Array.fromBase64(canvas.toDataURL("image/png").split(",")[1]);
    const bytes = new Uint8Array(size ?? png.length);
    if (!invalid) bytes.set(png);
    const transfer = new DataTransfer();
    transfer.items.add(new File([bytes], "cover.png", { type: "image/png" }));
    input.files = transfer.files;
    input.dispatchEvent(new Event("change", { bubbles: true }));
  }, options);
}

test("checks cover limits, preserves a failed replacement, and displays the saved cover in the Library", async ({
  page,
  request,
}) => {
  await page.goto("/create");
  const title = `Cover journey ${crypto.randomUUID()}`;
  await page.getByRole("combobox", { name: "Item language" }).selectOption("zh-CN");
  await page.getByLabel("Title", { exact: true }).fill(title);
  const fileInput = page.getByLabel("Cover image", { exact: true });
  const preview = page.getByRole("img", { name: "Cover preview" });
  const save = page.getByRole("button", { name: "Save Draft", exact: true });
  await selectCover(fileInput, { size: 1_000_000 });
  await expect(preview).toBeVisible();
  const original = (await preview.getAttribute("src"))!;
  await save.click();
  await expect(page).toHaveURL(/draftId=/);
  const id = new URL(page.url()).searchParams.get("draftId");
  const endpoint = `/api/v1/items/${id}`;
  const savedCover = async () =>
    ((await (await request.get(endpoint)).json()) as Item).coverImageId;
  expect(await savedCover()).toBe(original.split("/").at(-1));

  let uploadAttempts = 0;
  page.on("request", (request) => {
    if (request.method() === "POST" && request.url().endsWith("/api/v1/storage/images"))
      uploadAttempts++;
  });
  await selectCover(fileInput, { size: 1_000_001 });
  await expect(page.getByRole("alert")).toHaveText(
    "Cover images must be 1,000,000 bytes or smaller.",
  );
  await selectCover(fileInput, { invalid: true });
  await expect(page.getByRole("alert")).toHaveText("Choose a JPEG, PNG or WebP image.");
  await selectCover(fileInput, { width: 4097 });
  await expect(page.getByRole("alert")).toHaveText(
    "Cover images must be at most 4096 × 4096 pixels.",
  );
  expect(uploadAttempts).toBe(0);
  await expect(preview).toHaveAttribute("src", original);
  await expect(save).toBeDisabled();

  await page.route("**/api/v1/storage/images", (route) => route.fulfill({ status: 503 }), {
    times: 1,
  });
  await selectCover(fileInput);
  await expect(page.getByRole("alert")).toHaveText("Could not upload the cover. Try again.");
  await expect(preview).toHaveAttribute("src", original);
  expect(await savedCover()).toBe(original.split("/").at(-1));
  await page.getByRole("button", { name: "Retry cover upload" }).click();
  await expect(preview).not.toHaveAttribute("src", original);
  const replacement = (await preview.getAttribute("src"))!;
  expect(await savedCover()).toBe(original.split("/").at(-1));
  await save.click();
  await expect(page.getByText("Draft saved", { exact: true })).toBeVisible();
  expect(await savedCover()).toBe(replacement.split("/").at(-1));

  await page.goto(`/library?languageCode=zh-CN&q=${encodeURIComponent(title)}`);
  const article = page.getByRole("article").filter({ hasText: title });
  const thumbnail = article.locator("img");
  await expect(thumbnail).toHaveAttribute("src", replacement);
  await expect(thumbnail).toHaveAttribute("loading", "lazy");
  await expect
    .poll(() => thumbnail.evaluate((image: HTMLImageElement) => image.naturalWidth))
    .toBe(1);
  await page.route(`**${replacement}`, (route) => route.fulfill({ status: 404 }), { times: 1 });
  await page.reload();
  await expect(thumbnail).toHaveAttribute("hidden", "");
  await article.getByRole("link", { name: "Continue Editing" }).click();
  await page.getByRole("button", { name: "Remove cover" }).click();
  await expect(page.getByText("No cover selected")).toBeVisible();
  await save.click();
  await expect(page.getByText("Draft saved", { exact: true })).toBeVisible();
  expect(await savedCover()).toBeNull();
});

test("downloads published audio, confirms deletion, and returns to the remaining Library page", async ({
  page,
  request,
}) => {
  const prefix = `Library media ${crypto.randomUUID()}`;
  for (let number = 1; number <= 8; number++) {
    const response = await request.post("/api/v1/items", {
      data: { languageCode: "ko-KR", title: `${prefix} draft ${number}` },
    });
    expect(response.status()).toBe(201);
  }

  await page.goto("/create");
  const title = `${prefix} recording`;
  await page.getByRole("combobox", { name: "Item language" }).selectOption("ko-KR");
  await page.getByLabel("Title", { exact: true }).fill(title);
  await page.getByLabel("Source text", { exact: true }).fill("안녕하세요.");
  await selectCover(page.getByLabel("Cover image", { exact: true }));
  const preview = page.getByRole("img", { name: "Cover preview" });
  await expect(preview).toBeVisible();
  const cover = (await preview.getAttribute("src"))!;
  await page.getByRole("button", { name: "Generate Preview" }).click();
  const publish = page.getByRole("button", { name: "Publish to Library" });
  await expect(publish).toBeEnabled();
  await publish.click();
  await expect(page).toHaveURL(/\/items\/[0-9a-f-]+$/);
  const id = new URL(page.url()).pathname.split("/").at(-1);
  const endpoint = `/api/v1/items/${id}`;
  const item = (await (await request.get(endpoint)).json()) as Item;
  expect(item.coverImageId).toBe(cover.split("/").at(-1));
  const downloadURL = `${endpoint}/audio/download?audioAssetId=${item.audioAssetId}`;

  await page.goto(`/library?languageCode=ko-KR&q=${encodeURIComponent(prefix)}&sort=oldest&page=2`);
  await expect(page.getByText("Page 2 of 2")).toBeVisible();
  const card = page.getByRole("article").filter({ hasText: title });
  const downloadButton = card.getByRole("button", { name: "Download Audio" });
  await page.route(
    `**${downloadURL}`,
    (route) => route.fulfill({ status: 503, json: { error: { code: "AUDIO_UNAVAILABLE" } } }),
    { times: 1 },
  );
  await downloadButton.click();
  await expect(card.getByRole("alert")).toHaveText("Audio unavailable.");

  const downloadEvent = page.waitForEvent("download");
  await downloadButton.click();
  const download = await downloadEvent;
  expect(download.suggestedFilename()).toBe(`taotale-${id}.mp3`);
  const bytes: number[] = [];
  for await (const chunk of await download.createReadStream()) bytes.push(...chunk);
  const audio = await request.get(downloadURL);
  expect(audio.status()).toBe(200);
  expect(bytes).toEqual(Array.from(await audio.body()));

  const deleteButton = card.getByRole("button", { name: `Delete ${title}` });
  page.once("dialog", async (dialog) => {
    expect(dialog.message()).toContain("any recording will be removed");
    expect(dialog.message()).toContain("Linked vocabulary entries will remain");
    await dialog.dismiss();
  });
  await deleteButton.click();
  expect((await request.get(endpoint)).status()).toBe(200);

  await page.route(`**${endpoint}`, (route) => route.fulfill({ status: 503 }), { times: 1 });
  page.once("dialog", (dialog) => dialog.accept());
  await deleteButton.click();
  await expect(card.getByRole("alert")).toHaveText("Could not delete this article. Try again.");

  page.once("dialog", (dialog) => dialog.accept());
  await deleteButton.click();
  await expect(page).toHaveURL(/page=1$/);
  await expect(page.getByRole("article")).toHaveCount(8);
  await expect(card).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Download Audio" })).toHaveCount(0);
  expect((await request.get(endpoint)).status()).toBe(404);
  expect((await request.get(downloadURL)).status()).toBe(404);
  expect((await request.get(cover)).status()).toBe(404);
});
