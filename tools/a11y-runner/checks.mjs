// In-page checks shared by the component and live-page suites. Each one
// returns a list of problems as short strings; an empty list is a pass.
import AxeBuilder from "@axe-core/playwright";
import { AA_TAGS } from "./shell.mjs";

// The display settings a person can have, as Playwright media emulation.
// axe runs in light and dark. Forced colours is checked for focus rings
// only: there the person's own theme sets every colour, so contrast is not
// the page's to meet. Motion is reduced while checking, so arrivals and
// fades are finished and axe reads the colours a person ends up seeing.
export const MODES = {
  light: { colorScheme: "light", forcedColors: "none", reducedMotion: "reduce" },
  dark: { colorScheme: "dark", forcedColors: "none", reducedMotion: "reduce" },
  forced: { colorScheme: "light", forcedColors: "active", reducedMotion: "reduce" },
};

export async function setMode(page, mode) {
  await page.emulateMedia(MODES[mode]);
}

// Waits two frames, so a change just made to the page (a new size, a style,
// a focus) has been laid out before anything is measured.
export async function settle(page) {
  await page.evaluate(() => new Promise((done) => requestAnimationFrame(() => requestAnimationFrame(done))));
}

// axe in the page's current mode, at AA unless other tags are given,
// leaving out any rules named.
export async function axeProblems(page, skip = [], tags = AA_TAGS) {
  await settle(page);
  const res = await new AxeBuilder({ page }).withTags(tags).disableRules(skip).analyze();
  return res.violations.map((v) => `axe ${v.id} - ${v.help} (${v.nodes.length} node(s): ${v.nodes.slice(0, 3).map((n) => n.target.join(" ")).join(", ")})`);
}

// WCAG 1.4.10 Reflow: at 320 CSS pixels wide (1280 at 400% zoom) nothing
// makes the page scroll sideways. Content inside its own scroll region,
// such as a wide table, is the exception the criterion allows.
export async function reflowProblems(page) {
  const size = page.viewportSize();
  await page.setViewportSize({ width: 320, height: 256 });
  await settle(page);
  const found = await page.evaluate(() => {
    const doc = document.documentElement;
    if (doc.scrollWidth <= window.innerWidth + 1) return [];
    // Contained: inside a scroll region that itself fits the viewport. A
    // positioned element is only clipped by one that holds its offset parent.
    const scrolls = (el) => {
      const box = /(absolute|fixed)/.test(getComputedStyle(el).position) ? el.offsetParent : el.parentElement;
      for (let p = el.parentElement; p && p !== doc; p = p.parentElement) {
        if (/(auto|scroll|hidden|clip)/.test(getComputedStyle(p).overflowX)) return (p === box || (box && p.contains(box))) && p.getBoundingClientRect().right <= window.innerWidth + 1;
      }
      return false;
    };
    const wide = [...document.body.querySelectorAll("*")].filter((el) => el.getBoundingClientRect().right > window.innerWidth + 1 && !scrolls(el));
    // Report the outermost offenders; their children follow them out.
    const outer = wide.filter((el) => !wide.includes(el.parentElement));
    return [`page is ${doc.scrollWidth}px wide at 320px: ${outer.slice(0, 3).map(describe).join(", ")}`];
    function describe(el) { return el.tagName.toLowerCase() + (el.className && typeof el.className === "string" ? "." + el.className.trim().split(/\s+/).join(".") : ""); }
  });
  await page.setViewportSize(size);
  return found;
}

// WCAG 1.4.12 Text Spacing: with the spacing the criterion names forced
// on, no text is cut off by a box that hides its overflow.
const SPACING = "* { line-height: 1.5 !important; letter-spacing: 0.12em !important; word-spacing: 0.16em !important; } p { margin-bottom: 2em !important; } *, *::before, *::after { transition: none !important; }";

