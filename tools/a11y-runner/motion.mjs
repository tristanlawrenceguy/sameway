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
// comes back (Move). And a press answers at once (27-press.css, 13-mark.js):
// a tick strikes its row through before the server answers, and one the
// server refuses goes back and is said once; a pressed row is ringed in
// place, and a pressed button gives a little, except under reduced motion.
// Makes its own records and blocks, and deletes them.
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

  // A tick answers at once and, refused, goes back and says why, once.
  const refused = await post("/api/task", { title: `Motion refused ${mode} ${stamp}` });
  made.push(`/api/task/${refused.id}`);
  let release;
  const held = new Promise((r) => { release = r; });
  await page.route(`**/t/task/${refused.id}/props`, async (route) => {
    await held;
    await route.fulfill({ status: 200, contentType: "text/html", body: '<div id="outcome" class="sw-outcome" data-outcome="failed"><div class="sw-alert" role="alert"><p class="sw-alert__message">Not saved: the test refused it.</p></div></div>' });
  });
  await page.goto(base + "/t/task");
  const rbox = page.locator(`form.sw-mark[data-record-id="${refused.id}"] input[type=checkbox]`);
  await rbox.focus();
  await page.keyboard.press("Space");
  await settle(page, 100);
  check(await rbox.evaluate((b) => b.checked && b.closest(".sw-row").classList.contains("sw-row--done")), `${mode}: the row is struck through on the press, before the server answers`);
  release();
  await page.waitForFunction(() => (document.getElementById("sw-mark-failed")?.textContent || "").includes("refused"), null, { timeout: 5000 })
    .catch(() => check(false, `${mode}: a refused tick says why`));
  check(await rbox.evaluate((b) => !b.checked && !b.closest(".sw-row").classList.contains("sw-row--done") && b === document.activeElement), `${mode}: a refused tick goes back, with focus still on its box`);
  check(await page.evaluate(() => !!document.getElementById("outcome") && !document.querySelector("#outcome [role=alert], #outcome[role=alert]")), `${mode}: the refusal is shown, and said once, by its region, not again by the copy on the page`);
  await page.unroute(`**/t/task/${refused.id}/props`);

  // A press answers in the same frame and moves nothing around it.
  const link = page.locator(".sw-row__link").first();
  const before = await link.evaluate((a) => { const r = a.closest(".sw-row"); return { bg: getComputedStyle(r).backgroundColor, box: JSON.stringify(r.getBoundingClientRect()) }; });
  const lb = await link.boundingBox();
  await page.mouse.move(lb.x + lb.width / 2, lb.y + lb.height / 2);
  await page.mouse.down();
  const pressed = await link.evaluate((a) => { const r = a.closest(".sw-row"); return { bg: getComputedStyle(r).backgroundColor, ring: getComputedStyle(r).boxShadow, box: JSON.stringify(r.getBoundingClientRect()) }; });
  check(pressed.ring !== "none", `${mode}: a pressed row answers with a ring`);
  check(pressed.box === before.box, `${mode}: a pressed row keeps its place and size`);
  await page.mouse.move(0, 0);
  await page.mouse.up();
  const press = page.locator("main button.sw-pressable:visible").first();
  if (await press.count()) {
    const bb = await press.boundingBox();
    await page.mouse.move(bb.x + bb.width / 2, bb.y + bb.height / 2);
    await page.mouse.down();
    const t = await press.evaluate((b) => getComputedStyle(b).transform);
    check(reduced ? t === "none" : t !== "none", `${mode}: a pressed button ${reduced ? "does not change size" : "gives a little"} (${t})`);
    await page.mouse.move(0, 0);
    await page.mouse.up();
  }

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
