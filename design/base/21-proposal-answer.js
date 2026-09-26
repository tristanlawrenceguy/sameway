// A proposal's answer is sent without leaving the page.
//
// The two buttons are forms that work on their own. With scripts, the
// answer is sent where the person is: what it did is said through the
// chat's status, the page follows the change (17-refresh.js), and focus
// goes to the message box, the next thing to do, since the button that
// had it is gone. A failure says why. Every pending question goes with
// the answer, here and on the server, so at most one is ever waiting. A
// question the assistant asks during a live turn arrives with its end.
(function () {
  "use strict";

  function say(words) {
    var status = document.getElementById("chat-status");
    var text = status && (status.querySelector(".sw-status__text") || status);
    if (text) text.textContent = words;
    var said = status && status.querySelector(".sw-status__said");
    if (said) said.textContent = "";
  }

  function answer(btn) {
    var form = btn.closest("form");
    var card = btn.closest(".sw-proposal");
    // One answer: while it is sent, the others wait, still in reach.
    if (card._answering) return;
    card._answering = true;
    var all = card.querySelectorAll("[type=submit]");
    all.forEach(function (b) { b.setAttribute("aria-disabled", "true"); });
    function again() { card._answering = false; all.forEach(function (b) { b.removeAttribute("aria-disabled"); }); }
    var kind = form.action.split("/").pop();
    say("Sending your answer…");
    fetch(form.action, { method: "POST", body: new FormData(form), credentials: "same-origin" })
      .then(function (res) { return res.text().then(function (html) { return { ok: res.ok, html: html }; }); })
      .then(function (r) {
        // The page the answer returns to says how it went.
        var doc = new DOMParser().parseFromString(r.html, "text/html");
        var outcome = doc.querySelector(".sw-outcome[data-outcome=failed]");
        if (!r.ok || outcome) {
          say(outcome ? outcome.textContent.replace(/\s+/g, " ").replace("×", "").trim() : "That answer did not go through.");
          again();
          return;
        }
        document.querySelectorAll(".sw-proposal").forEach(function (p) { p.remove(); });
        say(kind === "dismiss" ? "Left as it was." : "Done: " + btn.textContent.trim() + ".");
        if (window.swRefresh) window.swRefresh(0);
        var box = document.querySelector("form.sw-compose textarea");
        if (box) box.focus();
      })
      .catch(function () { say("That answer did not go through."); again(); });
  }

  // A proposal's two buttons are forms that work on their own; with
  // scripts the answer is sent without leaving the page.
  function proposeHandler() {
    var btns = document.querySelectorAll(".sw-proposal [type=\"submit\"]");
    if (!btns || btns.length === 0) return;
    btns.forEach(function (btn) {
      if (btn._proposeArmed) return;
      btn._proposeArmed = true;
      btn.addEventListener("click", function (e) { e.preventDefault(); answer(btn); });
    });
  }

  // A question asked during a live turn comes with its end: put it where
  // the questions go, arm it, and say it.
  document.addEventListener("sw:turn-done", function (e) {
    var d = e.detail || {};
    if (!d.proposals) return;
    var form = document.querySelector("form.sw-compose");
    if (!form) return;
    var have = document.querySelector(".sw-proposals");
    var t = document.createElement("template");
    t.innerHTML = d.proposals.trim();
    var fresh = t.content.firstElementChild;
    if (!fresh) return;
    if (have) have.replaceWith(fresh); else form.parentNode.insertBefore(fresh, form);
    proposeHandler();
    var ask = fresh.querySelector(".sw-proposal__summary");
    if (ask) say("The assistant is asking: " + ask.textContent.trim());
  });

  function init() { proposeHandler(); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
  // Parts refreshed during a live turn are armed too; armed ones say so.
  document.addEventListener("sw:refresh", init);
})();
