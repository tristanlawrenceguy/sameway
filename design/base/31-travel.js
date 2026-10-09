// What a person does moves where it goes.
//
// A ticked task slides into Done when the person moves on, a card moved
// on a board glides to its new column, a list filtered or sorted moves its
// items into their new order, and a calendar's next month comes in from
// the side that was asked for. Each is a view transition: the browser
// draws the page as it was and the page as it is, and moves what the two
// have in common between them. This script only says what is the same
// thing on both: each item of a list is named by its block and its record
// for the moment of the transition, and loses the name after, so the page
// at rest is the page the server sent. How things move is CSS
// (30-travel.css): a glide well under a quarter of a second, a quicker one
// at the quick pace, and a cross-fade with no travel under reduced motion
// or the still pace. A browser without view transitions shows the
// finished page, as it always did; nothing here moves focus.
(function () {
  "use strict";
  var root = document.documentElement;
  // An item of a list, by the shapes lists take: a collection's rows,
  // cards and board cards, a type's listing and its group headings, an
  // agenda, and the activity log. Tables are left out: a row of one cannot
  // be lifted out of its table to move.
  var ITEMS = ".sw-collection__item, .sw-collection__cards > li, .sw-collection__column-items > li, .sw-rows > .sw-row, h2.sw-group, .sw-calendar__agenda > li, li > .sw-event";
  // A calendar's days, which slide as a whole to the next month or day.
  var DAYS = 'table[data-component="calendar"], .sw-calendar__list, .sw-calendar__allday';
  // More named things than this costs more to draw than it shows.
  var MOST = 200;
  var named = [];

  function ident(s) { return String(s).replace(/[^A-Za-z0-9_-]/g, "_"); }

  // key is what an item is, the same on the page before and after: its
  // id, its record, or its link, inside the block it is in.
  function key(el) {
    var rec = el.querySelector("[data-record-id]");
    var link = el.querySelector('a[href^="/t/"]');
    var what = el.id ||
      (rec && rec.getAttribute("data-record-type") + "-" + rec.getAttribute("data-record-id")) ||
      (el.classList.contains("sw-group") && (el.firstChild || {}).textContent) ||
      (link && link.getAttribute("href"));
    if (!what) return null;
    var block = el.closest("[data-block-id]");
    return "sw-m-" + ident((block ? block.getAttribute("data-block-id") : "page") + "-" + what.trim());
  }

  function set(el, name, kind) {
    named.push({ el: el, bare: !el.hasAttribute("style") });
    el.style.setProperty("view-transition-name", name);
    el.style.setProperty("view-transition-class", kind);
  }

  // clear takes every name off again, and the style attribute with it
  // when there was none, so a block compares equal to the server's.
  function clear() {
    named.forEach(function (n) {
      n.el.style.removeProperty("view-transition-name");
      n.el.style.removeProperty("view-transition-class");
      if (n.bare && !n.el.getAttribute("style")) n.el.removeAttribute("style");
    });
    named = [];
  }

  function name() {
    clear();
    var travel = root.getAttribute("data-travel"), seen = {};
    var all = document.querySelectorAll(ITEMS);
    for (var i = 0; i < all.length && named.length < MOST; i++) {
      var el = all[i].classList.contains("sw-event") ? all[i].parentElement : all[i];
      // Going to another month, the days slide as one; the events in them
      // are not the same things in another month.
      if (travel && el.closest(".sw-calendar")) continue;
      if (el.style.getPropertyValue("view-transition-name")) continue;
      var k = key(el);
      if (!k || seen[k]) continue;
      seen[k] = true;
      set(el, k, "sw-vt-move");
    }
    if (!travel) return;
    document.querySelectorAll(".sw-calendar").forEach(function (cal, c) {
      var parts = [].slice.call(cal.querySelectorAll(DAYS));
      var hour = cal.querySelector(".sw-calendar__hour");
      if (hour) parts.push(hour.parentElement);
      parts.forEach(function (p, i) { set(p, "sw-days-" + c + "-" + i, "sw-vt-days"); });
    });
  }

  // Which way a person went: the page left knows, when where it is going
  // is one of a calendar's previous or next, and keeps it for the page it
  // brings, for a few seconds at most.
  var KEEP = "sw-travel";
  function towards(to) {
    var links = document.querySelectorAll('.sw-calendar__months a[rel="prev"], .sw-calendar__months a[rel="next"]');
    for (var i = 0; i < links.length; i++) if (links[i].href === to) return links[i].getAttribute("rel");
    return null;
  }
  function went(forget) {
    try {
      var t = JSON.parse(sessionStorage.getItem(KEEP) || "null");
      if (forget) sessionStorage.removeItem(KEEP);
      return t && Date.now() - t.at < 5000 ? t : null;
    } catch (err) { return null; }
  }

  // Only a page that comes back as itself (a move, a filter, a month)
  // names its items; going to another page is the page cross-fading.
  function same(a, b) {
    try { var x = new URL(a, location.href), y = new URL(b, location.href); return x.pathname === y.pathname; } catch (err) { return true; }
  }

  window.addEventListener("pageswap", function (e) {
    if (!e.viewTransition) return;
    var to = e.activation && e.activation.entry && e.activation.entry.url;
    if (to && !same(to, location.href)) return;
    var dir = to && towards(to);
    if (dir) {
      root.setAttribute("data-travel", dir);
      try { sessionStorage.setItem(KEEP, JSON.stringify({ dir: dir, to: to, at: Date.now() })); } catch (err) { /* no storage: no direction */ }
    }
    name();
  });
  window.addEventListener("pagereveal", function (e) {
    if (!e.viewTransition) return;
    var from = window.navigation && navigation.activation && navigation.activation.from;
    if (from && from.url && !same(from.url, location.href)) return;
    var t = went(true);
    if (t && same(t.to, location.href)) root.setAttribute("data-travel", t.dir);
    name();
    var done = function () { clear(); root.removeAttribute("data-travel"); };
    e.viewTransition.finished.then(done, done);
  });
  // Back to a page kept whole by the browser: as the server sent it.
  window.addEventListener("pageshow", function (e) {
    if (e.persisted) { clear(); root.removeAttribute("data-travel"); }
  });

  // For changes made in the page itself (19-refresh.js): name what is
  // there, change it inside the transition, name what is there now.
  sw.travel = { name: name, clear: clear };
})();
