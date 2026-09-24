// Every page of a running sameway server (SAMEWAY_URL, default
// http://127.0.0.1:8080, llm.provider: none), in every mode, and the site
// as a whole.
//
// A canvas is seeded with every example of every component, the way the
// assistant would place them: the first of each in the main region at full
// size, the second in the right pane at compact size, the third as an icon.
// Then each page a person can reach is checked:
//   - axe at AA and AAA, in light and dark, with the quiet layer shown
//   - field edges, 44px targets, colour-only state, error wiring (visual.mjs)
//   - live regions that can announce, links with one name going one place,
//     and no two headings or controls alike
//   - reflow at 320px, text spacing, 200% text, reduced motion, forced colours
//   - a Tab walk at desktop (with focus appearance), at 320px, and on a phone
//     held sideways, for focus hidden under sticky parts
//   - /api/look, for what an agent is told
// and the site: every page titled after itself and no two alike, the main
// navigation the same everywhere, and every form that has required fields
// saying what is wrong when sent empty.
import { chromium } from "playwright";
import { readFileSync, readdirSync, existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { join, resolve } from "node:path";
import { AA_TAGS, AAA_TAGS } from "./shell.mjs";
import { setMode, settle, axeProblems, reflowProblems, spacingProblems, motionProblems, obscuredFocusProblems } from "./checks.mjs";
import { focusAppearance, visualProblems, forcedColourProblems, textZoomProblems, errorWiringProblems } from "./visual.mjs";

const root = resolve(fileURLToPath(new URL(".", import.meta.url)), "..", "..");
const base = (process.env.SAMEWAY_URL || "http://127.0.0.1:8080").replace(/\/$/, "");
const browser = await chromium.launch();
const page = await (await browser.newContext()).newPage();
let failures = 0;
const fail = (msg) => { failures++; console.log(`FAIL ${msg}`); };
const post = (path, body) => fetch(base + path, { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify(body) });

// ---- seed ---------------------------------------------------------------

const note = await (await post("/api/note", { title: "A note for every page", tags: ["a11y"] })).json();
const canvas = await (await post("/api/canvas", { name: "Everything" })).json();
const placings = [{ region: "main", size: "full", span: 12 }, { region: "right", size: "compact", span: 12 }, { region: "main", size: "icon", span: 4 }];
const blocks = [];
const waivedTargets = [];
const componentsDir = join(root, "design", "components");
for (const name of readdirSync(componentsDir).sort()) {
  const file = join(componentsDir, name, "manifest.json");
  if (!existsSync(file)) continue;
  const waivers = join(componentsDir, name, "examples", "a11y-waivers.json");
  if (existsSync(waivers) && JSON.parse(readFileSync(waivers, "utf8"))["target-size-44"]) waivedTargets.push(name);
  for (const [i, ex] of (JSON.parse(readFileSync(file, "utf8")).examples || []).entries()) {
    const res = await post("/api/block", { component: name, props: ex.props, canvas: canvas.id, position: blocks.length, ...(placings[i] || placings[0]) });
    // An example the store refuses is the component's own contract test's
    // business; here it is only left off the canvas.
    if (res.status !== 201) { console.log(`skip ${name}/${ex.name}: ${res.status} ${(await res.text()).slice(0, 120)}`); continue; }
    blocks.push({ id: (await res.json()).id, label: `${name}/${ex.name}` });
  }
}
if (!blocks.length) fail("no examples could be placed on a canvas");

const sitePages = ["/", "/chat", "/design", "/search", "/search?q=keyboard", "/activity", "/t/note", `/t/note/${note.id}`, "/t/note/import",
  "/help", "/workspaces", "/workspaces/new", "/workspaces/copy", "/workspaces/delete", "/t/nothing-here", `/c/${canvas.id}`];
const blockLabel = Object.fromEntries(blocks.map((b) => [`/canvas/${b.id}`, b.label]));

// ---- one page -------------------------------------------------------------

// Live regions that cannot announce, and links that share a name but not a
// place (2.4.9): what the page says about itself to a screen reader.
const pageSays = (seeded, styleguide) => page.evaluate(([seeded, styleguide]) => {
  const out = [];
  for (const el of document.querySelectorAll("[aria-live], [role=status], [role=alert], [role=log]")) {
    const cs = getComputedStyle(el);
    if (cs.display === "none" || cs.visibility === "hidden" || el.closest("[aria-hidden=true]")) out.push(`live region ${el.id ? "#" + el.id : el.className} is hidden, so nothing it says is announced`);
  }
  // The seeded canvas holds several of one component on purpose, and they
  // share names; there only the live regions are checked.
  if (seeded) return out;
  const places = new Map();
  for (const a of document.querySelectorAll("a[href]")) {
    const name = (a.getAttribute("aria-label") || a.textContent).replace(/\s+/g, " ").trim().toLowerCase();
    if (!name) continue;
    const url = new URL(a.href);
    if (!places.has(name)) places.set(name, new Set());
    places.get(name).add(url.pathname + url.search);
  }
  for (const [name, set] of places) if (set.size > 1) out.push(`links named "${name}" go to ${set.size} places: ${[...set].slice(0, 3).join(", ")} (2.4.9)`);
  // Two headings alike, or two controls of one kind with one name, cannot be
  // told apart by someone moving by headings or by controls (2.4.6). The
  // styleguide shows one component's examples side by side, each under its
  // own heading, so its controls repeat by design.
  if (styleguide) return out;
  const twice = (els, key) => { const seen = new Map(); for (const e of els) { const k = key(e); if (k) seen.set(k, (seen.get(k) || 0) + 1); } return [...seen].filter(([, n]) => n > 1); };
  const text = (e) => (e.getAttribute("aria-label") || (e.labels && e.labels[0] && e.labels[0].textContent) || e.textContent || "").replace(/\s+/g, " ").trim().toLowerCase();
  for (const [k, n] of twice(document.querySelectorAll("h1, h2, h3, h4, h5, h6"), (e) => `${e.tagName.toLowerCase()} "${text(e)}"`)) out.push(`${n} headings are ${k} (2.4.6)`);
  // Only what is drawn counts: a control shown only without scripts is not met.
  for (const [k, n] of twice(document.querySelectorAll("button, summary, input:not([type=hidden]), select, textarea"), (e) => e.getClientRects().length > 0 && text(e) && `${e.tagName.toLowerCase()}${e.type ? "[" + e.type + "]" : ""} "${text(e)}"`)) out.push(`${n} controls are ${k} (2.4.6)`);
  return out;
}, [seeded, styleguide]);

const titles = new Map();
let mainNav = null;

async function checkPage(path) {
  const label = blockLabel[path] ? `${path} (${blockLabel[path]})` : path;
  const seeded = path.startsWith("/c/") || blockLabel[path];
  // The seeded canvas stacks every example of a component together, so
  // several share a name by design; the styleguide shows examples as they
  // would sit on a page, own headings and all. Both rules are best
  // practice, not WCAG, and stay on everywhere else.
  const skip = seeded ? ["landmark-unique"] : path === "/design" ? ["heading-order"] : [];
  // Each mode gets a fresh load: some styles lag a mode switched under a
  // page already drawn. axe passes over anything at opacity 0, so the quiet
  // layer's controls are shown the way hover or focus shows them.
  for (const mode of ["dark", "light"]) {
    await setMode(page, mode);
    await page.goto(base + path);
    await page.evaluate(() => { document.documentElement.dataset.controls = "visible"; });
    for (const p of await axeProblems(page, skip, [...AA_TAGS, ...AAA_TAGS])) fail(`${label} ${mode}: ${p}`);
    for (const p of await visualProblems(page, waivedTargets)) fail(`${label} ${mode}: ${p}`);
  }
  for (const p of [...await errorWiringProblems(page), ...await pageSays(Boolean(seeded), path === "/design")]) fail(`${label}: ${p}`);

  // The site as a whole: titled after itself, the same main navigation.
  const head = await page.evaluate(() => ({
    title: document.title.trim(),
    h1: (document.querySelector("h1")?.textContent || "").replace(/\s+/g, " ").trim(),
    nav: [...document.querySelectorAll("nav[aria-label='Main'] a")].map((a) => `${a.textContent.trim()} ${new URL(a.href).pathname}`).join(" | "),
  }));
  if (!head.title || !head.title.toLowerCase().includes(head.h1.toLowerCase().slice(0, 40))) fail(`${label}: title "${head.title}" does not name the page ("${head.h1}") (2.4.2)`);
  if (!seeded) {
    if (titles.has(head.title)) fail(`${label}: title "${head.title}" is also ${titles.get(head.title)}'s (2.4.2)`);
    titles.set(head.title, path);
  }
  if (mainNav === null) mainNav = { path, nav: head.nav };
  else if (head.nav !== mainNav.nav) fail(`${label}: main navigation differs from ${mainNav.path}'s: ${head.nav} (3.2.3)`);

  for (const p of await reflowProblems(page)) fail(`${label} reflow: ${p}`);
  for (const p of await spacingProblems(page)) fail(`${label} text spacing: ${p}`);
  for (const p of await textZoomProblems(page)) fail(`${label} text zoom: ${p}`);
  for (const p of await motionProblems(page)) fail(`${label} reduced motion: ${p}`);
  for (const p of await forcedColourProblems(page)) fail(`${label} forced colours: ${p}`);
  for (const p of await obscuredFocusProblems(page, focusAppearance)) fail(`${label} focus: ${p}`);
  // A phone upright and on its side, where sticky parts take the most room.
  for (const [w, h, where] of [[320, 640, "at 320px"], [844, 390, "on a phone held sideways"]]) {
    await page.setViewportSize({ width: w, height: h });
    await settle(page);
    for (const p of await obscuredFocusProblems(page)) fail(`${label} focus ${where}: ${p}`);
    if (w > 320 && (await page.evaluate(() => document.documentElement.scrollWidth > innerWidth + 1))) fail(`${label}: scrolls sideways ${where}`);
  }
  await page.setViewportSize({ width: 1280, height: 720 });
  // What an agent is told about the page: /api/look lists any problems.
  const look = await (await fetch(`${base}/api/look?path=${encodeURIComponent(path)}`)).json();
  for (const p of look.problems || []) fail(`${label} look: ${typeof p === "string" ? p : JSON.stringify(p)}`);
}

for (const path of [...sitePages, ...Object.keys(blockLabel)]) await checkPage(path);

// ---- forms sent empty ------------------------------------------------------
// Every form with a required field, sent from the keyboard with that field
// empty, says what is wrong (3.3.1): the browser's own message on the field
// it moves focus to, or an error the page ties to the field and announces.
// Only required fields are emptied, so nothing is created or removed.
await setMode(page, "light");
let sent = 0;
for (const path of sitePages) {
  await page.goto(base + path);
  const count = await page.locator("main form:has([required])").count();
  for (let i = 0; i < count; i++) {
    await page.goto(base + path);
    const form = page.locator("main form:has([required])").nth(i);
    const what = `${path} form ${i + 1} (${await form.evaluate((f) => f.getAttribute("aria-label") || f.getAttribute("action") || "")})`;
    await form.evaluate((f) => { for (const el of f.querySelectorAll("[required]")) if (el.type !== "file") el.value = ""; });
    const submit = form.locator("[type=submit], button:not([type])").first();
    sent++;
    if (!(await submit.count())) { fail(`${what}: has required fields but no submit button`); continue; }
    await submit.focus();
    await page.keyboard.press("Enter");
    await page.waitForLoadState();
    await settle(page);
    const said = await page.evaluate(() => {
      const el = document.activeElement;
      if (el && el.matches(":invalid") && el.validationMessage) return "browser";
      const invalid = document.querySelector('[aria-invalid="true"]');
      const alert = [...document.querySelectorAll('[role=alert], [aria-live=assertive]')].find((a) => a.textContent.trim());
      if (invalid && (el === invalid || alert)) return "page";
      if (alert) return "page";
      return null;
    });
    if (!said) fail(`${what}: sent with its required fields empty, nothing says what is wrong or where (3.3.1)`);
    for (const p of await errorWiringProblems(page)) fail(`${what}: ${p}`);
  }
}

if (!sent) fail("no form with a required field was found to send empty");
console.log(`forms sent empty: ${sent}`);

await browser.close();
console.log(`site: ${failures} failure(s)`);
process.exit(failures > 0 ? 1 : 0);
