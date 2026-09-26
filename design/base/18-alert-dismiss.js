// Alerts that can be closed, and the outcome of the last action.
//
// The outcome (outcome.go) is where focus starts on the page an action
// returns to: a message already on the page when it loads is not read out
// by screen readers, so without focus a person heard nothing of "Changes
// saved" or "Not saved". An edit reopened to fix a refusal takes focus
// itself (16-drafts.js) and points its field at the outcome instead.
// Closing an alert puts focus back in the page, not nowhere.
(function () {
  "use strict";
  function arm() {
    document.querySelectorAll("[data-dismiss]").forEach(function (btn) {
      if (btn._dismissArmed) return;
      btn._dismissArmed = true;
      btn.addEventListener("click", function () {
        var alert = this.closest(".sw-outcome") || this.closest(".sw-alert");
        var had = alert && alert.contains(document.activeElement);
        if (alert) alert.remove();
        var main = document.getElementById("main");
        if (had && main) main.focus();
      });
    });
  }
  function start() {
    arm();
    var outcome = document.getElementById("outcome");
    if (!outcome) return;
    setTimeout(function () {
      if (document.querySelector(".sw-inline-form")) return;
      // Problems with a form are heard as their list, each leading to its field.
      (outcome.querySelector("[data-component=error-summary]") || outcome).focus();
    }, 50);
  }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", start); else start();
  document.addEventListener("sw:refresh", arm);
})();
