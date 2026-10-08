// The page follows changes made elsewhere.
//
// When another computer that hosts this workspace changes it, the server
// says so on /events and the page fetches itself and moves what changed
// into place, the way it does after a turn (17-refresh.js), glowing where
// it changed. Without scripts the page shows the change on the next load.
//
// The same connection is how the others know this person is here (see
// the presence component). After ten minutes with no key, pointer or
// wheel, it says ?idle=1: the page still follows, but its person is no
// longer said to be here, until they move again.
(function () {
  "use strict";
  if (!window.EventSource) return;
  var IDLE = 10 * 60 * 1000;
  var es = null, idle = false, moved = Date.now();
  function open() {
    if (es) return;
    es = new EventSource(idle ? "/events?idle=1" : "/events");
    es.addEventListener("changed", function () { sw.refresh(300); });
  }
  function close() { if (es) { es.close(); es = null; } }
  function become(now) {
    if (idle === now) return;
    idle = now;
    if (es) { close(); open(); }
  }
  ["keydown", "pointerdown", "pointermove", "wheel", "touchstart"].forEach(function (type) {
    window.addEventListener(type, function () { moved = Date.now(); become(false); }, { passive: true, capture: true });
  });
  setInterval(function () { if (Date.now() - moved >= IDLE) become(true); }, 60 * 1000);
  // A tab nobody is looking at does not hold a connection open; it
  // catches up when it is looked at again.
  document.addEventListener("visibilitychange", function () {
    if (document.hidden) { close(); } else { moved = Date.now(); idle = false; open(); sw.refresh(0); }
  });
  if (!document.hidden) open();
})();
