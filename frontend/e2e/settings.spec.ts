import { expect, test } from "@playwright/test";

test("persists settings, applies article defaults, and copies local data commands", async ({
  page,
  context,
}) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  await page.goto("/settings");
  const status = page.getByRole("region", { name: "System status" });
  await expect(status.getByText("Writable", { exact: true })).toBeVisible();
  await expect(status.getByText("Mock provider ready")).toBeVisible();
  await expect(status.getByText(/ko-kr-phrase/)).toBeVisible();
  const korean = page.getByRole("group", { name: "Korean" });
  const mandarin = page.getByRole("group", { name: "Mandarin Chinese" });
  await korean.getByLabel("Default speaking rate").selectOption("1.25");
  await mandarin.getByLabel("Default speaking rate").selectOption("0.75");
  await page.getByLabel("Active language").selectOption("ko-KR");
  await page.getByLabel("Default Reader speed").selectOption("1.5");
  await page.getByLabel("Reader vocabulary display").selectOption("acquired");
  await page.getByLabel("Open vocabulary details by default in the Reader").check();
  await page.reload();
  await expect(page.getByLabel("Active language")).toHaveValue("ko-KR");
  await expect(korean.getByLabel("Default speaking rate")).toHaveValue("1.25");
  await expect(mandarin.getByLabel("Default speaking rate")).toHaveValue("0.75");
  await expect(page.getByLabel("Default Reader speed")).toHaveValue("1.5");
  await expect(page.getByLabel("Open vocabulary details by default in the Reader")).toBeChecked();
  await page.getByText("Back up the local installation", { exact: true }).click();
  await page.getByRole("button", { name: "Copy backup commands" }).click();
  await expect(page.getByText("Commands copied.")).toBeVisible();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toContain("pg_dump");
  await page.getByText("Restore a backup into a fresh installation", { exact: true }).click();
  await page.getByRole("button", { name: "Copy restore commands" }).click();
  await expect
    .poll(() => page.evaluate(() => navigator.clipboard.readText()))
    .toContain("pg_restore");

  await page.goto("/create");
  await expect(page.getByLabel("Item language")).toHaveValue("ko-KR");
  await expect(page.getByLabel("Speaking rate")).toHaveValue("1.25");
  await page.getByLabel("Title", { exact: true }).fill(`Settings fixture ${crypto.randomUUID()}`);
  await page.getByLabel("Source text", { exact: true }).fill("안녕하세요. 설정을 확인합니다.");
  await page.getByRole("button", { name: "Generate Preview" }).click();
  await page.getByRole("button", { name: "Publish to Library" }).click();
  await expect(page).toHaveURL(/\/items\/[0-9a-f-]+$/);
  await expect(page.getByLabel("Playback speed")).toHaveValue("1.5");
  await expect(page.getByLabel("Show", { exact: true })).toBeVisible();
  await expect(page.getByLabel("Show", { exact: true })).toHaveValue("acquired");
});
