import { expect, test } from "@playwright/test";
import type { Item, WorkingRevision } from "../src/api/client";

test("revises a published article while its original recording stays available", async ({
  page,
  request,
}) => {
  const title = `Revision journey ${crypto.randomUUID()}`;
  const originalSource = "你好。欢迎来学习。";
  const revisedSource = "再见。我们明天继续。";
  await page.goto("/create");
  await page.getByLabel("Item language").selectOption("zh-CN");
  await page.getByLabel("Title", { exact: true }).fill(title);
  const source = page.getByLabel("Source text", { exact: true });
  await source.fill("你好。");
  await page.getByRole("button", { name: "Save Draft", exact: true }).click();
  await expect(page).toHaveURL(/draftId=/);
  await source.fill(originalSource);
  await page.getByRole("button", { name: "Save Draft", exact: true }).click();
  await expect(page.getByText("Draft saved", { exact: true })).toBeVisible();
  const generate = page.getByRole("button", { name: "Generate Audio", exact: true });
  await expect(generate).toBeEnabled();
  await generate.click();
  const publishDraft = page.getByRole("button", { name: "Publish to Library" });
  await expect(publishDraft).toBeEnabled();
  await publishDraft.click();
  await expect(page).toHaveURL(/\/items\/[0-9a-f-]+$/);
  const reader = new URL(page.url()).pathname;
  const itemEndpoint = `/api/v1${reader}`;
  const getItem = async () => (await (await request.get(itemEndpoint)).json()) as Item;
  const original = await getItem();
  const originalAudio = `${itemEndpoint}/audio?audioAssetId=${original.audioAssetId}`;
  const firstStream = await request.get(originalAudio);
  expect(firstStream.status()).toBe(200);
  const originalBytes = await firstStream.body();
  const resumeKey = `taotale.playback.v1.${original.id}.${original.audioAssetId}`;
  await page.evaluate((key) => localStorage.setItem(key, "250"), resumeKey);

  await page.getByRole("link", { name: "Edit article", exact: true }).click();
  await page.getByLabel("Title", { exact: true }).fill(`${title} updated`);
  await page.getByRole("button", { name: "Save Changes", exact: true }).click();
  await expect.poll(async () => (await getItem()).title).toBe(`${title} updated`);
  async function expectOriginal() {
    const current = await getItem();
    expect(current.sourceText).toBe(originalSource);
    expect(current.audioAssetId).toBe(original.audioAssetId);
    const stream = await request.get(originalAudio);
    expect(stream.status()).toBe(200);
    expect(await stream.body()).toEqual(originalBytes);
  }
  await expectOriginal();

  const open = page.getByRole("link", { name: "Edit Text & Voice", exact: true });
  await open.click();
  await expect(source).toHaveValue(originalSource);
  const publishRevision = page.getByRole("button", { name: "Publish Revision" });
  await expect(publishRevision).toBeDisabled();
  await source.fill(revisedSource);
  await expect(generate).toBeDisabled();
  await page.getByRole("link", { name: "Back to Reader" }).click();
  const leave = page.getByRole("dialog", { name: "Leave with unsaved changes?" });
  await expect(leave).toBeVisible();
  await leave.getByRole("button", { name: "Stay" }).click();
  await expect(source).toHaveValue(revisedSource);

  await page.route("**/revisions/*", (route) => route.fulfill({ status: 503 }), { times: 1 });
  await page.getByRole("link", { name: "Back to Reader" }).click();
  await leave.getByRole("button", { name: "Save Revision & Leave" }).click();
  await expect(page.getByRole("alert")).toBeVisible();
  await expect(source).toHaveValue(revisedSource);
  await page.getByRole("link", { name: "Back to Reader" }).click();
  await leave.getByRole("button", { name: "Save Revision & Leave" }).click();
  await expect(page).toHaveURL(reader);
  await expectOriginal();
  await page.getByRole("link", { name: "Edit article", exact: true }).click();
  await open.click();
  await expect(source).toHaveValue(revisedSource);

  await page.route(
    "**/revisions/*/audio/generate",
    (route) =>
      route.fulfill({
        status: 503,
        json: { error: { code: "TTS_UNAVAILABLE", message: "Speech synthesis is unavailable." } },
      }),
    { times: 1 },
  );
  await generate.click();
  await expect(page.getByText("Speech synthesis is unavailable.", { exact: true })).toBeVisible();
  await expect(publishRevision).toBeDisabled();
  await expectOriginal();

  await generate.click();
  await expect(publishRevision).toBeEnabled();
  const candidateAudio = await page.locator("audio").getAttribute("src");
  expect(candidateAudio).toMatch(/\/revisions\/[0-9a-f-]+\/audio\?audioAssetId=/);
  const candidate = await request.get(candidateAudio!);
  expect(candidate.status()).toBe(200);
  await expectOriginal();

  await page.getByRole("button", { name: "Discard Revision", exact: true }).click();
  const discard = page.getByRole("dialog", { name: "Discard this revision?" });
  await discard.getByRole("button", { name: "Keep Revision" }).click();
  await expect(publishRevision).toBeEnabled();
  await page.getByRole("button", { name: "Discard Revision", exact: true }).click();
  await discard.getByRole("button", { name: "Discard Revision", exact: true }).click();
  await expect(page).toHaveURL(`${reader}/edit`);
  expect((await request.get(candidateAudio!)).status()).toBe(404);
  await expectOriginal();
  expect(await page.evaluate((key) => localStorage.getItem(key), resumeKey)).toBe("250");

  await open.click();
  await expect(source).toHaveValue(originalSource);
  await source.fill(revisedSource);
  const save = page.getByRole("button", { name: "Save Revision", exact: true });
  await save.click();
  await expect(page.getByText("Revision saved", { exact: true })).toBeVisible();
  await generate.click();
  await expect(publishRevision).toBeEnabled();
  const revision = (await (
    await request.post(`${itemEndpoint}/revisions`)
  ).json()) as WorkingRevision;
  expect(revision.audioStatus).toBe("READY");
  await page.route(
    "**/revisions/*/publish",
    async (route) => {
      const result = await route.fetch();
      expect(result.status()).toBe(200);
      await route.abort();
    },
    { times: 1 },
  );
  await publishRevision.click();
  await page.getByRole("button", { name: "Check Publication" }).click();
  await expect(page).toHaveURL(reader);
  await expect(page.getByRole("heading", { name: `${title} updated` })).toBeVisible();
  const published = await getItem();
  expect(published.sourceText).toBe(revisedSource);
  expect(published.audioAssetId).toBe(revision.audioAssetId);
  await expect(page.locator("audio")).toHaveAttribute(
    "src",
    new RegExp(`audioAssetId=${published.audioAssetId}$`),
  );
  await expect
    .poll(() => page.locator("audio").evaluate((element: HTMLAudioElement) => element.currentTime))
    .toBe(0);
  expect(await page.evaluate((key) => localStorage.getItem(key), resumeKey)).toBeNull();
  expect((await request.get(originalAudio)).status()).toBe(404);
});
