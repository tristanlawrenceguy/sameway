// The page follows the turn.
//
// A block landing on the main canvas is shown the moment it exists, by
// 14-live.js. Everything else the assistant changes (a record a
// collection or a calendar shows, a block in a pane, a list that appears
// in the sidebar, the activity) used to wait for a reload. Now the page
// fetches itself as it is and moves what changed into place, inside a
// view transition: a block that moved slides to its new place, a new one
// arrives, one that has gone leaves, with the same motion the page has
// between navigations. The chat stays where it is, live, and a block that
// has not changed is left alone, so nothing flickers. Scripts that arm the
// page hear sw:refresh afterwards and arm what is new.
(function () {
  "use strict";
  if (!window.fetch || !window.DOMParser) return;
  var pace = document.documentElement.getAttribute("data-pace");
  var still = pace === "still" || (window.matchMedia && matchMedia("(prefers-reduced-motion: reduce)").matches);
  // On a narrow screen the whole page scrolls, under a browser bar that
  // comes and goes, and blocks sliding across it read as a glitch rather
  // than as motion; there the changes land without the transition.
  var wide = window.matchMedia ? matchMedia("(min-width: 64rem)") : { matches: true };

  var timer = null, running = false, again = false;

  // swRefresh asks for the page to follow, soon: several asks in a row
  // become one fetch, and one asked for during a fetch means another after.
  window.swRefresh = function (delay) {
    clearTimeout(timer);
    timer = setTimeout(refresh, delay === undefined ? 400 : delay);
  };

  function refresh() {
    if (running) { again = true; return; }
    running = true;
    fetch(location.pathname + location.search, { credentials: "same-origin", headers: { "X-Requested-With": "sameway-live" } })
      .then(function (r) { if (!r.ok) throw new Error("the page came back " + r.status); return r.text(); })
      .then(function (html) {
        var doc = new DOMParser().parseFromString(html, "text/html");
        if (document.startViewTransition && !still && wide.matches) {
          return document.startViewTransition(function () { merge(doc); }).finished.catch(function () {});
        }
        merge(doc);
      })
      .catch(function (err) { if (window.console) console.error("live page:", err); })
      .then(function () { running = false; if (again) { again = false; window.swRefresh(0); } });
  }

  // sameBlock says whether a block is as it was, apart from its place in
  // the order of arrival, which the server renumbers each time.
  function sameBlock(a, b) {
    var strip = function (n) { return n.outerHTML.replace(/ data-arrival="\d+"/g, ""); };
    return strip(a) === strip(b);
  }

  // move puts a node that is in the page in place of one from the fresh
  // page without taking it out first where the browser can (moveBefore),
  // so a focused field keeps its focus, and a phone its keyboard, and a
  // log keeps its scroll. Elsewhere it is an ordinary move, and merge puts
  // the scroll back itself.
  function move(have, spot) {
    if (spot.parentNode.moveBefore) {
      try { spot.parentNode.moveBefore(have, spot); spot.remove(); return; } catch (e) { /* not movable here */ }
    }
    spot.replaceWith(have);
  }

  // merge puts the fresh page in place of the old, keeping the chat and
  // every unchanged block as the nodes they were, and the person where
  // they were: scroll, focus, and the caret in what they were typing.
  function merge(doc) {
    var focused = document.activeElement;
    var focusId = focused && focused.id;
    var selStart = focused && focused.selectionStart, selEnd = focused && focused.selectionEnd;
    var y = window.scrollY;
    // The chat is what the person is reading: the page keeps it where it
    // was on the screen, whatever grew or went above it, and its log at
    // the end when that is where they were.
    var anchor = document.querySelector('[data-block-component="chat"]');
    var anchorTop = anchor && anchor.getBoundingClientRect().top;
    var logs = [];
    document.querySelectorAll(".sw-chat__log").forEach(function (log) {
      logs.push({ log: log, top: log.scrollTop, end: log.scrollHeight - log.scrollTop - log.clientHeight < 48 });
    });
    ["header.sw-header", ".sw-shell"].forEach(function (sel) {
      var fresh = doc.querySelector(sel), old = document.querySelector(sel);
      if (!fresh || !old) return;
      // The fresh page goes in beside the old one first, so a block kept
      // from the old moves between two parts of the page.
      fresh = document.importNode(fresh, true);
      old.after(fresh);
      fresh.querySelectorAll("[data-block-id]").forEach(function (block) {
        var have = old.querySelector('[data-block-id="' + block.getAttribute("data-block-id") + '"]');
        if (!have) return;
        if (block.getAttribute("data-block-component") === "chat" || sameBlock(have, block)) move(have, block);
      });
      old.remove();
    });
    if (doc.title) document.title = (document.title.indexOf("⏳ ") === 0 ? "⏳ " : "") + doc.title.replace(/^⏳ /, "");
    logs.forEach(function (l) { l.log.scrollTo({ top: l.end ? l.log.scrollHeight : l.top, behavior: "instant" }); });
    // Instant: the page's smooth scrolling would otherwise animate the
    // correction, and the page would be seen drifting back into place.
    if (anchor && anchor.isConnected) window.scrollBy({ top: anchor.getBoundingClientRect().top - anchorTop, behavior: "instant" });
    else window.scrollTo({ top: y, behavior: "instant" });
    if (focusId && document.activeElement && document.activeElement.id !== focusId) {
      var back = document.getElementById(focusId);
      if (back) {
        back.focus({ preventScroll: true });
        if (selStart !== null && selStart !== undefined && back.setSelectionRange) {
          try { back.setSelectionRange(selStart, selEnd); } catch (e) { /* not a text field */ }
        }
      }
    }
    document.dispatchEvent(new CustomEvent("sw:refresh"));
  }
})();
