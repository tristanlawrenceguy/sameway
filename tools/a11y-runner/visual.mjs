// What axe does not measure: focus ring strength, field edges, target
// size, colour-only state, forced colours, text zoom, and error wiring.
// Each check returns a list of problems as short strings.
import { settle } from "./checks.mjs";

// An in-page toolkit the checks share: colours as the browser computes
// them, contrast, and the background actually behind an element.
function toolkit() {
  if (window.__sw) return;
  const parse = (s) => {
    let m = s && s.match(/rgba?\(([^)]+)\)/);
    if (m) { const p = m[1].split(/[\s,/]+/).filter(Boolean).map(Number); return [p[0], p[1], p[2], p.length > 3 ? p[3] : 1]; }
    m = s && s.match(/color\(srgb ([^)]+)\)/);
    if (m) { const p = m[1].split(/[\s/]+/).filter(Boolean).map(Number); return [p[0] * 255, p[1] * 255, p[2] * 255, p.length > 3 ? p[3] : 1]; }
    return null;
  };
  const lum = (c) => { const f = (v) => { v /= 255; return v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4; }; return 0.2126 * f(c[0]) + 0.7152 * f(c[1]) + 0.0722 * f(c[2]); };
  const ratio = (a, b) => { const x = lum(a), y = lum(b); return (Math.max(x, y) + 0.05) / (Math.min(x, y) + 0.05); };
  const over = (top, under) => [0, 1, 2].map((i) => top[i] * top[3] + under[i] * (1 - top[3])).concat(1);
  // Every background from the root down, laid over each other.
  const behind = (el) => {
    const chain = [];
    for (let n = el; n; n = n.parentElement) chain.unshift(n);
    let c = [255, 255, 255, 1];
    for (const n of chain) { const b = parse(getComputedStyle(n).backgroundColor); if (b && b[3] > 0) c = over(b, c); }
    return c;
  };
  const hidden = (el) => {
    for (let n = el; n; n = n.parentElement) {
      const cs = getComputedStyle(n);
      if (cs.clipPath === "inset(50%)" || cs.display === "none" || cs.visibility === "hidden" || cs.opacity === "0") return true;
    }
    return false;
  };
  // The words a sighted person sees, leaving out text kept for screen readers.
  const seen = (el) => {
    let t = "";
    const walk = (n) => { for (const c of n.childNodes) { if (c.nodeType === 3) t += c.textContent; else if (c.nodeType === 1 && !hidden(c)) walk(c); } };
    walk(el);
    return t.replace(/\s+/g, " ").trim();
  };
  const name = (el) => el.tagName.toLowerCase() + (typeof el.className === "string" && el.className.trim() ? "." + el.className.trim().split(/\s+/).slice(0, 2).join(".") : "") + (seen(el) ? ` "${seen(el).slice(0, 30)}"` : "");
  window.__sw = { parse, ratio, over, behind, hidden, seen, name };
}

const ready = (page) => page.evaluate(toolkit);

// WCAG 2.4.13 Focus Appearance (AAA): the focused control's ring is at
// least 2px thick and stands 3:1 against what it is drawn over.
export async function focusAppearance(page) {
  await ready(page);
  return page.evaluate(() => {
    const el = document.activeElement;
    // Past the last control focus leaves the page, though activeElement stays.
    if (!el || el === document.body || !el.matches(":focus")) return null;
    const cs = getComputedStyle(el);
    const width = parseFloat(cs.outlineWidth);
    if (cs.outlineStyle === "none" || width < 2) return `${__sw.name(el)} focus ring is ${cs.outlineStyle === "none" ? "none" : width + "px"}, under 2px (2.4.13)`;
    const ring = __sw.parse(cs.outlineColor);
    // A ring drawn outside the control lies over what is around it.
    const under = parseFloat(cs.outlineOffset) >= 0 ? __sw.behind(el.parentElement) : __sw.behind(el);
    const r = ring ? __sw.ratio(__sw.over(ring, under), under) : 0;
    return r < 3 ? `${__sw.name(el)} focus ring is ${r.toFixed(2)}:1 against what is behind it, under 3:1 (2.4.13)` : null;
  });
}

