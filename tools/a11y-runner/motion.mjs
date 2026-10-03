// What a person does moves where it goes, and under reduced motion it does
// not travel (design/base/26-travel.js and .css), in a real browser against
// a running server (SAMEWAY_URL). Four everyday actions, each done by
// keyboard, once with motion and once with prefers-reduced-motion:
//   - tick a task on its type's page, then move on: the row goes to Done
//   - Move a card on a board to another column
//   - Apply a list's filters
//   - go to a calendar's next month
// Every view transition animation the page runs is recorded as it runs.
// With reduced motion none may move anything (a transform, a width or a
// height that changes), though a cross-fade may still show the change.
// With motion, the tick, the move and the month must travel, so the check
// is seen to be checking something. Focus stays on the control acted on
// where the page stays (the tick) and comes back to it where the page
// comes back (Move). Makes its own records and blocks, and deletes them.
import { chromium } from "playwright";

const base = (process.env.SAMEWAY_URL || "http://127.0.0.1:8080").replace(/\/$/, "");
let failures = 0;
const check = (ok, msg) => { if (!ok) { failures++; console.log(`FAIL motion: ${msg}`); } };
const post = (path, body) => fetch(base + path, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }).then((r) => r.json());
const today = new Date().toISOString().slice(0, 10);
const stamp = Date.now().toString(36);

// Records every view transition animation, in every document the tab
// loads, from before the page's own scripts run.
function recorder() {
  window.__swVT = [];
  const seen = new WeakSet();
  const moves = (kf, prop) => new Set(kf.map((k) => k[prop] === undefined || k[prop] === "none" ? "none" : k[prop])).size > 1 ||
    kf.some((k) => prop === "transform" && /translate|matrix/.test(k[prop] || "") && !/^matrix\(1, 0, 0, 1, 0, 0\)$/.test(k[prop]));
  const look = () => {
    for (const a of document.getAnimations()) {
      const pe = a.effect && a.effect.pseudoElement;
      if (!pe || !pe.startsWith("::view-transition") || seen.has(a)) continue;
      seen.add(a);
      let kf = [];
      try { kf = a.effect.getKeyframes(); } catch (e) { /* none */ }
      window.__swVT.push({ pe, name: a.animationName || "", travels: moves(kf, "transform") || moves(kf, "width") || moves(kf, "height") });
    }
    requestAnimationFrame(look);
  };
  requestAnimationFrame(look);
}

const settle = (page, ms = 700) => page.waitForTimeout(ms);
const record = (page) => page.evaluate(() => window.__swVT.splice(0));
const travels = (vt, part) => vt.filter((a) => a.travels && (!part || a.pe.includes(part)));

async function run(reduced) {
  const mode = reduced ? "reduced motion" : "motion";
  const browser = await chromium.launch();
  const context = await browser.newContext({ viewport: { width: 1280, height: 900 }, reducedMotion: reduced ? "reduce" : "no-preference" });
  await context.addInitScript(recorder);
  const page = await context.newPage();
  const made = [];

  // Tick, then move on.
  const task = await post("/api/task", { title: `Motion tick ${mode} ${stamp}`, due: today + "T00:00:00Z" });
  made.push(`/api/task/${task.id}`);
  await page.goto(base + "/t/task");
  const box = page.locator(`form.sw-mark[data-record-id="${task.id}"] input[type=checkbox]`);
  await box.focus();
  await record(page);
  await page.keyboard.press("Space");
  await page.waitForSelector("#outcome");
  await settle(page, 300);
  check(await box.evaluate((b) => b === document.activeElement && b.checked), `${mode}: focus stays on the box ticked`);
  check(travels(await record(page)).length === 0, `${mode}: nothing moves while focus is still in the list`);
  await page.evaluate(() => document.getElementById("main").focus());
  await page.waitForFunction((id) => {
    const row = document.querySelector(`form.sw-mark[data-record-id="${id}"]`)?.closest("ol");
    return row && /done/.test(row.getAttribute("aria-label") || "");
  }, task.id, { timeout: 5000 }).catch(() => check(false, `${mode}: the ticked row reaches Done once focus leaves the list`));
  await settle(page);
  let vt = await record(page);
  if (reduced) check(travels(vt).length === 0, `${mode}: the tick travels: ${JSON.stringify(travels(vt).slice(0, 3))}`);
  else check(travels(vt, "sw-m-").length > 0, `${mode}: the ticked row slides into Done (${vt.length} animations, none travel)`);

  // Move on a board.
  const proj = await post("/api/project", { title: `Motion move ${mode} ${stamp}`, status: "active" });
  const board = await post("/api/block", { component: "collection", props: { type: "project", as: "board", label: `Board ${mode}` } });
  made.push(`/api/project/${proj.id}`, `/api/block/${board.id}`);
  await page.goto(base + "/");
  const button = page.locator(`button:has-text("Move Motion move ${mode} ${stamp}")`);
  const select = page.locator("form.sw-move", { has: button }).locator("select");
  await select.focus();
  await select.selectOption("done");
  await page.keyboard.press("Tab");
  await record(page);
  await page.keyboard.press("Enter");
  await page.waitForURL(/#/);
  await page.waitForLoadState("load");
  await settle(page);
  check(await page.evaluate(() => document.activeElement.id === location.hash.slice(1) && document.activeElement.textContent.startsWith("Move")), `${mode}: focus comes back to the Move button`);
  vt = await record(page);
  if (reduced) check(travels(vt).length === 0, `${mode}: the move travels: ${JSON.stringify(travels(vt).slice(0, 3))}`);
  else check(travels(vt, "sw-m-").length > 0, `${mode}: the card glides to its column (${vt.length} animations, none travel)`);

  // Apply a list's filters.
  const list = await post("/api/block", { component: "collection", props: { type: "task", label: `Filtered ${mode}`, controls: true } });
  made.push(`/api/block/${list.id}`);
  await page.goto(base + "/");
  const apply = page.locator(`[data-block-id="${list.id}"] button:has-text("Apply")`);
  await apply.focus();
  await record(page);
  await page.keyboard.press("Enter");
  await page.waitForLoadState("load");
  await settle(page);
  vt = await record(page);
  if (reduced) check(travels(vt).length === 0, `${mode}: applying filters travels: ${JSON.stringify(travels(vt).slice(0, 3))}`);

  // A calendar's next month.
  const cal = await post("/api/block", { component: "calendar", props: { type: "task", detail: "page" } });
  made.push(`/api/block/${cal.id}`);
  await page.goto(`${base}/canvas/${cal.id}`);
  const next = page.locator('.sw-calendar__months a[rel="next"]').first();
  await next.focus();
  await record(page);
  await page.keyboard.press("Enter");
  await page.waitForLoadState("load");
  await settle(page);
  vt = await record(page);
  if (reduced) {
    check(travels(vt).length === 0, `${mode}: the next month travels: ${JSON.stringify(travels(vt).slice(0, 3))}`);
    check(vt.length > 0, `${mode}: the next month still cross-fades in`);
  } else check(travels(vt, "sw-days-").length > 0, `${mode}: the next month slides in (${vt.length} animations, none travel)`);

  await browser.close();
  for (const path of made) await fetch(base + path, { method: "DELETE" });
}

await run(false);
await run(true);
console.log(failures ? `motion: ${failures} failure(s)` : "motion: ok");
process.exitCode = failures ? 1 : 0;
