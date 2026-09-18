// The turn as it happens.
//
// The composer is a form and works as one: post, wait, page. With scripts
// the same form posts to /chat/stream and reads the turn as it goes: the
// person's message appears as recorded, the reply's words come as the
// model says them, each tool shows as it starts and ticks as it lands, and
// each block the assistant makes or changes arrives on the canvas the
// moment it exists, one at a time with a breath between when several come
// at once, so there is time to take each in. If anything about the stream
// fails, the form is sent the ordinary way and the page comes back whole.
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

  // The log the messages live in, made when the conversation was empty.
  function logFor(form) {
    var body = form.closest(".sw-chat__body") || form.parentElement;
    var log = body.querySelector(".sw-chat__log");
    if (log) return log;
    var empty = body.querySelector(".sw-empty");
    log = el('<ol class="sw-plain sw-stack sw-chat__log" aria-label="Messages" tabindex="-1"></ol>');
    if (empty) empty.replaceWith(log); else body.insertBefore(log, form);
    return log;
  }

  function liveMessage(log) {
    var li = el('<li class="sw-live"><article class="sw-message sw-message--assistant" data-component="message" data-role="assistant" data-actor="assistant" aria-busy="true">' +
      '<p class="sw-message__meta"><span class="sw-message__author">Assistant</span></p>' +
      '<div class="sw-message__body"><p class="sw-live__text"></p><ol class="sw-plain sw-live__steps" aria-label="What the assistant is doing"></ol></div></article></li>');
    log.appendChild(li);
    return { li: li, text: li.querySelector(".sw-live__text"), steps: li.querySelector(".sw-live__steps") };
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
      if (have) have.replaceWith(node); else canvas.appendChild(node);
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

  function send(form) {
    var log = logFor(form);
    var live = liveMessage(log);
    var land = lander();
    var text = "";
    var settled = false, heard = false;
    function step(label) {
      live.steps.querySelectorAll('[data-state="running"]').forEach(function (s) { s.setAttribute("data-state", "done"); });
      live.steps.appendChild(el('<li class="sw-live__step" data-state="running"><span class="sw-live__dot" aria-hidden="true"></span>' + label.replace(/[&<>]/g, function (c) { return { "&": "&amp;", "<": "&lt;", ">": "&gt;" }[c]; }) + '</li>'));
    }
    function settle(d) {
      settled = true;
      form._sending = false;
      if (d.html) live.li.innerHTML = d.html; else live.li.remove();
      var status = document.getElementById("chat-status");
      if (status && d.status) status.outerHTML = d.status;
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
          var first = document.querySelector("a.sw-skip");
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
      var ta = form.querySelector("textarea");
      if (ta && !d.text) ta.value = "";
      var file = form.querySelector('input[type="file"]');
      if (file) file.value = "";
      log.scrollTop = log.scrollHeight;
    }
    function handle(msg) {
      var d = msg.data;
      heard = true;
      switch (msg.event) {
        case "said": if (d.html) live.li.before(el("<li>" + d.html + "</li>")); break;
        case "delta": text += d.text || ""; live.text.textContent = text; break;
        case "text": text = d.text || ""; live.text.textContent = text; break;
        case "tool": step(d.label || d.tool || "Working"); break;
        case "change":
          live.steps.querySelectorAll('[data-state="running"]').forEach(function (s) { s.setAttribute("data-state", "done"); });
          if (d.block) land(d);
          break;
        case "done": settle(d); break;
        case "error": settle(d); break;
      }
      log.scrollTop = log.scrollHeight;
    }
    var data = new FormData(form);
    fetch(form.action.replace(/\/chat$/, "/chat/stream"), { method: "POST", body: data, headers: { Accept: "text/event-stream" }, credentials: "same-origin" })
      .then(function (res) {
        if (!res.ok || !res.body) throw new Error("no stream");
        var reader = res.body.getReader(), decoder = new TextDecoder(), buffer = "";
        function pump() {
          return reader.read().then(function (r) {
            if (r.done) { if (!settled) settle({}); return; }
            buffer += decoder.decode(r.value, { stream: true });
            var frames = buffer.split("\n\n");
            buffer = frames.pop();
            frames.forEach(function (f) {
              if (!f.trim()) return;
              // A slip in showing one event must not lose the rest.
              try { handle(parse(f)); } catch (err) { if (window.console) console.error("live turn:", err); }
            });
            return pump();
          });
        }
        return pump();
      })
      .catch(function (err) {
        if (window.console) console.error("live turn:", err);
        // Once the server has heard the message the turn is under way and
        // must not be sent twice: show what arrived and stop. Before that,
        // the ordinary way, whole: the page comes back with the turn done.
        if (heard) { if (!settled) settle({}); return; }
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
      if (form._sending) return;
      form._sending = true;
      e.preventDefault();
      // The status enhancement hears this same submit and shows the busy
      // state, but only if the form is not busy yet when it looks: mark it
      // a tick later, for a page without that enhancement.
      setTimeout(function () { form.setAttribute("aria-busy", "true"); }, 0);
      send(form);
    });
  }
  function init() { document.querySelectorAll("form.sw-compose").forEach(arm); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init); else init();
})();
