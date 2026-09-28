// Runs axe-core against every component example in design/components.
//
// AA and AAA violations (axe tags wcag2a to wcag22aaa) fail the run unless
// the component has a waiver in examples/a11y-waivers.json that names the
// rule and a reason; AA rules cannot be waived. Each example is also
// checked with axe in dark mode, for reflow at 320px, for clipped text
// under WCAG text spacing and at 200% text, for motion when reduced motion
// is asked for (checks.mjs), for field edges, 44px targets, colour-only
// state, marks lost in forced colours and errors tied to their fields
// (visual.mjs), and against the role its manifest declares.
// Any problem fails the run.
//
// Usage: cd tools/a11y-runner && npm install && npm test
import { chromium } from "playwright";
import AxeBuilder from "@axe-core/playwright";
import { readFileSync, readdirSync, existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { join, resolve } from "node:path";

const __dirname = fileURLToPath(new URL(".", import.meta.url));
const root = resolve(__dirname, "..", "..");
import { shell, AA_TAGS, AAA_TAGS } from "./shell.mjs";
import { setMode, settle, axeProblems, reflowProblems, spacingProblems, motionProblems } from "./checks.mjs";
import { visualProblems, forcedColourProblems, textZoomProblems, errorWiringProblems } from "./visual.mjs";

// ARIA roles a manifest's a11y.role may name, and the ones worth holding
// an example's outermost element to: landmarks, live regions and widgets.
const ROLES = ["alert", "article", "button", "checkbox", "combobox", "figure", "form", "group", "heading", "img", "link", "list", "listitem", "meter", "navigation", "progressbar", "radio", "paragraph", "region", "search", "searchbox", "spinbutton", "status", "table", "term", "definition", "textbox"];
const PLAIN = new Set(["text", "generic", "paragraph", "listitem", "none"]);

// The roles an example's accessibility tree holds, outermost first.
async function rolesOf(page) {
  const snap = await page.locator("#example").ariaSnapshot();
  // A line whose name holds a colon ("Error: …") comes back quoted, as YAML
  // would write it: the quote is not part of the role.
  return snap.split("\n").map((l) => l.trim().replace(/^- /, "").replace(/^'/, "").split(/[\s:"]/)[0]).filter(Boolean);
}

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
let waived = 0;

for (const name of readdirSync(componentsDir).sort()) {
  const dir = join(componentsDir, name);
  const examplesDir = join(dir, "examples");
  if (!existsSync(examplesDir)) continue;
  const waiverFile = join(examplesDir, "a11y-waivers.json");
  const waivers = existsSync(waiverFile) ? JSON.parse(readFileSync(waiverFile, "utf8")) : {};
  const manifest = JSON.parse(readFileSync(join(dir, "manifest.json"), "utf8"));
  // Field-like components need a form context; wrap everything in one. The
  // page shell above supplies h1 and h2 so components that default to level
  // 3 sit in a valid outline, the same way they do inside a real page section.
  for (const file of readdirSync(examplesDir).filter((f) => f.endsWith(".html")).sort()) {
    const body = readFileSync(join(examplesDir, file), "utf8");
    await tab.setContent(shell(`<form id="example">${body}</form>`));
    // Give the renderer a chance to apply CSS before axe-core reads computed styles.
    // Without this, setContent can race with style application on reused pages,
    // causing non-deterministic color-contrast failures (backlog 0161).
    await new Promise((resolve) => setTimeout(resolve, 250));
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
        waived++;
        console.log(`waived ${name}/${file}: ${v.id} - ${waivers[v.id]}`);
        continue;
      }
      failures++;
      const diags = await diagnosticNodes(tab, v.nodes);
      for (const d of diags) {
        console.log(`  ${JSON.stringify(d)}`);
      }
      console.log(`FAIL ${name}/${file}: ${v.id} - ${v.help} (AAA)`);
    }
    // What the manifest says a reader meets is what the tree holds.
    const other = [];
    const declared = ROLES.filter((r) => new RegExp(`\\b${r}s?\\b`).test(manifest.a11y.role));
    const found = await rolesOf(tab);
    // An example of plain text (a badge, an empty chart's words) holds no
    // role worth naming.
    const plainOnly = found.every((r) => PLAIN.has(r));
    if (!plainOnly && !found.some((r) => declared.includes(r))) other.push(`manifest a11y.role "${manifest.a11y.role}" names none of the roles the example has (${found.slice(0, 5).join(", ")})`);
    if (found[0] && !PLAIN.has(found[0]) && !declared.includes(found[0])) other.push(`the example is a ${found[0]}, which manifest a11y.role "${manifest.a11y.role}" does not say`);
    // Light and dark: what axe does not measure.
    const excused = waivers["target-size-44"] ? [name] : [];
    for (const mode of ["light", "dark"]) {
      await setMode(tab, mode);
      await tab.setContent(shell(`<form id="example">${body}</form>`));
      await new Promise((resolve) => setTimeout(resolve, 250));
      if (mode === "dark") for (const p of await axeProblems(tab, Object.keys(waivers).filter((k) => k !== "target-size-44"), [...AA_TAGS, ...AAA_TAGS])) other.push(`dark: ${p}`);
      for (const p of await visualProblems(tab, excused)) other.push(`${mode}: ${p}`);
    }
    // The same example at the sizes, spacing, zoom and motion settings WCAG
    // asks it to survive, and in forced colours.
    await setMode(tab, "light");
    await tab.setContent(shell(`<form id="example">${body}</form>`));
    await settle(tab);
    for (const p of await reflowProblems(tab)) other.push(`reflow: ${p}`);
    for (const p of await spacingProblems(tab)) other.push(`text spacing: ${p}`);
    for (const p of await textZoomProblems(tab)) other.push(`text zoom: ${p}`);
    for (const p of await motionProblems(tab)) other.push(`reduced motion: ${p}`);
    for (const p of await forcedColourProblems(tab)) other.push(`forced colours: ${p}`);
    for (const p of await errorWiringProblems(tab)) other.push(`errors: ${p}`);
    for (const p of other) {
      failures++;
      console.log(`FAIL ${name}/${file}: ${p}`);
    }
  }
}

await browser.close();
console.log(`a11y: ${failures} violation(s), ${waived} waived`);
process.exit(failures > 0 ? 1 : 0);
