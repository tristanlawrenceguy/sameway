// A board's Move, in a real browser against a running server (SAMEWAY_URL):
// a card moved by keyboard alone comes back to its own Move button, now in
// its new column, which hears where it went as its description, once; the
// message and its Undo stay at the top without taking focus. Makes three
// projects and a board, so it runs after pages.mjs and site.mjs.
import { chromium } from "playwright";

const base = (process.env.SAMEWAY_URL || "http://127.0.0.1:8080").replace(/\/$/, "");
let failures = 0;
const check = (ok, msg) => { if (!ok) { failures++; console.log(`FAIL move: ${msg}`); } };
const post = (path, body) => fetch(base + path, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }).then((r) => r.json());

const title = "Move test " + Date.now().toString(36);
const rec = await post("/api/project", { title, status: "active" });
const block = await post("/api/block", { component: "collection", props: { type: "project", as: "board", label: "Board to move on" } });

const browser = await chromium.launch();
const page = await (await browser.newContext()).newPage();
await page.goto(base + "/");
const button = page.locator(`button:has-text("Move ${title}")`);
const form = page.locator("form.sw-move", { has: button });
const select = form.locator("select");
await select.focus();
await select.selectOption("done");
await page.keyboard.press("Tab");
check(await button.evaluate((b) => b === document.activeElement), "Tab from the choices reaches Move");
await page.keyboard.press("Enter");
await page.waitForURL(/#/);
await page.waitForLoadState("load");
await page.waitForTimeout(300); // past the outcome's own focus, 50ms after load

const now = await page.evaluate(() => {
  const el = document.activeElement;
  const said = el.getAttribute("aria-describedby");
  return {
    text: el.textContent, id: el.id, hash: location.hash.slice(1),
    column: (el.closest(".sw-collection__column")?.querySelector(".sw-collection__column-title")?.textContent || "").trim(),
    said: said && document.getElementById(said)?.textContent,
    outcome: !!document.getElementById("outcome"),
  };
});
check(now.text === `Move ${title}` && now.id === now.hash, `focus returns to the Move button pressed, not ${JSON.stringify(now)}`);
check(now.column.startsWith("Done"), `the button is in the Done column now, not ${now.column}`);
check(now.said === `${title} moved from Active to Done.`, `the button hears where it went, not ${now.said}`);
check(now.outcome, "the message stays at the top, with its Undo");
await page.keyboard.press("Tab");
check(await page.evaluate(() => !document.querySelector("[aria-describedby=outcome-words]")), "said once: the description goes when focus leaves");

// A move to where it already is says so and changes nothing.
await page.locator("form.sw-move", { has: button }).locator("button").press("Enter");
await page.waitForLoadState("load");
await page.waitForTimeout(300);
check(await page.evaluate(() => document.activeElement.getAttribute("aria-describedby") && document.getElementById(document.activeElement.getAttribute("aria-describedby")).textContent) === "Status is already Done. Nothing changed.", "a move to where it is says nothing changed");

await browser.close();
await fetch(`${base}/api/project/${rec.id}`, { method: "DELETE" });
await fetch(`${base}/api/block/${block.id}`, { method: "DELETE" });
console.log(failures ? `move: ${failures} failure(s)` : "move: ok");
process.exitCode = failures ? 1 : 0;
