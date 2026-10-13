// What the page says while things happen out of the person's hands: a
// turn that thinks for long says it is still going (17-live.js), a change
// out of sight is pointed at for eyes only (22-offscreen.js), and the
// page's own measuring sends numbers and ids, never words (29-measure.js).
import { open, page } from "./harness.mjs";

const chat = `<div class="sw-shell"><main id="main" tabindex="-1"><div class="sw-chat__body"><ol class="sw-plain sw-stack sw-chat__log" aria-label="Messages" tabindex="0"></ol>
<p class="sw-status sw-status--idle" data-component="status" data-state="idle" role="status" aria-live="polite" id="chat-status"><span class="sw-status__text"></span></p>
<form method="post" action="/chat" class="sw-compose" data-busy-target="chat-status"><label for="m">Message</label><textarea id="m" name="message"></textarea><button type="submit">Send</button></form></div>
</main></div>`;

export const cases = {
  "live: a model that thinks for long is said to be still going, not stuck": async (browser, check) => {
    // The turn's stream says the message was heard, then nothing more for
    // as long as the case runs: a model thinking.
    const tab = await open(browser, page(chat), {}, () => {
      const real = window.fetch;
      window.fetch = (url, o) => !String(url).endsWith("/chat/stream") ? real(url, o) : Promise.resolve(new Response(new ReadableStream({
        start(c) { c.enqueue(new TextEncoder().encode('event: said\ndata: {"id":"m1"}\n\n')); },
      }), { headers: { "Content-Type": "text/event-stream" } }));
    });
    await tab.fill("#m", "Plan my week");
    await tab.click("text=Send");
    const thinking = () => tab.evaluate(() => (document.querySelector("[data-thinking]") || {}).textContent || "");
    await tab.waitForFunction(() => document.querySelector("[data-thinking]"));
    check(/^Thinking$/.test((await thinking()).trim()), `at first it says Thinking: ${await thinking()}`);
    await tab.waitForFunction(() => /Still thinking/.test(document.querySelector("[data-thinking]").textContent), null, { timeout: 12000 }).catch(() => {});
    check((await thinking()).trim() === "Still thinking. Some models take a minute.", `after a while, that this is how it goes: ${await thinking()}`);
    await tab.close();
  },

  // A turn you can watch without noise (34-turn.js); its steps, held places
  // and landing are checked against a server in motion.mjs.
  "turn: the reply's words grow a few times a second outside any live region, and writing it is said once": async (browser, check) => {
    const tab = await open(browser, page(chat.replace('class="sw-plain sw-stack sw-chat__log"', 'class="sw-plain sw-stack sw-chat__log" style="max-height: 120px; overflow-y: auto"')), {}, () => {
      // The stream: heard, twenty pieces of the reply 10 ms apart, done.
      const real = window.fetch;
      window.fetch = (url, o) => !String(url).endsWith("/chat/stream") ? real(url, o) : Promise.resolve(new Response(new ReadableStream({
        async start(c) {
          const send = (event, data) => c.enqueue(new TextEncoder().encode(`event: ${event}\ndata: ${JSON.stringify(data)}\n\n`));
          send("said", { id: "m1" });
          for (let i = 0; i < 20; i++) { await new Promise((r) => setTimeout(r, 10)); send("delta", { text: `word${i} ` + "and more ".repeat(4) }); }
          await new Promise((r) => setTimeout(r, 300));
          send("done", {});
          c.close();
        },
      }), { headers: { "Content-Type": "text/event-stream" } }));
    });
    await tab.evaluate(() => {
      sw.refresh = () => {};
      window.statuses = [];
      const s = document.getElementById("chat-status");
      new MutationObserver(() => window.statuses.push(s.textContent.trim())).observe(s, { childList: true, characterData: true, subtree: true });
      window.grew = 0;
      new MutationObserver((rs) => { window.grew += rs.filter((r) => r.type === "characterData" && r.target.parentElement && r.target.parentElement.classList.contains("sw-live__text")).length; }).observe(document.body, { characterData: true, subtree: true });
    });
    await tab.fill("#m", "Plan my week");
    await tab.click("text=Send");
    await tab.waitForFunction(() => /word19/.test((document.querySelector(".sw-live__text") || {}).textContent || ""), null, { timeout: 10000 });
    await tab.waitForTimeout(150);
    const seen = await tab.evaluate(() => {
      const words = document.querySelector(".sw-live__text"), log = words.closest(".sw-chat__log");
      return { grew: window.grew, live: !!words.closest("[aria-live], [role=status], [role=log], [role=alert]"), still: document.getElementById("chat-status").getAttribute("data-still"), atEnd: log.scrollHeight - log.scrollTop - log.clientHeight < 2 };
    });
    await tab.waitForTimeout(3000);
    seen.statuses = await tab.evaluate(() => window.statuses);
    check(seen.grew > 0 && seen.grew < 10, `the words land a few at a time, not each piece (${seen.grew} times for 20 pieces)`);
    check(!seen.live, "the words grow outside any live region; they are heard once, whole");
    check(seen.statuses.filter((s) => s === "Writing the reply…").length === 1, `writing the reply is said once: ${JSON.stringify(seen.statuses)}`);
    check(seen.still === "Still writing the reply…", `and a long one says it is still going: ${seen.still}`);
    check(seen.atEnd, "the log keeps up with the words");
    await tab.close();
  },

  "offscreen: a change out of sight is pointed at, for eyes only, and goes once seen": async (browser, check) => {
    const tab = await open(browser, page(`<main id="main"><div style="height: 3000px">Top</div><div id="far" data-block-id="b9" data-actor="assistant" style="height: 200px">Far down</div></main>`));
    await tab.evaluate(() => document.getElementById("far").setAttribute("data-changed", ""));
    await tab.waitForFunction(() => { const s = document.querySelector(".sw-offscreen--down"); return s && !s.hidden; });
    const sign = await tab.evaluate(() => {
      const s = document.querySelector(".sw-offscreen--down");
      return { words: s.textContent, hidden: s.closest("[aria-hidden=true]") === s, focusable: s.tabIndex >= 0 || !!s.querySelector("a, button, [tabindex]"), tag: s.tagName };
    });
    check(sign.words === "↓ Change below", `it says which way: ${sign.words}`);
    check(sign.hidden && !sign.focusable && sign.tag !== "BUTTON", `out of the accessibility tree and the tab order: ${JSON.stringify(sign)}`);
    for (let i = 0; i < 5; i++) {
      await tab.keyboard.press("Tab");
      check(!(await tab.evaluate(() => !!document.activeElement.closest(".sw-offscreen"))), "Tab never lands on it");
    }
    await tab.click(".sw-offscreen--down");
    await tab.waitForFunction(() => document.querySelector(".sw-offscreen--down").hidden, null, { timeout: 5000 }).catch(() => {});
    check(await tab.evaluate(() => document.querySelector(".sw-offscreen--down").hidden), "pressed, it goes there, and the sign goes");
    await tab.close();
  },

  "measure: a page is measured in numbers and ids, never its words, and not when a robot drives it": async (browser, check) => {
    const words = "SECRETWORDS";
    const body = `<main id="main" data-measure="tab" data-measure-canvas="c1"><div class="sw-page"><ol class="sw-canvas">
<li class="sw-block" data-block-id="b1" data-block-label="${words} label" data-measure-v="0123456789abcdef" aria-label="${words} aria"><h3>${words} title</h3><input value="${words} typed"><p>${words} text</p></li></ol></div></main>`;
    const sent = [];
    const routes = { "/canvas/measure": (route, url, req) => { sent.push(req.postData()); return route.fulfill({ status: 204, body: "" }); } };
    const driven = await open(browser, page(body), routes);
    await driven.waitForTimeout(3000);
    check(sent.length === 0, `driven by a robot, nothing is sent of itself: ${sent.length}`);
    await driven.evaluate(() => sw.measure.send());
    await driven.waitForFunction(() => true);
    await driven.waitForTimeout(500);
    check(sent.length === 1, `asked, it sends once: ${sent.length}`);
    const got = JSON.parse(sent[0] || "{}");
    check(!sent.join().includes(words), `no words are sent: ${sent[0]}`);
    const strings = [];
    const walk = (v, k) => { if (typeof v === "string") strings.push(k + "=" + v); else if (v && typeof v === "object") for (const [kk, vv] of Object.entries(v)) walk(vv, kk); };
    walk(got, "");
    check(strings.every((s) => /^(view=tab|canvas=c1|id=b1|v=0123456789abcdef)$/.test(s)), `every field is a number, an id or which page it is: ${strings.join(", ")}`);
    check(got.blocks && got.blocks.length === 1 && typeof got.blocks[0].h === "number", `the block's size is in it: ${sent[0]}`);
    await driven.close();

    // A person's own browser sends of itself, unless it asks to save data.
    // An init script runs in the page, so what it needs is written into it.
    const person = (saveData) => `Object.defineProperty(navigator, "webdriver", { get: () => false });
      Object.defineProperty(navigator, "userAgent", { get: () => "Mozilla/5.0 Chrome/130" });
      Object.defineProperty(navigator, "connection", { get: () => ({ saveData: ${saveData} }) });`;
    for (const saveData of [false, true]) {
      sent.length = 0;
      const tab = await open(browser, page(body), routes, person(saveData));
      await tab.waitForTimeout(4000);
      check(saveData ? sent.length === 0 : sent.length === 1, `${saveData ? "saving data, nothing is sent" : "a person's browser sends once of itself"}: ${sent.length}`);
      await tab.close();
    }
  },
};
