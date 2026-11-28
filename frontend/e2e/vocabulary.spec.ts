import { expect, test, type APIRequestContext, type Locator } from "@playwright/test";
import type { Item, Topic, VocabularyDetail, VocabularyStats } from "../src/api/client";

const vocabulary = "/api/v1/vocabulary";

async function publish(
  request: APIRequestContext,
  languageCode: Item["languageCode"],
  sourceText: string,
  title: string,
) {
  const draft = await request.post("/api/v1/items", { data: { languageCode, sourceText, title } });
  expect(draft.status()).toBe(201);
  const { id, version } = (await draft.json()) as Item;
  const path = `/api/v1/items/${id}`;
  const job = await request.post(`${path}/audio/generate`, {
    headers: { "Idempotency-Key": crypto.randomUUID() },
    data: { expectedVersion: version },
  });
  expect(job.status()).toBe(202);
  const { generationId } = await job.json();
  await expect
    .poll(
      async () =>
        (await (await request.get(`${path}/audio/generations/${generationId}`)).json()).status,
      { timeout: 15_000 },
    )
    .toBe("READY");
  expect(
    (await request.post(`${path}/publish`, { data: { expectedVersion: version } })).status(),
  ).toBe(200);
  return id;
}

async function selectText(transcript: Locator, text: string) {
  await transcript.evaluate((root, text) => {
    const start = root.textContent!.indexOf(text);
    if (start < 0) throw new Error("Selected text not present in the transcript");
    const end = start + text.length;
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
    const range = document.createRange();
    let offset = 0;
    let hasStart = false;
    for (let node = walker.nextNode(); node; node = walker.nextNode()) {
      const length = node.textContent!.length;
      if (!hasStart && start < offset + length) {
        range.setStart(node, start - offset);
        hasStart = true;
      }
      if (end <= offset + length) {
        range.setEnd(node, end - offset);
        break;
      }
      offset += length;
    }
    const selection = window.getSelection()!;
    selection.removeAllRanges();
    // Backwards native selections have the same ordered Range boundaries.
    selection.setBaseAndExtent(
      range.endContainer,
      range.endOffset,
      range.startContainer,
      range.startOffset,
    );
  }, text);
}

