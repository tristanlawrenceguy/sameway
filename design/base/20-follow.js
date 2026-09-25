// The page follows changes made elsewhere.
//
// When another computer that hosts this workspace changes it, the server
// says so on /events and the page fetches itself and moves what changed
// into place, the way it does after a turn (17-refresh.js), glowing where
// it changed. Without scripts the page shows the change on the next load.
(function () {
  "use strict";
  if (!window.EventSource || !window.swRefresh) return;
  var es = null;
  function open() {
    if (es) return;
    es = new EventSource("/events");
    es.addEventListener("changed", function () { window.swRefresh(300); });
  }
  function close() { if (es) { es.close(); es = null; } }
  // A tab nobody is looking at does not hold a connection open; it
  // catches up when it is looked at again.
  document.addEventListener("visibilitychange", function () {
    if (document.hidden) { close(); } else { open(); window.swRefresh(0); }
  });
  if (!document.hidden) open();
})();
