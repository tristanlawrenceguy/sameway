// The page's small core: the one place the other scripts meet.
//
// Every script here and in a component's enhance.js is concatenated, in
// order, into /design/sameway.js, and each used to arm its own nodes on
// load, listen for the page's refresh to arm the new ones, and hang what
// others needed on window. Now they share this:
//
//   sw.arm(selector, fn)  fn(el) once for each element that matches, on load
//                         and whenever the page brings nodes in (a refresh,
//                         an outcome shown); an element is armed only once.
//   sw.scan()             arm what a script has just put in the page itself.
//   sw.ready(fn)          fn once the page is parsed.
//   sw.on / sw.off / sw.emit(name, detail)
//                         a tiny bus: "refresh" when the page has followed
//                         a change (19-refresh.js), "turn-done" when a turn
//                         ends (17-live.js). Each is also dispatched on the
//                         document as "sw:<name>", for a workspace's own
//                         scripts and the browser tests.
//   sw.status(el, state, words, said)
//                         the one way a status changes (the status component).
//   sw.refresh(delay)     ask the page to follow, soon (19-refresh.js); does
//                         nothing in a browser that cannot.
//
// No framework and no build: this is a plain script, first in the bundle,
// and every other script finds window.sw already there.
(function () {
  "use strict";
  if (window.sw) return; // a page that loads a base file again keeps the one core
  var sw = window.sw = {};
  var handlers = {};
  var scans = [];

  sw.on = function (name, fn) { (handlers[name] = handlers[name] || []).push(fn); };
  sw.off = function (name, fn) {
    var list = handlers[name];
    if (list) handlers[name] = list.filter(function (f) { return f !== fn; });
  };
  // A slip in one handler must not stop the rest, as with separate listeners.
  sw.emit = function (name, detail) {
    (handlers[name] || []).slice().forEach(function (fn) {
      try { fn(detail); } catch (err) { if (window.console) console.error("sw " + name + ":", err); }
    });
    document.dispatchEvent(new CustomEvent("sw:" + name, { detail: detail }));
  };

  sw.ready = function (fn) {
    if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", fn);
    else fn();
  };

  // Each arm keeps the elements it has armed, so a node kept through a
  // refresh is not armed twice and one that arrives is armed once.
  sw.arm = function (selector, fn) {
    var seen = new WeakSet();
    function scan() {
      document.querySelectorAll(selector).forEach(function (el) {
        if (seen.has(el)) return;
        seen.add(el);
        try { fn(el); } catch (err) { if (window.console) console.error("sw arm " + selector + ":", err); }
      });
    }
    scans.push(scan);
    sw.ready(scan);
    sw.on("refresh", scan);
  };
  sw.scan = function () { scans.forEach(function (scan) { scan(); }); };

  sw.refresh = function () {};

  // sw.status was the status component's: state, look, words and what is
  // read out after them but not drawn, all at once, in the region already
  // on the page, so it is heard and never shows one state's mark beside
  // another's words. A wait still going after fifteen seconds says so once.
  sw.status = function (el, state, words, said) {
    if (!el) return;
    if (state) {
      el.setAttribute("data-state", state);
      el.className = el.className.replace(/sw-status--\w+/, "sw-status--" + state);
    }
    el.setAttribute("aria-live", "polite");
    var text = el.querySelector(".sw-status__text");
    if (text && words != null) text.textContent = words;
    var hidden = el.querySelector(".sw-status__said");
    if (said && !hidden) {
      hidden = document.createElement("span");
      hidden.className = "sw-status__said sw-visually-hidden";
      el.appendChild(hidden);
    }
    if (hidden) hidden.textContent = said ? " " + said : "";
    clearTimeout(el._still);
    if ((state || el.getAttribute("data-state")) === "working" && state) {
      el._still = setTimeout(function () {
        if (el.getAttribute("data-state") === "working") sw.status(el, null, el.getAttribute("data-still") || "Still working…");
      }, 15000);
    }
  };
})();
