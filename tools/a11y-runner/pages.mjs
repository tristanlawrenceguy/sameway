// Live page tests against a running sameway server (SAMEWAY_URL, default
// http://127.0.0.1:8080). The workspace must have llm.provider: none so no
// model is dialled.
//
// Two passes over the same pages:
//   person: keyboard only, skip links, landmarks, forms, error recovery, axe
//   agent:  reads /api/describe, then finds and operates the same controls by
//           role and accessible name, and checks every rendered component is
//           one the description lists.
// Every page in every mode, and the site as a whole, is site.mjs.
import { chromium } from "playwright";
import AxeBuilder from "@axe-core/playwright";
import { AA_TAGS, AAA_TAGS } from "./shell.mjs";
import { axeProblems } from "./checks.mjs";
import { visualProblems } from "./visual.mjs";

const base = (process.env.SAMEWAY_URL || "http://127.0.0.1:8080").replace(/\/$/, "");
const browser = await chromium.launch();
const page = await (await browser.newContext()).newPage();
let failures = 0;
const fail = (msg) => { failures++; console.log(`FAIL ${msg}`); };
const check = (ok, msg) => { if (!ok) fail(msg); };

async function axe(label) {
  const res = await new AxeBuilder({ page }).withTags(AA_TAGS).analyze();
  for (const v of res.violations) fail(`${label}: axe ${v.id} - ${v.help} (${v.nodes.length} node(s))`);
}

async function shellChecks(label) {
  check(await page.locator("h1").count() === 1, `${label}: exactly one h1`);
  check(await page.locator("main#main").count() === 1, `${label}: one main landmark`);
  check(await page.locator("nav[aria-label='Main']").count() === 1, `${label}: labelled nav`);
  await axe(label);
}

// ---- person -------------------------------------------------------------

await page.goto(base + "/");
await shellChecks("home");
await page.keyboard.press("Tab");
check(await page.evaluate(() => document.activeElement.textContent) === "Skip to main content", "home: first Tab lands on the skip link");
await page.keyboard.press("Enter");
check(await page.evaluate(() => document.activeElement.id) === "main", "home: skip link moves focus into main");
check(await page.getByRole("region", { name: "Assistant" }).count() === 1, "home: the canvas carries a named chat region");
check(await page.locator(".sw-canvas > li[data-block-id]").count() >= 1, "home: the canvas is a list of blocks");

