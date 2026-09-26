// Remembers whether each disclosure was open, moves only when opened, and
// prints what is inside.
//
// Every action in Sameway is a full-page navigation, so without this a log
// you opened would close again on the next reload, and after each reply of
// the assistant, which swaps the log in anew. State is per browser, and per
// page unless the disclosure is the same one on every page, as a canvas's
// side pane is (data-remember="site"); nothing is sent anywhere. The
// disclosure works without this script, it just forgets.
(function () {
  "use strict";
  function key(el) {
    return "sw-disclosure:" + (el.getAttribute("data-remember") === "site" ? "" : location.pathname) + ":" + el.id;
  }
  var printing = false;
  function init() {
    document.querySelectorAll("details[data-component=disclosure][id]").forEach(function (el) {
      if (el._remembered) return;
      el._remembered = true;
      try {
        var was = localStorage.getItem(key(el));
        if (was === "open") el.open = true;
        else if (was === "closed") el.open = false;
      } catch (e) { /* private mode or blocked storage: leave it as served */ }
      el.addEventListener("toggle", function () {
        if (printing) return;
        try { localStorage.setItem(key(el), el.open ? "open" : "closed"); } catch (e) {}
      });
    });
    // The body moves in when a person opens it, not when a page arrives
    // with it open: motion explains a change and never decorates a load.
    document.querySelectorAll("details[data-component=disclosure] > summary").forEach(function (s) {
      if (s._moves) return;
      s._moves = true;
      s.addEventListener("click", function () { s.parentNode.setAttribute("data-opened", ""); });
    });
  }
  // On paper, what is folded away is shown, then folded again after.
  var opened = [];
  window.addEventListener("beforeprint", function () {
    printing = true;
    opened = [];
    document.querySelectorAll("details[data-component=disclosure]:not([open])").forEach(function (el) {
      el.open = true;
      opened.push(el);
    });
  });
  window.addEventListener("afterprint", function () {
    opened.forEach(function (el) { el.open = false; });
    opened = [];
    setTimeout(function () { printing = false; }, 0);
  });
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
  document.addEventListener("sw:refresh", init);
  // A reply of the assistant swaps the log in anew; it keeps how it was.
  document.addEventListener("sw:turn-done", init);
})();
