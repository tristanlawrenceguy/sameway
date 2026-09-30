// Measured, not guessed.
//
// How tall a block is was guessed on the server from what it is: a
// calendar tall, a clock short. Only the browser that draws the page
// knows, at the window's width, the zoom, the text size and spacing the
// person chose (ui.text, ui.spacing), the fonts, the phone. So on a page
// of blocks this measures each one where it is drawn: its size, where its
// row starts, anything inside it that scrolls or is cut off and how much
// is hidden, and how far it runs past the side of the screen. It sends
// numbers and ids, never what a block says, to POST /canvas/measure, and
// the assistant reads them in Layout now (chat/layout_measured.go).
//
// Only a page the server marked is measured (data-measure): one the person
// looking may change. A published page, or someone who may only look, is
// never marked, and the route refuses them as well. Nothing is sent when
// the browser is asked to save data, or when a program drives it (a test,
// the look tool): its window is not a person's.
//
// What scrolls by design says so with data-scrolls (the conversation's
// history, a wide table's own region), as fields and code do by being
// what they are; it is counted apart and never flagged.
//
// window.swMeasure.read() gives what would be sent, so the accessibility
// runner checks pages with the same measuring the person's browser does.
(function () {
  "use strict";
  // .sw-table-wrap is the one box whose job is to scroll a wide table
  // sideways, wherever a table is drawn (tables, calendars, prose).
  var DESIGN = "[data-scrolls], .sw-table-wrap, textarea, select, input, pre";
  var MOST = 50000, BLOCKS = 200, NODES = 3000;
  var QUIET = 1500, GAP = 5000;

  function n(v) { return Math.max(0, Math.min(MOST, Math.round(v || 0))); }
  function scrolls(v) { return v === "auto" || v === "scroll"; }
  function clips(v) { return v === "hidden" || v === "clip"; }

  // shown says an element is drawn: not in a folded disclosure, not
  // hidden, and more than the pixel or two a box hidden on purpose for
  // screen readers is.
  function shown(el) {
    if (el.checkVisibility && !el.checkVisibility()) return false;
    return el.getClientRects().length > 0 && el.clientWidth > 2 && el.clientHeight > 2;
  }

  // inside is what a block hides: the worst box that scrolls when it was
  // not meant to (its content and its height), what is hidden sideways,
  // what is cut off where nothing scrolls, what scrolls by design, and
  // how far anything drawn runs past the right edge of the screen.
  function inside(block, vw) {
    var out = { sh: 0, sb: 0, sx: 0, cut: 0, d: 0, past: 0 };
    var els = [block].concat(Array.prototype.slice.call(block.querySelectorAll("*"), 0, NODES));
    for (var i = 0; i < els.length; i++) {
      var el = els[i];
      var right = el.getBoundingClientRect().right;
      if (right > vw + 1 && shown(el)) out.past = Math.max(out.past, right - vw);
      var y = el.scrollHeight - el.clientHeight, x = el.scrollWidth - el.clientWidth;
      if (y <= 1 && x <= 1) continue;
      if (!shown(el)) continue;
      var cs = getComputedStyle(el);
      var designed = el.closest(DESIGN) || cs.textOverflow === "ellipsis" || (cs.webkitLineClamp && cs.webkitLineClamp !== "none");
      var sy = scrolls(cs.overflowY) && y > 1, sx = scrolls(cs.overflowX) && x > 1;
      if (designed) {
        if (sy || sx) out.d = Math.max(out.d, sy ? y : 0, sx ? x : 0);
        continue;
      }
      if (sy && y > out.sh - out.sb) { out.sh = el.scrollHeight; out.sb = el.clientHeight; }
      if (sx) out.sx = Math.max(out.sx, x);
      if (clips(cs.overflowY) && y > 1) out.cut = Math.max(out.cut, y);
      if (clips(cs.overflowX) && x > 1) out.cut = Math.max(out.cut, x);
    }
    return out;
  }

  // paneBox is the height of the pane a block sits in, when that pane
  // scrolls on its own (the side panes on a wide screen); 0 otherwise.
  function paneBox(block) {
    var pane = block.closest(".sw-pane");
    return pane && scrolls(getComputedStyle(pane).overflowY) ? pane.clientHeight : 0;
  }

  function read() {
    var page = document.querySelector("[data-measure]");
    if (!page) return null;
    var root = document.documentElement;
    var vw = root.clientWidth || window.innerWidth;
    var grid = document.querySelector(".sw-page .sw-canvas");
    var cols = grid ? getComputedStyle(grid).gridTemplateColumns.split(" ").filter(Boolean).length : 1;
    var blocks = [];
    var els = document.querySelectorAll("[data-measure-v]");
    for (var i = 0; i < els.length && blocks.length < BLOCKS; i++) {
      var el = els[i];
      if (!el.getClientRects().length) continue; // in a folded pane: not drawn
      var r = el.getBoundingClientRect();
      var list = el.closest(".sw-canvas") || el.parentElement;
      var hid = inside(el, vw);
      blocks.push({
        id: el.getAttribute("data-block-id") || el.getAttribute("data-measure-block"), v: el.getAttribute("data-measure-v"),
        w: n(r.width), h: n(r.height), top: n(r.top - list.getBoundingClientRect().top),
        sh: n(hid.sh), sb: n(hid.sb), sx: n(hid.sx), cut: n(hid.cut), d: n(hid.d),
        past: n(hid.past), pane: n(paneBox(el))
      });
    }
    return {
      view: page.getAttribute("data-measure"), canvas: page.getAttribute("data-measure-canvas") || "",
      vw: n(vw), dpr: Math.round((window.devicePixelRatio || 1) * 100) / 100,
      font: parseFloat(getComputedStyle(root).fontSize) || 0, cols: Math.min(12, cols), blocks: blocks
    };
  }

  var last = "", timer = null, sentAt = 0;
  function send(again) {
    var body = read();
    if (!body || !body.blocks.length) return false;
    var json = JSON.stringify(body);
    if (json === last && !again) return false;
    last = json;
    sentAt = Date.now();
    var url = "/canvas/measure";
    if (navigator.sendBeacon && navigator.sendBeacon(url, new Blob([json], { type: "application/json" }))) return true;
    if (window.fetch) {
      fetch(url, { method: "POST", body: json, headers: { "Content-Type": "application/json" }, credentials: "same-origin", keepalive: true })["catch"](function () {});
    }
    return true;
  }

  // soon sends once the page has been still a moment, and never more
  // often than every few seconds, however much it moves.
  function soon() {
    clearTimeout(timer);
    timer = setTimeout(function () {
      var wait = GAP - (Date.now() - sentAt);
      if (wait > 0) timer = setTimeout(send, wait);
      else send();
    }, QUIET);
  }

  window.swMeasure = { read: read, send: function () { return send(true); } };

  var driven = navigator.webdriver || /HeadlessChrome/.test(navigator.userAgent || "");
  var saving = navigator.connection && navigator.connection.saveData;
  if (driven || saving || !window.ResizeObserver || !document.querySelector("[data-measure]")) return;

  // Only a real change of size counts: a few pixels either way is a
  // clock's digits or a font settling, not a new layout.
  var sizes = new WeakMap();
  var watcher = new ResizeObserver(function (entries) {
    var moved = false;
    entries.forEach(function (e) {
      var w = e.contentRect.width, h = e.contentRect.height, was = sizes.get(e.target);
      if (!was || Math.abs(was[0] - w) >= 4 || Math.abs(was[1] - h) >= 4) {
        sizes.set(e.target, [w, h]);
        moved = true;
      }
    });
    if (moved) soon();
  });
  function watch() {
    watcher.observe(document.documentElement);
    document.querySelectorAll("[data-measure-v]").forEach(function (el) { watcher.observe(el); });
  }
  watch();
  document.addEventListener("sw:refresh", watch);
  if (document.fonts && document.fonts.ready) document.fonts.ready.then(soon);
  window.addEventListener("load", soon);
})();
