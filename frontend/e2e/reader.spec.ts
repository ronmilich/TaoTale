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

  test(`synchronizes ${language} highlights and seeks within 150 ms at every playback speed`, async ({
    page,
    request,
  }, testInfo) => {
    const api = await publishedArticle(request, language, source);
    const item = (await (await request.get(api)).json()) as Item;
    const segments = item.transcript!.segments;
    await page.goto(api.replace("/api/v1", ""));
    const player = page.getByRole("region", { name: "Article audio" });
    await expect(player.getByRole("button", { name: "Play", exact: true })).toBeEnabled();
    const measurements = [];
    for (const rate of [0.75, 1, 1.25, 1.5]) {
      await player.getByLabel("Playback speed").selectOption(String(rate));
      const report = await page.evaluate(
        async ({ segments, rate }) => {
          const root = document.querySelector<HTMLElement>('[aria-label="Transcript"]')!;
          const audio = document.querySelector("audio")!;
          const controls = document.querySelector('[aria-label="Article audio"]')!;
          const play = controls.querySelector("button")!;
          const parts = [...root.querySelectorAll<HTMLElement>("[data-segment-index]")];
          const original = parts.map((part) => part.firstChild);
          let structuralWrites = 0;
          const observer = new MutationObserver((records) => {
            structuralWrites += records.length;
          });
          observer.observe(root, { subtree: true, childList: true, characterData: true });
          const frame = () =>
            new Promise<void>((resolve) => requestAnimationFrame(() => resolve()));
          const active = () => parts.findIndex((part) => part.dataset.active === "true");
          const latencies = [];
          // Alternating destinations tests both directions while paused.
          for (const index of [1, 0]) {
            const start = performance.now();
            parts[index].dispatchEvent(new MouseEvent("click", { bubbles: true, detail: 1 }));
            while (active() !== index && performance.now() - start < 500) await frame();
            if (
              active() !== index ||
              Math.abs(audio.currentTime * 1000 - segments[index].startMs) > 25 ||
              !audio.paused
            )
              throw new Error("Paused seek did not settle at its stored start");
            latencies.push(performance.now() - start);
          }
          play.click();
          let samples = 0;
          let previousExpected = -2;
          let changedAt = performance.now();
          let worst = 0;
          const seen = new Set<number>();
          const deadline = performance.now() + 5000;
          while (!audio.ended && performance.now() < deadline) {
            await frame();
            const ms = audio.currentTime * 1000;
            const index = segments.findIndex(
              (segment) => segment.startMs <= ms && ms < segment.endMs,
            );
            const now = performance.now();
            if (index !== previousExpected) {
              changedAt = now;
              previousExpected = index;
            }
            if (active() === index) seen.add(index);
            else worst = Math.max(worst, now - changedAt);
            samples++;
          }
          await frame();
          observer.disconnect();
          return {
            rate,
            nativeRate: audio.playbackRate,
            latencies,
            worst,
            samples,
            seen: [...seen],
            ended: audio.ended,
            finalActive: active(),
            structuralWrites,
            stable: parts.every(
              (part, i) => root.contains(part) && part.firstChild === original[i],
            ),
          };
        },
        { segments, rate },
      );
      measurements.push(report);
      expect(report.nativeRate).toBe(rate);
      expect(Math.max(...report.latencies, report.worst)).toBeLessThanOrEqual(150);
      expect(report.samples).toBeGreaterThan(3);
      for (const { index } of segments) expect(report.seen).toContain(index);
      expect(report.ended).toBe(true);
      expect(report.finalActive).toBe(-1);
      expect(report.structuralWrites).toBe(0);
      expect(report.stable).toBe(true);
    }
    await testInfo.attach("Native clock measurements", {
      body: JSON.stringify(measurements),
      contentType: "application/json",
    });
  });
}

