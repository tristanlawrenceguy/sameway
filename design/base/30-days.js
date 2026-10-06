// Days that stay true on a page left open, and a zone said when it differs.
//
// The server says a day the way a person plans by it: Today, Tomorrow, a
// task Overdue once its time has gone (design/foundations/glance.md).
// Words like those go stale on a page left open overnight, or past a
// task's time: Today becomes yesterday's, and an overdue task still reads
// as due. So the page follows itself (swRefresh, 17-refresh.js, which
// keeps scroll, focus and what is being typed) at the moments its words
// change: when this workspace's day turns, and when a time the page shows
// passes. A hidden tab waits until it is looked at again.
//
// Every time on the page is the workspace's, the computer it runs on. When
// the browser reading it is in another zone (a phone abroad, a person let
// in from elsewhere), the page says so once, near its top, with how far
// apart the two are, as GOV.UK adds "UK time" for readers elsewhere.
(function () {
  "use strict";
  var root = document.documentElement;
  var zone = parseInt(root.getAttribute("data-zone"), 10);
  if (isNaN(zone)) return;

  // today is the workspace's date now, as the server wrote data-today.
  function today() { return new Date(Date.now() + zone * 60000).toISOString().slice(0, 10); }
  var shown = root.getAttribute("data-today") || today();

  // next is the soonest moment the page's words change: the workspace's
  // next midnight, or a time shown on the page that is still to come.
  function next() {
    var now = Date.now();
    var here = new Date(now + zone * 60000);
    var soonest = Date.UTC(here.getUTCFullYear(), here.getUTCMonth(), here.getUTCDate() + 1) - zone * 60000;
    document.querySelectorAll("time[datetime*='T']").forEach(function (t) {
      var at = Date.parse(t.getAttribute("datetime"));
      if (!isNaN(at) && at > now && at < soonest) soonest = at;
    });
    return soonest;
  }

  var timer = null;
  function follow() {
    if (document.visibilityState === "hidden" || !window.swRefresh) return;
    shown = today();
    window.swRefresh(0);
  }
  function arm() {
    clearTimeout(timer);
    // A second after, so the server's now is past it too; at most a day,
    // which a browser's timer can hold.
    timer = setTimeout(follow, Math.min(next() - Date.now() + 1000, 86400000));
  }
  document.addEventListener("visibilitychange", function () {
    if (document.visibilityState !== "visible") return;
    if (today() !== shown) follow(); else arm();
  });
  document.addEventListener("sw:refresh", arm);
  arm();

  // The zone, said once when the reader's differs from the workspace's.
  var mine = -new Date().getTimezoneOffset();
  if (mine === zone) return;
  function offset(min) {
    var sign = min < 0 ? "−" : "+", a = Math.abs(min);
    return "UTC" + sign + Math.floor(a / 60) + (a % 60 ? ":" + (a % 60 < 10 ? "0" : "") + (a % 60) : "");
  }
  function apart(min) {
    var a = Math.abs(min), h = Math.floor(a / 60), m = a % 60;
    var words = (h ? h + (h === 1 ? " hour" : " hours") : "") + (h && m ? " " : "") + (m ? m + " minutes" : "");
    return words + (min > 0 ? " ahead of yours" : " behind yours");
  }
  function say() {
    var main = document.querySelector("main");
    if (!main || !main.querySelector("time") || document.getElementById("zone-note")) return;
    var note = document.createElement("p");
    note.id = "zone-note";
    note.className = "sw-muted sw-small";
    note.textContent = "Times here are this workspace's, " + offset(zone) + ": " + apart(zone - mine) + ".";
    var head = main.querySelector(".sw-page-head");
    if (head) head.after(note); else main.prepend(note);
  }
  say();
  // A page that follows itself comes back without it.
  document.addEventListener("sw:refresh", say);
})();