// Field edges (1.4.11), 44px targets (2.5.5) and colour-only state (1.4.1),
// over everything on the page at once. waived names components whose
// a11y-waivers.json gives a reason 44px targets cannot be met there.
export async function visualProblems(page, waived = []) {
  await ready(page);
  return page.evaluate((waived) => {
    const out = [];
    const visible = (el) => { const b = el.getBoundingClientRect(); return b.width > 1 && b.height > 1 && !__sw.hidden(el); };

    // A field's edge, or its fill, shows where it is: 3:1 against its surface.
    for (const el of document.querySelectorAll("input:not([type=hidden]):not([type=checkbox]):not([type=radio]):not([type=file]):not([type=submit]), select, textarea")) {
      if (!visible(el)) continue;
      const cs = getComputedStyle(el);
      const around = __sw.behind(el.parentElement);
      const edge = __sw.parse(cs.borderTopColor);
      const edgeRatio = edge && cs.borderTopStyle !== "none" && parseFloat(cs.borderTopWidth) > 0 ? __sw.ratio(__sw.over(edge, around), around) : 0;
      const best = Math.max(edgeRatio, __sw.ratio(__sw.behind(el), around));
      if (best < 3) out.push(`${__sw.name(el)} edge is ${best.toFixed(2)}:1 against its surface, under 3:1 (1.4.11)`);
    }

    // Every target at least 44 by 44. What can be pressed counts, not only
    // the element's box: a positioned ::before or ::after widens the reach
    // (a fill link's fills its row), and a field's label presses it too. A
    // text link is left to the inline exception: it keeps to its line so
    // links on neighbouring lines do not overlap.
    const reach = (el) => {
      const r = el.getBoundingClientRect();
      let [l, t, rt, b] = [r.left, r.top, r.right, r.bottom];
      for (const pseudo of ["::before", "::after"]) {
        const cs = getComputedStyle(el, pseudo);
        if (cs.content === "none" || cs.position !== "absolute" || cs.pointerEvents === "none") continue;
        const holder = getComputedStyle(el).position === "static" ? el.offsetParent : el;
        if (!holder) continue;
        const h = holder.getBoundingClientRect();
        const px = (v) => (v === "auto" ? null : parseFloat(v));
        const [il, it, ir, ib] = [px(cs.left), px(cs.top), px(cs.right), px(cs.bottom)];
        if ([il, it, ir, ib].some((v) => v === null)) continue;
        l = Math.min(l, h.left + il); t = Math.min(t, h.top + it); rt = Math.max(rt, h.right - ir); b = Math.max(b, h.bottom - ib);
      }
      for (const lab of el.labels || []) { const lb = lab.getBoundingClientRect(); l = Math.min(l, lb.left); t = Math.min(t, lb.top); rt = Math.max(rt, lb.right); b = Math.max(b, lb.bottom); }
      return { width: rt - l, height: b - t };
    };
    const excused = (el) => { const c = el.closest("[data-component]"); return c && waived.includes(c.dataset.component); };
    for (const el of document.querySelectorAll("a[href], button, summary, select, input:not([type=hidden])")) {
      if (!visible(el) || el.closest(".shell") || excused(el)) continue;
      const textLink = el.tagName === "A" && getComputedStyle(el).display === "inline" && !/sw-link--(fill|button)/.test(el.className);
      if (textLink) continue;
      const box = reach(el);
      if (box.width < 43.5 || box.height < 43.5) out.push(`${__sw.name(el)} can be pressed over ${Math.round(box.width)}x${Math.round(box.height)}, under 44x44 (2.5.5)`);
    }

    // A control shows what it does, and what it shows is in its name so a
    // person can say it to speech control (2.5.3). An icon counts as showing.
    // A glyph in an aria-hidden span is an icon, not words to say.
    const words = (el) => {
      let t = "";
      const walk = (n) => { for (const c of n.childNodes) { if (c.nodeType === 3) t += c.textContent; else if (c.nodeType === 1 && c.getAttribute("aria-hidden") !== "true" && !__sw.hidden(c)) walk(c); } };
      walk(el);
      return t.replace(/\s+/g, " ").trim();
    };
    for (const el of document.querySelectorAll("a[href], button, summary")) {
      if (!visible(el) || el.closest(".shell")) continue;
      const shown = words(el);
      const drawn = el.querySelector("img, svg, [aria-hidden=true]") || ["::before", "::after"].some((p) => { const c = getComputedStyle(el, p).content; return c !== "none" && c !== '""' && c !== "normal"; });
      const label = el.getAttribute("aria-label");
      if (!shown && !drawn) out.push(`${__sw.name(el)}${label ? ` "${label}"` : ""} shows nothing: its name is only for screen readers (2.5.3)`);
      else if (label && shown && !label.toLowerCase().includes(shown.toLowerCase())) out.push(`${__sw.name(el)} shows "${shown}" but is named "${label}" (2.5.3)`);
    }

    // Two elements alike but for a state must differ in more than colour,
    // or in the words they show.
    const colourish = (p) => /color|fill|stroke|shadow|background|^--|^caret/.test(p);
    for (const attr of ["data-met", "data-state", "data-aim", "aria-current", "data-changed", "data-kind", "data-tone"]) {
      const groups = new Map();
      for (const el of document.querySelectorAll(`[${attr}]`)) {
        if (!visible(el)) continue;
        const key = el.tagName.toLowerCase() + "." + String(el.className).trim().split(/\s+/)[0];
        if (!groups.has(key)) groups.set(key, new Map());
        if (!groups.get(key).has(el.getAttribute(attr))) groups.get(key).set(el.getAttribute(attr), el);
      }
      for (const [key, byValue] of groups) {
        const els = [...byValue.values()];
        for (let i = 1; i < els.length; i++) {
          const diff = [];
          for (const pseudo of [null, "::before", "::after"]) {
            const a = getComputedStyle(els[0], pseudo), b = getComputedStyle(els[i], pseudo);
            for (const prop of a) if (a.getPropertyValue(prop) !== b.getPropertyValue(prop)) diff.push(prop);
          }
          // A filled mark against a hollow one differs in shape, not colour.
          const filled = (el) => (__sw.parse(getComputedStyle(el).backgroundColor) || [0, 0, 0, 0])[3] > 0;
          if (filled(els[0]) !== filled(els[i])) continue;
          if (diff.length && diff.every(colourish) && __sw.seen(els[0]) === __sw.seen(els[i])) {
            out.push(`${key} ${attr}="${els[0].getAttribute(attr)}" and "${els[i].getAttribute(attr)}" differ only in colour (1.4.1)`);
          }
        }
      }
    }
    return [...new Set(out)];
  }, waived);
}

