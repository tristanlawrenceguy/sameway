// The page following itself (design/base/19-refresh.js): it fetches itself
// and moves what changed into place, keeping the person where they were.
// An unchanged block stays the same node, and so does the chat; focus on
// a control with no id is found again in the fresh page; a field keeps its
// caret; the scroll is put back at once, not glided to; a status keeps its
// node and takes the server's words; and the names a transition gives
// items (31-travel.js) are taken off before blocks are compared and are
// gone after.
import { open, page, refreshed } from "./harness.mjs";

const item = (id, words) => `<li class="sw-collection__item"><a href="/t/task/${id}">${words}</a></li>`;
const blocks = (n) => `<ol class="sw-canvas">
<li class="sw-block" data-block-id="kept"><ul class="sw-collection">${item("t1", "Sow beans")}${item("t2", "Water")}</ul></li>
<li class="sw-block" data-block-id="changed"><p>Version ${n}</p><div style="height: ${n * 100}px"></div><button type="button">Open</button></li>
<li class="sw-block" data-block-id="chat" data-block-component="chat"><p>Chat as it was served ${n}</p></li></ol>`;
const shell = (n) => page(`<style>html { scroll-behavior: smooth; } .tall { height: 3000px; }</style>
<div class="sw-shell"><main id="main" tabindex="-1">
<p class="sw-status" data-component="status" data-state="idle" role="status" id="say"><span class="sw-status__text">Words ${n}</span></p>
<label for="q">Search</label><input id="q" value="beans and peas">
${blocks(n)}<div class="tall"></div></main></div>`);

async function follow(tab) {
  const done = refreshed(tab);
  await tab.evaluate(() => sw.refresh(0));
  return done;
}

export const cases = {
  "refresh: unchanged blocks and the chat stay the same nodes; what changed is the server's": async (browser, check) => {
    const tab = await open(browser, (n) => shell(n));
    await tab.evaluate(() => { for (const b of document.querySelectorAll("[data-block-id]")) b.__was = true; });
    check(await follow(tab), "the page followed itself");
    const kept = await tab.evaluate(() => Object.fromEntries([...document.querySelectorAll("[data-block-id]")].map((b) => [b.getAttribute("data-block-id"), !!b.__was])));
    check(kept.kept && kept.chat && !kept.changed, `kept: ${JSON.stringify(kept)}`);
    check(await tab.textContent("[data-block-id=changed] p") === "Version 2", "the changed block is the server's");
    check(await tab.textContent("[data-block-id=chat] p") === "Chat as it was served 1", "the chat is left as it is");
    check(await tab.textContent("#say .sw-status__text") === "Words 2", "the status takes the server's words");
    check(await tab.evaluate(() => document.querySelectorAll("#say").length === 1 && !!document.getElementById("say")), "in its own node");
    await tab.close();
  },

  "refresh: focus on a control with no id is found again; a field keeps its caret; the scroll is put back at once": async (browser, check) => {
    const tab = await open(browser, (n) => shell(n));
    await tab.focus("[data-block-id=changed] button");
    check(await follow(tab), "followed");
    check(await tab.evaluate(() => document.activeElement.textContent === "Open" && !!document.activeElement.closest("[data-block-id=changed]") && !document.activeElement.__was), "focus is on the fresh Open button");
    await tab.evaluate(() => { const q = document.getElementById("q"); q.focus(); q.setSelectionRange(4, 7); window.scrollTo({ top: 600, behavior: "instant" }); window.__chatTop = document.querySelector("[data-block-id=chat]").getBoundingClientRect().top; });
    const done = refreshed(tab);
    await tab.evaluate(() => sw.refresh(0));
    await done;
    const where = await tab.evaluate(() => ({ id: document.activeElement.id, start: document.activeElement.selectionStart, end: document.activeElement.selectionEnd, moved: Math.round(document.querySelector("[data-block-id=chat]").getBoundingClientRect().top - window.__chatTop) }));
    check(where.id === "q" && where.start === 4 && where.end === 7, `the caret is where it was: ${JSON.stringify(where)}`);
    check(where.moved === 0, `the chat is where it was on the screen the moment the page is followed, though the block above it grew: moved ${where.moved}px`);
    await tab.close();
  },

  "refresh: a transition's names come off before blocks are compared, and are gone after": async (browser, check) => {
    const tab = await open(browser, (n) => shell(n));
    await tab.evaluate(() => { document.querySelector("[data-block-id=kept]").__was = true; });
    for (let i = 0; i < 2; i++) check(await follow(tab), "followed");
    await tab.waitForTimeout(400);
    const after = await tab.evaluate(() => ({
      named: [...document.querySelectorAll("*")].filter((el) => el.style && el.style.getPropertyValue("view-transition-name")).length,
      styled: document.querySelectorAll(".sw-collection__item[style]").length,
      kept: !!document.querySelector("[data-block-id=kept]").__was,
    }));
    check(after.named === 0 && after.styled === 0, `no names left on the page at rest: ${JSON.stringify(after)}`);
    check(after.kept, "a list named for the transition still compares equal and is kept");
    await tab.close();
  },
};
