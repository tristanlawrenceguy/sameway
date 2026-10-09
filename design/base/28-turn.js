// A turn you can watch, without noise (14-live.js shows the turn; this is
// how it is said, how its words grow and where it is marked).
//
// The status line says what the assistant is doing in words its calls
// give ("Adding a chart of water…", "Looking up your tasks…") rather than
// a spinner. The status is a polite live region, so every change to it is
// read out: it changes at most once every few seconds, keeps only the
// newest words when several come close together, and never says the same
// words twice in one turn. What is shown is exactly what is heard, since
// the words drawn are the region's own.
//
// A block the assistant is about to add or change is marked in its place
// while the call runs: a new block as an outline in the assistant's colour
// where it will land (the first stage of its arrival, held), a block being
// changed with an outline round it. When the block lands, its staged
// arrival plays from that outline (03-motion.css). Nothing moves: the
// outline fades in, and under the still pace it is simply there.
(function () {
  "use strict";
  var GAP = 2500; // between two things said: time for one to be heard

  // The log follows the turn only while the person is reading its end.
  // Someone who has scrolled up, or is selecting text, is left where they
  // are: text that moves under the pointer cannot be selected.
  window.swFollower = function (log) {
    var stick = true, pressed = false;
    function nearEnd() { return log.scrollHeight - log.scrollTop - log.clientHeight < 48; }
    log.addEventListener("scroll", function () { stick = nearEnd(); });
    log.addEventListener("pointerdown", function () { pressed = true; });
    document.addEventListener("pointerup", function () { pressed = false; });
    document.addEventListener("pointercancel", function () { pressed = false; });
    return function () {
      if (pressed) return;
      var sel = window.getSelection && getSelection();
      if (sel && !sel.isCollapsed && sel.anchorNode && log.contains(sel.anchorNode)) return;
      // At once, not smoothly: a log gliding after every few words is a
      // page that never stops moving.
      if (stick) { if (log.scrollTo) log.scrollTo({ top: log.scrollHeight, behavior: "instant" }); else log.scrollTop = log.scrollHeight; }
    };
  };

  // The reply's words go on the page a few times a second, not as each
  // piece arrives: the text grows calmly, with no caret and nothing drawn
  // letter by letter, and the log is not scrolled for every word. The
  // words are not in a live region: a screen reader hears the finished
  // reply once, when the turn ends (19-live-join.js, swSay).
  window.swWriter = function (words, follow, watch) {
    var queued = "", timer = null;
    function flush() { timer = null; words.data += queued; queued = ""; follow(); }
    function write(text) {
      if (!text) return;
      if (watch && !words.data && !queued) watch.wrote();
      queued += text;
      if (!timer) timer = setTimeout(flush, 80);
    }
    // A model that does not stream gives a round's words whole.
    write.all = function (text) { clearTimeout(timer); timer = null; queued = ""; words.data = text || ""; if (text && watch) watch.wrote(); };
    write.stop = function () { clearTimeout(timer); timer = null; };
    return write;
  };

  function canvas() { return document.querySelector(".sw-main .sw-canvas:not(.sw-canvas--strip)"); }

  window.swWatchTurn = function (form) {
    var status = document.getElementById(form.getAttribute("data-busy-target"));
    var said = {}, last = 0, waiting = null, timer = null, early = 0;

    // say puts words on the status line, at most every GAP, the newest
    // words winning, each only once a turn.
    function put(words) {
      if (!status || said[words]) return;
      said[words] = true;
      last = Date.now();
      // After fifteen quiet seconds the line says it is still at it.
      status.setAttribute("data-still", "Still " + words.charAt(0).toLowerCase() + words.slice(1));
      sw.status(status, "working", words, "");
    }
    function say(words) {
      if (said[words]) return;
      waiting = words;
      if (timer) return;
      timer = setTimeout(function () { timer = null; if (waiting) put(waiting); waiting = null; }, Math.max(0, GAP - (Date.now() - last)));
    }

    // A new block's place: where the lander will put it, at the end of the
    // main canvas, as wide as it will be. A refresh of the page while the
    // turn runs (17-refresh.js) brings the server's page, which has no such
    // place and no marks, so they are put back after it.
    var held = [], marked = {};
    function hold(d) {
      var c = canvas();
      if (!c) return;
      var li = document.createElement("li");
      li.className = "sw-block sw-block--pending";
      li.setAttribute("data-actor", "assistant");
      li.style.setProperty("--sw-span", d.span || 6);
      // Its shape, in grey bars, rather than empty space (29-skeleton.css);
      // busy, and saying in words what is coming, until it lands.
      li.setAttribute("aria-busy", "true");
      li.innerHTML = '<p class="sw-visually-hidden"></p><div class="sw-skeleton" aria-hidden="true"><span></span><span></span><span></span></div>';
      li.firstChild.textContent = (d.label || "Adding a block") + "…";
      c.appendChild(li);
      held.push(li);
    }
    function mark(id, how) {
      var blk = document.querySelector('[data-block-id="' + id.replace(/["\\]/g, "") + '"]');
      if (blk) { blk.setAttribute("data-pending", how); blk.setAttribute("aria-busy", "true"); }
    }
    function again() {
      var c = canvas();
      held = held.filter(function (li) { return !li._landed; });
      held.forEach(function (li) { if (!li.isConnected && c) c.appendChild(li); });
      Object.keys(marked).forEach(function (id) { mark(id, marked[id]); });
    }
    sw.on("refresh", again);

    return {
      // A step of the turn: said, and its place marked.
      step: function (d, label) {
        say(label + "…");
        if (d.tool === "add_component") {
          if (d.early) { early++; hold(d); return; }
          if (early > 0) {
            early--;
            var last = held[held.length - 1];
            // The call, whole, says where it goes: a pane is not the main canvas.
            if (last && d.region && d.region !== "main") { last.remove(); held.pop(); }
            else if (last) last.style.setProperty("--sw-span", d.span || 6);
          } else if (!d.region || d.region === "main") hold(d);
        }
        if (d.block) { marked[d.block] = d.tool === "remove_component" ? "remove" : "change"; mark(d.block, marked[d.block]); }
      },
      // A block has landed: its place and its mark are its own now.
      // A new one takes the first place held, as the lander puts it there,
      // so a refresh before it lands does not hold that place again.
      landed: function (id, added) {
        delete marked[id];
        if (added && held.length) held.shift()._landed = true;
      },
      // The reply's words have started: said once.
      wrote: function () { say("Writing the reply…"); },
      // The turn is over: nothing more is said by this, and every mark goes.
      done: function () {
        clearTimeout(timer); timer = null; waiting = null;
        sw.off("refresh", again);
        held = []; marked = {};
        if (status) status.removeAttribute("data-still");
        document.querySelectorAll(".sw-block--pending").forEach(function (p) { p.remove(); });
        document.querySelectorAll("[data-pending]").forEach(function (b) { b.removeAttribute("data-pending"); b.removeAttribute("aria-busy"); });
      }
    };
  };
})();
