import { expect, test, type APIRequestContext } from "@playwright/test";
import type { Item } from "../src/api/client";

async function publishedArticle(
  request: APIRequestContext,
  languageCode: Item["languageCode"],
  sourceText: string,
) {
  const draft = await request.post("/api/v1/items", {
    data: { languageCode, title: "Reader fixture", sourceText },
  });
  expect(draft.status()).toBe(201);
  const { id, version } = (await draft.json()) as Item;
  const api = `/api/v1/items/${id}`;
  const job = await request.post(`${api}/audio/generate`, {
    headers: { "Idempotency-Key": crypto.randomUUID() },
    data: { expectedVersion: version },
  });
  expect(job.status()).toBe(202);
  const { generationId } = await job.json();
  await expect
    .poll(
      async () => {
        const response = await request.get(`${api}/audio/generations/${generationId}`);
        return (await response.json()).status;
      },
      { timeout: 15_000 },
    )
    .toBe("READY");
  const published = await request.post(`${api}/publish`, { data: { expectedVersion: version } });
  expect(published.status()).toBe(200);
  return api;
}

const chinese = "  第一行𠀀。🙂\r\n\r\n第二行，保留空行。\t ";
const korean = "  첫 줄이에요.🙂\n\n둘째 줄도 그대로예요.\r ";

for (const [language, source] of [
  ["zh-CN", chinese],
  ["ko-KR", korean],
] as const) {
  test(`renders exact published ${language} text from stored segments`, async ({
    page,
    request,
  }) => {
    const api = await publishedArticle(request, language, source);
    const item = (await (await request.get(api)).json()) as Item;
    expect(item.transcript?.segments.length).toBeGreaterThan(1);
    await page.goto(api.replace("/api/v1", ""));
    const transcript = page.getByRole("region", { name: "Transcript" });
    await expect(transcript).toHaveAttribute("lang", language);
    await expect(transcript).toHaveCSS("white-space", "pre-wrap");
    const utf8 = new TextEncoder();
    expect(utf8.encode((await transcript.textContent())!)).toEqual(utf8.encode(source));
    const segments = transcript.locator("[data-segment-index]");
    expect(await segments.allTextContents()).toEqual(
      item.transcript!.segments.map(({ text }) => text),
    );
    const player = page.getByRole("region", { name: "Article audio" });
    await expect(player).toHaveCSS("position", "fixed");
    await expect(player.locator("audio")).toHaveAttribute(
      "src",
      new RegExp(`audioAssetId=${item.audioAssetId}$`),
    );
    await expect
      .poll(() => player.locator("audio").evaluate((audio: HTMLAudioElement) => audio.duration))
      .toBeGreaterThan(0);
    if (language === "zh-CN") {
      await expect(transcript).toHaveCSS("font-family", /TaoTale Noto Serif SC/);
      expect(
        await page.evaluate(async () =>
          (await document.fonts.load('22px "TaoTale Noto Serif SC"', "第一行")).some(
            (font) => font.status === "loaded",
          ),
        ),
      ).toBe(true);
    } else {
      await expect(transcript).toHaveCSS("font-family", /Noto Serif KR.*Batang.*Malgun Gothic/);
      await expect(transcript).toHaveCSS("word-break", "keep-all");
    }
  });
}

test("keeps the transcript readable through corrupt timing and a missing recording", async ({
  page,
  request,
}) => {
  const api = await publishedArticle(request, "zh-CN", chinese);
  await page.route(
    `**${api}`,
    async (route) => {
      const response = await route.fetch();
      const item = (await response.json()) as Item;
      // Split the non-BMP Han character at its high surrogate.
      const first = item.transcript!.segments[0];
      first.textEnd = chinese.indexOf("𠀀") + 1;
      first.text = chinese.slice(first.textStart, first.textEnd);
      await route.fulfill({ response, json: item });
    },
    { times: 1 },
  );
  await page.route(`**${api}/audio?*`, (route) => route.fulfill({ status: 404 }), { times: 1 });
  await page.goto(api.replace("/api/v1", ""));
  const transcript = page.getByRole("region", { name: "Transcript" });
  const player = page.getByRole("region", { name: "Article audio" });
  const audio = player.locator("audio");
  const retry = player.getByRole("button", { name: "Retry", exact: true });
  await expect(retry).toBeVisible();
  expect(await transcript.textContent()).toBe(chinese);
  await expect(transcript.locator("[data-segment-index]")).toHaveCount(0);
  await expect(audio).toHaveCount(0);
  const failure = page.waitForResponse(
    (response) => response.url().includes(`${api}/audio?`) && response.status() === 404,
  );
  await retry.click();
  await failure;
  await expect(retry).toBeVisible();
  await expect(audio).toHaveCount(0);
  expect(await transcript.textContent()).toBe(chinese);
  await retry.click();
  await expect
    .poll(() => audio.evaluate((element: HTMLAudioElement) => element.duration))
    .toBeGreaterThan(0);
  expect(await transcript.textContent()).toBe(chinese);
});

test("offers editing from a draft URL and reports unknown item IDs", async ({ page, request }) => {
  const created = await request.post("/api/v1/items", { data: { languageCode: "ko-KR" } });
  expect(created.status()).toBe(201);
  const { id } = (await created.json()) as Item;
  await page.goto(`/items/${id}`);
  await expect(page.getByRole("link", { name: "Continue Editing" })).toHaveAttribute(
    "href",
    `/create?draftId=${id}`,
  );
  await expect(page.getByRole("region", { name: "Transcript" })).toHaveCount(0);
  await page.goto(`/items/${crypto.randomUUID()}`);
  await expect(page.getByRole("heading", { name: "Page not found" })).toBeVisible();
});
