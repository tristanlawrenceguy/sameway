// A turn under way, reached from the page: stopped, or followed from
// any page.
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
  function init() { document.querySelectorAll("form.sw-compose[data-turn]").forEach(join); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init); else init();
})();
