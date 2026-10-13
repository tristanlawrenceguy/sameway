// What the stylesheet does as a person uses the page, read from the page
// as drawn: what a pointer reveals and Escape puts away, a press, a lift,
// a block's tone, an arrival, the header on a phone and a wide screen,
// and nothing pulsing for ever; under reduced motion and forced colours
// too, as Playwright emulates them.
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { open, page } from "./harness.mjs";

const example = (component, name) => readFileSync(fileURLToPath(new URL(`../../design/components/${component}/examples/${name}.html`, import.meta.url)), "utf8");
const settle = (tab) => tab.waitForTimeout(400); // past the longest transition, motion-base

export const cases = {
  "quiet: Escape puts away the controls the pointer revealed, until the pointer comes back": async (browser, check) => {
    const tab = await open(browser, page(`<main id="main"><ol class="sw-canvas"><li class="sw-block sw-reveal" data-block-id="b1"><div class="sw-bar sw-quiet"><button type="button">Remove</button></div><p>Words</p></li></ol><p>Elsewhere</p></main>`));
    const shown = async () => { await settle(tab); return tab.evaluate(() => getComputedStyle(document.querySelector(".sw-bar")).opacity); };
    await tab.hover("text=Words");
    check(await shown() === "1", `the pointer on the block shows its bar: ${await shown()}`);
    await tab.keyboard.press("Escape");
    check(await shown() === "0", `Escape puts it away: ${await shown()}`);
    await tab.hover("text=Elsewhere");
    await tab.hover("text=Words");
    check(await shown() === "1", `the pointer coming back shows it again: ${await shown()}`);
    await tab.close();
  },

  "quiet: a block's bar, shown by the pointer, lies over none of the block's controls": async (browser, check) => {
    const tab = await open(browser, page(`<main id="main"><ol class="sw-canvas"><li class="sw-block sw-reveal" data-block-id="b1"><div class="sw-bar sw-quiet"><button type="button">Expand</button><button type="button">Remove</button></div><form style="display: flex; justify-content: space-between"><label for="q">Search</label><input id="q"><button type="submit">Search</button></form></li></ol></main>`));
    for (const width of [1280, 800, 390]) {
      await tab.setViewportSize({ width, height: 700 });
      const at = await tab.locator("label[for=q]").boundingBox();
      await tab.mouse.move(at.x + 2, at.y + 2);
      const over = await tab.evaluate(() => {
        const bar = document.querySelector(".sw-bar").getBoundingClientRect();
        return [...document.querySelectorAll("form label, form input, form button")].filter((el) => { const r = el.getBoundingClientRect(); return r.left < bar.right && r.right > bar.left && r.top < bar.bottom && r.bottom > bar.top; }).map((el) => el.textContent || el.id);
      });
      check(over.length === 0, `${width}px: the bar lies over ${over.join(", ")}`);
    }
    await tab.close();
  },

  "chat: the log scrolls smoothly to where it is sent, inside itself": async (browser, check) => {
    const tab = await open(browser, page(`<main id="main"><section class="sw-chat" data-component="chat"><div class="sw-chat__body"><ol class="sw-plain sw-chat__log" tabindex="0" aria-label="Messages"><li>One</li></ol></div></section></main>`));
    check(await tab.evaluate(() => getComputedStyle(document.querySelector(".sw-chat__log")).scrollBehavior) === "smooth", "Skip to latest message glides there in the log");
    await tab.close();
  },

  "empty: a link in an empty state goes where it says": async (browser, check) => {
    const tab = await open(browser, page(`<main id="main"><div class="sw-empty" data-component="empty"><p class="sw-empty__message">No notes yet. <a href="/chat">Ask the assistant</a> to add one.</p></div></main>`), {
      "/chat": (route) => route.fulfill({ contentType: "text/html", body: page("<main id=\"main\"><h1>Chat</h1></main>") }),
    });
    await tab.click("text=Ask the assistant");
    await tab.waitForURL("**/chat", { timeout: 5000 }).catch(() => {});
    check(new URL(tab.url()).pathname === "/chat", `the press goes to the chat: ${tab.url()}`);
    await tab.close();
  },

  "tone: a block's tone is an edge as well as a tint, and forced colours keep the edge": async (browser, check) => {
    const tab = await open(browser, page(`<main id="main"><ol class="sw-canvas"><li class="sw-block" data-block-id="b1" data-tone="danger"><p class="sw-visually-hidden">Important.</p><h3>Rent is due</h3></li><li class="sw-block" data-block-id="b2"><h3>Plain</h3></li></ol></main>`));
    const edges = () => tab.evaluate(() => [...document.querySelectorAll(".sw-block")].map((b) => { const s = getComputedStyle(b); return s.borderInlineStartWidth + " " + s.borderInlineStartStyle; }));
    for (const forcedColors of ["none", "active"]) {
      await tab.emulateMedia({ forcedColors });
      const [toned, plain] = await edges();
      // Under forced colours every block is edged, the tint being gone.
      check(toned === "4px solid" && (forcedColors === "active" || toned !== plain), `forced colours ${forcedColors}: the toned block has its edge (${toned} against ${plain})`);
    }
    await tab.close();
  },

  "arrival: a block arriving is in the accessibility tree from the start": async (browser, check) => {
    const tab = await open(browser, page(`<main id="main"><ol class="sw-canvas"></ol></main>`));
    await tab.evaluate(() => document.querySelector(".sw-canvas").insertAdjacentHTML("beforeend", `<li class="sw-block" data-block-id="b1" data-changed="added" data-arrival="1"><h3>Just arrived</h3></li>`));
    const early = await tab.evaluate(() => { const h = document.querySelector("h3"); return { visibility: getComputedStyle(h).visibility, running: h.getAnimations().length }; });
    check(early.running > 0, `the arrival is under way: ${JSON.stringify(early)}`);
    check(early.visibility === "visible", `while it fades in, it is there to be read: ${JSON.stringify(early)}`);
    check(await tab.getByRole("heading", { name: "Just arrived" }).count() === 1, "a screen reader finds it at once");
    await tab.close();
  },

  "card: a card that can be pressed lifts under the pointer and moves nothing around it; not under reduced motion; edged in forced colours": async (browser, check) => {
    const card = example("card", "linked").replace('data-component="card"', 'data-component="card" data-press');
    const tab = await open(browser, page(`<main id="main" style="width: 30rem">${card}<p id="after">Below the card</p></main>`));
    const read = () => tab.evaluate(() => { const c = document.querySelector(".sw-card"); const s = getComputedStyle(c); return { transform: s.transform, shadow: s.boxShadow, border: s.borderColor, after: document.getElementById("after").getBoundingClientRect().top }; });
    for (const reducedMotion of ["no-preference", "reduce"]) {
      await tab.emulateMedia({ reducedMotion, forcedColors: "none" });
      await tab.mouse.move(0, 0);
      await settle(tab);
      const rest = await read();
      await tab.hover(".sw-card__title a");
      await settle(tab);
      const lifted = await read();
      check(lifted.shadow !== rest.shadow, `${reducedMotion}: it lifts, with a shadow (${rest.shadow} to ${lifted.shadow})`);
      check(reducedMotion === "reduce" ? lifted.transform === "none" : lifted.transform !== "none", `${reducedMotion}: ${reducedMotion === "reduce" ? "it does not rise" : "it rises a little"} (${lifted.transform})`);
      check(lifted.after === rest.after, `${reducedMotion}: what is below stays where it was`);
    }
    // Dark has a lift of its own, not light's carried over.
    const lift = async (colorScheme) => {
      await tab.emulateMedia({ reducedMotion: "no-preference", colorScheme });
      await tab.mouse.move(0, 0);
      await tab.hover(".sw-card__title a");
      await settle(tab);
      return tab.evaluate(() => { const s = getComputedStyle(document.querySelector(".sw-card")); return s.backgroundColor + " " + s.borderColor; });
    };
    const light = await lift("light"), dark = await lift("dark");
    check(light !== dark, `a lifted card in dark is dark's own (${light} against ${dark})`);
    await tab.emulateMedia({ reducedMotion: "no-preference", forcedColors: "active", colorScheme: "light" });
    await tab.mouse.move(0, 0);
    await settle(tab);
    const rest = await read();
    await tab.hover(".sw-card__title a");
    await settle(tab);
    check((await read()).border !== rest.border, "under forced colours the edge says it is lifted");
    await tab.close();
  },

  "press: a pressed row is ringed in place, under forced colours in the system's colour": async (browser, check) => {
    const tab = await open(browser, page(`<main id="main"><ul class="sw-rows"><li class="sw-row"><a class="sw-row__link" href="#x">Water the plants</a></li></ul></main>`));
    for (const forcedColors of ["none", "active"]) {
      await tab.emulateMedia({ forcedColors });
      const link = await tab.locator(".sw-row__link").boundingBox();
      const before = await tab.evaluate(() => { const s = getComputedStyle(document.querySelector(".sw-row")); return s.boxShadow + "|" + s.outlineStyle; });
      await tab.mouse.move(link.x + 5, link.y + 5);
      await tab.mouse.down();
      const pressed = await tab.evaluate(() => { const s = getComputedStyle(document.querySelector(".sw-row")); return s.boxShadow + "|" + s.outlineStyle; });
      check(pressed !== before, `forced colours ${forcedColors}: the press shows at once (${before} to ${pressed})`);
      await tab.mouse.up();
    }
    await tab.close();
  },

  "mark: the tick is drawn on the box when checked, and forced colours draw the box and its fill": async (browser, check) => {
    const tab = await open(browser, page(`<main id="main">${example("mark", "default")}${example("mark", "checked")}</main>`));
    const boxes = () => tab.evaluate(() => [...document.querySelectorAll(".sw-mark__input")].map((b) => { const s = getComputedStyle(b), t = getComputedStyle(b, "::before"); return { checked: b.checked, appearance: s.appearance, edge: s.borderTopWidth + " " + s.borderTopStyle, fill: s.backgroundColor, tick: t.clipPath }; }));
    for (const forcedColors of ["none", "active"]) {
      await tab.emulateMedia({ forcedColors });
      await settle(tab);
      const [off, on] = await boxes();
      check(!off.checked && on.checked && on.appearance === "none", `the box is the page's own: ${JSON.stringify(on)}`);
      check(on.edge === "2px solid" && off.edge === "2px solid", `forced colours ${forcedColors}: each box has an edge (${off.edge}, ${on.edge})`);
      check(on.fill !== off.fill, `forced colours ${forcedColors}: checked is filled (${off.fill} against ${on.fill})`);
      check(/inset\(0(px)?\)/.test(on.tick) && on.tick !== off.tick, `forced colours ${forcedColors}: the tick shows when checked (${off.tick} to ${on.tick})`);
    }
    await tab.close();
  },

  "shell: on a phone the header scrolls away with the page; on a wide screen the lists stay beside it, over none of it": async (browser, check) => {
    const shell = page(`<header class="sw-header"><div class="sw-header__inner"><a class="sw-header__brand" href="/">Sameway</a><nav class="sw-nav" aria-label="Main"><ul><li><a class="sw-link" href="/t/note">Notes</a></li></ul></nav></div></header>
<div class="sw-shell"><main id="main" class="sw-main" tabindex="-1"><h1>Notes</h1><button type="button" id="first">Edit block</button><div style="height: 3000px"></div></main></div>`);
    const tab = await open(browser, shell);
    const read = () => tab.evaluate(() => {
      const h = document.querySelector(".sw-header").getBoundingClientRect(), b = document.getElementById("first").getBoundingClientRect(), brand = document.querySelector(".sw-header__brand").getBoundingClientRect();
      return { headerBottom: h.bottom, headerRight: h.right, button: b, onButton: document.elementFromPoint(b.x + b.width / 2, b.y + b.height / 2) === document.getElementById("first"), brandInView: brand.bottom > 0 };
    });
    await tab.setViewportSize({ width: 390, height: 800 });
    const top = await read();
    check(top.onButton, "on a phone the first control is there to press");
    await tab.evaluate(() => scrollTo(0, 600));
    check(!(await read()).brandInView, "and the header scrolls away, leaving the screen to the page");
    await tab.setViewportSize({ width: 1280, height: 800 });
    await tab.evaluate(() => scrollTo(0, 0));
    const wide = await read();
    check(wide.onButton && wide.button.x >= wide.headerRight, `on a wide screen the lists sit beside the page: ${JSON.stringify(wide)}`);
    await tab.evaluate(() => scrollTo(0, 600));
    check((await read()).brandInView, "and stay in view as it scrolls");
    await tab.close();
  },

  // Travel (30-travel.css): a view transition the browser draws between
  // where a thing was and where it is now, of the classes 31-travel.js
  // gives it. Here one row moves down, and one month turns.
  "travel: a move is short and travels; under reduced motion or the still pace it cross-fades in place": async (browser, check) => {
    const tab = await open(browser, page(`<main id="main"><div id="row" style="view-transition-name: row; view-transition-class: sw-vt-move">Row</div><div id="month" style="view-transition-name: month; view-transition-class: sw-vt-days">October</div></main>`));
    const run = () => tab.evaluate(async () => {
      const t = document.startViewTransition(() => { const r = document.getElementById("row"); r.style.marginTop = r.style.marginTop ? "" : "200px"; });
      await t.ready;
      const out = document.getAnimations().filter((a) => a.effect.pseudoElement).map((a) => ({
        on: a.effect.pseudoElement, name: a.animationName, ms: a.effect.getComputedTiming().duration,
        props: [...new Set(a.effect.getKeyframes().flatMap((k) => Object.keys(k).filter((p) => !["offset", "computedOffset", "easing", "composite"].includes(p))))],
      }));
      await t.finished;
      return out;
    });
    const fades = (all) => all.every((a) => a.props.every((p) => p === "opacity")) && !all.some((a) => /group/.test(a.on));
    for (const pace of ["calm", "quick"]) {
      await tab.evaluate((p) => { document.documentElement.dataset.pace = p; }, pace);
      const all = await run();
      const group = all.find((a) => a.on === "::view-transition-group(row)");
      check(group && group.ms > 0 && group.ms <= (pace === "quick" ? 120 : 220), `${pace}: the row travels, in ${pace === "quick" ? "motion-fast" : "motion-base"} at most (${JSON.stringify(group)})`);
      const days = all.filter((a) => /sw-days/.test(a.name));
      check(days.length === 2 && days.every((a) => a.props.every((p) => p === "opacity" || p === "transform")), `${pace}: a month turns by transform and opacity only (${JSON.stringify(days)})`);
    }
    await tab.evaluate(() => { document.documentElement.dataset.pace = "still"; });
    const still = await run();
    check(fades(still), `still pace: nothing travels, things cross-fade (${JSON.stringify(still)})`);
    await tab.evaluate(() => { document.documentElement.dataset.pace = "calm"; });
    await tab.emulateMedia({ reducedMotion: "reduce" });
    const reduced = await run();
    check(fades(reduced), `reduced motion: nothing travels, things cross-fade (${JSON.stringify(reduced)})`);
    await tab.close();
  },

  // The skeleton held in a block's place during a turn (35-skeleton.css);
  // that it is busy, hidden from screen readers and lands is motion.mjs's.
  "skeleton: the bars shimmer for under five seconds, not at all at the still pace, and show under forced colours": async (browser, check) => {
    const tab = await open(browser, page(`<main id="main"><div class="sw-skeleton" aria-hidden="true"><span></span><span></span><span></span></div></main>`));
    const read = () => tab.evaluate(() => ({ runs: document.getAnimations().map((a) => a.effect.getComputedTiming().endTime), bar: getComputedStyle(document.querySelector(".sw-skeleton > span")).backgroundColor + " " + getComputedStyle(document.querySelector(".sw-skeleton > span")).backgroundImage.slice(0, 20), page: getComputedStyle(document.body).backgroundColor }));
    const calm = await read();
    check(calm.runs.length === 3 && calm.runs.every((ms) => ms < 5000), `each bar shimmers, for under five seconds: ${calm.runs}`);
    await tab.evaluate(() => { document.documentElement.dataset.pace = "still"; });
    check((await read()).runs.length === 0, "at the still pace nothing shimmers");
    await tab.emulateMedia({ forcedColors: "active" });
    const forced = await read();
    check(forced.runs.length === 0 && /^rgb/.test(forced.bar) && !forced.bar.startsWith(forced.page) && !/rgba\([^)]*, 0\)/.test(forced.bar), `under forced colours the bars keep a colour of their own: ${JSON.stringify(forced)}`);
    await tab.close();
  },

  "pulse: nothing pulses on its own for more than five seconds": async (browser, check) => {
    const tab = await open(browser, page(`<main id="main">${example("status", "working")}${example("clock", "ringing")}<ol class="sw-plain sw-live__steps"><li class="sw-live__step" data-state="running"><span class="sw-live__dot" aria-hidden="true"></span><span>Thinking</span></li></ol></main>`));
    const runs = await tab.evaluate(() => document.getAnimations().map((a) => { const t = a.effect.getComputedTiming(); return { name: a.animationName, on: a.effect.target.className, total: t.endTime }; }));
    for (const dot of ["sw-status__dot", "sw-clock__bell", "sw-live__dot"]) check(runs.some((r) => r.on.includes(dot)), `${dot} pulses at all: ${JSON.stringify(runs)}`);
    for (const r of runs) check(r.total < 5000, `${r.on} ${r.name} runs ${r.total} ms; it must stop before five seconds`);
    await tab.close();
  },
};