test("preserves selections and play state, resumes per recording and restarts after ending", async ({
  page,
  request,
}) => {
  const api = await publishedArticle(request, "ko-KR", korean);
  const item = (await (await request.get(api)).json()) as Item;
  const key = `taotale.playback.v1.${item.id}.${item.audioAssetId}`;
  const route = api.replace("/api/v1", "");
  await page.goto(route);
  const player = page.getByRole("region", { name: "Article audio" });
  const audio = player.locator("audio");
  const play = player.getByRole("button", { name: "Play", exact: true });
  const pause = player.getByRole("button", { name: "Pause", exact: true });
  const parts = page.locator("[data-segment-index]");
  await expect(play).toBeEnabled();
  const second = item.transcript!.segments[1];
  await parts.nth(1).focus();
  await page.keyboard.press("Enter");
  await expect(parts.nth(1)).toHaveAttribute("data-active", "true");
  expect(await audio.evaluate((el: HTMLAudioElement) => el.paused)).toBe(true);
  await expect
    .poll(() => page.evaluate((key) => Number(localStorage.getItem(key)), key))
    .toBe(second.startMs);
  await page.reload();
  await expect(play).toBeEnabled();
  await expect(parts.nth(1)).toHaveAttribute("data-active", "true");
  const readTime = () => audio.evaluate((el: HTMLAudioElement) => el.currentTime);
  expect(await readTime()).toBeCloseTo(second.startMs / 1000, 2);
  const firstBounds = (await parts.first().boundingBox())!;
  const y = firstBounds.y + firstBounds.height / 2;
  await page.mouse.move(firstBounds.x + 1, y);
  await page.mouse.down();
  await page.mouse.move(firstBounds.x + firstBounds.width - 1, y, { steps: 8 });
  await page.mouse.up();
  expect(await page.evaluate(() => window.getSelection()?.toString())).toBeTruthy();
  expect(await readTime()).toBeCloseTo(second.startMs / 1000, 2);
  await page.evaluate(() => window.getSelection()?.removeAllRanges());
  await parts.first().dblclick();
  expect(await page.evaluate(() => window.getSelection()?.toString())).toBeTruthy();
  // Wait past the single-click reaction budget to catch an unintended delayed seek.
  await audio.evaluate(() => new Promise((resolve) => setTimeout(resolve, 150)));
  expect(await readTime()).toBeCloseTo(second.startMs / 1000, 2);
  await page.evaluate(() => window.getSelection()?.removeAllRanges());
  await player.getByRole("button", { name: "Back 5 seconds" }).click();
  expect(await readTime()).toBe(0);
  await player.getByRole("button", { name: "Forward 5 seconds" }).click();
  expect(await readTime()).toBeCloseTo(item.transcript!.durationMs / 1000, 1);
  await player.getByRole("slider").focus();
  await page.keyboard.press("Home");
  expect(await readTime()).toBe(0);
  await page.keyboard.press("End");
  expect(await readTime()).toBeCloseTo(item.transcript!.durationMs / 1000, 3);
  await page.keyboard.press("Home");
  await play.click();
  await parts.nth(1).click();
  await expect
    .poll(() =>
      audio.evaluate(
        (el: HTMLAudioElement, startMs) => !el.paused && el.currentTime * 1000 >= startMs - 1,
        second.startMs,
      ),
    )
    .toBe(true);
  await expect(play).toBeEnabled();
  await expect.poll(() => audio.evaluate((el: HTMLAudioElement) => el.ended)).toBe(true);
  expect(await page.evaluate((key) => localStorage.getItem(key), key)).toBeNull();
  await play.click();
  await expect(pause).toBeVisible();
  expect(await readTime()).toBeLessThan(second.startMs / 1000);
  await pause.click();
  const link = page.getByRole("link", { name: "Download Audio" });
  expect(await link.getAttribute("href")).toBe(
    `${api}/audio/download?audioAssetId=${item.audioAssetId}`,
  );
  const oldAudio = (await audio.elementHandle())!;
  await play.click();
  await page.getByRole("link", { name: "Back to Library" }).click();
  await expect(page).toHaveURL(/\/library/);
  expect(
    await oldAudio.evaluate(
      (el) => (el as HTMLAudioElement).paused && !el.isConnected && !el.hasAttribute("src"),
    ),
  ).toBe(true);
  // A position for another asset in the same item must never be restored.
  await page.evaluate(
    ({ key, id }) => {
      localStorage.removeItem(key);
      localStorage.setItem(`taotale.playback.v1.${id}.previous`, "450");
    },
    { key, id: item.id },
  );
  await page.goto(route);
  await expect(play).toBeEnabled();
  expect(await readTime()).toBe(0);
});

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
  await expect(page.getByRole("link", { name: "Download Audio" })).toHaveCount(0);
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

