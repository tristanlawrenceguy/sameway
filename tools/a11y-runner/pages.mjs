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
import { readFileSync, readdirSync, existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { join, resolve } from "node:path";
import { AA_TAGS } from "./shell.mjs";
import { setMode, axeProblems, reflowProblems, spacingProblems, motionProblems, obscuredFocusProblems } from "./checks.mjs";

const root = resolve(fileURLToPath(new URL(".", import.meta.url)), "..", "..");
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

// Navigate to notes list and verify shell invariants.
await page.getByRole("link", { name: "notes" }).click();
await shellChecks("notes list");

// Create a note via the API so there is one to navigate to on the detail page.
const apiRec = await fetch(base + "/api/note", {
  method: "POST",
  headers: { "content-type": "application/json" },
  body: JSON.stringify({ title: "Created by keyboard test", tags: ["a11y"] }),
});
check(apiRec.status === 201, `person API note creation returns 201 (got ${apiRec.status})`);
const apiNote = await apiRec.json();

// Navigate to the detail page and verify shell invariants.
await page.goto(base + "/t/note/" + apiNote.id);
await shellChecks("detail");

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

// ---- every page, every mode ---------------------------------------------
// A canvas holding every example of every component, the way the assistant
// would place them: the first of each in the main region at full size, the
// second in the right pane at compact size, the third as an icon. Then every
// page a person can reach is checked in light, dark and forced colours, at
// 320px wide, with WCAG text spacing, with reduced motion, by walking it
// with Tab for focus hidden under sticky parts, and by /api/look.

const post = (path, body) => fetch(base + path, { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify(body) });
const canvas = await (await post("/api/canvas", { name: "Everything" })).json();
const placings = [{ region: "main", size: "full", span: 12 }, { region: "right", size: "compact", span: 12 }, { region: "main", size: "icon", span: 4 }];
const blocks = [];
const componentsDir = join(root, "design", "components");
for (const name of readdirSync(componentsDir).sort()) {
  const file = join(componentsDir, name, "manifest.json");
  if (!existsSync(file)) continue;
  const examples = JSON.parse(readFileSync(file, "utf8")).examples || [];
  for (const [i, ex] of examples.entries()) {
    const res = await post("/api/block", { component: name, props: ex.props, canvas: canvas.id, position: blocks.length, ...(placings[i] || placings[0]) });
    // An example the store refuses is the component's own contract test's
    // business; here it is only left off the canvas.
    if (res.status !== 201) { console.log(`skip ${name}/${ex.name}: ${res.status} ${(await res.text()).slice(0, 120)}`); continue; }
    blocks.push({ id: (await res.json()).id, label: `${name}/${ex.name}` });
  }
}
check(blocks.length > 0, "every page: examples placed on a canvas");

const everyPage = ["/", "/chat", "/design", "/search", "/search?q=keyboard", "/activity", "/t/note", `/t/note/${apiNote.id}`, "/t/note/import",
  "/workspaces", "/workspaces/new", "/workspaces/copy", "/workspaces/delete", "/t/nothing-here", `/c/${canvas.id}`,
  ...blocks.map((b) => `/canvas/${b.id}`)];
const blockLabel = Object.fromEntries(blocks.map((b) => [`/canvas/${b.id}`, b.label]));

for (const path of everyPage) {
  const label = blockLabel[path] ? `${path} (${blockLabel[path]})` : path;
  // Each mode gets a fresh load: some styles lag a mode switched under a
  // page that is already drawn. axe passes over anything at opacity 0, so
  // the quiet layer's controls are shown the way hover or focus shows them.
  for (const mode of ["dark", "light"]) {
    await setMode(page, mode);
    await page.goto(base + path);
    await page.evaluate(() => { document.documentElement.dataset.controls = "visible"; });
    // The seeded canvas stacks every example of a component together, so
    // several share a name by design; landmark-unique is best practice,
    // not WCAG, and stays on for every other page.
    // The styleguide shows every example as it would sit on a page, own
    // headings and all, between its h4 labels; heading-order is likewise
    // best practice and is left out there alone.
    const seeded = path.startsWith("/c/") || blockLabel[path];
    const skip = seeded ? ["landmark-unique"] : path === "/design" ? ["heading-order"] : [];
    for (const p of await axeProblems(page, skip)) fail(`${label} ${mode}: ${p}`);
  }
  for (const p of await reflowProblems(page)) fail(`${label} reflow: ${p}`);
  for (const p of await spacingProblems(page)) fail(`${label} text spacing: ${p}`);
  for (const p of await motionProblems(page)) fail(`${label} reduced motion: ${p}`);
  for (const p of await obscuredFocusProblems(page)) fail(`${label} focus: ${p}`);
  await page.setViewportSize({ width: 320, height: 640 });
  for (const p of await obscuredFocusProblems(page)) fail(`${label} focus at 320px: ${p}`);
  await page.setViewportSize({ width: 1280, height: 720 });
  // What an agent is told about the page: /api/look lists any problems.
  const look = await (await fetch(`${base}/api/look?path=${encodeURIComponent(path)}`)).json();
  for (const p of look.problems || []) fail(`${label} look: ${typeof p === "string" ? p : JSON.stringify(p)}`);
}

await browser.close();
console.log(`pages: ${failures} failure(s)`);
process.exit(failures > 0 ? 1 : 0);
