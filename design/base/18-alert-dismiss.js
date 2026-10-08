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
  sw.arm("[data-dismiss]", function (btn) {
    btn.addEventListener("click", function () {
      var alert = this.closest(".sw-outcome") || this.closest(".sw-alert");
      var had = alert && alert.contains(document.activeElement);
      // The outcome of an edit hands focus back to the Edit it came from.
      var from = alert && alert.getAttribute("data-outcome-for");
      var edit = from && document.querySelector('[data-edit-action="' + from + '"] [data-edit]');
      if (alert) alert.remove();
      var main = document.getElementById("main");
      if (had && edit) edit.focus(); else if (had && main) main.focus();
    });
  });
  sw.ready(function () {
    var outcome = document.getElementById("outcome");
    if (!outcome) return;
    setTimeout(function () {
      // Told where the person came back to (23-back.js), it keeps its place.
      if (document.querySelector(".sw-inline-form") || outcome.hasAttribute("data-outcome-told")) return;
      // Problems with a form are heard as their list, each leading to its field.
      (outcome.querySelector("[data-component=error-summary]") || outcome).focus();
    }, 50);
  });
})();
