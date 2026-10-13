// Page scripts that keep a page left open true: its days follow the clock
// and the zone is said once when the reader's differs (36-days.js), the
// connect card follows what the person does about it (37-connect-wait.js),
// and Install is offered where the browser can (38-install.js).
import { open, page, refreshed } from "./harness.mjs";

// hideable lets a case say whether the tab is looked at: window.__vis.
const hideable = () => {
  Object.defineProperty(document, "visibilityState", { get: () => window.__vis || "visible" });
  Object.defineProperty(document, "hidden", { get: () => window.__vis === "hidden" });
};
const here = -new Date().getTimezoneOffset();
const shown = (time, today = "") => page(`<main id="main"><p>Call at <time datetime="${time}">soon</time></p></main>`, ` data-zone="${here}"${today ? ` data-today="${today}"` : ""}`);

export const cases = {
  "days: a time shown passing brings the page up to date, and a hidden tab waits": async (browser, check) => {
    const soon = () => new Date(Date.now() + 1500).toISOString();
    const tab = await open(browser, shown(soon()), {}, hideable);
    check(await refreshed(tab, 5000), "the page followed itself once the time had passed");
    await tab.close();
    const hidden = await open(browser, shown(soon()), {}, hideable);
    await hidden.evaluate(() => { window.__vis = "hidden"; });
    check(!(await refreshed(hidden, 4000)), "a hidden tab does not");
    await hidden.close();
  },

  "days: a day that turned while the tab was hidden is caught up when it is seen": async (browser, check) => {
    const tab = await open(browser, shown("2000-01-01T09:00:00Z", "2000-01-01"), {}, hideable);
    await tab.evaluate(() => { window.__vis = "hidden"; document.dispatchEvent(new Event("visibilitychange")); });
    check(!(await refreshed(tab, 1500)), "nothing while hidden");
    const done = refreshed(tab, 3000);
    await tab.evaluate(() => { window.__vis = "visible"; document.dispatchEvent(new Event("visibilitychange")); });
    check(await done, "seen again on another day, the page follows itself");
    await tab.close();
  },

  "install: offered where the browser can install, and pressed, the browser asks": async (browser, check) => {
    const help = page(`<main id="main"><button type="button" data-install hidden>Install Sameway</button></main>`);
    const tab = await open(browser, help);
    check(await tab.isHidden("[data-install]"), "hidden where the browser has not offered");
    await tab.evaluate(() => {
      const e = new Event("beforeinstallprompt", { cancelable: true });
      e.prompt = () => { window.prompted = (window.prompted || 0) + 1; };
      e.userChoice = Promise.resolve({ outcome: "accepted" });
      window.dispatchEvent(e);
    });
    check(await tab.isVisible("[data-install]"), "shown once the browser offers");
    await tab.click("text=Install Sameway");
    check(await tab.evaluate(() => window.prompted) === 1, "pressed, the browser is asked to install");
    await tab.waitForFunction(() => document.querySelector("[data-install]").hidden, null, { timeout: 2000 }).catch(() => {});
    check(await tab.isHidden("[data-install]"), "answered, the button goes");
    await tab.close();
  },

  "days: a reader in another zone is told once, near the top, how far apart they are, and again after a refresh": async (browser, check) => {
    const zone = -new Date().getTimezoneOffset() + 90;
    const body = `<div class="sw-shell"><main id="main"><div class="sw-page-head"><h1>Today</h1></div><p>Call at <time datetime="2026-10-09">9 Oct</time></p></main></div>`;
    const tab = await open(browser, page(body, ` data-zone="${zone}"`));
    const notes = () => tab.locator("#zone-note").allTextContents();
    let said = await notes();
    check(said.length === 1 && /Times here are this workspace's, UTC.*: 1 hour 30 minutes ahead of yours\./.test(said[0]), `said once, in words: ${said.join(" | ")}`);
    const done = refreshed(tab);
    await tab.evaluate(() => sw.refresh(0));
    await done;
    said = await notes();
    check(said.length === 1, `said once after the page followed itself: ${said.length}`);
    await tab.close();
  },

  "connect: the card follows what the server says, but not while a key is being typed into it": async (browser, check) => {
    const card = (typed) => page(`<div class="sw-shell"><main id="main"><div class="sw-connect" data-wait="Looking for Ollama"><label for="k">Key</label><input id="k" name="key" value="${typed}"></div></main></div>`);
    let asked = 0;
    const tab = await open(browser, card(""), { "/model/wait": (route) => { asked++; return route.fulfill({ contentType: "text/plain", body: "Ollama found" }); } });
    check(await refreshed(tab, 7000), "the page followed the change");
    check(asked >= 1, "it asked the server");
    await tab.close();

    const typing = await open(browser, card("sk-test"), { "/model/wait": (route) => route.fulfill({ contentType: "text/plain", body: "Ollama found" }) });
    check(!(await refreshed(typing, 6000)), "a key half typed is not emptied by a fresh card");
    await typing.close();
  },
};
