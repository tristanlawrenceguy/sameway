// Days that stay true on a page left open, and a zone said when it differs.
//
// The server says a day the way a person plans by it: Today, Tomorrow, a
// task Overdue once its time has gone (design/foundations/glance.md).
// Words like those go stale on a page left open overnight, or past a
// task's time: Today becomes yesterday's, and an overdue task still reads
// as due. So the page follows itself (sw.refresh, 19-refresh.js, which
// keeps scroll, focus and what is being typed) at the moments its words
// change: when this workspace's day turns, and when a time the page shows
// passes. A hidden tab waits until it is looked at again.
//
// Every time on the page is the workspace's, in its time zone (ui.zone,
// else the computer's it runs on). When the browser reading it is in
// another zone (a phone abroad, a person let in from elsewhere), the page
// says so once, near its top, with how far apart the two are, as GOV.UK
// adds "UK time" for readers elsewhere. The owner of a workspace with no
// zone chosen (data-zone-ask) is asked instead whether theirs is its, as
// Google Calendar offers to update a time zone: the computer's may be a
// server's in UTC. Nothing changes until they press (design/foundations/
// today.md); a viewer who is not the owner is never asked.
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
    if (document.visibilityState === "hidden") return;
    shown = today();
    sw.refresh(0);
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
  sw.on("refresh", arm);
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
  // place is the browser's zone by name, Europe/Madrid, when it says one.
  var place = "";
  try { place = Intl.DateTimeFormat().resolvedOptions().timeZone || ""; } catch (e) { /* none said */ }
  var ask = root.hasAttribute("data-zone-ask") && place.indexOf("/") > 0;
  var called = place.slice(place.lastIndexOf("/") + 1).replace(/_/g, " ") + " time";
  // choice is a press that sets the workspace's zone, back to this page.
  function choice(label, value, variant) {
    var form = document.createElement("form");
    form.method = "post";
    form.action = "/zone";
    [["from", location.pathname + location.search], ["zone", value]].forEach(function (f) {
      var input = document.createElement("input");
      input.type = "hidden";
      input.name = f[0];
      input.value = f[1];
      form.append(input);
    });
    var button = document.createElement("button");
    button.type = "submit";
    button.className = "sw-button sw-button--" + variant + " sw-pressable";
    button.textContent = label;
    form.append(button);
    return form;
  }
  function say() {
    var main = document.querySelector("main");
    if (!main || !main.querySelector("time") || document.getElementById("zone-note")) return;
    var note = document.createElement(ask ? "div" : "p");
    note.id = "zone-note";
    note.className = ask ? "sw-stack--tight" : "sw-muted sw-small";
    if (ask) {
      var words = document.createElement("p");
      words.textContent = "Times here are this computer's, " + offset(zone) + ": " + apart(zone - mine) + ". Use " + called + " for this workspace?";
      var presses = document.createElement("div");
      presses.className = "sw-cluster";
      presses.append(choice("Use " + called, place, "primary"), choice("Keep this computer's time", "Local", "secondary"));
      note.append(words, presses);
    } else {
      note.textContent = "Times here are this workspace's, " + offset(zone) + ": " + apart(zone - mine) + ".";
    }
    var head = main.querySelector(".sw-page-head");
    if (head) head.after(note); else main.prepend(note);
  }
  say();
  // A page that follows itself comes back without it.
  sw.on("refresh", say);
})();
