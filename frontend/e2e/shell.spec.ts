import { expect, test } from "@playwright/test";

test("serves the Library, native controls and refreshed Reader from Gin", async ({
  page,
  request,
}) => {
  const unknownApi = await request.get("/api/v1/missing");
  expect(unknownApi.status()).toBe(404);
  expect((await unknownApi.json()).error.code).toBe("NOT_FOUND");
  await page.goto("/");
  await expect(page).toHaveURL(/\/library$/);
  await expect(page.getByRole("heading", { name: "Your reading library" })).toBeVisible();
  const favorite = page.getByRole("button", { name: "Favorite sample" });
  await expect(favorite).toHaveCSS("border-radius", "11px");
  await favorite.click();
  await expect(favorite).toHaveAttribute("aria-pressed", "true");
  const language = page.getByLabel("Reading language");
  await expect(language).toBeEnabled();
  await language.selectOption("ko-KR");
  await page.getByRole("link", { name: "Open reader" }).click();
  await expect(page).toHaveURL(/\/items\/hangzhou-morning-ko$/);
  await page.reload();
  await expect(page.getByRole("heading", { name: "A quiet morning in Hangzhou" })).toBeVisible();
  await expect(language).toHaveValue("ko-KR");
  await expect(page.locator("main [lang='ko-KR']")).toContainText("재스민");
  await expect
    .poll(() => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth))
    .toBe(true);
});

test("persists appearance and initializes the theme before app scripts run", async ({ page }) => {
  await page.goto("/settings");
  const theme = page.getByRole("combobox", { name: "Appearance" });
  const root = page.locator("html");
  await expect(theme).toHaveValue("system");
  await theme.selectOption("dark");
  await expect(root).toHaveAttribute("data-theme", "dark");
  await page.getByRole("link", { name: "Library" }).click();
  await expect(page.getByRole("heading", { name: "Your reading library" })).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  await page.route(/\/assets\/.*\.js$/, (route) => route.abort());
  await page.reload({ waitUntil: "domcontentloaded" });
  await expect(root).toHaveAttribute("data-theme", "dark");
});

test("navigates to route shells and shows a client not-found", async ({ page }) => {
  await page.goto("/library");
  await page.getByRole("link", { name: "New article" }).click();
  await expect(page.getByRole("heading", { name: "New article" })).toBeVisible();
  await page.getByRole("link", { name: "Vocabulary" }).click();
  await expect(page.getByRole("heading", { name: "Vocabulary" })).toBeVisible();
  await page.goto("/items/hangzhou-morning/edit");
  await expect(page.getByRole("heading", { name: "Edit article" })).toBeVisible();
  await page.goto("/generate");
  await expect(page.getByRole("heading", { name: "Page not found" })).toBeVisible();
});
