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
        if (document.startViewTransition && !still) {
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

  // merge puts the fresh page in place of the old, keeping the chat and
  // every unchanged block as the nodes they were, and the person where
  // they were: scroll, focus, and the caret in what they were typing.
  function merge(doc) {
    var focused = document.activeElement;
    var focusId = focused && focused.id;
    var selStart = focused && focused.selectionStart, selEnd = focused && focused.selectionEnd;
    var y = window.scrollY;
    ["header.sw-header", ".sw-shell"].forEach(function (sel) {
      var fresh = doc.querySelector(sel), old = document.querySelector(sel);
      if (!fresh || !old) return;
      fresh.querySelectorAll("[data-block-id]").forEach(function (block) {
        var have = old.querySelector('[data-block-id="' + block.getAttribute("data-block-id") + '"]');
        if (!have) return;
        if (block.getAttribute("data-block-component") === "chat" || sameBlock(have, block)) block.replaceWith(have);
      });
      old.replaceWith(fresh);
    });
    if (doc.title) document.title = (document.title.indexOf("⏳ ") === 0 ? "⏳ " : "") + doc.title.replace(/^⏳ /, "");
    window.scrollTo(0, y);
    if (focusId) {
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
