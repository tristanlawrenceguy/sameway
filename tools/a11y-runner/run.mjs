// Runs axe-core against every component example in design/components.
//
// AA violations (tags wcag2a, wcag2aa, wcag21a, wcag21aa, wcag22aa) fail the
// run. AAA findings are printed as warnings unless the component has a
// waiver in examples/a11y-waivers.json that names the rule and a reason.
//
// Usage: cd tools/a11y-runner && npm install && npm test
import { chromium } from "playwright";
import AxeBuilder from "@axe-core/playwright";
import { readFileSync, readdirSync, existsSync } from "node:fs";
import { join, resolve } from "node:path";
import { shell, AA_TAGS, AAA_TAGS } from "./shell.mjs";

const root = resolve(process.cwd(), "..", "..");
const componentsDir = join(root, "design", "components");

const browser = await chromium.launch();
const ctx = await browser.newContext();
const tab = await ctx.newPage();
let failures = 0;
let warnings = 0;

for (const name of readdirSync(componentsDir).sort()) {
  const dir = join(componentsDir, name);
  const examplesDir = join(dir, "examples");
  if (!existsSync(examplesDir)) continue;
  const css = existsSync(join(dir, "style.css")) ? readFileSync(join(dir, "style.css"), "utf8") : "";
  const waiverFile = join(examplesDir, "a11y-waivers.json");
  const waivers = existsSync(waiverFile) ? JSON.parse(readFileSync(waiverFile, "utf8")) : {};
  // Field-like components need a form context; wrap everything in one. The
  // page shell above supplies h1 and h2 so components that default to level
  // 3 sit in a valid outline, the same way they do inside a real page section.
  for (const file of readdirSync(examplesDir).filter((f) => f.endsWith(".html")).sort()) {
    const body = readFileSync(join(examplesDir, file), "utf8");
    await tab.setContent(shell(css, `<form>${body}</form>`));
    const aa = await new AxeBuilder({ page: tab }).withTags(AA_TAGS).analyze();
    for (const v of aa.violations) {
      failures++;
      console.log(`FAIL ${name}/${file}: ${v.id} - ${v.help} (${v.nodes.length} node(s))`);
    }
    const aaa = await new AxeBuilder({ page: tab }).withTags(AAA_TAGS).analyze();
    for (const v of aaa.violations) {
      if (waivers[v.id]) {
        console.log(`waived ${name}/${file}: ${v.id} - ${waivers[v.id]}`);
        continue;
      }
      warnings++;
      console.log(`WARN ${name}/${file}: ${v.id} - ${v.help} (AAA)`);
    }
  }
}

await browser.close();
console.log(`a11y: ${failures} AA violation(s), ${warnings} AAA warning(s)`);
process.exit(failures > 0 ? 1 : 0);
