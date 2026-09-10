// Keyboard tests for every component example.
//
// For each example: every focusable element is reachable with Tab in DOM
// order and shows a visible focus ring; then the component is operated the
// way its manifest keyboard map promises (Enter/Space on buttons, Space on
// checkboxes, Enter in a textarea inserts a newline, Enter in a text field
// submits, ArrowDown changes a select, Enter follows a link).
//
// Usage: cd tools/a11y-runner && npm install && node keyboard.mjs
import { chromium } from "playwright";
import { readFileSync, readdirSync, existsSync } from "node:fs";
import { join, resolve } from "node:path";
import { shell } from "./shell.mjs";

const root = resolve(process.cwd(), "..", "..");
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

const focusables = () => page.evaluate(() => {
  const sel = 'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';
  return [...document.querySelectorAll(sel)].filter((el) => !el.closest(".shell")).map((el) => el.tagName.toLowerCase() + (el.id ? "#" + el.id : ""));
});

const active = () => page.evaluate(() => {
  const el = document.activeElement;
  if (!el || el === document.body) return { tag: "body", ring: false };
  const cs = getComputedStyle(el);
  return { tag: el.tagName.toLowerCase() + (el.id ? "#" + el.id : ""), ring: cs.outlineStyle !== "none" && parseFloat(cs.outlineWidth) > 0 };
});

async function tabOrder(where) {
  const expected = await focusables();
  await page.evaluate(() => document.body.focus());
  // Skip the shell's own focusables (heading links do not exist, so none).
  const seen = [];
  for (let i = 0; i < expected.length + 2; i++) {
    await page.keyboard.press("Tab");
    const a = await active();
    if (a.tag === "body") break;
    seen.push(a.tag);
    if (!a.ring) fail(where, `${a.tag} focused via keyboard without a visible focus ring`);
    if (seen.length === expected.length) break;
  }
  if (seen.join(",") !== expected.join(",")) fail(where, `tab order ${seen.join(",")} differs from DOM order ${expected.join(",")}`);
  return expected;
}

async function operate(name, where) {
  const clicks = () => page.evaluate(() => window.__clicks);
  const submits = () => page.evaluate(() => window.__submits);
  switch (name) {
    case "button": {
      const btn = page.locator("button[data-component=button]:not([disabled])");
      if (await btn.count() === 0) return; // disabled example
      await btn.focus();
      await page.keyboard.press("Enter");
      await page.keyboard.press("Space");
      if (await clicks() !== 2) fail(where, `Enter and Space should each activate the button, got ${await clicks()} clicks`);
      const disabled = page.locator("button[data-component=button][disabled]");
      if (await disabled.count() && (await focusables()).some((f) => f.startsWith("button"))) fail(where, "disabled button must leave the tab order");
      break;
    }
    case "link": {
      await page.locator("a[data-component=link]").focus();
      await page.keyboard.press("Enter");
      if (await clicks() !== 1) fail(where, "Enter should follow the link");
      break;
    }
    case "checkbox": {
      const box = page.locator("input[type=checkbox]");
      const before = await box.isChecked();
      await box.focus();
      await page.keyboard.press("Space");
      if (await box.isChecked() === before) fail(where, "Space should toggle the checkbox");
      await page.locator("label").click();
      if (await box.isChecked() !== before) fail(where, "clicking the label should toggle the checkbox");
      break;
    }
    case "textarea": {
      const ta = page.locator("textarea");
      await ta.focus();
      await page.keyboard.type("a");
      await page.keyboard.press("Enter");
      await page.keyboard.type("b");
      if (!(await ta.inputValue()).includes("\n")) fail(where, "Enter in a textarea should insert a newline");
      if (await submits() !== 0) fail(where, "Enter in a textarea must not submit the form");
      break;
    }
    case "text-field": {
      const input = page.locator("input");
      await input.focus();
      await page.keyboard.type("x");
      await page.keyboard.press("Enter");
      if (await submits() !== 1) fail(where, "Enter in a text field should submit its form");
      break;
    }
    case "select": {
      const sel = page.locator("select");
      const before = await sel.inputValue();
      await sel.focus();
      await page.keyboard.press("ArrowDown");
      if (await sel.inputValue() === before) fail(where, "ArrowDown should move to the next option");
      break;
    }
    case "table": {
      const wrap = page.locator(".sw-table-wrap");
      await wrap.focus();
      if ((await active()).tag !== "div") fail(where, "the table scroll region should be focusable");
      break;
    }
  }
}

for (const name of readdirSync(componentsDir).sort()) {
  const dir = join(componentsDir, name);
  const examplesDir = join(dir, "examples");
  if (!existsSync(examplesDir)) continue;
  const css = existsSync(join(dir, "style.css")) ? readFileSync(join(dir, "style.css"), "utf8") : "";
  for (const file of readdirSync(examplesDir).filter((f) => f.endsWith(".html")).sort()) {
    const where = `${name}/${file}`;
    const body = readFileSync(join(examplesDir, file), "utf8");
    // novalidate: examples such as an invalid email show the error state on
    // purpose; here we test keyboard mechanics, not constraint validation.
    await page.setContent(shell(css, `<form action="#" method="post" novalidate>${body}</form>`));
    await arm();
    const expected = await tabOrder(where);
    if (expected.length > 0) await operate(name, where);
  }
}

await browser.close();
console.log(`keyboard: ${failures} failure(s)`);
process.exit(failures > 0 ? 1 : 0);