for (const languageCode of ["zh-CN", "ko-KR"] as const) {
  test(`collects ${languageCode} selections and syncs manual vocabulary status without remounting audio`, async ({
    page,
    request,
    context,
  }) => {
    test.setTimeout(60_000);
    const token = crypto.randomUUID().slice(0, 6);
    const isMandarin = languageCode === "zh-CN";
    const other = isMandarin ? "ko-KR" : "zh-CN";
    const word = isMandarin ? `𠀀${token}、新词` : `한글𠀀${token}？\n\n  다음 문장`;
    const source = isMandarin ? `前言。${word}。 后记。` : `앞말. ${word}. 또 봐요.`;
    const title = `Collect ${token}`;
    expect(
      (
        await request.post(vocabulary, {
          data: { languageCode: other, word, meaning: "Same spelling in the other language" },
        })
      ).status(),
    ).toBe(201);
    const id = await publish(request, languageCode, source, title);
    const route = `/items/${id}`;
    await page.addInitScript((language) => {
      localStorage.setItem("taotale.preferences.language", language);
    }, other);
    await page.goto(route);
    const transcript = page.getByRole("region", { name: "Transcript" });
    await expect(transcript).toHaveText(source, { useInnerText: false });
    expect(await transcript.textContent()).toBe(source);
    const choose = page.getByRole("button", { name: "Add to Vocabulary" });
    await expect(choose).toBeDisabled();
    await expect(choose).toHaveCSS("opacity", "0.55");
    if (isMandarin) {
      await selectText(transcript, `前言。${word}`);
      await expect(page.getByText(/sentence punctuation/)).toBeVisible();
      await expect(choose).toBeDisabled();
    } else {
      expect(await transcript.locator("[data-segment-index]").count()).toBeGreaterThan(2);
    }
    await selectText(transcript, word);
    await expect(choose).toBeEnabled();
    await choose.click();
    const add = page.getByRole("dialog", { name: "Add to Vocabulary" });
    await expect(add.getByLabel("Word or phrase")).toHaveValue(word);
    await add.getByLabel("Meaning").fill("Cancelled meaning");
    await add.getByRole("button", { name: "Cancel" }).click();
    await expect(add).toHaveCount(0);
    const lookupURL = `${vocabulary}/lookup?${new URLSearchParams({ languageCode, word })}`;
    expect(await (await request.get(lookupURL)).json()).toEqual({ entry: null });
    expect(await (await request.get(`/api/v1${route}/vocabulary`)).json()).toMatchObject({
      items: [],
    });

    await selectText(transcript, word);
    await choose.click();
    await add.getByLabel("Reading", { exact: true }).fill("Manual reading");
    await add.getByLabel("Reading system").fill(isMandarin ? "pinyin" : "romanization");
    await add.getByLabel("Meaning").fill(`Meaning ${token}`);
    await add.getByRole("checkbox", { name: "Culture" }).check();
    await add.getByRole("button", { name: "Save Word" }).click();
    await expect(add).toHaveCount(0);
    const { entry } = (await (await request.get(lookupURL)).json()) as { entry: VocabularyDetail };
    expect(entry).toMatchObject({
      word,
      languageCode,
      meaning: `Meaning ${token}`,
      status: "LEARNING",
    });
    expect(entry.topicIds).toHaveLength(1);

    const otherId = await publish(request, languageCode, source, `Also ${title}`);
    const second = await context.newPage();
    if (!isMandarin) {
      await second.route(`**/api/v1/items/${otherId}`, async (route) => {
        const response = await route.fetch();
        const item = (await response.json()) as Item;
        delete item.transcript;
        await route.fulfill({ response, json: item });
      });
    }
    await second.goto(`/items/${otherId}`);
    const secondTranscript = second.getByRole("region", { name: "Transcript" });
    await expect(secondTranscript).toBeVisible();
    if (!isMandarin) {
      await expect(second.getByText("Audio unavailable. You can keep reading.")).toBeVisible();
    }
    await selectText(secondTranscript, word);
    await second.getByRole("button", { name: "Add to Vocabulary" }).click();
    const existing = second.getByRole("dialog", { name: "Add to Vocabulary" });
    await expect(existing.getByText("Already in Vocabulary")).toBeVisible();
    await expect(existing.getByText(`Meaning ${token}`)).toBeVisible();
    await existing.getByRole("button", { name: "Link to Article" }).click();
    await expect(existing).toHaveCount(0);
    const detail = (await (
      await request.get(`${vocabulary}/${entry.id}`)
    ).json()) as VocabularyDetail;
    expect(detail.meaning).toBe(`Meaning ${token}`);
    expect(detail.references.map(({ itemId }) => itemId).sort()).toEqual([id, otherId].sort());
    await second.goto(`/vocabulary?${new URLSearchParams({ languageCode, q: token })}`);
    const row = second.getByRole("row").filter({ hasText: `Meaning ${token}` });
    await expect(row.getByText("Learning", { exact: true })).toBeVisible();
    await row.getByRole("button").click();
    const editor = second.getByRole("region", { name: "Word details" });
    await expect(editor.getByRole("link", { name: title, exact: true })).toHaveAttribute(
      "href",
      route,
    );

    const audio = page.locator("audio");
    await expect(
      page
        .getByRole("region", { name: "Article audio" })
        .getByRole("button", { name: "Play", exact: true }),
    ).toBeEnabled();
    const pinned = (await audio.elementHandle())!;
    const panel = page
      .locator("details")
      .filter({ has: page.getByText("Article vocabulary", { exact: true }) });
    await page.getByText("Article vocabulary", { exact: true }).click();
    if (!isMandarin) {
      await expect(panel.getByRole("button", { name: "Sync vocabulary" })).toHaveCount(0);
      await expect(transcript.locator("[data-vocabulary-id]")).toHaveCount(0);
    }
    await expect(panel.getByText("Manual reading", { exact: false })).toBeVisible();
    await expect(panel.getByText("Culture", { exact: true })).toBeVisible();
    const acquire = panel.getByRole("button", { name: /as Acquired$/ });
    let release!: () => void;
    const pendingWrite = new Promise<void>((resolve) => {
      release = resolve;
    });
    await page.route(
      `**${vocabulary}/${entry.id}`,
      async (route) => {
        if (route.request().method() === "PATCH") {
          await pendingWrite;
          await route.fulfill({ status: 503 });
        } else await route.continue();
      },
      { times: 1 },
    );
    try {
      await acquire.click();
      await expect(acquire).toHaveAttribute("aria-busy", "true");
      await expect(acquire).toBeDisabled();
      await expect(
        panel.getByRole("listitem").getByText("Learning", { exact: true }),
      ).toBeVisible();
    } finally {
      release();
    }
    await expect(panel.getByRole("alert")).toContainText("Last confirmed status is Learning");
    await expect(row.getByText("Learning", { exact: true })).toBeVisible();
    await acquire.click();
    await expect(panel.getByRole("listitem").getByText("Acquired", { exact: true })).toBeVisible();
    await expect(row.getByText("Acquired", { exact: true })).toBeVisible();
    await expect(editor.getByRole("button", { name: "Mark as Learning" })).toBeEnabled();
    expect(await pinned.evaluate((element) => element === document.querySelector("audio"))).toBe(
      true,
    );

    await panel.getByLabel("Show").selectOption("acquired");
    await expect(panel.getByRole("listitem")).toHaveCount(1);
    await editor.getByRole("button", { name: "Mark as Learning" }).click();
    await expect(panel.getByText("No linked words match this display.")).toBeVisible();
    expect(await pinned.evaluate((element) => element === document.querySelector("audio"))).toBe(
      true,
    );
    await page.reload();
    await page.getByText("Article vocabulary", { exact: true }).click();
    await expect(panel.getByLabel("Show")).toHaveValue("acquired");
    await panel.getByLabel("Show").selectOption("both");
    await expect(panel.getByRole("button", { name: /as Acquired$/ })).toBeVisible();
    await second.close();
  });
}

