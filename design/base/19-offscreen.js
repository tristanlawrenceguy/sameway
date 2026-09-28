// A change out of sight is pointed at until it is seen.
//
// The glow says what changed and who changed it, but only to someone
// looking at that place. A block that changes above or below what is on
// screen (further down a phone, in a pane scrolled elsewhere) would glow
// for nobody. So while a changed block is out of view, a small sign at
// the edge of the screen says there is a change that way, in the colour
// of whoever made it, and tapping it goes there. Once the block has been
// in view the sign goes, and the block glows again for the one who has
// just arrived at it.
//
// The sign is for eyes only: it is aria-hidden and outside the tab order.
// A screen reader already hears each change in the reply's receipt and
// the activity log, and an agent reads those too; a second, visual-only
// announcement would be noise to both.
(function () {
  "use strict";
  if (!window.IntersectionObserver || !window.MutationObserver) return;
  var still = document.documentElement.getAttribute("data-pace") === "still" ||
    (window.matchMedia && matchMedia("(prefers-reduced-motion: reduce)").matches);

  // What has been seen is kept for the tab, so a reload does not point at
  // a change the person has already looked at.
  var KEY = "sw-seen-changes", seen = {};
  try { seen = JSON.parse(sessionStorage.getItem(KEY) || "{}") || {}; } catch (e) { seen = {}; }
  function remember(k) {
    seen[k] = 1;
    try { sessionStorage.setItem(KEY, JSON.stringify(seen)); } catch (e) { /* storage off */ }
  }
  function keyOf(el) {
    var html = el.outerHTML.replace(/ data-arrival="\d+"/g, ""), h = 0;
    for (var i = 0; i < html.length; i++) h = (h * 31 + html.charCodeAt(i)) | 0;
    return (el.getAttribute("data-block-id") || "") + ":" + h;
  }

  var pending = [], watched = new WeakSet();
  var signs = { up: sign("up"), down: sign("down") };

  function sign(way) {
    var s = document.createElement("div");
    s.className = "sw-offscreen sw-offscreen--" + way;
    s.setAttribute("aria-hidden", "true");
    s.hidden = true;
    s.addEventListener("click", function () { go(way); });
    return s;
  }

  // Seen is half the block in view, or half the screen filled by it.
  var io = new IntersectionObserver(function (entries) {
    entries.forEach(function (e) {
      var el = e.target, i = pending.indexOf(el);
      var enough = e.isIntersecting && (e.intersectionRatio >= 0.5 || e.intersectionRect.height >= 0.5 * (e.rootBounds ? e.rootBounds.height : innerHeight));
      if (enough) {
        io.unobserve(el);
        remember(el._swKey);
        if (i >= 0) { pending.splice(i, 1); glowAgain(el); }
      } else if (i < 0 && !el._swFirstSeen) {
        pending.push(el);
      }
      el._swFirstSeen = true;
    });
    draw();
  }, { threshold: [0, 0.5, 1] });

  // The key is taken once, when the block is first met, so buttons that
  // other scripts add to it later do not make it a different change.
  function watch(el) {
    if (watched.has(el)) return;
    el._swKey = el._swKey || keyOf(el);
    if (seen[el._swKey]) return;
    // A block in a closed disclosure has no place on screen to go to.
    if (!el.getClientRects().length) return;
    watched.add(el);
    io.observe(el);
  }
  function scan(root) {
    if (root.nodeType !== 1) return;
    if (root.hasAttribute("data-changed")) watch(root);
    root.querySelectorAll("[data-changed]").forEach(watch);
  }

  // Which way a block is: above or below the part of its scrolling
  // container (a pane, the chat, or the page) that is on screen.
  function way(el) {
    var r = el.getBoundingClientRect(), top = 0, bottom = innerHeight, p = el.parentElement;
    for (; p && p !== document.body; p = p.parentElement) {
      var o = getComputedStyle(p).overflowY;
      if ((o === "auto" || o === "scroll") && p.scrollHeight > p.clientHeight) {
        var c = p.getBoundingClientRect();
        top = Math.max(top, c.top); bottom = Math.min(bottom, c.bottom);
        break;
      }
    }
    return r.bottom <= top + 1 ? "up" : r.top >= bottom - 1 ? "down" : "";
  }

  function draw() {
    pending = pending.filter(function (el) { return el.isConnected; });
    var groups = { up: [], down: [] };
    pending.forEach(function (el) { var w = way(el); if (w) groups[w].push(el); });
    ["up", "down"].forEach(function (w) {
      var s = signs[w], n = groups[w].length;
      if (!s.isConnected) document.body.appendChild(s);
      s.hidden = n === 0;
      if (!n) return;
      var nearest = nearestOf(groups[w], w);
      s.setAttribute("data-actor", nearest.getAttribute("data-actor") || "assistant");
      // Someone else's change points in their colour.
      if (nearest.getAttribute("data-person")) s.setAttribute("data-person", nearest.getAttribute("data-person"));
      else s.removeAttribute("data-person");
      s.textContent = (w === "up" ? "↑ " : "↓ ") + (n === 1 ? "Change " : n + " changes ") + (w === "up" ? "above" : "below");
    });
  }

  function nearestOf(list, w) {
    return list.reduce(function (a, b) {
      var ra = a.getBoundingClientRect(), rb = b.getBoundingClientRect();
      return w === "up" ? (rb.bottom > ra.bottom ? b : a) : (rb.top < ra.top ? b : a);
    });
  }

  function go(w) {
    var list = pending.filter(function (el) { return el.isConnected && way(el) === w; });
    if (list.length) nearestOf(list, w).scrollIntoView({ block: "center", behavior: still ? "auto" : "smooth" });
  }

  // The glow played while nobody could see it; it plays again on arrival.
  function glowAgain(el) {
    if (still || !el.getAnimations) return;
    el.getAnimations().forEach(function (a) {
      if (a.animationName === "sw-glow") { a.currentTime = 0; a.play(); }
    });
  }

  var queued = false;
  function redraw() {
    if (queued) return;
    queued = true;
    requestAnimationFrame(function () { queued = false; draw(); });
  }
  addEventListener("scroll", redraw, { passive: true, capture: true });
  addEventListener("resize", redraw);

  function init() {
    scan(document.body);
    new MutationObserver(function (records) {
      records.forEach(function (r) {
        if (r.type === "attributes") scan(r.target);
        else r.addedNodes.forEach(scan);
      });
    }).observe(document.body, { childList: true, subtree: true, attributes: true, attributeFilter: ["data-changed"] });
  }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
})();
