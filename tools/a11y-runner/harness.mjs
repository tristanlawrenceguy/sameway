// The harness for behave.mjs: the page scripts as the server joins them,
// a case page around them, and a made-up origin that answers as the case
// says.
import { readFileSync, readdirSync, existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { join, resolve } from "node:path";

const design = join(resolve(fileURLToPath(new URL(".", import.meta.url)), "..", ".."), "design");
const baseJS = readdirSync(join(design, "base")).filter((f) => f.endsWith(".js")).sort()
  .map((f) => `\n/* base: ${f} */\n` + readFileSync(join(design, "base", f), "utf8")).join("");
const componentJS = readdirSync(join(design, "components")).sort()
  .filter((c) => existsSync(join(design, "components", c, "enhance.js")))
  .map((c) => `/* component: ${c} */\n` + readFileSync(join(design, "components", c, "enhance.js"), "utf8")).join("\n");
export const bundle = `/* base */\n${baseJS}\n${componentJS}`;

export const ORIGIN = "http://sw.test";

// page wraps a body the way the layout does, with the script deferred.
export function page(body, attrs = "") {
  return `<!doctype html><html lang="en"${attrs}><head><meta charset="utf-8"><title>Case</title>
<script src="/design/sameway.js" defer></script></head><body>${body}</body></html>`;
}

// open loads a case: html is the page, or a function of how many times the
// page has been asked for (1 on load, 2 on its first refresh...); routes
// answer other paths, each (route, url, request) => fulfilled. Anything
// else is answered 204, so an EventSource that is not wanted closes.
export async function open(browser, html, routes = {}) {
  const tab = await browser.newPage();
  let asked = 0;
  const errors = [];
  tab.on("pageerror", (e) => errors.push(String(e)));
  await tab.route(ORIGIN + "/**", (route) => {
    const url = new URL(route.request().url());
    if (url.pathname === "/design/sameway.js") return route.fulfill({ contentType: "text/javascript", body: bundle });
    if (routes[url.pathname]) return routes[url.pathname](route, url, route.request());
    if (url.pathname === "/" && route.request().method() === "GET") {
      asked++;
      return route.fulfill({ contentType: "text/html", body: typeof html === "function" ? html(asked) : html });
    }
    return route.fulfill({ status: 204, body: "" });
  });
  await tab.goto(ORIGIN + "/");
  await tab.waitForFunction(() => window.sw && document.readyState === "complete");
  tab.errors = errors;
  return tab;
}

// refreshed resolves once the page has followed itself (sw:refresh).
export function refreshed(tab, ms = 5000) {
  return tab.evaluate((ms) => new Promise((r) => {
    document.addEventListener("sw:refresh", () => r(true), { once: true });
    setTimeout(() => r(false), ms);
  }), ms);
}

