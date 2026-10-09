// The page's one connection (design/base/01-connect.js): one EventSource
// whatever listens, the page following a change made elsewhere, opened
// again when the server ends it and left closed when refused, closed in a
// hidden tab unless a ring is listened for, and a turn's stream read whole
// even when one event slips.
import { open, page, refreshed } from "./harness.mjs";

// Counts every EventSource the page makes, before its scripts run.
const counting = () => {
  const Real = window.EventSource;
  window.__sources = [];
  window.EventSource = function (url) { const es = new Real(url); window.__sources.push(String(url)); return es; };
  window.EventSource.prototype = Real.prototype;
};
// hide or show the tab, as the browser would say it.
const seen = (tab, hidden) => tab.evaluate((hidden) => {
  Object.defineProperty(document, "hidden", { configurable: true, get: () => hidden });
  Object.defineProperty(document, "visibilityState", { configurable: true, get: () => (hidden ? "hidden" : "visible") });
  document.dispatchEvent(new Event("visibilitychange"));
}, hidden);
const sse = (body) => ({ status: 200, headers: { "Content-Type": "text/event-stream", "Cache-Control": "no-cache" }, body });
const shell = `<div class="sw-shell"><main id="main"><p>Page</p></main></div>`;

export const cases = {
  "connect: following and ringing share one connection, and a change elsewhere is followed": async (browser, check) => {
    const asked = [];
    const tab = await open(browser, page(shell), {
      "/events": (route, url) => { asked.push(url.search); return route.fulfill(sse(asked.length === 1 ? "retry: 60000\nevent: hello\ndata: {}\n\n" : "retry: 60000\nevent: ring\ndata: {\"id\":\"r1\"}\n\nevent: changed\ndata: {}\n\n")); },
    }, counting);
    const done = refreshed(tab);
    await tab.evaluate(() => { window.rang = []; sw.listen("ring", (d) => window.rang.push(d.id), true); });
    check(await done, "the page followed the change");
    const got = await tab.evaluate(() => ({ sources: window.__sources, rang: window.rang }));
    check(got.rang.join() === "r1", `the ring was heard: ${got.rang.join()}`);
    check(got.sources.length === 2 && got.sources[1].includes("ring=1"), `one connection at a time, opened again to carry rings: ${got.sources.join(" ")}`);
    check(await tab.evaluate(() => window.__sources.length) === 2, "no third connection");
    await tab.close();
  },

  "connect: a connection the server ends is opened again; one refused from the start is not": async (browser, check) => {
    let n = 0;
    const tab = await open(browser, page(shell), {
      "/events": (route) => { n++; return n === 2 ? route.fulfill({ status: 503, body: "" }) : route.fulfill(sse("retry: 50\nevent: hello\ndata: {}\n\n")); },
    }, counting);
    await tab.waitForFunction(() => window.__sources.length >= 2, null, { timeout: 5000 });
    const sources = await tab.evaluate(() => window.__sources.length);
    check(sources === 2 && n >= 3, `after the server refused a reconnection, the page opened its connection again: ${sources} connections, ${n} requests`);
    await tab.close();

    let m = 0;
    const refused = await open(browser, page(shell), { "/events": (route) => { m++; return route.fulfill({ status: 404, body: "" }); } }, counting);
    await refused.waitForTimeout(2500);
    check(m === 1, `a page with no /events asks once: ${m}`);
    await refused.close();
  },

  "connect: a hidden tab lets go unless a ring is listened for, and catches up when seen": async (browser, check) => {
    const tab = await open(browser, page(shell), { "/events": (route) => route.fulfill(sse("retry: 60000\nevent: hello\ndata: {}\n\n")) }, counting);
    await seen(tab, true);
    check(await tab.evaluate(() => window.__sources.length) === 1, "hidden: no new connection");
    const done = refreshed(tab);
    await seen(tab, false);
    check(await done, "seen again, the page catches up");
    check(await tab.evaluate(() => window.__sources.length) === 2, "and connects again");
    await tab.evaluate(() => sw.listen("ring", () => {}, true));
    await seen(tab, true);
    const last = await tab.evaluate(() => window.__sources[window.__sources.length - 1]);
    check(/idle=1/.test(last) && /ring=1/.test(last), `hidden with a ring listened for: kept open, said idle: ${last}`);
    await tab.close();
  },

  "stream: a turn's events are read whole, across pieces, past one that slips": async (browser, check) => {
    const tab = await open(browser, page(shell));
    const got = await tab.evaluate(async () => {
      const parts = ["event: said\ndata: {\"id\":1}\n", "\nevent: delta\ndata: {\"te", "xt\":\"Hi\"}\n\nevent: delta\ndata: {\"text\":\" there\"}\n\nevent: done\ndata: {}\n\n"];
      const body = new ReadableStream({ start(c) { parts.forEach((p) => c.enqueue(new TextEncoder().encode(p))); c.close(); } });
      const heard = [];
      console.error = () => {};
      await sw.stream(new Response(body), (m) => { heard.push(m.event + (m.data.text || "")); if (m.event === "said") throw new Error("slip"); });
      return heard.join(",");
    });
    check(got === "said,deltaHi,delta there,done", `every event, in order: ${got}`);
    await tab.close();
  },
};
