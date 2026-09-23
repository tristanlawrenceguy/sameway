// A turn under way, reached from the page: stopped, followed from any
// page, and told when it is done to a person who looked away.
//
// Going elsewhere while the assistant works does not stop it: the turn
// runs on at the server. A page made while it runs, the one the person
// went to or the one they came back to, says the assistant is working and
// names the turn on its composer; this follows that turn from
// /chat/live, from its start to its reply, busy as the page that asked
// for it would be. See 14-live.js, which shows it.
(function () {
  "use strict";
  // Stop, beside the status while the turn runs: the turn ends where it
  // is, the reply says so, and what it did stays.
  window.swStopControl = function (form, turn) {
    var status = document.getElementById(form.getAttribute("data-busy-target"));
    if (!status || !turn) return null;
    var btn = document.createElement("button");
    btn.type = "button";
    btn.className = "sw-button sw-button--quiet sw-pressable sw-live__stop";
    btn.textContent = "Stop";
    btn.onclick = function () {
      btn.setAttribute("aria-disabled", "true");
      btn.textContent = "Stopping…";
      fetch(form.action.replace(/\/chat$/, "/chat/stop"), { method: "POST", body: new URLSearchParams({ turn: turn }), credentials: "same-origin" });
    };
    status.after(btn);
    return btn;
  };

  function join(form) {
    var turn = form.getAttribute("data-turn");
    if (!turn || !window.swFollowTurn || form._sending) return;
    form._sending = true;
    form.setAttribute("aria-busy", "true");
    form.querySelectorAll("button[type=submit]").forEach(function (b) { b.setAttribute("aria-disabled", "true"); });
    var region = form.closest("[data-region]");
    if (region) region.setAttribute("data-state", "working");
    if (document.title.indexOf("⏳ ") !== 0) document.title = "⏳ " + document.title;
    var from = form.querySelector('input[name="from"]');
    var url = form.action.replace(/\/chat$/, "/chat/live") + "?" + new URLSearchParams({ turn: turn, from: from ? from.value : "/" });
    window.swFollowTurn(form, fetch(url, { headers: { Accept: "text/event-stream" }, credentials: "same-origin" }));
  }
  // The end of a turn, for a person who looked away while it ran: the
  // system's notification with the reply's first words, and a mark on the
  // tab until they look again. Someone watching the page needs neither.
  // With no page left to follow the turn, the server tells it instead.
  document.addEventListener("sw:turn-done", function (e) {
    var d = e.detail || {};
    // A stream that broke with no reply has nothing to tell.
    if (!document.hidden || (!d.id && !d.text)) return;
    var failed = !!d.text, words = d.text || "";
    if (!failed && d.html) {
      var t = document.createElement("template");
      t.innerHTML = d.html;
      var body = t.content.querySelector(".sw-message__body");
      words = body ? body.textContent.replace(/\s+/g, " ").trim() : "";
    }
    if (words.length > 140) words = words.slice(0, 139) + "…";
    if (window.Notification && Notification.permission === "granted") {
      try {
        var n = new Notification(failed ? "The assistant could not finish" : "Assistant replied", { body: words, tag: "sameway-turn" });
        n.onclick = function () { window.focus(); if (d.id) location.hash = "msg-" + d.id; n.close(); };
      } catch (err) { /* a notification is a courtesy */ }
    }
    if (document.title.indexOf("✓ ") !== 0) document.title = "✓ " + document.title;
  });
  document.addEventListener("visibilitychange", function () {
    if (!document.hidden) document.title = document.title.replace(/^✓ /, "");
  });

  // Whether to notify is asked once, the first time something is sent:
  // the moment it is wanted, and a press, which browsers ask for.
  document.addEventListener("submit", function (e) {
    if (e.target.matches && e.target.matches("form.sw-compose") && window.Notification && Notification.permission === "default") {
      try { Notification.requestPermission(); } catch (err) { /* not here */ }
    }
  }, true);

  function init() { document.querySelectorAll("form.sw-compose[data-turn]").forEach(join); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init); else init();
})();
