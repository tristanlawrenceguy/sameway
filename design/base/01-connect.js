// The page's one connection to the server, and the reading of a turn.
//
// Several scripts want to hear from the server as things happen: the page
// follows a change made elsewhere (17-refresh.js), the clock rings (the
// clock component), and the server knows who is here by the page holding
// /events open (the presence component). Each used to open its own
// EventSource; now they share one:
//
//   sw.listen(name, fn, keep)
//               fn(data) for each server event of that name on the one
//               /events connection, opened when the first is asked for.
//               keep: the event matters while the tab is hidden (a ring
//               does), so the connection stays open then, said idle.
//   sw.stream(response, fn)
//               reads a turn's server-sent events from a fetch (14-live.js,
//               19-live-join.js), fn({event, data}) for each, a slip in one
//               not losing the rest; resolves when the stream ends.
//
// The connection says ?idle=1 after ten minutes with no key, pointer or
// wheel, or while the tab is hidden: the page still follows, but its
// person is no longer said to be here. A tab nobody is looking at does not
// hold it open unless something kept wants it; it catches up when looked
// at again ("back" on the bus). A connection the server ends is opened
// again, after one second, then two, four, up to a minute; one refused
// from the start (a published page has no /events) is left closed.
// Measuring (25-measure.js) sends its own beacons and is not this.
(function () {
  "use strict";
  var IDLE = 10 * 60 * 1000, MOST = 60 * 1000;
  var es = null, url = "", opened = false, tries = 0, again = null, moved = Date.now();
  var subs = {}, kept = {};

  function idle() { return document.hidden || Date.now() - moved >= IDLE; }
  function wanted() {
    var q = [];
    if (idle()) q.push("idle=1");
    if (kept.ring) q.push("ring=1");
    return "/events" + (q.length ? "?" + q.join("&") : "");
  }
  function hear(name) {
    es.addEventListener(name, function (e) {
      var data = {};
      try { data = JSON.parse(e.data || "{}"); } catch (err) { data = {}; }
      (subs[name] || []).forEach(function (fn) {
        try { fn(data); } catch (err) { if (window.console) console.error("sw listen " + name + ":", err); }
      });
    });
  }
  function close() { clearTimeout(again); if (es) { es.close(); es = null; } }
  // open brings the connection to what is wanted now: open, closed, or
  // opened again when what it should say has changed.
  function open() {
    if (!window.EventSource || !Object.keys(subs).length) return;
    var keep = Object.keys(kept).length > 0;
    if (document.hidden && !keep) { close(); return; }
    var want = wanted();
    if (es && want === url) return;
    close();
    url = want;
    es = new EventSource(url);
    var mine = es;
    es.onopen = function () { opened = true; tries = 0; };
    es.onerror = function () {
      if (mine !== es || es.readyState !== 2) return; // the browser tries again itself
      es = null;
      if (!opened) return;
      again = setTimeout(open, Math.min(MOST, 1000 * Math.pow(2, tries++)));
    };
    Object.keys(subs).forEach(hear);
  }

  sw.listen = function (name, fn, keep) {
    var fresh = !subs[name];
    (subs[name] = subs[name] || []).push(fn);
    if (keep) kept[name] = true;
    if (es && fresh && wanted() === url) hear(name);
    sw.ready(open);
  };

  ["keydown", "pointerdown", "pointermove", "wheel", "touchstart"].forEach(function (type) {
    window.addEventListener(type, function () {
      var was = idle();
      moved = Date.now();
      if (was && es) open();
    }, { passive: true, capture: true });
  });
  setInterval(function () { if (es && idle() && url.indexOf("idle=1") < 0) open(); }, 60 * 1000);
  document.addEventListener("visibilitychange", function () {
    if (!document.hidden) moved = Date.now();
    open();
    if (!document.hidden) sw.emit("back");
  });

  sw.stream = function (response, fn) {
    var reader = response.body.getReader(), decoder = new TextDecoder(), buffer = "";
    function parse(frame) {
      var event = "message", data = "";
      frame.split("\n").forEach(function (line) {
        if (line.indexOf("event:") === 0) event = line.slice(6).trim();
        else if (line.indexOf("data:") === 0) data += line.slice(5).trim();
      });
      var body = {};
      try { body = JSON.parse(data); } catch (e) { body = {}; }
      return { event: event, data: body };
    }
    function pump() {
      return reader.read().then(function (r) {
        if (r.done) return;
        buffer += decoder.decode(r.value, { stream: true });
        var frames = buffer.split("\n\n");
        buffer = frames.pop();
        frames.forEach(function (f) {
          if (!f.trim()) return;
          try { fn(parse(f)); } catch (err) { if (window.console) console.error("live turn:", err); }
        });
        return pump();
      });
    }
    return pump();
  };
})();
