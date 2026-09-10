// Remembers whether each disclosure was open.
//
// Every action in Sameway is a full-page navigation, so without this a log
// you opened would close again on the next reload. State is per browser and
// per page; nothing is sent anywhere. The disclosure works without this
// script, it just forgets.
(function () {
  "use strict";
  function key(el) { return "sw-disclosure:" + location.pathname + ":" + el.id; }
  function init() {
    document.querySelectorAll("details[data-component=disclosure][id]").forEach(function (el) {
      try {
        if (localStorage.getItem(key(el)) === "open") el.open = true;
      } catch (e) { /* private mode or blocked storage: leave it closed */ }
      el.addEventListener("toggle", function () {
        try { localStorage.setItem(key(el), el.open ? "open" : "closed"); } catch (e) {}
      });
    });
  }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
})();