test("keeps the recording and transcript mounted while changing favorite, notes and theme", async ({
  page,
  request,
}) => {
  const api = await publishedArticle(request, "ko-KR", "차 한 잔을 마셔요. ".repeat(95));
  const initial = (await (await request.get(api)).json()) as Item;
  await page.goto(api.replace("/api/v1", ""));
  const favorite = page.getByRole("button", { name: "Favorite", exact: true });
  const audio = page.locator("audio");
  const transcript = page.getByRole("region", { name: "Transcript" });
  const play = page.getByRole("button", { name: "Play", exact: true });
  await expect(play).toBeEnabled();
  const originalAudio = (await audio.elementHandle())!;
  const originalText = (await transcript.locator("[data-segment-index]").first().elementHandle())!;
  await page.route(`**${api}/favorite`, (route) => route.fulfill({ status: 503 }), { times: 1 });
  await favorite.click();
  await expect(page.getByRole("alert")).toContainText("Could not update favorite");
  await expect(favorite).toHaveAttribute("aria-pressed", "false");
  await play.click();
  await favorite.click();
  const unfavorite = page.getByRole("button", { name: "Remove from Favorites" });
  await expect(unfavorite).toHaveAttribute("aria-pressed", "true");
  const favorited = (await (await request.get(api)).json()) as Item;
  expect(favorited.version).toBe(initial.version);
  expect(favorited.updatedAt).toBe(initial.updatedAt);
  const before = await audio.evaluate((el: HTMLAudioElement) => el.currentTime);
  await page.getByRole("button", { name: "Notes", exact: true }).click();
  const notes = page.getByRole("region", { name: "Article notes" });
  await notes.getByRole("button", { name: "Edit Notes" }).click();
  await notes.getByRole("textbox").fill("<em>plain text</em>\n둘째 줄");
  await page.getByRole("link", { name: "Back to Library" }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Stay" }).click();
  await page.emulateMedia({
    colorScheme: test.info().project.name === "tablet" ? "light" : "dark",
  });
  await expect(page.locator("html")).toHaveAttribute(
    "data-theme",
    test.info().project.name === "tablet" ? "light" : "dark",
  );
  await notes.getByRole("button", { name: "Save Notes", exact: true }).click();
  await expect(notes.getByRole("status")).toHaveText("Notes saved.");
  await expect(notes.getByText("<em>plain text</em>")).toBeVisible();
  expect(await originalAudio.evaluate((el: HTMLAudioElement) => el.isConnected && !el.paused)).toBe(
    true,
  );
  expect(await originalText.evaluate((el) => el.isConnected)).toBe(true);
  expect(await audio.evaluate((el: HTMLAudioElement) => el.currentTime)).toBeGreaterThan(before);
  const saved = (await (await request.get(api)).json()) as Item;
  expect(saved).toMatchObject({
    notes: "<em>plain text</em>\n둘째 줄",
    isFavorite: true,
    audioAssetId: initial.audioAssetId,
    version: initial.version + 1,
  });
  await page.reload();
  await expect(unfavorite).toHaveAttribute("aria-pressed", "true");
  await unfavorite.click();
  await expect(favorite).toHaveAttribute("aria-pressed", "false");
});

test("guards unsaved notes and leaves only after the save is confirmed", async ({
  page,
  request,
}) => {
  const api = await publishedArticle(request, "zh-CN", chinese);
  await page.goto(api.replace("/api/v1", ""));
  await page.getByRole("button", { name: "Notes", exact: true }).click();
  const notes = page.getByRole("region", { name: "Article notes" });
  const edit = notes.getByRole("button", { name: "Edit Notes" });
  await edit.click();
  const textbox = notes.getByRole("textbox");
  await textbox.fill("cancelled");
  await notes.getByRole("button", { name: "Cancel" }).click();
  await expect(notes.getByText("No notes yet.")).toBeVisible();
  await edit.click();
  const published = (await (await request.get(api)).json()) as Item;
  const elsewhere = await request.patch(api, {
    data: { expectedVersion: published.version, notes: "Changed elsewhere." },
  });
  expect(elsewhere.ok()).toBe(true);
  await textbox.fill("Stale note.");
  await notes.getByRole("button", { name: "Save Notes", exact: true }).click();
  await expect(notes.getByRole("alert")).toContainText("This article changed elsewhere");
  await expect(textbox).toHaveValue("Stale note.");
  await notes.getByRole("button", { name: "Discard Edits and Load Latest" }).click();
  await expect(notes.getByText("Changed elsewhere.")).toBeVisible();
  await edit.click();
  await textbox.fill("Keep this note.");
  const back = page.getByRole("link", { name: "Back to Library" });
  await back.click();
  const dialog = page.getByRole("dialog", { name: "Leave with unsaved notes?" });
  await dialog.getByRole("button", { name: "Stay" }).click();
  await expect(textbox).toHaveValue("Keep this note.");
  await page.route(`**${api}`, (route) => route.fulfill({ status: 503 }), { times: 1 });
  await back.click();
  const saveAndLeave = dialog.getByRole("button", { name: "Save Notes & Leave" });
  await saveAndLeave.click();
  await expect(notes.getByRole("alert")).toContainText("Could not save notes");
  await expect(textbox).toHaveValue("Keep this note.");
  await back.click();
  await saveAndLeave.click();
  await expect(page).toHaveURL(/\/library/);
  expect(((await (await request.get(api)).json()) as Item).notes).toBe("Keep this note.");
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
