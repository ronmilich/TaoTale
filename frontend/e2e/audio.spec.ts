import { expect, test, type Page } from "@playwright/test";
import type { Item } from "../src/api/client";

async function draftEditor(page: Page, id: string) {
  await page.goto(`/create?draftId=${id}`);
  const audio = page.getByRole("region", { name: "Audio", exact: true });
  await expect(audio.getByLabel("Voice")).toHaveValue("zh-CN-mock");
  return audio;
}

test("recovers an admitted generation with the same key and body and plays pinned audio", async ({
  page,
  request,
}) => {
  const seeded = await request.post("/api/v1/items", {
    data: { languageCode: "zh-CN", title: "Pinned audio", sourceText: "你好，世界。" },
  });
  expect(seeded.status()).toBe(201);
  const { id } = (await seeded.json()) as Item;
  const audio = await draftEditor(page, id);
  const generate = audio.getByRole("button", { name: "Generate Audio" });
  await audio.getByLabel("Speaking rate").selectOption("0.75");
  await expect(generate).toBeDisabled();
  await page.getByRole("button", { name: "Save Draft", exact: true }).click();
  await expect(generate).toBeEnabled();

  const attempts: { key: string; body: string | null; job?: string }[] = [];
  await page.route(`**/api/v1/items/${id}/audio/generate`, async (route) => {
    const submitted = route.request();
    const reply = await route.fetch();
    attempts.push({
      key: submitted.headers()["idempotency-key"],
      body: submitted.postData(),
      job: (await reply.json()).generationId,
    });
    if (attempts.length === 1) await route.abort();
    else await route.fulfill({ response: reply });
  });
  await generate.click();
  const recover = audio.getByRole("button", { name: "Recover generation" });
  await expect(recover).toBeVisible();
  expect(attempts).toHaveLength(1);
  await expect(generate).toBeDisabled();
  await recover.click();
  await expect(audio.getByRole("status").filter({ hasText: /^Ready$/ })).toBeVisible();
  expect(attempts).toHaveLength(2);
  expect(attempts[0].key).toMatch(/^[0-9a-f-]{36}$/);
  expect(attempts[0].job).toMatch(/^[0-9a-f-]{36}$/);
  expect(JSON.parse(attempts[0].body!)).toEqual({ expectedVersion: 2 });
  expect(attempts[1]).toEqual(attempts[0]);

  const player = audio.locator("audio");
  await expect(player).toHaveAttribute("src", /\/audio\?audioAssetId=[0-9a-f-]{36}$/);
  await expect
    .poll(() => player.evaluate((element) => (element as HTMLAudioElement).duration))
    .toBeGreaterThan(0);
  await player.evaluate(async (element) => {
    const media = element as HTMLAudioElement;
    await media.play();
    media.pause();
  });
  const download = await audio.getByRole("link", { name: "Download Audio" }).getAttribute("href");
  const ranged = await request.get(download!, { headers: { Range: "bytes=0-5" } });
  expect(ranged.status()).toBe(206);
  expect(ranged.headers()["content-type"]).toBe("audio/mpeg");

  await page.getByLabel("Source text", { exact: true }).fill("新的文字。");
  await expect(audio.getByText("Outdated", { exact: true })).toBeVisible();
  await expect(player).toHaveCount(0);
  await expect(generate).toBeDisabled();
  await page.getByRole("button", { name: "Save Draft", exact: true }).click();
  await expect(generate).toBeEnabled();
  await expect(audio.getByText("Outdated", { exact: true })).toBeVisible();
});