// In forced colours the person's theme repaints every background, so a mark
// drawn only as a patch of background colour (a bar's fill, a dot) vanishes
// unless it keeps its colour or has an edge.
export async function forcedColourProblems(page) {
  await ready(page);
  const marks = await page.evaluate(() => {
    let n = 0;
    for (const el of document.body.querySelectorAll("*")) {
      const b = el.getBoundingClientRect();
      if (b.width < 2 || b.height < 2 || __sw.hidden(el) || __sw.seen(el)) continue;
      // Decoration hidden from assistive technology carries no meaning to lose.
      if (el.closest("[aria-hidden=true]") || el.querySelector("img, svg, input, select, textarea, button, a")) continue;
      const own = __sw.parse(getComputedStyle(el).backgroundColor);
      if (!own || own[3] === 0 || __sw.ratio(__sw.behind(el), __sw.behind(el.parentElement)) < 1.5) continue;
      el.dataset.swMark = n++;
    }
    return n;
  });
  if (!marks) return [];
  await page.emulateMedia({ forcedColors: "active" });
  await settle(page);
  const lost = await page.evaluate(() => {
    const out = [];
    for (const el of document.querySelectorAll("[data-sw-mark]")) {
      const cs = getComputedStyle(el);
      const edged = ["Top", "Right", "Bottom", "Left"].some((s) => cs[`border${s}Style`] !== "none" && parseFloat(cs[`border${s}Width`]) > 0) || (cs.outlineStyle !== "none" && parseFloat(cs.outlineWidth) > 0);
      if (!edged && __sw.ratio(__sw.behind(el), __sw.behind(el.parentElement)) < 1.5) out.push(`${__sw.name(el)} is drawn only in background colour and vanishes in forced colours`);
      delete el.dataset.swMark;
    }
    return out;
  });
  await page.emulateMedia({ forcedColors: "none" });
  return [...new Set(lost)];
}