// Status live region and the busy enhancement: submitting flips the status
// to "working" and marks the region before the page reloads.
check(await page.locator("#chat-status[role=status][data-state=idle]").count() === 1, "home: status region starts idle");
await page.getByRole("textbox", { name: /Your message/ }).fill("hello from playwright");
const busy = page.evaluate(() => new Promise((resolve) => {
  const form = document.querySelector("form[data-busy-target]");
  form.addEventListener("submit", () => setTimeout(() => resolve({
    busy: form.getAttribute("aria-busy"),
    region: form.closest("[data-region]").getAttribute("data-state"),
    status: document.getElementById("chat-status").getAttribute("data-state"),
    text: document.getElementById("chat-status").textContent,
  }), 0), { once: true });
}));
await page.getByRole("button", { name: "Send" }).click();
const busyState = await busy;
check(busyState.busy === "true" && busyState.region === "working" && busyState.status === "working" && /working/i.test(busyState.text), `chat: submit marks the form, region, and status as working (${JSON.stringify(busyState)})`);
await page.waitForURL(/#msg-/);
check(await page.locator("#chat-status[data-state=error]").count() === 1, "chat: status reports the failed request");
check(await page.locator("[data-component=message][data-actor=human]").count() === 1, "chat: the person's message carries data-actor=human");
check(await page.locator("[data-component=event][data-actor=human]").count() >= 1, "chat: the person's action appears in recent activity");
check(page.url().includes("#msg-"), "chat: redirect targets the newest message");
const errorMsg = page.locator("[data-component=message][data-role=error]");
check(await errorMsg.count() === 1 && /no model configured/.test(await errorMsg.textContent()), "chat: missing model is recorded as a system message");
check(await page.getByRole("link", { name: "Skip to latest message" }).count() === 1, "chat: skip link to latest message present");
// Pressing it takes focus to the newest message, not just to the top of the page.
await page.getByRole("link", { name: "Skip to latest message" }).focus();
await page.keyboard.press("Enter");
check(await page.evaluate(() => {
  const all = document.querySelectorAll("[data-component=message]");
  return all.length > 0 && all[all.length - 1].contains(document.activeElement);
}), "chat: the skip link moves focus to the newest message");
await axe("home after chat");

// Navigate to notes list and verify shell invariants.
await page.getByRole("link", { name: "notes" }).click();
await shellChecks("notes list");

// Create a note via the API so there is one to navigate to on the detail page.
const apiRec = await fetch(base + "/api/note", {
  method: "POST",
  headers: { "content-type": "application/json" },
  body: JSON.stringify({ title: "Created by keyboard test", body: "Written by the test.", tags: ["a11y"] }),
});
check(apiRec.status === 201, `person API note creation returns 201 (got ${apiRec.status})`);
const apiNote = await apiRec.json();

// Navigate to the detail page and verify shell invariants.
await page.goto(base + "/t/note/" + apiNote.id);
await shellChecks("detail");

// Editing a note by keyboard alone: Edit puts focus in the form's first
// field, Tab moves forward through the fields to the body and on to Save,
// and Save from the keyboard keeps what was typed.
const focused = () => page.evaluate(() => {
  const el = document.activeElement;
  return (el.getAttribute("aria-label") || (el.labels && el.labels[0] && el.labels[0].textContent) || el.textContent || "").trim().replace(/\s+/g, " ");
});
const editable = () => page.evaluate(() => { const el = document.activeElement; return el.isContentEditable || /^(INPUT|TEXTAREA|SELECT)$/.test(el.tagName); });
await page.getByRole("button", { name: /^Edit/ }).first().focus();
await page.keyboard.press("Enter");
check(await editable(), `edit: Enter on Edit puts focus in a field of the form (got "${await focused()}")`);
let inBody = (await focused()) === "Body";
for (let i = 0; i < 8 && !inBody; i++) {
  await page.keyboard.press("Tab");
  inBody = (await focused()) === "Body";
}
check(inBody, "edit: Tab forward from the first field reaches the body");
await page.keyboard.press("End");
await page.keyboard.type(" Typed by keyboard.");
let reachedSave = false;
for (let i = 0; i < 10 && !reachedSave; i++) {
  await page.keyboard.press("Tab");
  reachedSave = (await focused()) === "Save";
}
check(reachedSave, "edit: Tab from the body reaches Save");
if (reachedSave) {
  // The save goes by script without leaving the page; wait for it to land.
  const saved = page.waitForResponse((r) => r.request().method() === "POST", { timeout: 10000 }).catch(() => null);
  const reloaded = page.waitForEvent("load", { timeout: 10000 }).catch(() => null);
  await page.keyboard.press("Enter");
  await saved;
  await reloaded;
  const stored = await (await fetch(`${base}/api/note/${apiNote.id}`)).json();
  check(String(stored.fields.body).includes("Typed by keyboard."), `edit: what was typed is saved (stored ${JSON.stringify(stored.fields.body)})`);
}

// ---- the editor is the design system -------------------------------------
// Editing a habit, which has every kind of field: each one the editor makes
// is a design-system component, a choice offers names rather than the word
// the machine stores, and the open form passes the same checks as a page.
const habit = await (await fetch(base + "/api/habit", {
  method: "POST", headers: { "content-type": "application/json" },
  body: JSON.stringify({ name: "Water", target: 8, unit: "glasses", aim: "limit" }),
})).json();
await page.goto(base + "/t/habit/" + habit.id);
await page.getByRole("button", { name: /^Edit/ }).first().click();
const fields = await page.evaluate(() => [...document.querySelectorAll(".sw-inline-form .sw-inline-field")].map((f) => ({
  component: f.dataset.component || (f.classList.contains("sw-prose-field") ? "prose" : ""),
  label: (f.querySelector("label") || {}).textContent,
})));
check(fields.length > 3, `editor: a habit opens with its fields (${fields.length})`);
for (const f of fields) check(["text-field", "when-field", "textarea", "select", "checkbox", "prose"].includes(f.component), `editor: "${f.label}" is not a design-system control`);
const aim = page.getByRole("combobox", { name: "Aim" });
const offered = await aim.evaluate((s) => [...s.options].map((o) => o.textContent));
check(offered.includes("At most the target") && !offered.includes("limit"), `editor: aim offers names, not stored words (${offered.join(", ")})`);
check(await aim.inputValue() === "limit", "editor: the habit's own aim is the one chosen");
for (const p of await axeProblems(page, [], [...AA_TAGS, ...AAA_TAGS])) fail(`editor: ${p}`);
for (const p of await visualProblems(page)) fail(`editor: ${p}`);

// A day is the when-field component: its picker appears once the script is
// there to wire it, and picking a day writes it into the words.
const task = await (await fetch(base + "/api/task", {
  method: "POST", headers: { "content-type": "application/json" },
  body: JSON.stringify({ title: "Plant garlic", due: "2026-10-02" }),
})).json();
await page.goto(base + "/t/task/" + task.id);
await page.getByRole("button", { name: /^Edit/ }).first().click();
const due = page.locator(".sw-inline-form [data-component=when-field]").filter({ hasText: "Due" });
check(await due.count() === 1, "editor: a task's Due is the when-field component");
const picker = due.locator(".sw-when-field__pick");
check(await picker.isVisible(), "editor: the day picker is shown once its script runs");
await picker.fill("2026-10-09");
await picker.dispatchEvent("change");
check((await due.locator("input[type=text]").inputValue()).startsWith("9 Oct 2026"), "editor: picking a day writes it into the words");

// ---- the quiet layer ----------------------------------------------------
// Per-item controls are faded until hovered or focused, but must stay
// present, focusable, named, and clickable for everyone.
await page.goto(base + "/");
const bar = page.locator(".sw-bar").first();
if (await bar.count()) {
  check(await bar.evaluate((el) => getComputedStyle(el).opacity) === "0", "quiet: control bars start faded");
  check(await bar.isVisible(), "quiet: a faded control bar is still reported visible to automation");
  const hidden = await bar.evaluate((el) => {
    const bad = [];
    el.querySelectorAll("*").forEach((n) => {
      const cs = getComputedStyle(n);
      if (cs.display === "none" || cs.visibility === "hidden") bad.push(n.tagName + ":style");
      if (n.hasAttribute("aria-hidden") || n.hasAttribute("inert")) bad.push(n.tagName + ":aria");
      if (n.tabIndex === -1 && n.matches("a,button")) bad.push(n.tagName + ":tabindex");
    });
    return bad;
  });
  check(hidden.length === 0, `quiet: nothing may be hidden from assistive technology (${hidden.join(", ")})`);

  // Hover reveals it.
  await bar.locator("..").hover();
  await page.waitForTimeout(400);
  check(await bar.evaluate((el) => getComputedStyle(el).opacity) === "1", "quiet: hovering the block reveals its controls");

  // Keyboard focus reveals it too.
  await page.mouse.move(0, 0);
  await page.waitForTimeout(400);
  await bar.locator("a,button").first().focus();
  await page.waitForTimeout(400);
  check(await bar.evaluate((el) => getComputedStyle(el).opacity) === "1", "quiet: focusing a control reveals it for keyboard users");

  // An agent can operate it by role and name without hovering first.
  const names = await bar.locator("a,button").evaluateAll((els) => els.map((e) => e.textContent.replace(/\s+/g, " ").trim()));
  for (const n of names) check(/ /.test(n), `quiet: compact control "${n}" should carry its context in the accessible name`);
}

// The activity log is collapsed, and its full contents live on their own page.
const disclosure = page.locator("details[data-component=disclosure]").first();
if (await disclosure.count()) {
  check(!(await disclosure.evaluate((el) => el.open)), "disclosure: the activity log starts closed");
  await disclosure.locator("summary").click();
  check(await disclosure.evaluate((el) => el.open), "disclosure: clicking the summary opens it");
}

// ---- agent --------------------------------------------------------------

const describe = await (await fetch(base + "/api/describe")).json();
const known = new Set(describe.components.map((c) => c.name));
check(describe.types.some((t) => t.name === "note") && known.has("button"), "describe: lists types and components");

const created = await fetch(base + "/api/note", {
  method: "POST",
  headers: { "content-type": "application/json" },
  body: JSON.stringify({ title: "Created by an agent", tags: ["api"] }),
});
check(created.status === 201, `agent: POST /api/note returns 201 (got ${created.status})`);
const rec = await created.json();

for (const path of ["/", "/chat", "/design", "/search", "/activity", "/t/note", `/t/note/${rec.id}`]) {
  await page.goto(base + path);
  const names = await page.locator("[data-component]").evaluateAll((els) => els.map((e) => e.dataset.component));
  for (const n of new Set(names)) check(known.has(n), `${path}: renders component ${n} that /api/describe does not list`);
  for (const c of describe.components) {
    const selector = c.machine.selector;
    const count = await page.locator(selector).count();
    if (names.includes(c.name)) check(count > 0, `${path}: machine.selector for ${c.name} matches nothing`);
  }
  // Every focusable control has an accessible name an agent can target by role.
  const unnamed = await page.evaluate(() => {
    // A hidden input carries a form value; it is not a control.
    const sel = 'a[href], button, input:not([type=hidden]), select, textarea, summary';
    return [...document.querySelectorAll(sel)].filter((el) => {
      const labelled = el.labels && el.labels.length > 0;
      return !(labelled || el.getAttribute("aria-label") || el.getAttribute("aria-labelledby") || el.textContent.trim());
    }).map((el) => el.outerHTML.slice(0, 80));
  });
  for (const u of unnamed) fail(`${path}: control without an accessible name: ${u}`);
}

// The agent-created record is reachable by a person through the same UI.
await page.goto(base + "/t/note");
await page.getByRole("link", { name: "Created by an agent" }).click();
check(await page.locator("h1").textContent() === "Created by an agent", "agent record: person can open it from the list");

const removed = await fetch(`${base}/api/note/${rec.id}`, { method: "DELETE" });
check(removed.status === 200, "agent: DELETE returns 200");

await browser.close();
console.log(`pages: ${failures} failure(s)`);
process.exit(failures > 0 ? 1 : 0);
