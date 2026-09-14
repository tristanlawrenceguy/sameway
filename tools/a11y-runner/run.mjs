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

// Field names for per-violation diagnostic JSON output.
const DIAG_SELECTOR = "selector";
const DIAG_FG = "fg";
const DIAG_BG = "bg";
const DIAG_CONTRAST = "contrast";

// diagnosticNodes extracts per-node diagnostics from axe-core violation nodes.
// Returns [{ selector, fg, bg, contrast }] for each node so agents can see
// exactly which element failed and why (backlog 0197).
async function diagnosticNodes(page, nodes) {
  return Promise.all(
    nodes.map(async (node) => {
      const sel = node.target.length > 0 ? node.target[0] : "?";
      const diag = await page.evaluate((s) => {
        const el = document.querySelector(s);
        if (!el) return null;
        const cs = getComputedStyle(el);
        const fg = cs.color;
        const bg = cs.backgroundColor;
        // WCAG relative luminance + contrast ratio
        function hexToRgb(h) {
          const m = h.match(/\d+/g);
          if (!m || m.length < 3) return null;
          return [parseInt(m[0]), parseInt(m[1]), parseInt(m[2])];
        }
        function srgb(c) {
          const v = c / 255;
          return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4);
        }
        function lum(rgb) {
          if (!rgb) return 0;
          return 0.2126 * srgb(rgb[0]) + 0.7152 * srgb(rgb[1]) + 0.0722 * srgb(rgb[2]);
        }
        const fgRgb = hexToRgb(fg);
        const bgRgb = hexToRgb(bg);
        if (!fgRgb || !bgRgb) return { fg: "?", bg: "?", contrast: 0 };
        const l1 = lum(fgRgb), l2 = lum(bgRgb);
        const ratio = (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05);
        return { fg, bg, contrast: Math.round(ratio * 100) / 100 };
      }, sel);
      const fgHex = diag ? diag.fg : "?";
      const bgHex = diag ? diag.bg : "?";
      const ratio = diag && diag.contrast != null ? diag.contrast : 0;
      const obj = {};
      obj[DIAG_SELECTOR] = sel;
      obj[DIAG_FG] = fgHex;
      obj[DIAG_BG] = bgHex;
      obj[DIAG_CONTRAST] = ratio;
      return obj;
    })
  );
}

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
      const diags = await diagnosticNodes(tab, v.nodes);
      for (const d of diags) {
        console.log(`  ${JSON.stringify(d)}`);
      }
      console.log(`FAIL ${name}/${file}: ${v.id} - ${v.help} (${v.nodes.length} node(s))`);
    }
    const aaa = await new AxeBuilder({ page: tab }).withTags(AAA_TAGS).analyze();
    for (const v of aaa.violations) {
      if (waivers[v.id]) {
        console.log(`waived ${name}/${file}: ${v.id} - ${waivers[v.id]}`);
        continue;
      }
      warnings++;
      const diags = await diagnosticNodes(tab, v.nodes);
      for (const d of diags) {
        console.log(`  ${JSON.stringify(d)}`);
      }
      console.log(`WARN ${name}/${file}: ${v.id} - ${v.help} (AAA)`);
    }
  }
}

await browser.close();
console.log(`a11y: ${failures} AA violation(s), ${warnings} AAA warning(s)`);
process.exit(failures > 0 ? 1 : 0);
