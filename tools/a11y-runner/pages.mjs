// Live page tests against a running sameway server (SAMEWAY_URL, default
// http://127.0.0.1:8080). The workspace must have llm.provider: none so no
// model is dialled.
//
// Two passes over the same pages:
//   person: keyboard only, skip links, landmarks, forms, error recovery, axe
//   agent:  reads /api/describe, then finds and operates the same controls by
//           role and accessible name, and checks every rendered component is
//           one the description lists.
import { chromium } from "playwright";
import AxeBuilder from "@axe-core/playwright";
import { AA_TAGS } from "./shell.mjs";

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
check(await page.getByRole("region").count() >= 0 && await page.locator("section[aria-labelledby]").count() === 2, "home: conversation and canvas regions are labelled");

// Chat with no model: the failure must land in the transcript.
await page.getByRole("textbox", { name: /Your message/ }).fill("hello from playwright");
await page.getByRole("button", { name: "Send" }).click();
await page.waitForURL(/#msg-/);
check(page.url().includes("#msg-"), "chat: redirect targets the newest message");
const errorMsg = page.locator("[data-component=message][data-role=error]");
check(await errorMsg.count() === 1 && /no model configured/.test(await errorMsg.textContent()), "chat: missing model is recorded as a system message");
check(await page.getByRole("link", { name: "Skip to latest message" }).count() === 1, "chat: skip link to latest message present");
await axe("home after chat");

// Keyboard-only note creation.
await page.getByRole("link", { name: "notes" }).click();
await shellChecks("notes list");
await page.getByRole("link", { name: "New note" }).click();
await shellChecks("new note");
const title = page.getByRole("textbox", { name: /Title/ });
await title.focus();
await page.keyboard.type("Typed by keyboard");
await page.keyboard.press("Tab");
await page.keyboard.type("Body line one");
await page.keyboard.press("Enter");
await page.keyboard.type("Body line two");
check((await page.getByRole("textbox", { name: "Body" }).inputValue()).includes("\n"), "new note: Enter in body inserts a newline");
await page.getByRole("combobox", { name: "Status" }).selectOption("published");
await page.getByRole("checkbox", { name: "Pinned" }).check();
await title.focus();
await page.keyboard.press("Enter");
// Enter starts a navigation; wait for the detail URL rather than a load
// state that the form page already satisfies.
await page.waitForURL(/\/t\/note\/[a-z0-9]+$/, { timeout: 10000 }).catch(() => {});
check(/\/t\/note\/[a-z0-9]+$/.test(page.url()), `new note: Enter in the title submits and lands on the detail page (${page.url()})`);
check(await page.locator("h1").textContent() === "Typed by keyboard", "detail: h1 is the note title");
const detailText = await page.locator("main").textContent();
check(detailText.includes("published") && detailText.includes("yes"), "detail: status and pinned saved");
await shellChecks("detail");

// Validation failure with values preserved and errors linked.
await page.getByRole("link", { name: "Edit note" }).click();
await page.getByRole("textbox", { name: /Title/ }).fill("x".repeat(201));
await page.getByRole("button", { name: "Save" }).click();
await page.getByRole("alert").first().waitFor({ timeout: 10000 }).catch(() => {});
const invalid = page.getByRole("textbox", { name: /Title/ });
check(await invalid.getAttribute("aria-invalid") === "true", "edit: over-long title marks the field invalid");
const describedBy = await invalid.getAttribute("aria-describedby");
check(describedBy && await page.locator("#" + describedBy.split(" ").pop()).count() === 1, "edit: error text is linked via aria-describedby");
check((await invalid.inputValue()).length === 201, "edit: submitted value is preserved on error");
check(await page.getByRole("alert").count() >= 1, "edit: failed submit announces an alert");
await axe("edit with errors");

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

for (const path of ["/", "/t/note", "/t/note/new", `/t/note/${rec.id}`, `/t/note/${rec.id}/edit`]) {
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
    const sel = 'a[href], button, input, select, textarea';
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
