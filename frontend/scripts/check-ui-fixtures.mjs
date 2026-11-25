import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const checker = fileURLToPath(new URL("./check-ui.mjs", import.meta.url));
const cases = [
  ["native Solid", { dependencies: { "solid-js": "2.0.0" } }, "", 0],
  ["headless UI dependency", { dependencies: { "@kobalte/core": "1.0.0" } }, "", 1],
  ["npm alias", { devDependencies: { shortcuts: "npm:@radix-ui/react-dialog@1.0.0" } }, "", 1],
  ["unapproved runtime package", { dependencies: { "some-ui": "1.0.0" } }, "", 1],
  ["unapproved development package", { devDependencies: { "some-ui": "1.0.0" } }, "", 1],
  ["dynamic import", {}, 'const dialog = import(\n"@kobalte/core/dialog");', 1],
  ["Tailwind CSS import", {}, '@import "tailwindcss";', 1],
];

for (const [name, manifest, source, status] of cases) {
  test(name, (t) => {
    const dir = mkdtempSync(join(tmpdir(), "taotale-ui-"));
    t.after(() => rmSync(dir, { recursive: true, force: true }));
    writeFileSync(join(dir, "package.json"), JSON.stringify(manifest));
    writeFileSync(join(dir, name.endsWith("CSS import") ? "app.css" : "app.tsx"), source);
    const result = spawnSync(process.execPath, [checker, dir], { encoding: "utf8" });
    assert.equal(result.status, status, result.stderr);
    if (status) assert.match(result.stderr, /prohibited UI|requires review/);
  });
}