test("creates a temporary preview without a draft and claims its asset on Save Draft", async ({
  page,
  request,
}) => {
  await page.goto("/create");
  await page.getByRole("combobox", { name: "Item language" }).selectOption("ko-KR");
  const audio = page.getByRole("region", { name: "Audio", exact: true });
  await expect(audio.getByLabel("Voice")).toHaveValue("ko-KR-mock");
  await audio.getByLabel("Speaking rate").selectOption("1.25");
  await page.getByLabel("Source text", { exact: true }).fill("안녕하세요.");
  const items: string[] = [];
  page.on("request", (request) => {
    if (request.method() === "POST" && new URL(request.url()).pathname === "/api/v1/items")
      items.push(request.url());
  });
  await audio.getByRole("button", { name: "Generate Preview" }).click();
  await expect(audio.getByText("Ready", { exact: true })).toBeVisible();
  expect(items).toEqual([]);
  await expect(page).toHaveURL(/\/create$/);
  const temporary = await audio.locator("audio").getAttribute("src");
  expect(temporary).toMatch(/\/tts\/previews\/[0-9a-f-]{36}\/audio$/);
  const statusURL = temporary!.replace(/\/audio$/, "");
  const preview = await (await request.get(statusURL)).json();

  await page.getByRole("button", { name: "Save Draft", exact: true }).click();
  await expect(page).toHaveURL(/\/create\?draftId=/);
  await expect(audio.locator("audio")).toHaveAttribute(
    "src",
    new RegExp(`audioAssetId=${preview.audioAssetId}$`),
  );
  expect((await request.get(statusURL)).status()).toBe(404);
  expect(items).toHaveLength(1);
  await page.goto("/create");
  await page.getByRole("combobox", { name: "Item language" }).selectOption("ko-KR");
  await expect(audio.getByLabel("Speaking rate")).toHaveValue("1.25");
});

test("continues status reads across a restart without resubmitting generation", async ({
  page,
  request,
}) => {
  const seeded = await request.post("/api/v1/items", {
    data: { languageCode: "zh-CN", title: "Interrupt fixture", sourceText: "你好。" },
  });
  expect(seeded.status()).toBe(201);
  const { id } = (await seeded.json()) as Item;
  const audio = await draftEditor(page, id);
  let submissions = 0;
  let polls = 0;
  const generationId = "a2230000-0000-4000-8000-000000000001";
  await page.route(`**/api/v1/items/${id}/audio/generate`, (route) => {
    submissions++;
    return route.fulfill({ status: 202, json: { generationId } });
  });
  await page.route(`**/api/v1/items/${id}/audio/generations/${generationId}`, (route) => {
    polls++;
    if (polls === 1)
      return route.fulfill({ status: 503, json: { error: { code: "SERVICE_UNAVAILABLE" } } });
    return route.fulfill({
      json: {
        generationId,
        itemId: id,
        status: "FAILED",
        jobStatus: "FAILED",
        sourceFingerprint: "",
        audioAssetId: null,
        errorCode: "GENERATION_INTERRUPTED",
      },
    });
  });
  await audio.getByRole("button", { name: "Generate Audio" }).click();
  await expect(audio.getByText("Connection lost. Checking audio status…")).toBeVisible();
  await expect(
    audio.getByText("The backend restarted during generation.", { exact: false }),
  ).toBeVisible();
  await expect(audio.getByText("Failed", { exact: true })).toBeVisible();
  expect(submissions).toBe(1);
  expect(polls).toBe(2);
});

test("shows unavailable voices and admission rate limits without automatic retries", async ({
  page,
  request,
}) => {
  const seeded = await request.post("/api/v1/items", {
    data: { languageCode: "zh-CN", sourceText: "你好。" },
  });
  expect(seeded.status()).toBe(201);
  const { id } = (await seeded.json()) as Item;
  await page.route(
    "**/api/v1/tts/voices?*",
    (route) => route.fulfill({ status: 503, json: { error: { code: "TTS_UNAVAILABLE" } } }),
    { times: 1 },
  );
  await page.goto(`/create?draftId=${id}`);
  const audio = page.getByRole("region", { name: "Audio", exact: true });
  const generate = audio.getByRole("button", { name: "Generate Audio" });
  await expect(audio.getByText("Speech synthesis is unavailable.", { exact: false })).toBeVisible();
  await expect(generate).toBeDisabled();
  await audio.getByRole("button", { name: "Retry voices" }).click();
  await expect(generate).toBeEnabled();

  let attempts = 0;
  await page.route(`**/api/v1/items/${id}/audio/generate`, (route) => {
    attempts++;
    return route.fulfill({
      status: 429,
      json: {
        error: { code: "TTS_RATE_LIMITED", message: "The speech generation limit was reached." },
      },
    });
  });
  await generate.click();
  await expect(audio.getByRole("alert")).toContainText("The speech generation limit was reached.");
  await expect(audio.getByRole("button", { name: "Recover generation" })).toHaveCount(0);
  await expect(generate).toBeEnabled();
  expect(attempts).toBe(1);
});
