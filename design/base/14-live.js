// The turn as it happens.
//
// The composer is a form and works as one: post, wait, page. With scripts
// the same form posts to /chat/stream and reads the turn as it goes: the
// person's message appears as recorded and the box clears for the next
// one, the reply's words come as the model says them, each tool shows the
// moment the model names it and fills in as it runs, and each block the
// assistant makes or changes arrives on the canvas the moment it exists,
// one at a time with a breath between when several come at once, so there
// is time to take each in. A message sent while the assistant is working
// waits and goes when the turn is done. If anything about the stream
// fails, the form is sent the ordinary way and the page comes back whole.
// Stopping a turn, and following one from another page: 19-live-join.js.
(function () {
  "use strict";
  if (!window.fetch || !window.ReadableStream || !window.TextDecoder) return;
  var pace = document.documentElement.getAttribute("data-pace");
  var still = pace === "still" || (window.matchMedia && matchMedia("(prefers-reduced-motion: reduce)").matches);
  var GAP = still ? 0 : pace === "quick" ? 400 : 1200;

  function el(html) {
    var t = document.createElement("template");
    t.innerHTML = html.trim();
    return t.content.firstElementChild;
  }
  function escape(s) {
    return String(s).replace(/[&<>"]/g, function (c) { return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]; });
  }

  // The log the messages live in, made when the conversation was empty.
  function logFor(form) {
    var body = form.closest(".sw-chat__body") || form.parentElement;
    var log = body.querySelector(".sw-chat__log");
    if (log) return log;
    var empty = body.querySelector(".sw-empty");
    log = el('<ol class="sw-plain sw-stack sw-chat__log" aria-label="Messages" tabindex="0"></ol>');
    if (empty) empty.replaceWith(log); else body.insertBefore(log, form);
    body.querySelectorAll(".sw-chat__starts").forEach(function (x) { x.remove(); }); // things to ask go with the empty
    return log;
  }

  function liveMessage(log) {
    var li = el('<li class="sw-live"><article class="sw-message sw-message--assistant" data-component="message" data-role="assistant" data-actor="assistant" aria-busy="true">' +
      '<h3 class="sw-message__meta"><span class="sw-message__author">Assistant</span></h3>' +
      '<div class="sw-message__body"><p class="sw-live__text"></p><ol class="sw-plain sw-live__steps" aria-label="What the assistant is doing"></ol></div></article></li>');
    log.appendChild(li);
    // The words grow in one text node, so a selection elsewhere in the
    // log survives each piece that arrives.
    var words = document.createTextNode("");
    li.querySelector(".sw-live__text").appendChild(words);
    return { li: li, words: words, steps: li.querySelector(".sw-live__steps") };
  }

  // Blocks land one at a time, with a breath between when several come
  // at once; under reduced motion or a still pace they land as they come.
  function lander() {
    var queue = [], busy = false, last = 0;
    function land(item) {
      // The canvas in the main region, not the strip in the header, which
      // is a canvas too and is where the first try put a new card.
      var canvas = document.querySelector(".sw-main .sw-canvas:not(.sw-canvas--strip)");
      if (!canvas || (item.region && item.region !== "main")) return;
      var have = canvas.querySelector('[data-block-id="' + item.block + '"]');
      if (item.action === "removed" || !item.html) {
        if (have) { have.classList.add("sw-exit"); setTimeout(function () { have.remove(); }, still ? 0 : 220); }
        return;
      }
      var node = el(item.html);
      node.classList.add("sw-landed"); // under reduced motion it fades in (28-turn.css)
      // A new block takes the place held for it while its call ran (28-turn.js).
      have = have || canvas.querySelector(".sw-block--pending");
      if (have) { have._landed = true; have.replaceWith(node); } else canvas.appendChild(node);
      var page = canvas.closest(".sw-page");
      if (page && page.getAttribute("data-layout") === "solo") page.setAttribute("data-layout", "canvas");
    }
    function next() {
      if (busy || !queue.length) return;
      busy = true;
      setTimeout(function () { land(queue.shift()); last = Date.now(); busy = false; next(); }, Math.max(0, GAP - (Date.now() - last)));
    }
    return function (item) { queue.push(item); next(); };
  }

  // The rest of the page follows the turn too: see 17-refresh.js, which
  // fetches the page as it now is and moves what changed into place, with
  // a transition. A block landing on the main canvas is shown at once,
  // above; this brings the rest.
  function refreshSoon(delay) { sw.refresh(delay); }

  function statusText(form, message) {
    sw.status(document.getElementById(form.getAttribute("data-busy-target")), null, message, "");
  }

  // send runs a turn on the page: the one the form asks for, or, with a
  // request for its events, one under way that the page is joining late.
  function send(form, joining) {
    var log = logFor(form);
    var live = liveMessage(log);
    var land = lander();
    var follow = sw.follower(log);
    var ta = form.querySelector("textarea");
    var asked = ta ? ta.value : "";
    var settled = false, heard = false, stop = null;
    var watch = sw.watchTurn(form);
    var write = sw.writer(live.words, follow, watch);
    // The steps: what the assistant is doing, one dot each. A step is
    // early while the model is still saying what it wants; the same tool
    // fills the step in when it runs. Until anything arrives, a dot says
    // the model is thinking.
    function running() { return live.steps.querySelectorAll('[data-state="running"]:not([data-early])'); }
    function finish() { running().forEach(function (s) { s.setAttribute("data-state", "done"); }); }
    // A model on the person's own computer can think for most of a minute
    // before its first word, and a dot that only says Thinking for that
    // long reads as stuck; after a while it says that is how it goes.
    var slow = null;
    function thinking(on) {
      var have = live.steps.querySelector("[data-thinking]");
      if (on && !have) {
        var li = el('<li class="sw-live__step" data-state="running" data-thinking><span class="sw-live__dot" aria-hidden="true"></span><span>Thinking</span></li>');
        live.steps.appendChild(li);
        clearTimeout(slow);
        slow = setTimeout(function () { if (li.isConnected) li.lastChild.textContent = "Still thinking. Some models take a minute."; }, 10000);
      }
      if (!on && have) { clearTimeout(slow); have.remove(); }
    }
    function step(d) {
      thinking(false);
      var label = d.label || d.tool || "Working";
      if (watch) watch.step(d, label);
      if (!d.early) {
        finish();
        var early = d.tool && live.steps.querySelector('[data-early][data-tool="' + escape(d.tool) + '"]');
        if (early) {
          early.removeAttribute("data-early");
          early.lastChild.nodeValue = label;
          return;
        }
      }
      live.steps.appendChild(el('<li class="sw-live__step" data-state="running"' + (d.early ? " data-early" : "") + (d.tool ? ' data-tool="' + escape(d.tool) + '"' : "") +
        '><span class="sw-live__dot" aria-hidden="true"></span>' + escape(label) + "</li>"));
    }
    function settle(d) {
      settled = true;
      form._sending = false;
      write.stop();
      if (watch) watch.done();
      if (stop) { if (stop === document.activeElement && ta) ta.focus(); stop.remove(); }
      live.li.classList.remove("sw-live");
      if (d.html) live.li.innerHTML = d.html; else live.li.remove();
      var status = document.getElementById("chat-status");
      if (status && d.status) sw.say(d.status, d.html, d.text);
      // What a reload would have brought: the recent activity, the skip
      // link to the newest message, and the address naming it.
      var activity = document.querySelector(".sw-activity");
      if (!activity && d.activity) {
        // A page that had nothing to report yet has no place for it.
        activity = el('<div class="sw-activity"></div>');
        var page = document.querySelector(".sw-main .sw-page") || document.querySelector(".sw-main .sw-empty");
        if (page) page.after(activity); else if (document.querySelector(".sw-main")) document.querySelector(".sw-main").appendChild(activity);
      }
      if (activity && d.activity) activity.innerHTML = d.activity;
      if (d.id) {
        var skips = document.querySelectorAll('a.sw-skip[href^="#msg-"]');
        if (!skips.length) {
          // The first message on a page brings the way to the newest one.
          var first = null, j, allSkips = document.querySelectorAll(".sw-skip");
          for (j = 0; j < allSkips.length; j++) { if (allSkips[j].tagName === "A") { first = allSkips[j]; break; } }
          if (first) first.after(el('<a class="sw-skip" href="#msg-' + d.id + '">Skip to latest message</a>'));
        }
        document.querySelectorAll('a.sw-skip[href^="#msg-"]').forEach(function (a) { a.setAttribute("href", "#msg-" + d.id); });
        if (window.history && history.replaceState) history.replaceState(null, "", "#msg-" + d.id);
        var newest = document.getElementById("msg-" + d.id);
        if (newest && newest.scrollIntoView) newest.scrollIntoView({ block: "nearest" });
      }
      form.removeAttribute("aria-busy");
      form.querySelectorAll("button[type=submit]").forEach(function (b) { b.removeAttribute("aria-disabled"); });
      var region = form.closest("[data-region]");
      if (region) region.setAttribute("data-state", "idle");
      document.title = document.title.replace(/^⏳ /, "");
      // A turn that failed gives the words back, so they can be sent again.
      if (ta && d.text && ta.value === "" && !joining) ta.value = asked;
      follow();
      // The turn is over: 19-live-join.js tells a person who looked away.
      sw.emit("turn-done", d);
      // A message written while the assistant was working goes now.
      if (form._queued) {
        form._queued = false;
        if (ta && ta.value.trim() && !d.text) setTimeout(function () { if (form.requestSubmit) form.requestSubmit(); else form.submit(); }, 0);
      }
    }
    function handle(msg) {
      var d = msg.data;
      heard = true;
      switch (msg.event) {
        case "said":
          // A page joining late already shows the message, and the box
          // may hold something new.
          if (joining && document.getElementById("msg-" + d.id)) { thinking(true); stop = sw.stopControl(form, d.turn); break; }
          if (d.html) live.li.before(el("<li>" + d.html + "</li>"));
          // The message is recorded; the box is ready for the next one.
          if (ta && !joining) ta.value = "";
          var file = form.querySelector('input[type="file"]');
          if (file && !joining) file.value = "";
          thinking(true);
          stop = sw.stopControl(form, d.turn);
          break;
        case "delta": thinking(false); write(d.text); break;
        case "text": thinking(false); write.all(d.text); break;
        case "tool": step(d); break;
        case "change":
          finish();
          if (watch && d.block) watch.landed(d.block, d.action === "added" && (!d.region || d.region === "main"));
          if (d.block && d.html) land(d);
          // Whatever else the change touched follows, once things settle.
          refreshSoon();
          break;
        case "done": settle(d); refreshSoon(0); break;
        case "error": settle(d); refreshSoon(0); break;
      }
      follow();
    }
    (joining || fetch(form.action.replace(/\/chat$/, "/chat/stream"), { method: "POST", body: new FormData(form), headers: { Accept: "text/event-stream" }, credentials: "same-origin" }))
      .then(function (res) {
        // The turn ended between the page and this: the page as it now is
        // has the reply.
        if (res.status === 204) { settle({}); refreshSoon(0); return; }
        if (!res.ok || !res.body) throw new Error("no stream");
        // Read through the page's connection (01-connect.js), a slip in
        // showing one event not losing the rest.
        return sw.stream(res, handle).then(function () { if (!settled) settle({}); });
      })
      .catch(function (err) {
        if (window.console) console.error("live turn:", err);
        // Once the server has heard the message the turn is under way and
        // must not be sent twice: show what arrived and stop. Before that,
        // the ordinary way, whole: the page comes back with the turn done.
        if (heard || joining) { if (!settled) settle({}); refreshSoon(0); return; }
        live.li.remove();
        form._sending = false;
        form.setAttribute("data-live", "off");
        form.removeAttribute("aria-busy");
        if (form.requestSubmit) form.requestSubmit(); else form.submit();
      });
  }

  function arm(form) {
    if (form._live) return;
    form._live = true;
    form.addEventListener("submit", function (e) {
      if (form.getAttribute("data-live") === "off") return;
      e.preventDefault();
      if (form._sending) {
        // Sent while the assistant is working: it waits its turn.
        form._queued = true;
        statusText(form, "Assistant is working; your next message goes when it is done");
        return;
      }
      form._sending = true;
      // The status enhancement hears this same submit and shows the busy
      // state, but only if the form is not busy yet when it looks: mark it
      // a tick later, for a page without that enhancement.
      setTimeout(function () { form.setAttribute("aria-busy", "true"); }, 0);
      send(form);
    });
  }
  // A turn already under way, for 19-live-join.js: joining is the request
  // for what it has done and does next.
  sw.followTurn = send;
  sw.arm("form.sw-compose", arm);
})();
