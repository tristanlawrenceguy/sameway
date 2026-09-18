// The chat, shaped by the person.
//
// Pop out opens the conversation in a small window of its own, so it can
// sit beside whatever else is on the screen; without scripts the link
// opens a tab, which is the same conversation. The log can be dragged
// taller or shorter (that is CSS), and the height a person settles on is
// kept in this browser so it is the same next time.
(function () {
  "use strict";
  var KEY = "sameway:chat-height";

  function popout(a) {
    a.addEventListener("click", function (e) {
      if (!window.open) return;
      var win = window.open(a.getAttribute("href"), "sameway-chat", "popup=yes,width=460,height=760");
      if (win) e.preventDefault();
    });
  }

  function remember(log) {
    var saved = null;
    try { saved = localStorage.getItem(KEY); } catch (err) { saved = null; }
    if (saved && log.querySelector("li")) log.style.height = saved;
    if (!window.ResizeObserver) return;
    var first = true;
    new ResizeObserver(function () {
      // The first call reports the size as laid out, not a choice.
      if (first) { first = false; return; }
      if (!log.style.height) return;
      try { localStorage.setItem(KEY, log.style.height); } catch (err) { /* private windows keep nothing */ }
    }).observe(log);
  }

  function init() {
    document.querySelectorAll("a[data-popout]").forEach(popout);
    document.querySelectorAll(".sw-chat__log").forEach(remember);
  }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init); else init();
})();
