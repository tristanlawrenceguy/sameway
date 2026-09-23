// Keyboard tests for every component example.
//
// For each example: every focusable element is reachable with Tab in DOM
// order with a visible focus ring, in the default, dark, and forced-colours
// modes, 2px thick and 3:1 against what it is drawn over (WCAG 2.4.13);
// Shift+Tab walks the same order back and Tab leaves the last one, so
// nothing traps focus. Then every control is operated the way the platform
// promises for its kind (Enter follows a link, Enter and Space press a
// button, Space toggles a checkbox, Enter and Space open a summary, Up
// changes a date part, Down changes a select, Space opens a file chooser,
// typing fills a field, Enter in a textarea is a new line), and Enter in a
// field submits its form when the component's manifest says Enter does.
// A component whose examples can be focused must say what Tab reaches in
// its manifest keyboard map.
//
// Usage: cd tools/a11y-runner && npm install && node keyboard.mjs
import { chromium } from "playwright";
import { readFileSync, readdirSync, existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { join, resolve } from "node:path";
import { shell } from "./shell.mjs";
import { setMode } from "./checks.mjs";
import { focusAppearance } from "./visual.mjs";

const __dirname = fileURLToPath(new URL(".", import.meta.url));
const root = resolve(__dirname, "..", "..");
const componentsDir = join(root, "design", "components");
const browser = await chromium.launch();
const page = await (await browser.newContext()).newPage();
let failures = 0;

function fail(where, msg) {
  failures++;
  console.log(`FAIL ${where}: ${msg}`);
}

// Instrument the page: count clicks and submits, block real navigation.
// Only link clicks are cancelled; cancelling a checkbox click would undo
// the toggle the test is looking for.
async function arm() {
  await page.evaluate(() => {
    window.__clicks = 0;
    window.__submits = 0;
    document.addEventListener("click", (e) => {
      window.__clicks++;
      if (e.target.closest("a")) e.preventDefault();
    });
    document.addEventListener("submit", (e) => { window.__submits++; e.preventDefault(); });
  });
}

// summary is focusable; a hidden input is a form value, not a control.
const FOCUSABLE = 'a[href], button:not([disabled]), summary, input:not([disabled]):not([type=hidden]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

const focusables = () => page.evaluate((sel) => {
  return [...document.querySelectorAll(sel)].filter((el) => !el.closest(".shell")).map((el) => el.tagName.toLowerCase() + (el.id ? "#" + el.id : ""));
}, FOCUSABLE);

const active = () => page.evaluate(() => {
  const el = document.activeElement;
  if (!el || el === document.body) return { tag: "body", ring: false };
  const cs = getComputedStyle(el);
  return { tag: el.tagName.toLowerCase() + (el.id ? "#" + el.id : ""), ring: cs.outlineStyle !== "none" && parseFloat(cs.outlineWidth) > 0 };
});

async function tabOrder(where, mode) {
  const expected = await focusables();
  await page.evaluate(() => document.body.focus());
  const seen = [];
  for (let i = 0; i < expected.length + 2; i++) {
    await page.keyboard.press("Tab");
    const a = await active();
    if (a.tag === "body") break;
    seen.push(a.tag);
    if (!a.ring) fail(where, `${a.tag} focused via keyboard without a visible focus ring (${mode})`);
    // In forced colours the ring is the person's own colour; elsewhere it
    // must be 2px and 3:1 against what it is drawn over (2.4.13).
    const weak = mode === "forced" ? null : await focusAppearance(page);
    if (weak) fail(where, `${weak} (${mode})`);
    if (seen.length === expected.length) break;
  }
  if (seen.join(",") !== expected.join(",")) fail(where, `tab order ${seen.join(",")} differs from DOM order ${expected.join(",")} (${mode})`);
  if (mode !== "light" || expected.length === 0) return expected;
  // Tab from the last control leaves the example; nothing holds focus. A
  // date field takes a Tab for each of its parts before it lets go.
  let out;
  for (let i = 0; i < 4; i++) {
    await page.keyboard.press("Tab");
    out = await active();
    if (out.tag === "body") break;
  }
  if (out.tag !== "body") fail(where, `Tab from the last control stays on ${out.tag}: a keyboard trap`);
  // Shift+Tab from the end walks the same order backwards.
  const back = [];
  for (let i = 0; i < expected.length; i++) {
    await page.keyboard.press("Shift+Tab");
    back.push((await active()).tag);
  }
  if (back.reverse().join(",") !== expected.join(",")) fail(where, `Shift+Tab order ${back.join(",")} is not the Tab order reversed`);
  return expected;
}

// Operates every control in the example by the kind of element it is.
async function operate(where, enterSubmits) {
  const count = await page.evaluate((sel) => {
    const els = [...document.querySelectorAll(sel)].filter((el) => !el.closest(".shell"));
    els.forEach((el, i) => { el.dataset.kb = i; });
    return els.length;
  }, FOCUSABLE);
  const counts = () => page.evaluate(() => ({ clicks: window.__clicks, submits: window.__submits }));
  for (let i = 0; i < count; i++) {
    const el = page.locator(`[data-kb="${i}"]`);
    const kind = await el.evaluate((e) => {
      const t = e.tagName.toLowerCase();
      if (t === "input") return "input:" + (e.type || "text");
      if (t === "a" || t === "summary" || t === "button" || t === "select" || t === "textarea") return t;
      return "region";
    });
    const what = `${kind} ${i + 1} of ${count}`;
    await el.focus();
    const before = await counts();
    switch (kind) {
      case "a": {
        await page.keyboard.press("Enter");
        if ((await counts()).clicks !== before.clicks + 1) fail(where, `Enter should follow the link (${what})`);
        break;
      }
      case "button": {
        await page.keyboard.press("Enter");
        await el.focus();
        await page.keyboard.press("Space");
        if ((await counts()).clicks !== before.clicks + 2) fail(where, `Enter and Space should each press the button (${what})`);
        break;
      }
      case "summary": {
        const open = () => el.evaluate((e) => e.parentElement.open);
        const was = await open();
        await page.keyboard.press("Enter");
        if (await open() === was) fail(where, `Enter on the summary should open or close it (${what})`);
        await page.keyboard.press("Space");
        if (await open() !== was) fail(where, `Space on the summary should open or close it (${what})`);
        break;
      }
      case "input:checkbox":
      case "input:radio": {
        const was = await el.isChecked();
        await page.keyboard.press("Space");
        if (await el.isChecked() === was && !(kind === "input:radio" && was)) fail(where, `Space should check the ${kind.slice(6)} (${what})`);
        break;
      }
      case "input:file": {
        const chooser = page.waitForEvent("filechooser", { timeout: 2000 }).then(() => true, () => false);
        await page.keyboard.press("Space");
        if (!(await chooser)) fail(where, `Space on the file field should open the file chooser (${what})`);
        break;
      }
      case "input:date": {
        const was = await el.inputValue();
        if (!was) break; // an empty date has no part to step
        await page.keyboard.press("ArrowUp");
        if (await el.inputValue() === was) fail(where, `ArrowUp should change the date part under the cursor (${what})`);
        break;
      }
      case "select": {
        const last = await el.evaluate((s) => s.selectedIndex >= s.options.length - 1);
        if (last) break;
        const was = await el.inputValue();
        await page.keyboard.press("ArrowDown");
        if (await el.inputValue() === was) fail(where, `ArrowDown should move to the next option (${what})`);
        break;
      }
      case "textarea": {
        await page.keyboard.type("a");
        await page.keyboard.press("Enter");
        await page.keyboard.type("b");
        if (!(await el.inputValue()).includes("a\nb")) fail(where, `Enter in a textarea should insert a newline (${what})`);
        if ((await counts()).submits !== before.submits) fail(where, `Enter in a textarea must not submit the form (${what})`);
        break;
      }
      case "region": break; // a focusable scroll region: reaching it is the promise
      default: {
        // Text-like fields: typing fills them; Enter submits when promised.
        const typeable = await el.evaluate((e) => ["text", "search", "email", "url", "tel", "password", "number"].includes(e.type) && !e.readOnly);
        if (!typeable) break;
        const was = await el.inputValue();
        await page.keyboard.type(kind === "input:number" ? "1" : "x");
        if (await el.inputValue() === was) fail(where, `typing should change the field (${what})`);
        if (enterSubmits) {
          await page.keyboard.press("Enter");
          if ((await counts()).submits !== before.submits + 1) fail(where, `Enter in the field should submit its form, as the manifest says (${what})`);
        }
      }
    }
  }
}

for (const name of readdirSync(componentsDir).sort()) {
  const dir = join(componentsDir, name);
  const examplesDir = join(dir, "examples");
  if (!existsSync(examplesDir)) continue;
  const manifest = JSON.parse(readFileSync(join(dir, "manifest.json"), "utf8"));
  const keys = (manifest.a11y && manifest.a11y.keyboard) || [];
  const tabDocumented = keys.some((k) => /\bTab\b/.test(k.key));
  // Enter is promised to submit when the map names Enter and says it does
  // something other than follow a link or open a picker.
  const enterSubmits = keys.some((k) => /\bEnter\b/.test(k.key) && /submit|search|sets|logs/i.test(k.does));
  for (const file of readdirSync(examplesDir).filter((f) => f.endsWith(".html")).sort()) {
    const where = `${name}/${file}`;
    const body = readFileSync(join(examplesDir, file), "utf8");
    // novalidate: examples such as an invalid email show the error state on
    // purpose; here we test keyboard mechanics, not constraint validation.
    // An example with forms of its own is left unwrapped: a form inside a
    // form is dropped by the parser, and its end tag closes the outer one.
    const html = shell(/<form[\s>]/.test(body) ? body : `<form action="#" method="post" novalidate>${body}</form>`);
    let expected = [];
    for (const mode of ["dark", "forced", "light"]) {
      await setMode(page, mode);
      await page.setContent(html);
      await arm();
      expected = await tabOrder(where, mode);
    }
    if (expected.length > 0 && !tabDocumented) fail(where, "can be focused, but the manifest keyboard map does not say what Tab reaches");
    if (expected.length > 0) await operate(where, enterSubmits);
  }
}

await browser.close();
console.log(`keyboard: ${failures} failure(s)`);
process.exit(failures > 0 ? 1 : 0);