// WCAG 1.4.4 Resize Text: at twice the root text size, which every rem in
// the tokens follows, nothing is cut off and no text stays small.
export async function textZoomProblems(page) {
  // Each element's text size now, to see which ones grow.
  await page.evaluate(() => { for (const el of document.body.querySelectorAll("*")) el.__swSize = parseFloat(getComputedStyle(el).fontSize); });
  // Transitions off, so sizes land at once rather than a frame or two later.
  const style = await page.addStyleTag({ content: "html { font-size: 200% !important; } *, *::before, *::after { transition: none !important; }" });
  await settle(page);
  const found = await page.evaluate(() => {
    const out = [];
    const label = (el) => `${el.tagName.toLowerCase()}.${String(el.className).trim().split(/\s+/)[0]}`;
    for (const el of document.body.querySelectorAll("*")) {
      const cs = getComputedStyle(el);
      const box = el.getBoundingClientRect();
      if (box.width <= 1 || box.height <= 1 || cs.clipPath !== "none" || cs.clip !== "auto") continue;
      if (/(hidden|clip)/.test(cs.overflowX + cs.overflowY) && el.textContent.trim() && (el.scrollHeight > el.clientHeight + 1 || el.scrollWidth > el.clientWidth + 1)) {
        out.push(`${label(el)} clips "${el.textContent.trim().replace(/\s+/g, " ").slice(0, 40)}" at 200% text (1.4.4)`);
      }
      // Text set in px does not grow with the person's text size. A chart's
      // labels are drawn to its scale; its Numbers table carries the same
      // words as text that does grow.
      if (el.closest("svg")) continue;
      const own = [...el.childNodes].some((n) => n.nodeType === 3 && n.textContent.trim());
      if (own && el.__swSize && parseFloat(cs.fontSize) < el.__swSize * 1.5) out.push(`${label(el)} text stays ${cs.fontSize} at 200% (1.4.4)`);
    }
    return [...new Set(out)];
  });
  await style.evaluate((el) => el.remove());
  return found;
}

// An error is tied to its field: aria-invalid points at a message that is
// there and says something (3.3.1).
export async function errorWiringProblems(page) {
  return page.evaluate(() => {
    const out = [];
    for (const el of document.querySelectorAll('[aria-invalid="true"]')) {
      const ids = `${el.getAttribute("aria-errormessage") || ""} ${el.getAttribute("aria-describedby") || ""}`.trim().split(/\s+/).filter(Boolean);
      const text = ids.map((id) => document.getElementById(id)?.textContent.trim() || "").join(" ").trim();
      if (!text) out.push(`${el.tagName.toLowerCase()}#${el.id} is marked invalid but no message is tied to it (3.3.1)`);
    }
    return out;
  });
}
