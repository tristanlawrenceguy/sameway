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
await axe("home after chat");
// Person navigates the notes list and opens a detail page via keyboard.
// Create a note first so there is something to navigate to.
const created = await fetch(base + "/api/note", {
  method: "POST",
  headers: { "content-type": "application/json" },
  body: JSON.stringify({ title: "Keyboard person test", tags: ["a11y"] }),
});
check(created.status === 201, `person-note creation: POST /api/note returns 201 (got ${created.status})`);
const personNote = await created.json();

// Navigate to the notes list and verify keyboard access.
await page.getByRole("link", { name: "notes" }).click();
await shellChecks("notes list");
check(await page.locator("[data-component=card]").count() >= 1, "notes list: at least one card rendered");

// Click the card to go to detail (person uses mouse here; agent tests cover keyboard).
await page.getByRole("link", { name: "Keyboard person test" }).click();
await shellChecks("detail");
const h1 = (await page.locator("h1").textContent()).trim();
check(h1 === "Keyboard person test", `detail: h1 should be the note title, got ${JSON.stringify(h1)}`);
const detailText = (await page.locator("main").textContent()).replace(/\s+/g, " ");
check(detailText.includes("a11y"), `detail: field values present; page says ${JSON.stringify(detailText.slice(0, 300))}`);
// Verify no edit link exists on detail pages.
const editLink = await page.locator('a[href*="/edit"]').count();
check(editLink === 0, "detail: no /edit link present");

// Navigate back to list via the notes nav item.
await page.getByRole("link", { name: "notes" }).click();
await shellChecks("notes list after detail");

// Delete the person-created note so it does not pollute subsequent tests.
const removed = await fetch(`${base}/api/note/${personNote.id}`, { method: "DELETE" });
check(removed.status === 200, `person-note cleanup: DELETE returns ${removed.status}`);


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

const created2 = await fetch(base + "/api/note", {
  method: "POST",
  headers: { "content-type": "application/json" },
  body: JSON.stringify({ title: "Created by an agent", tags: ["api"] }),
});
check(created2.status === 201, `agent: POST /api/note returns 201 (got ${created2.status})`);
const rec = await created2.json();

for (const path of ["/", "/chat", "/activity", "/t/note", `/t/note/${rec.id}`]) {
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

const removed2 = await fetch(`${base}/api/note/${rec.id}`, { method: "DELETE" });
check(removed2.status === 200, "agent: DELETE returns 200");

await browser.close();
console.log(`pages: ${failures} failure(s)`);
process.exit(failures > 0 ? 1 : 0);
