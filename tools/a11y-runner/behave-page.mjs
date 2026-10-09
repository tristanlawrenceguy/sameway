// Page scripts that keep a page left open true: the zone said once when the
// reader's differs (36-days.js), and the connect card following what the
// person does about it (37-connect-wait.js).
import { open, page, refreshed } from "./harness.mjs";

export const cases = {
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
