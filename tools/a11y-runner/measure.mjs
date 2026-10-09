// Measured, not guessed: pages of blocks checked with the page's own
// measuring (design/base/29-measure.js, sw.measure), the same the
// person's browser sends, at a desktop and a phone width.
//
// - innerScrollProblems: on the seeded tabs, no block scrolls inside when
//   it was not meant to, is cut off where nothing scrolls, scrolls
//   sideways inside, or runs past the right edge of the screen. What
//   scrolls by design (data-scrolls, a wide table's region) is not counted.
// - scrollFlagProblems: a long list in a box made to scroll, on a phone,
//   is sent the way the person's browser sends it, and the server says so:
//   /api/look under measured, and the layout line with its fix.
import { settle } from "./checks.mjs";

const widths = [[1280, 720, "desktop"], [390, 844, "phone"]];

async function measured(page) {
  await page.evaluate(() => document.fonts && document.fonts.ready);
  await settle(page);
  return page.evaluate(() => ({
    reading: window.sw && sw.measure ? sw.measure.read() : null,
    names: Object.fromEntries([...document.querySelectorAll("[data-block-id]")].map((e) => [e.dataset.blockId, e.dataset.blockLabel || e.dataset.blockComponent])),
  }));
}

export async function innerScrollProblems(page, base, paths) {
  const out = [];
  for (const [width, height, where] of widths) {
    await page.setViewportSize({ width, height });
    for (const path of paths) {
      await page.goto(base + path);
      const { reading, names } = await measured(page);
      if (!reading) { out.push(`${path}: not marked for measuring, though the owner may change it`); continue; }
      for (const b of reading.blocks) {
        const name = `${path} on a ${where}: ${names[b.id] || b.id}`;
        if (b.sb > 0 && b.sh > b.sb + 1) out.push(`${name} scrolls inside (${b.sh}px of content in a ${b.sb}px box)`);
        if (b.cut) out.push(`${name} has ${b.cut}px cut off where nothing scrolls`);
        if (b.sx) out.push(`${name} scrolls sideways inside (${b.sx}px hidden)`);
        if (b.past) out.push(`${name} runs ${b.past}px past the right edge of the screen (1.4.10)`);
      }
    }
  }
  await page.setViewportSize({ width: 1280, height: 720 });
  return out;
}

export async function scrollFlagProblems(page, base, post) {
  const out = [];
  const tab = await (await post("/api/canvas", { name: "Measured" })).json();
  for (let i = 1; i <= 30; i++) await post("/api/note", { title: `Measured note ${i}` });
  const list = await (await post("/api/block", { component: "collection", props: { type: "note", label: "Long list", limit: 30, controls: false }, canvas: tab.id, span: 6 })).json();
  const beside = await (await post("/api/block", { component: "text", props: { content: "Beside it." }, canvas: tab.id, span: 6, position: 1 })).json();
  if (!list.id || !beside.id) return [`could not set up the long list: ${JSON.stringify(list).slice(0, 200)}`];

  await page.setViewportSize({ width: 390, height: 700 });
  await page.goto(`${base}/c/${tab.id}`);
  // A box made to scroll, the way a workspace's own component might make one.
  await page.addStyleTag({ content: `[data-block-id="${list.id}"] .sw-collection__items { max-height: 240px; overflow-y: auto; }` });
  const { reading } = await measured(page);
  const mine = reading && reading.blocks.find((b) => b.id === list.id);
  if (!mine || !(mine.sh > mine.sb && mine.sb > 0)) out.push(`the long list in a 240px box on a phone should measure as scrolling inside, got ${JSON.stringify(mine)}`);
  if (!(await page.evaluate(() => sw.measure.send()))) out.push("the page did not send its measurements");

  let person = [];
  for (let i = 0; i < 30 && !person.some((r) => r.block === list.id); i++) {
    await page.waitForTimeout(100);
    person = ((await (await fetch(`${base}/api/look?path=/c/${tab.id}`)).json()).measured || {}).person || [];
  }
  const kept = person.find((r) => r.block === list.id);
  if (!kept || kept.device !== "phone" || !(kept.content > kept.box)) out.push(`/api/look should give the phone's reading of the long list, got ${JSON.stringify(person).slice(0, 300)}`);

  const res = await fetch(`${base}/api/block/${beside.id}`, { method: "PATCH", headers: { "content-type": "application/json" }, body: JSON.stringify({ props: { content: "Beside it, still." } }) });
  const layout = (await res.json()).layout || "";
  for (const want of ['"Long list" scrolls inside on your phone', "give it span 12", "arrange_canvas"]) {
    if (!layout.includes(want)) out.push(`the layout line should say ${want}: ${layout.slice(0, 400)}`);
  }
  await page.setViewportSize({ width: 1280, height: 720 });
  return out;
}