test("browses vocabulary by language, edits metadata, and applies atomic bulk actions", async ({
  page,
  request,
}) => {
  test.setTimeout(45_000);
  const key = `Batch ${crypto.randomUUID().slice(0, 8)}`;
  const { items } = (await (await request.get("/api/v1/topics")).json()) as { items: Topic[] };
  const science = items.find(({ systemKey }) => systemKey === "science")!;
  for (let i = 1; i <= 12; i++) {
    const word = `${key} ${String(i).padStart(2, "0")}`;
    expect(
      (
        await request.post(vocabulary, {
          data: {
            languageCode: "zh-CN",
            word,
            reading: `Reading ${i}`,
            meaning: `Meaning ${i}`,
            topicIds: i % 2 === 0 ? [science.id] : [],
          },
        })
      ).status(),
    ).toBe(201);
  }
  expect(
    (
      await request.post(vocabulary, {
        data: { languageCode: "ko-KR", word: `${key} Korean` },
      })
    ).status(),
  ).toBe(201);
  const statistics = page.waitForResponse(`**${vocabulary}/stats?languageCode=zh-CN`);
  await page.goto(
    `/vocabulary?${new URLSearchParams({ languageCode: "zh-CN", q: key, page: "2" })}`,
  );
  const table = page.getByRole("table", { name: "Vocabulary entries" });
  await expect(table.getByRole("row")).toHaveCount(3);
  await expect(page.getByText("Page 2 of 2", { exact: true })).toBeVisible();
  await expect(page.getByText("Total words")).toBeVisible();
  const stats = (await (await statistics).json()) as VocabularyStats;
  const total = page.locator("dl").filter({ hasText: "Total words" }).locator("dd").first();
  await expect(total).toHaveText(String(stats.total));
  await page.getByLabel("Sort by").selectOption("word_asc");
  await expect(page).toHaveURL(/sort=word_asc&page=1$/);
  await expect(table.getByRole("row")).toHaveCount(11);
  const first = table.getByRole("button", { name: `${key} 01`, exact: true });
  await first.click();
  const editor = page.getByRole("region", { name: "Word details" });
  await expect(editor.getByLabel("Word or phrase")).toHaveValue(`${key} 01`);
  await editor.getByLabel("Word or phrase").fill(`${key} 01 revised`);
  await editor.getByLabel("Reading", { exact: true }).fill("Reading revised");
  await editor.getByLabel("Reading system").fill("pinyin");
  await editor.getByLabel("Meaning").fill("Meaning revised");
  await editor.getByLabel("Note").fill("Saved note");
  await editor.getByRole("checkbox", { name: "Science" }).check();
  await editor.getByRole("button", { name: "Save Word" }).click();
  const revised = table.getByRole("row").filter({ hasText: `${key} 01 revised` });
  await expect(revised.getByText("Meaning revised")).toBeVisible();
  await expect(revised.getByText("Science")).toBeVisible();

  const acquire = editor.getByRole("button", { name: "Mark as Acquired" });
  await page.route(
    /\/api\/v1\/vocabulary\/[0-9a-f-]+$/,
    async (route) => {
      if (route.request().method() === "PATCH") await route.fulfill({ status: 503 });
      else await route.continue();
    },
    { times: 1 },
  );
  await acquire.click();
  await expect(editor.getByRole("alert")).toContainText("Last confirmed status is Learning");
  await expect(revised.getByText("Learning", { exact: true })).toBeVisible();
  await expect(editor.getByLabel("Note")).toHaveValue("Saved note");
  await acquire.click();
  await expect(revised.getByText("Acquired", { exact: true })).toBeVisible();
  await editor.getByRole("button", { name: "Close details" }).click();

  await page.getByText("Filter by Topics", { exact: true }).click();
  await page.getByRole("checkbox", { name: "Science" }).check();
  await expect(page).toHaveURL(new RegExp(`topics=${science.id}&page=1$`));
  await expect(table.getByRole("row")).toHaveCount(8);
  await page.getByRole("checkbox", { name: "Select all words on this page" }).check();
  await expect(page.getByText("7 selected")).toBeVisible();
  await page.route(`**${vocabulary}/bulk`, (route) => route.fulfill({ status: 503 }), { times: 1 });
  await page.getByRole("button", { name: "Mark selected Acquired" }).click();
  await expect(page.getByRole("alert")).toContainText("last confirmed status");
  await expect(page.getByText("7 selected")).toBeVisible();
  await expect(table.getByText("Acquired", { exact: true })).toHaveCount(1);
  await page.getByRole("button", { name: "Mark selected Acquired" }).click();
  await expect(page.getByText("7 words marked Acquired.")).toBeVisible();
  await expect(table.getByText("Acquired", { exact: true })).toHaveCount(7);
  await page.getByLabel("Status", { exact: true }).selectOption("LEARNING");
  await expect(page.getByText("No words match your filters.")).toBeVisible();
  await page.getByLabel("Status", { exact: true }).selectOption("ACQUIRED");
  await page.getByRole("checkbox", { name: "Select all words on this page" }).check();
  page.once("dialog", (dialog) => dialog.dismiss());
  await page.getByRole("button", { name: "Delete selected" }).click();
  await expect(table.getByRole("row")).toHaveCount(8);
  page.once("dialog", (dialog) => dialog.accept());
  await page.getByRole("button", { name: "Delete selected" }).click();
  await expect(page.getByText("7 words deleted.")).toBeVisible();
  await expect(page.getByText("No words match your filters.")).toBeVisible();

  await page.getByLabel("Reading language").selectOption("ko-KR");
  await expect(page).toHaveURL(/languageCode=ko-KR&page=1$/);
  await page.getByLabel("Search vocabulary").fill(key);
  await expect(page).toHaveURL((url) => url.searchParams.get("q") === key);
  await expect(table.getByRole("row")).toHaveCount(2);
  await expect(table.getByText(`${key} Korean`, { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Add word", exact: true }).click();
  const add = page.getByRole("region", { name: "Add word" });
  await add.getByLabel("Word or phrase").fill(`${key} 직접 추가`);
  await add.getByRole("button", { name: "Save Word" }).click();
  await expect(table.getByRole("button", { name: `${key} 직접 추가` })).toBeVisible();
  await expect(editor.getByLabel("Word or phrase")).toHaveValue(`${key} 직접 추가`);
  page.once("dialog", (dialog) => dialog.accept());
  await editor.getByRole("button", { name: "Delete Word" }).click();
  await expect(table.getByRole("button", { name: `${key} 직접 추가` })).toHaveCount(0);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
});

test("highlights and syncs exact Mandarin matches while keeping native selections and seek targets", async ({
  page,
  request,
}) => {
  test.setTimeout(45_000);
  const token = crypto.randomUUID().slice(0, 6);
  const shorter = `𠀀${token}学习`;
  const compound = `${shorter}。 \n  语言`;
  const later = `复习${token}`;
  const source = `${compound}。 ${later}。`;
  const add = async (word: string, status: "LEARNING" | "ACQUIRED") => {
    const result = await request.post(vocabulary, {
      data: { languageCode: "zh-CN", word, status },
    });
    expect(result.status()).toBe(201);
    return ((await result.json()) as VocabularyDetail).id;
  };
  const shorterID = await add(shorter, "LEARNING");
  const compoundID = await add(compound, "ACQUIRED");
  const id = await publish(request, "zh-CN", source, `Automatic vocabulary ${token}`);
  await page.goto(`/items/${id}`);
  const transcript = page.getByRole("region", { name: "Transcript" });
  const annotation = (wordID: string) => transcript.locator(`[data-vocabulary-id="${wordID}"]`);
  await expect(annotation(compoundID)).toHaveCount(3);
  await expect(annotation(shorterID)).toHaveCount(0);
  expect((await annotation(compoundID).allTextContents()).join("")).toBe(compound);
  expect(await transcript.textContent()).toBe(source);
  await expect(annotation(compoundID).first()).toHaveCSS("text-decoration-style", "solid");
  await expect(page.getByLabel("Vocabulary highlight legend")).toBeVisible();

  const player = page.getByRole("region", { name: "Article audio" });
  await expect(player.getByRole("button", { name: "Play", exact: true })).toBeEnabled();
  const audio = (await player.locator("audio").elementHandle())!;
  const secondSegment = transcript.locator('[data-segment-index="1"]');
  const timedNode = (await secondSegment.elementHandle())!;
  const lateID = await add(later, "LEARNING");
  await page.getByText("Article vocabulary", { exact: true }).click();
  const panel = page
    .locator("details")
    .filter({ has: page.getByText("Article vocabulary", { exact: true }) });
  await expect(panel.getByText("Matched in article")).toHaveCount(2);

  let release!: () => void;
  const gate = new Promise<void>((resolve) => {
    release = resolve;
  });
  await page.route(
    `**/api/v1/items/${id}/vocabulary/sync`,
    async (route) => {
      const response = await route.fetch();
      await gate;
      await route.fulfill({ response });
    },
    { times: 1 },
  );
  try {
    await panel.getByRole("button", { name: "Sync vocabulary" }).click();
    await expect(panel.getByRole("button", { name: "Syncing…" })).toBeDisabled();
    await selectText(transcript, shorter);
    await expect(page.getByRole("button", { name: "Add to Vocabulary" })).toBeEnabled();
  } finally {
    release();
  }
  await expect(page.getByText("3 vocabulary matches in this article.")).toBeVisible();
  await expect(panel.getByText(later, { exact: true })).toBeVisible();
  expect(await page.evaluate(() => window.getSelection()?.toString())).toBe(shorter);
  await expect(annotation(lateID)).toHaveCount(0);
  await page.evaluate(() => window.getSelection()?.removeAllRanges());
  await expect(annotation(lateID)).toHaveCSS("text-decoration-style", "dotted");
  expect(
    await timedNode.evaluate(
      (element) => element === document.querySelector('[data-segment-index="1"]'),
    ),
  ).toBe(true);
  await secondSegment.press("Enter");
  await expect(secondSegment).toHaveAttribute("data-active", "true");
  await expect(annotation(compoundID).last()).toHaveCSS("text-decoration-style", "solid");
  const compoundRow = panel.getByRole("listitem").filter({ hasText: compound });
  await compoundRow.getByRole("button", { name: /as Learning$/ }).click();
  await expect(annotation(compoundID).first()).toHaveCSS("text-decoration-style", "dotted");
  await panel.getByLabel("Show").selectOption("acquired");
  await expect(annotation(compoundID).first()).toHaveCSS("text-decoration-line", "none");
  expect(await transcript.textContent()).toBe(source);
  expect(await audio.evaluate((element) => element === document.querySelector("audio"))).toBe(true);

  // With bad alignment, the same ranges decorate plain source text without seek targets.
  await page.route(
    `**/api/v1/items/${id}`,
    async (route) => {
      const response = await route.fetch();
      const item = (await response.json()) as Item;
      delete item.transcript;
      await route.fulfill({ response, json: item });
    },
    { times: 1 },
  );
  await page.reload();
  await expect(player.getByText("Audio unavailable. You can keep reading.")).toBeVisible();
  await page.getByText("Article vocabulary", { exact: true }).click();
  await panel.getByLabel("Show").selectOption("both");
  await expect(annotation(compoundID)).toHaveCount(1);
  await expect(annotation(compoundID)).toHaveCSS("text-decoration-style", "dotted");
  expect(await transcript.textContent()).toBe(source);
});
