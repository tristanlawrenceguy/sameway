// The core (design/base/00-sw.js): sw.arm arms a node once, whether it was
// there at load or arrives later, and the bus carries its events in order.
import { open, page, refreshed } from "./harness.mjs";

export const cases = {
  "arm: a node there at load and one that arrives are each armed once": async (browser, check) => {
    const tab = await open(browser, page(`<main id="main"><p class="t">one</p></main>`));
    const counts = await tab.evaluate(() => {
      sw.arm(".t", (el) => { el.dataset.n = String(Number(el.dataset.n || 0) + 1); });
      const late = document.createElement("p");
      late.className = "t";
      document.getElementById("main").appendChild(late);
      const before = late.dataset.n || "0";
      sw.emit("refresh");
      sw.emit("refresh");
      return { before, all: [...document.querySelectorAll(".t")].map((el) => el.dataset.n) };
    });
    check(counts.before === "0", `a late node is armed when the page brings it in, not before: ${counts.before}`);
    check(counts.all.join() === "1,1", `each node armed once over two refreshes: ${counts.all.join()}`);
    check(!tab.errors.length, `no errors: ${tab.errors.join("; ")}`);
    await tab.close();
  },

  "arm: a slip in one arm does not stop the next node or the next arm": async (browser, check) => {
    const tab = await open(browser, page(`<p class="t">a</p><p class="t">b</p>`));
    const got = await tab.evaluate(() => {
      const seen = [];
      console.error = () => {};
      sw.arm(".t", (el) => { seen.push(el.textContent); if (el.textContent === "a") throw new Error("slip"); });
      sw.arm(".t", (el) => seen.push("second " + el.textContent));
      return seen.join();
    });
    check(got === "a,b,second a,second b", `every node and every arm still run: ${got}`);
    await tab.close();
  },

  "bus: on, emit in order with its detail, off, and heard on the document": async (browser, check) => {
    const tab = await open(browser, page(`<p>bus</p>`));
    const got = await tab.evaluate(() => {
      const heard = [];
      console.error = () => {};
      const a = (d) => heard.push("a" + d.n);
      sw.on("thing", a);
      sw.on("thing", () => { throw new Error("slip"); });
      sw.on("thing", (d) => heard.push("b" + d.n));
      document.addEventListener("sw:thing", (e) => heard.push("doc" + e.detail.n));
      sw.emit("thing", { n: 1 });
      sw.off("thing", a);
      sw.emit("thing", { n: 2 });
      return heard.join();
    });
    check(got === "a1,b1,doc1,b2,doc2", `handlers in order, a slip skipped, off honoured, the document told: ${got}`);
    await tab.close();
  },

  "compose: Enter sends, Shift+Enter and an empty box do not, and a composer a refresh brings sends once": async (browser, check) => {
    const composer = `<div class="sw-shell"><main id="main"><form class="sw-compose" action="/chat" method="post" data-live="off">
<label for="m">Message</label><textarea id="m" name="message"></textarea><p id="m-hint">Ask anything.</p>
<button type="submit">Send</button></form></main></div>`;
    const tab = await open(browser, page(composer));
    await tab.evaluate(() => {
      window.sent = 0;
      document.addEventListener("submit", (e) => { e.preventDefault(); window.sent++; }, true);
    });
    const sent = () => tab.evaluate(() => window.sent);
    await tab.focus("#m");
    await tab.keyboard.press("Enter");
    check(await sent() === 0, "an empty box sends nothing");
    await tab.keyboard.type("hello");
    await tab.keyboard.press("Shift+Enter");
    check(await sent() === 0, "Shift+Enter starts a line, it does not send");
    await tab.keyboard.press("Enter");
    check(await sent() === 1, "Enter sends");
    check(/Enter sends/.test(await tab.textContent("#m-hint")), "the hint says Enter sends");
    const done = refreshed(tab);
    await tab.evaluate(() => sw.refresh(0));
    check(await done, "the page followed itself");
    const again = refreshed(tab);
    await tab.evaluate(() => sw.refresh(0));
    await again;
    await tab.focus("#m");
    await tab.keyboard.type("again");
    await tab.keyboard.press("Enter");
    check(await sent() === 2, `the composer the refresh brought sends once: ${await sent()}`);
    check((await tab.textContent("#m-hint")).split("Enter sends").length === 2, "its hint says it once");
    await tab.close();
  },
};