export async function spacingProblems(page) {
  const style = await page.addStyleTag({ content: SPACING });
  await settle(page);
  const found = await page.evaluate(() => {
    const out = [];
    for (const el of document.body.querySelectorAll("*")) {
      const cs = getComputedStyle(el);
      if (!/(hidden|clip)/.test(cs.overflowX + cs.overflowY)) continue;
      if (!el.textContent.trim()) continue;
      // Visually hidden text is for screen readers and is meant to be clipped.
      const box = el.getBoundingClientRect();
      if (box.width <= 1 || box.height <= 1 || cs.clipPath !== "none" || cs.clip !== "auto") continue;
      if (el.scrollHeight > el.clientHeight + 1 || el.scrollWidth > el.clientWidth + 1) {
        out.push(`${el.tagName.toLowerCase()}${el.className && typeof el.className === "string" ? "." + el.className.trim().split(/\s+/).join(".") : ""} clips "${el.textContent.trim().replace(/\s+/g, " ").slice(0, 40)}"`);
      }
    }
    return out;
  });
  await style.evaluate((el) => el.remove());
  return found;
}

// prefers-reduced-motion: with it set, nothing animates or transitions for
// longer than the 0.01ms the base styles allow, and nothing waits to appear.
export async function motionProblems(page) {
  await page.emulateMedia({ reducedMotion: "reduce" });
  await settle(page);
  const found = await page.evaluate(() => {
    const secs = (list) => Math.max(...list.split(",").map((t) => (t.trim().endsWith("ms") ? parseFloat(t) / 1000 : parseFloat(t))));
    const out = [];
    for (const el of document.querySelectorAll("*")) {
      for (const pseudo of [null, "::before", "::after"]) {
        const cs = getComputedStyle(el, pseudo);
        const name = el.tagName.toLowerCase() + (pseudo || "") + (el.className && typeof el.className === "string" ? "." + el.className.trim().split(/\s+/)[0] : "");
        if (cs.animationName !== "none" && (secs(cs.animationDuration) > 0.001 || secs(cs.animationDelay) > 0)) out.push(`${name} animates for ${cs.animationDuration} after ${cs.animationDelay}`);
        if (secs(cs.transitionDuration) > 0.001) out.push(`${name} transitions for ${cs.transitionDuration}`);
      }
    }
    return [...new Set(out)];
  });
  await page.emulateMedia({ reducedMotion: "no-preference" });
  return found;
}

// WCAG 2.4.11 Focus Not Obscured: walk the page with Tab and check each
// focused control is not entirely covered by something else, such as a
// sticky header. Five points across it are sampled; one showing is enough.
// onFocus, when given, is asked about each focused control on the way and
// returns a problem or null.
export async function obscuredFocusProblems(page, onFocus = null, limit = 300) {
  // Reduced motion makes the scroll to a focused control instant rather
  // than smooth, so it is in place when it is measured.
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.evaluate(() => { document.activeElement && document.activeElement.blur(); window.scrollTo(0, 0); });
  const out = [];
  for (let i = 0; i < limit; i++) {
    await page.keyboard.press("Tab");
    // Styles that move a control on focus (a skip link) apply first.
    await settle(page);
    const r = await page.evaluate(() => {
      const el = document.activeElement;
      // Back at the start, or out of the page: the walk is done.
      if (!el || el === document.body || el.__swFocused) return null;
      el.__swFocused = true;
      const b = el.getBoundingClientRect();
      if (b.width === 0 && b.height === 0) return { shown: true };
      // What is painted on top at a point, passing over anything faded to
      // nothing (a quiet control at rest covers nothing a person can see).
      const faded = (n) => { for (; n; n = n.parentElement) if (getComputedStyle(n).opacity === "0") return true; return false; };
      const top = (x, y) => document.elementsFromPoint(x, y).find((n) => !faded(n));
      const pts = [[0.5, 0.5], [0.1, 0.1], [0.9, 0.1], [0.1, 0.9], [0.9, 0.9]];
      const shown = pts.some(([x, y]) => {
        const hit = top(b.left + b.width * x, b.top + b.height * y);
        return hit && (hit === el || el.contains(hit) || hit.contains(el) || (hit.control === el));
      });
      const name = (el.getAttribute("aria-label") || el.textContent || el.getAttribute("name") || "").trim().replace(/\s+/g, " ").slice(0, 40);
      return { shown, name: `${el.tagName.toLowerCase()} "${name}"` };
    });
    if (!r) break;
    if (!r.shown) out.push(`${r.name} is hidden behind something else when focused`);
    const more = onFocus && (await onFocus(page));
    if (more) out.push(more);
  }
  await page.emulateMedia({ reducedMotion: "no-preference" });
  return out;
}
