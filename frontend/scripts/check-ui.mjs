import { readdirSync, readFileSync } from "node:fs";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

const paths = process.argv[2] ?? fileURLToPath(new URL("../", import.meta.url));
const manifest = JSON.parse(readFileSync(join(paths, "package.json"), "utf8"));
const runtimePackages = new Set(["solid-js", "@solidjs/meta", "@solidjs/router", "@solidjs/web"]);
const toolingPackages = new Set([
  "@solidjs/diagnostics",
  "@solidjs/testing-library",
  "@solidjs/vite-plugin",
  "@testing-library/jest-dom",
  "eslint-plugin-solid",
  "filesystem-routing",
  "jsdom",
  "oxfmt",
  "oxlint",
  "typescript",
  "vite",
  "vitest",
]);
const forbidden =
  /^(?:@(?:kobalte|radix-ui|headlessui|mui|chakra-ui|mantine|ark-ui|ariakit|heroui|react-aria|zag-js|tailwindcss|floating-ui)\/|(?:eroxo-ui|kobalte|tailwindcss|daisyui|shadcn(?:-ui)?|bootstrap|flowbite|solid-headless|solid-aria|solid-ui|react-aria-components|class-variance-authority|tailwind-merge)(?:\/|$))/i;
const problems = [];

for (const section of [
  "dependencies",
  "devDependencies",
  "optionalDependencies",
  "peerDependencies",
]) {
  for (const [name, spec] of Object.entries(manifest[section] ?? {})) {
    const alias = /^npm:(@[^/]+\/[^@]+|[^@]+)/.exec(spec)?.[1];
    if (forbidden.test(name) || (alias && forbidden.test(alias))) {
      problems.push(`package.json: prohibited UI dependency ${name}`);
    } else if (
      alias ||
      !(section === "devDependencies" ? toolingPackages : runtimePackages).has(name)
    ) {
      problems.push(
        `package.json: ${section} entry ${name} requires review and an allowlist entry`,
      );
    }
  }
}

function scan(dir) {
  for (const item of readdirSync(join(paths, dir), { withFileTypes: true })) {
    const file = join(dir, item.name);
    if (item.isDirectory()) {
      if (!["node_modules", "dist", ".git", "scripts"].includes(item.name)) scan(file);
      continue;
    }
    if (!/\.(?:[cm]?[jt]sx?|css|scss|html)$/.test(file)) continue;
    const content = readFileSync(join(paths, file), "utf8");
    const imports =
      /(?:\b(?:from|import)\s*|\b(?:import|require)\s*\(\s*|@(?:import|plugin|use)\s*(?:url\(\s*)?)['"`]([^'"`\r\n]+)['"`]/g;
    for (const [, specifier] of content.matchAll(imports)) {
      if (forbidden.test(specifier)) problems.push(`${file}: prohibited UI import ${specifier}`);
    }
  }
}

scan(".");
if (problems.length) {
  console.error(problems.join("\n"));
  process.exitCode = 1;
}
